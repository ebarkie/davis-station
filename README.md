# Davis Instruments weather station

[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat)](http://choosealicense.com/licenses/mit/)
[![Build Status](https://travis-ci.org/ebarkie/davis-station.svg?branch=master)](https://travis-ci.org/ebarkie/davis-station)

This uses the weatherlink package to create a modern Personal
Weather Station.

Features include:

* Console clock synchronization.
* Storing archive data in a [BoltDB](https://github.com/boltdb/bolt) database.
* Primitive Quality Control.
* Pulling loop packets using HTTP GET requests.
* Pulling archive data using HTTP GET requests.
* Pushed archive and loop packets using HTTP Server-sent events (EventSource).
* All data is delivered in structured and easily parsable JSON.
* Telnet server for direct access to data and debugging the sever.

## Building

### Binary from source

```bash
$ go get
$ go generate
$ go build
```

### Debian/Ubuntu packages

```bash
$ debuild -uc -us -b
```

## Usage

### Daemon

```
Usage of ./davis-station:
  -addr string
    	server bind address
  -db string
    	bolt database file (default "weather.db")
  -debug
    	enable debug mode
  -dev string
    	weather station device (REQUIRED)
  -res string
    	resources path (default ".")
  -trace
    	enable trace mode

$ ./davis-station -dev /dev/ttyUSB0
```

### HTTP

Refer to the [swagger](http://petstore.swagger.io/?url=https://raw.githubusercontent.com/ebarkie/davis-station/master/doc/swagger.json) specification for HTTP endpoint information.

### Telnet

![Telnet Session](doc/telnet.gif)

## License

Copyright (c) 2016-2018 Eric Barkie. All rights reserved.  
Use of this source code is governed by the MIT license
that can be found in the [LICENSE](LICENSE) file.
