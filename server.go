// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"sync"
	"time"

	"github.com/ebarkie/weatherlink"
)

const (
	archiveMax   = 5 * 512         // Matches memory in Vantage Pro2 console
	loopsMin     = 3               // Minimum number of samples received before responding
	loopsMax     = 2 * 135         // Store up to about 10 minutes of loop sample history
	loopStaleAge = 5 * time.Minute // Stop responding if most recent loop sample was > 5 minutes
)

type loopData struct {
	loops        []WrappedLoop // Loops slice that gets served to HTTP GET requests
	sync.RWMutex               // Mutex to coordinate adding and reading loops
}

// serverContext contains a shared context that is made available to
// the HTTP endpoint handlers and telnet connections.
type serverContext struct {
	ad        *ArchiveData
	ld        *loopData
	eb        *eventBroker
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
	Sequence  int64     `json:"sequence"`
}

func server(bindAddress string, weatherStationAddress string, dbFile string) {
	ad, err := OpenArchive(dbFile)
	if err != nil {
		Error.Fatalf("Unable to open archive file %s: %s", dbFile, err.Error())
	}
	defer ad.Close()
	sc := serverContext{
		ad: &ad,
		ld: &loopData{},
		eb: &eventBroker{
			events: make(chan event, 8),
			subs:   make(map[chan event]string),
			sub:    make(chan eventSub),
			unsub:  make(chan eventSub),
		},
		startTime: time.Now(),
	}

	// Buffer archive channel to max all of the memory can be downloaded
	// quickly without waiting for database syncs.
	archive := make(chan weatherlink.Archive, archiveMax)
	loops := make(chan weatherlink.Loop, 8)

	// HTTP server
	go httpServer(bindAddress, sc)

	// Telnet server
	go telnetServer(bindAddress, sc)

	// Server-sent events broker
	go eventsBroker(sc.eb)

	// Retrieve data from weather station
	go func() {
		// If a device name of "/dev/null" is specified launch
		// a primitive test server instead of attaching to the
		// Weatherlink.
		if weatherStationAddress == "/dev/null" {
			Info.Print("Test weather station poller started")

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
			Info.Print("Weather station poller started")

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

			// Update HTTP server loop slice
			sc.ld.Lock()
			wrappedLoop := WrappedLoop{}
			wrappedLoop.Update.Timestamp = time.Now()
			wrappedLoop.Update.Sequence = seq
			wrappedLoop.Loop = l
			// Delete oldest sample at the end of the slice if we've reached
			// reached the max size for loops.
			if len(sc.ld.loops) >= loopsMax {
				sc.ld.loops = sc.ld.loops[0 : len(sc.ld.loops)-1]
			}
			// Add the latest sample to the front of the slice
			sc.ld.loops = append([]WrappedLoop{wrappedLoop}, sc.ld.loops...)
			sc.ld.Unlock()

			// Update Server-sent events broker
			sc.eb.publish(event{event: "loop", data: wrappedLoop})
		}
	}()

	// Block forever
	select {}
}
