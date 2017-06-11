# Davis Instruments weather station server

[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat)](http://choosealicense.com/licenses/mit/)
[![Build Status](https://travis-ci.org/ebarkie/davis-station.svg?branch=master)](https://travis-ci.org/ebarkie/davis-station)

This uses the weatherlink package to talk to a Davis weather
station over a Weatherlink IP, serial, or USB interface.

Features include:

* Console clock synchronization.
* Storing archive data in a Bolt database
* Pulling loop packets using HTTP GET requests.
* Pulling archive data using HTTP GET requests.
* Pushed archive and loop packets using HTTP Server-sent events (EventSource).
* All data is delivered in structured and easily parsable JSON.

## Installation

```
$ go get
$ go generate
$ go build
```

See [contrib](contrib) directory for sample systemd service.

## Usage

```
Usage of ./davis-station:
  -bindaddress string
        server bind address (default "[::]")
  -database string
        sqlite database file (default "weather.db")
  -debug
        enable debug mode
  -device string
        weather station device (REQUIRED)

$ ./davis-station -device /dev/ttyUSB0
```

See [swagger](http://petstore.swagger.io/?url=https://raw.githubusercontent.com/ebarkie/davis-station/master/doc/swagger.json) specification for endpoints.

## License

Copyright (c) 2016-2017 Eric Barkie. All rights reserved.  
Use of this source code is governed by the MIT license
that can be found in the [LICENSE](LICENSE) file.
