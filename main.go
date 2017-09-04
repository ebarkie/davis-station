// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Davis Instruments weather station.
package main

//go:generate ./version.sh

import (
	"flag"
	"fmt"
	"os"
)

var banner = fmt.Sprintf("Davis Instruments weather station (version %s)", version)

func main() {
	bindAddress := flag.String("bindaddress", "[::]", "server bind address")
	dbFile := flag.String("database", "weather.db", "sqlite database file")
	debug := flag.Bool("debug", false, "enable debug mode")
	device := flag.String("device", "", "weather station device (REQUIRED)")
	trace := flag.Bool("trace", false, "enable trace mode")
	flag.Parse()

	switch {
	case *trace:
		Trace.addOutput(os.Stdout)
		fallthrough
	case *debug:
		Debug.addOutput(os.Stdout)
	}

	if len(*device) == 0 {
		flag.Usage()
	} else {
		Info.Println(banner)
		server(*bindAddress, *device, *dbFile)
	}
}
