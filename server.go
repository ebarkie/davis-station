// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"errors"
	"time"

	"github.com/ebarkie/davis-station/internal/events"
)

const (
	loopsMin     = 3               // Minimum number of samples received before responding
	loopsMax     = 2 * 135         // Store up to about 10 minutes of loop sample history
	loopStaleAge = 5 * time.Minute // Stop responding if most recent loop sample was > 5 minutes
)

// Errors.
var (
	ErrLoopsAge = errors.New("Samples are too old")
	ErrLoopsMin = errors.New("Not enough samples yet")
)

// serverCtx contains a shared context that is made available to
// the HTTP endpoint handlers and telnet connections.
type serverCtx struct {
	ad        *ArchiveData
	lb        *loopBuffer
	eb        *events.Broker
	startTime time.Time
}

func server(bindAddr string, device string, dbFile string) {
	ad, err := OpenArchive(dbFile)
	if err != nil {
		Error.Fatalf("Unable to open archive file %s: %s", dbFile, err.Error())
	}
	defer ad.Close()
	sc := serverCtx{
		ad:        &ad,
		lb:        &loopBuffer{},
		eb:        events.New(),
		startTime: time.Now(),
	}

	// HTTP server
	go httpServer(sc, bindAddr)

	// Telnet server
	go telnetServer(sc, bindAddr)

	// Weather station server
	go stationServer(sc, device)

	select {}
}
