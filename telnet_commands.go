// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

func (c telnetCtx) archive(h uint) {
	d := time.Duration(h) * time.Hour
	ac := c.ad.NewGet(time.Now().Add(-d), time.Now())
	c.tmpl("archive", ac)
}

func (c telnetCtx) commandPrompt() {
	defer c.conn.Close()
	defer Debug.Printf("Telnet connection from %s closed", c.conn.RemoteAddr())

	Debug.Printf("Telnet connection from %s opened", c.conn.RemoteAddr())
	c.uname()

	for {
		c.tmpl("prompt", nil)
		in, err := c.readLine()
		if err != nil {
			// Client closed the connection
			return
		}

		tokens := strings.Split(strings.TrimSpace(in), " ")
		cmd := tokens[0]
		args := tokens[1:]
		Debug.Printf("Telnet command from %s: %s%s", c.conn.RemoteAddr(), cmd, args)

		switch strings.ToUpper(cmd) {
		case "":
			// NOOP
		case "?", "HELP":
			c.help()
		case "ARCHIVE", "TREND":
			if len(args) != 1 {
				c.archive(2)
			} else if h, _ := strconv.Atoi(args[0]); h > 0 {
				c.archive(uint(h))
			} else {
				fmt.Fprintf(c.conn, "%s: invalid range %s\n", cmd, args[0])
			}
		case "COND", "LOOP":
			c.loop(false)
		case "DATE", "TIME":
			c.time()
		case "LOGOFF", "LOGOUT", "QUIT", "\x04":
			c.quit()
			return
		case "UNAME":
			c.uname()
		case "UPTIME":
			c.uptime()
		case "VER", "VERS":
			c.ver()
		case "WATCH":
			if len(args) == 1 {
				switch strings.ToUpper(args[0]) {
				case "COND", "LOOP":
					c.loop(true)
				case "DEBUG":
					c.debug()
				default:
					fmt.Fprintf(c.conn, "%s: invalid item %s\n", cmd, args[0])
				}
			} else {
				fmt.Fprintf(c.conn, "%s: an item is required\n", cmd)
			}
		case "WHOAMI":
			c.whoami()
		default:
			fmt.Fprintf(c.conn, "%s: command not found.\n", cmd)
		}
	}
}

func (c telnetCtx) debug() {
	fmt.Fprintf(c.conn, "Watching log at debug level.  Press <ENTER> to end.\n\n")
	debugLoggers := []*logger{Debug, Info, Warn, Error}
	for _, l := range debugLoggers {
		l.addOutput(c.conn)
	}
	defer func() {
		for _, l := range debugLoggers {
			l.removeOutput(c.conn)
		}
	}()

	c.readLine()
	fmt.Fprintln(c.conn, "Watch log ended.")
}

func (c telnetCtx) help() {
	c.tmpl("help", nil)
}

func (c telnetCtx) loop(watch bool) {
	l := c.lb.loops()
	if len(l) > 0 {
		c.tmpl("loop", l[0])
	}

	if watch {
		inEvents := c.eb.subscribe(c.conn.RemoteAddr().String())
		defer c.eb.unsubscribe(inEvents)

		go func() {
			for e := range inEvents {
				if e.event == "loop" {
					c.tmpl("loop", e.data)
					fmt.Fprintf(c.conn, "\nWatching conditions.  Press <ENTER> to end.")
				}
			}
		}()

		c.readLine()
	}
}

func (c telnetCtx) quit() {
	c.tmpl("quit", nil)
}

func (c telnetCtx) time() {
	c.tmpl("time",
		struct {
			Time time.Time
		}{
			time.Now(),
		},
	)
}

func (c telnetCtx) uname() {
	c.tmpl("uname",
		struct {
			Banner    string
			LocalAddr net.Addr
		}{
			banner,
			c.conn.LocalAddr(),
		},
	)
}

func (c telnetCtx) uptime() {
	c.tmpl("uptime",
		struct {
			Uptime    time.Duration
			StartTime time.Time
		}{
			time.Since(c.startTime),
			c.startTime,
		},
	)
}

func (c telnetCtx) ver() {
	c.tmpl("ver",
		struct {
			Version string
		}{
			version,
		},
	)
}

func (c telnetCtx) whoami() {
	c.tmpl("whoami",
		struct {
			RemoteAddr net.Addr
		}{
			c.conn.RemoteAddr(),
		},
	)
}
