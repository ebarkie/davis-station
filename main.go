// Copyright (c) 2016 Eric Barkie. All rights reserved.
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

type config struct {
	addr  string
	dev   string
	db    string
	res   string
	debug bool
	trace bool
}

func main() {
	var cfg config
	flag.StringVar(&cfg.addr, "addr", "[::]", "server bind address")
	flag.StringVar(&cfg.dev, "dev", "", "weather station device (REQUIRED)")
	flag.StringVar(&cfg.db, "db", "weather.db", "bolt database file")
	flag.StringVar(&cfg.res, "res", ".", "resources path")
	flag.BoolVar(&cfg.debug, "debug", false, "enable debug mode")
	flag.BoolVar(&cfg.trace, "trace", false, "enable trace mode")
	flag.Parse()

	switch {
	case cfg.trace:
		Trace.addOutput(os.Stdout)
		fallthrough
	case cfg.debug:
		Debug.addOutput(os.Stdout)
	}

	if cfg.dev == "" {
		flag.Usage()
		return
	}

	Info.Println(banner)
	server(cfg)
}
