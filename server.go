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

// Loop is a weatherlink.Loop with a sequence and timestamp
// added in.
type Loop struct {
	Seq       int64     `json:"sequence"`
	Timestamp time.Time `json:"timestamp"`
	weatherlink.Loop
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

	// HTTP server
	go httpServer(bindAddress, sc)

	// Telnet server
	go telnetServer(bindAddress, sc)

	// Events server
	go eventsServer(sc.eb)

	// Open and setup events channel for weather station
	var ec chan interface{}

	// If a device name of "/dev/null" is specified launch
	// a primitive test server instead of attaching to the
	// Weatherlink.
	if weatherStationAddress == "/dev/null" {
		Info.Print("Test poller started")

		ec = make(chan interface{})

		// Send a mostly empty loop packet, except for a few
		// things initialized so it passes QC,  every 2s.
		l := weatherlink.Loop{}
		l.Bar.Altimeter = 6.8 // QC minimums
		l.Bar.SeaLevel = 25.0
		l.Bar.Station = 6.8

		go func() {
			for {
				ec <- l
				time.Sleep(2 * time.Second)
			}
		}()
	} else {
		Info.Print("Weatherlink poller started")

		// Connect the weatherlink loggers
		weatherlink.Trace.SetOutput(Trace)
		weatherlink.Debug.SetOutput(Debug)
		weatherlink.Info.SetOutput(Info)
		weatherlink.Warn.SetOutput(Warn)
		weatherlink.Error.SetOutput(Error)

		// Open connection and start command broker
		wl, err := weatherlink.Dial(weatherStationAddress)
		if err != nil {
			Error.Fatal(err)
		}
		defer wl.Close()
		wl.LastDmpTime = ad.Last()
		ec = wl.Start()
	}

	// Receive events forever
	var seq int64
	for e := range ec {
		switch e.(type) {
		case weatherlink.Archive:
			a := e.(weatherlink.Archive)

			// Add record to archive database
			err := ad.Add(a)
			if err != nil {
				Error.Printf("Unable to add archive record to database: %s", err.Error())
			}

			// Update events broker
			sc.eb.publish(event{event: "archive", data: a})
		case weatherlink.Loop:
			// Create Loop with sequence and timestamp
			l := Loop{}
			l.Timestamp = time.Now()
			l.Seq = seq
			l.Loop = e.(weatherlink.Loop)

			// Quality control validity check
			qc := validityCheck(l)
			if !qc.passed {
				// Log and ignore bad packets
				Error.Printf("QC %s", qc.errs)
				continue
			}

			// Update loop buffer
			sc.lb.add(l)

			// Publish to events broker
			sc.eb.publish(event{event: "loop", data: l})

			// Increment loop sequence
			seq++
		default:
			Warn.Printf("Unhandled event type: %T", e)
		}
	}
	Error.Fatal("Weatherlink command broker died")
}
