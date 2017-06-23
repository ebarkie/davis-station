// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"time"

	"github.com/ebarkie/weatherlink"
)

const (
	loopsMin     = 3               // Minimum number of samples received before responding
	loopsMax     = 2 * 135         // Store up to about 10 minutes of loop sample history
	loopStaleAge = 5 * time.Minute // Stop responding if most recent loop sample was > 5 minutes
)

// serverCtx contains a shared context that is made available to
// the HTTP endpoint handlers and telnet connections.
type serverCtx struct {
	ad        *ArchiveData
	lb        *loopBuffer
	eb        *eventsBroker
	startTime time.Time
}

// WrappedLoop is a wrapper struct for a Loop packet so Update
// information can be added.
type WrappedLoop struct {
	Update Update           `json:"update"`
	Loop   weatherlink.Loop `json:"loop"`
}

// Update is a struct for representing the update timestamp
// and sequence of a loop sample.
type Update struct {
	Timestamp time.Time `json:"timestamp"`
	Seq       int64     `json:"sequence"`
}

func server(bindAddress string, weatherStationAddress string, dbFile string) {
	ad, err := OpenArchive(dbFile)
	if err != nil {
		Error.Fatalf("Unable to open archive file %s: %s", dbFile, err.Error())
	}
	defer ad.Close()
	sc := serverCtx{
		ad: &ad,
		lb: &loopBuffer{},
		eb: &eventsBroker{
			events: make(chan event, 8),
			subs:   make(map[chan event]string),
			sub:    make(chan eventsSub),
			unsub:  make(chan eventsSub),
		},
		startTime: time.Now(),
	}

	// The archive channel is buffered to the maximum records a Vantage
	// Pro 2 console can hold in memory.  This can speed up large
	// downloads which would otherwise be I/O bound by database writes.
	archive := make(chan weatherlink.Archive, 5*512)
	loops := make(chan weatherlink.Loop, 8)

	// HTTP server
	go httpServer(bindAddress, sc)

	// Telnet server
	go telnetServer(bindAddress, sc)

	// Events server
	go eventsServer(sc.eb)

	// Retrieve data from weather station
	go func() {
		// If a device name of "/dev/null" is specified launch
		// a primitive test server instead of attaching to the
		// Weatherlink.
		if weatherStationAddress == "/dev/null" {
			Info.Print("Test poller started")

			// Send a mostly empty loop packet every 2s but a few
			// things need to be initialized to pass QC checks.
			l := weatherlink.Loop{}
			l.Bar.Altimeter = 6.8 // QC minimums
			l.Bar.SeaLevel = 25.0
			l.Bar.Station = 6.8

			for {
				loops <- l
				time.Sleep(2 * time.Second)
			}
		} else {
			Info.Print("Weatherlink poller started")

			// Connect to weatherlink logging
			weatherlink.Trace.SetOutput(Trace)
			weatherlink.Debug.SetOutput(Debug)
			weatherlink.Info.SetOutput(Info)
			weatherlink.Warn.SetOutput(Warn)
			weatherlink.Error.SetOutput(Error)

			wl, err := weatherlink.Dial(weatherStationAddress)
			if err != nil {
				Error.Fatal(err)
			}
			defer wl.Close()
			wl.Archive = archive
			wl.LastDmpTime = ad.Last()
			wl.Loops = loops

			err = wl.Start()
			if err != nil {
				Error.Fatalf("Weatherlink command broker error: %s", err.Error())
			}
		}
	}()

	// Monitor archive channel for new records
	go func() {
		for {
			a := <-archive

			// Add to archive database
			err := ad.Add(a)
			if err != nil {
				Error.Printf("Unable to add archive record to database: %s", err.Error())
			}

			// Update Server-sent events broker
			sc.eb.publish(event{event: "archive", data: a})
		}
	}()

	// Monitor loop channel for new packets
	go func() {
		for seq := int64(0); true; seq++ {
			l := <-loops

			// Quality control validity check
			qc := validityCheck(l)
			if !qc.passed {
				// Log and ignore bad packets
				Error.Printf("QC %s", qc.errs)
				continue
			}

			// Wrap with sequence and timestamp
			wrappedLoop := WrappedLoop{}
			wrappedLoop.Update.Timestamp = time.Now()
			wrappedLoop.Update.Seq = seq
			wrappedLoop.Loop = l

			// Update loop buffer
			sc.lb.add(wrappedLoop)

			// Publish to Server-sent events broker
			sc.eb.publish(event{event: "loop", data: wrappedLoop})
		}
	}()

	// Block forever
	select {}
}
