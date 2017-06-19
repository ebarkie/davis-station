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
	debug := flag.Bool("debug", false, "enable debug mode")
	bindAddress := flag.String("bindaddress", "[::]", "server bind address")
	device := flag.String("device", "", "weather station device (REQUIRED)")
	dbFile := flag.String("database", "weather.db", "sqlite database file")
	flag.Parse()

	if *debug {
		Debug.addOutput(os.Stdout)
	}

	if len(*device) == 0 {
		flag.Usage()
	} else {
		Info.Print(banner)
		server(*bindAddress, *device, *dbFile)
	}
}
