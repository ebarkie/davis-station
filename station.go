// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"time"

	"github.com/ebarkie/davis-station/internal/events"
	"github.com/ebarkie/weatherlink"
	"github.com/ebarkie/weatherlink/data"
)

// Loop is a weatherlink.Loop with a sequence and timestamp
// added in.
type Loop struct {
	Seq       int64     `json:"sequence"`
	Timestamp time.Time `json:"timestamp"`
	data.Loop
}

func weatherlinkEvents(sc serverCtx, dev string) (<-chan interface{}, error) {
	Info.Println("Weatherlink poller started")

	// Connect the weatherlink loggers
	weatherlink.Trace.SetOutput(Trace)
	weatherlink.Debug.SetOutput(Debug)
	weatherlink.Info.SetOutput(Info)
	weatherlink.Warn.SetOutput(Warn)
	weatherlink.Error.SetOutput(Error)

	// Open connection and start command broker
	w, err := weatherlink.Dial(dev)
	if err != nil {
		return nil, err
	}

	w.LastDmp = sc.ad.Last()
	ec := w.Start(weatherlink.StdIdle)
	w.CmdQ <- weatherlink.GetDmps

	return ec, nil
}

func stationServer(sc serverCtx, cfg config) error {
	// Setup events channel for weather station.
	ec, err := weatherlinkEvents(sc, cfg.dev)
	if err != nil {
		Error.Fatalf("Weatherlink command broker failed to start: %s", err.Error())
		return err
	}

	// Receive events forever
	var seq int64
	for e := range ec {
		switch e.(type) {
		case data.Archive:
			a := e.(data.Archive)

			// Add record to archive database
			err := sc.ad.Add(a)
			if err != nil {
				Error.Printf("Unable to add archive record to database: %s", err.Error())
			}

			// Update events broker
			sc.eb.Publish(events.Event{Event: "archive", Data: a})
		case data.Loop:
			// Create Loop with sequence and timestamp
			l := Loop{}
			l.Timestamp = time.Now()
			l.Seq = seq
			l.Loop = e.(data.Loop)

			// Quality control validity check
			qc := validityCheck(l)
			if !qc.passed {
				// Log and ignore bad packets
				Error.Printf("QC %s", qc.errs)
				continue
			}

			// Update loop buffer
			sc.lb.Add(l)

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
