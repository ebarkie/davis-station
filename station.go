// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"time"

	"github.com/ebarkie/davis-station/internal/events"
	"github.com/ebarkie/weatherlink"
)

// Loop is a weatherlink.Loop with a sequence and timestamp
// added in.
type Loop struct {
	Seq       int64     `json:"sequence"`
	Timestamp time.Time `json:"timestamp"`
	weatherlink.Loop
}

func stationEvents(sc serverCtx, device string) (ec chan interface{}, err error) {
	// If a device name of "/dev/null" is specified launch
	// a primitive test server instead of attaching to the
	// Weatherlink.
	switch device {
	case "/dev/null":
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
	default:
		Info.Print("Weatherlink poller started")

		// Connect the weatherlink loggers
		weatherlink.Trace.SetOutput(Trace)
		weatherlink.Debug.SetOutput(Debug)
		weatherlink.Info.SetOutput(Info)
		weatherlink.Warn.SetOutput(Warn)
		weatherlink.Error.SetOutput(Error)

		// Open connection and start command broker
		var wl weatherlink.Weatherlink
		wl, err = weatherlink.Dial(device)
		if err != nil {
			return
		}
		defer wl.Close()
		wl.LastDmpTime = sc.ad.Last()
		ec = wl.Start()
	}

	return
}

func stationServer(sc serverCtx, device string) error {
	// Open and setup events channel for weather station
	ec, err := stationEvents(sc, device)
	if err != nil {
		Error.Fatalf("Weatherlink command broker failed to start: %s", err.Error())
		return err
	}

	// Receive events forever
	var seq int64
	for e := range ec {
		switch e.(type) {
		case weatherlink.Archive:
			a := e.(weatherlink.Archive)

			// Add record to archive database
			err := sc.ad.Add(a)
			if err != nil {
				Error.Printf("Unable to add archive record to database: %s", err.Error())
			}

			// Update events broker
			sc.eb.Publish(events.Event{Event: "archive", Data: a})
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
			sc.eb.Publish(events.Event{Event: "loop", Data: l})

			// Increment loop sequence - this intentionally only occurs
			// if it passed QC.
			seq++
		default:
			Warn.Printf("Unhandled event type: %T", e)
		}
	}

	Error.Fatal("Weatherlink command broker unexpectedly exited")
	return nil
}
