// Copyright (c) 2016 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"errors"
	"time"

	"github.com/ebarkie/davis-station/internal/archive"
	"github.com/ebarkie/davis-station/internal/events"
	"github.com/ebarkie/weatherlink"
)

const (
	loopsMin     = 3               // Minimum number of samples received before responding
	loopsMax     = 2 * 135         // Store up to about 10 minutes of loop sample history
	loopStaleAge = 5 * time.Minute // Stop responding if most recent loop sample was > 5 minutes
)

// Errors.
var (
	errLoopsAge = errors.New("samples are too old")
	errLoopsMin = errors.New("not enough samples yet")
)

// serverCtx contains a shared context that is made available to
// the HTTP endpoint handlers and telnet connections.
type serverCtx struct {
	ar *archive.Records
	lb *loopBuffer
	eb *events.Broker
	wl *weatherlink.Conn

	firmTime  time.Time
	firmVer   string
	startTime time.Time
}

func server(cfg config) {
	// Open archive database
	ar, err := archive.Open(cfg.db)
	if err != nil {
		Error.Fatalf("Unable to open archive file %s: %s", cfg.db, err.Error())
	}
	defer ar.Close()

	// Open weather station
	wl, err := stationOpen(cfg.dev)
	if err != nil {
		Error.Fatalf("Unable to open Weatherlink: %s", err.Error())
	}

	sc := serverCtx{
		ar:        &ar,
		lb:        &loopBuffer{},
		eb:        events.New(),
		wl:        &wl,
		startTime: time.Now(),
	}

	// Query firmware information
	if t, err := sc.wl.GetFirmTime(); err == nil {
		sc.firmTime = time.Time(t)
	} else {
		Warn.Printf("Unable to get firmware build time: %s", err.Error())
	}
	if v, err := sc.wl.GetFirmVer(); err == nil {
		sc.firmVer = string(v)
	} else {
		Warn.Printf("Unable to get firmware version: %s", err.Error())
	}

	// Start weather station events handler
	go stationEvents(sc)

	// Start HTTP server
	go httpServer(sc, cfg)

	// Start Telnet server
	go telnetServer(sc, cfg)

	select {}
}
