# Davis Instruments weather station server

[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat)](http://choosealicense.com/licenses/mit/)
[![Build Status](https://travis-ci.org/ebarkie/davis-station.svg?branch=master)](https://travis-ci.org/ebarkie/davis-station)

This uses the weatherlink package to create a modern Personal
Weather Station.

Features include:

* Console clock synchronization.
* Storing archive data in a Bolt database.
* Primitive Quality Control.
* Pulling loop packets using HTTP GET requests.
* Pulling archive data using HTTP GET requests.
* Pushed archive and loop packets using HTTP Server-sent events (EventSource).
* All data is delivered in structured and easily parsable JSON.
* Telnet server for direct access to data and debugging the sever.

## Installation

```
$ go get
$ go generate
$ go build
```

Refer to the [contrib](contrib) directory for a sample systemd service.

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

Refer to the [swagger](http://petstore.swagger.io/?url=https://raw.githubusercontent.com/ebarkie/davis-station/master/doc/swagger.json) specification for HTTP endpoint information.

```
$ telnet wx 8023
Trying 192.168.1.254...
Connected to wx.
Escape character is '^]'.
Davis Instruments Weather station server (version ab06ac1) on 192.168.1.254:8023.

> help
Command	                Argument(s)     Description
----------------------- --------------- -----------------------------------
ARCHIVE, TREND         [h=2]            Show last h hours of observations
                                        at 5 minute intervals
COND, LOOP                              Show detailed latest observation
                                        information
DATE, TIME                              Show current date and time
HELP                                    Show this help information
LOGOFF, LOGOUT, QUIT                    Gracefully close the connection
UNAME                                   Show server information
UPTIME                                  Show server uptime
VER, VERS                               Show server version
WATCH                   <cond|loop|     Continuously watch an item
                         debug>
WHOAMI                                  Show client source IP address and
                                        port

> trend 1
Trend (5 minute interval):

Timestamp   Bar(in) Tem(F) Hum(%) Rn(in) Wind/Gus(mph)  Sol(wm2) UV(i)
----------- ------- ------ ------ ------ -------------- -------- -----
06/18 23:15 29.938  78.30  84     0.00   E   at 2/6     0        0.0  
06/18 23:10 29.939  78.30  84     0.00   SE  at 1/7     0        0.0  
06/18 23:05 29.942  78.40  84     0.00   ESE at 1/6     0        0.0  
06/18 23:00 29.946  78.50  84     0.00   S   at 3/10    0        0.0  
06/18 22:55 29.946  78.60  83     0.00   ESE at 2/7     0        0.0  
06/18 22:50 29.941  78.70  83     0.00   SSE at 2/6     0        0.0  
06/18 22:45 29.945  78.80  83     0.00   SSE at 3/9     0        0.0  
06/18 22:40 29.944  79.00  83     0.00   WSW at 1/6     0        0.0  
06/18 22:35 29.946  79.10  82     0.00   ESE at 3/10    0        0.0  
06/18 22:30 29.941  79.30  82     0.00   ESE at 3/8     0        0.0  
06/18 22:25 29.939  79.30  82     0.00   SE  at 4/10    0        0.0  
06/18 22:20 29.939  79.30  83     0.00   SE  at 3/13    0        0.0  
----------- ------- ------ ------ ------ -------------- -------- -----

> loop
Weather conditions on Sunday, June 18 2017 at 23:17:04            Seq:     4440
                      Sunrise at 05:59, sunset at 20:34

   Forecast: Partly cloudy and cooler.

  Barometer: 29.518in Rising Slowly

Temperature: 78.30 °F     Humidity: 84 %      Dew Point: 73.00 °F
 Heat Index: 82.00 °F
 Wind Chill: 78.00 °F

  Solar Rad: 0   wm/2     UV Index: 0.0        ET Today: 0.1

 Rain Today: 0.10in           Rate: 0.00in/h

       Wind: 124° SE  at 8  mph
    Gusting: 157° SSE at 1  mph

> quit
Bye!
Connection closed by foreign host.
```

## License

Copyright (c) 2016-2017 Eric Barkie. All rights reserved.  
Use of this source code is governed by the MIT license
that can be found in the [LICENSE](LICENSE) file.
