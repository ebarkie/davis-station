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
* Telnet server for direct access to data and debugging the sever.

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

```
telnet wx 8023
Trying 192.168.1.254...
Connected to wx.
Escape character is '^]'.
Davis Instruments Weather station server (version ab06ac1) on 192.168.1.254:8023.

> help
Command	                Argument(s)     Description
----------------------- --------------- -----------------------------------
ARCHIVES, TREND         [h=2]           Show last h hours of observations
                                        at 5 minute intervals
COND, LOOP                              Show detailed latest observation
                                        information
DATE, TIME                              Show current date and time
DEBUG                                   Watch debug log messages
HELP                                    Show this help information
LOGOFF, LOGOUT, QUIT                    Gracefully close the connection
UNAME                                   Show server information
UPTIME                                  Show server uptime
VER, VERS                               Show server version
WHOAMI                                  Show client source IP address and
                                        port

> trend 1
Trend (5 minute interval):

Timestamp   Bar(in) Tem(F) Hum(%) Rn(in) Dir(D) Spd(mph) Gus(mph) Sol(wm2) UV(i)
----------- ------- ------ ------ ------ ------ -------- -------- -------- -----
06/11 10:45 30.218  83.00  63     0.00   293    1        3        811      4.0  
06/11 10:40 30.216  82.50  63     0.00   225    1        3        796      3.9  
06/11 10:35 30.216  82.20  68     0.00   203    1        2        769      3.7  
06/11 10:30 30.215  81.90  68     0.00   158    1        2        761      3.6  
06/11 10:25 30.215  82.00  71     0.00   158    0        2        755      3.5  
06/11 10:20 30.216  81.90  66     0.00   180    1        2        744      3.3  
06/11 10:15 30.217  81.20  73     0.00   270    1        2        735      3.2  
06/11 10:10 30.219  81.20  69     0.00   270    1        2        721      3.1  
06/11 10:05 30.219  80.70  73     0.00   270    1        1        703      2.9  
06/11 10:00 30.217  80.00  73     0.00   270    1        3        685      2.8  
06/11 09:55 30.217  80.10  70     0.00   248    1        3        667      2.7  
06/11 09:50 30.217  80.40  72     0.00   248    1        2        653      2.5  
----------- ------- ------ ------ ------ ------ -------- -------- -------- -----

> quit
Bye!
Connection closed by foreign host.
```

## License

Copyright (c) 2016-2017 Eric Barkie. All rights reserved.  
Use of this source code is governed by the MIT license
that can be found in the [LICENSE](LICENSE) file.
