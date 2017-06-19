// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"text/template"
	"time"
)

// Telnet server for accessing weather station data old-school style,
// and additional debugging capabilities.

type telnetContext struct {
	serverContext
	templates *template.Template
}

func (c telnetContext) archive(conn io.Writer, h uint) {
	d := time.Duration(h) * time.Hour
	ac := c.ad.NewGet(time.Now().Add(-d), time.Now())
	c.tmpl(conn, "archive", ac)
}

func (c telnetContext) commandPrompt(conn net.Conn) {
	defer conn.Close()

	Debug.Printf("Telnet connection from %s opened", conn.RemoteAddr())
	c.uname(conn)

commandLoop:
	for {
		c.tmpl(conn, "prompt", nil)
		in, err := c.readLine(conn)
		if err != nil {
			// Client closed the connection
			break
		}

		tokens := strings.Split(strings.TrimSpace(in), " ")
		cmd := tokens[0]
		args := tokens[1:]
		Debug.Printf("Telnet command from %s: %s%s", conn.RemoteAddr(), cmd, args)

		switch strings.ToUpper(cmd) {
		case "":
			// NOOP
		case "?", "HELP":
			c.help(conn)
		case "ARCHIVE", "TREND":
			if len(args) != 1 {
				c.archive(conn, 2)
			} else if h, _ := strconv.Atoi(args[0]); h > 0 {
				c.archive(conn, uint(h))
			} else {
				fmt.Fprintf(conn, "%s: invalid range %s\n", cmd, args[0])
			}
		case "COND", "LOOP":
			c.loop(conn, false)
		case "DATE", "TIME":
			c.time(conn)
		case "LOGOFF", "LOGOUT", "QUIT", "\x04":
			c.quit(conn)
			break commandLoop
		case "UNAME":
			c.uname(conn)
		case "UPTIME":
			c.uptime(conn)
		case "VER", "VERS":
			c.ver(conn)
		case "WATCH":
			if len(args) == 1 {
				switch strings.ToUpper(args[0]) {
				case "COND", "LOOP":
					c.loop(conn, true)
				case "DEBUG":
					c.debug(conn)
				default:
					fmt.Fprintf(conn, "%s: invalid item %s\n", cmd, args[0])
				}
			} else {
				fmt.Fprintf(conn, "%s: an item is required\n", cmd)
			}
		case "WHOAMI":
			c.whoami(conn)
		default:
			fmt.Fprintf(conn, "%s: command not found.\n", cmd)
		}
	}

	Debug.Printf("Telnet connection from %s closed", conn.RemoteAddr())
}

func (c telnetContext) debug(conn io.ReadWriter) {
	fmt.Fprintf(conn, "Watching log at debug level.  Press <ENTER> to end.\n\n")
	debugLoggers := []*logger{Debug, Info, Warn, Error}
	for _, l := range debugLoggers {
		l.addOutput(conn)
	}
	defer func() {
		for _, l := range debugLoggers {
			l.removeOutput(conn)
		}
	}()

	c.readLine(conn)
	fmt.Fprintln(conn, "Watch log ended.")
}

func (c telnetContext) help(conn io.Writer) {
	c.tmpl(conn, "help", nil)
}

func (c telnetContext) loop(conn net.Conn, watch bool) {
	l := c.lb.loops()
	if len(l) > 0 {
		c.tmpl(conn, "loop", l[0])
	}

	if watch {
		inEvents := c.eb.subscribe(conn.RemoteAddr().String())
		defer c.eb.unsubscribe(inEvents)

		go func() {
			for e := range inEvents {
				if e.event == "loop" {
					c.tmpl(conn, "loop", e.data)
					fmt.Fprintf(conn, "\nWatching conditions.  Press <ENTER> to end.")
				}
			}
		}()

		c.readLine(conn)
	}
}

func (c telnetContext) quit(conn io.Writer) {
	c.tmpl(conn, "quit", nil)
}

func (c telnetContext) time(conn io.Writer) {
	c.tmpl(conn, "time",
		struct {
			Time time.Time
		}{
			time.Now(),
		},
	)
}

func (c telnetContext) uname(conn net.Conn) {
	c.tmpl(conn, "uname",
		struct {
			Banner    string
			LocalAddr net.Addr
		}{
			banner,
			conn.LocalAddr(),
		},
	)
}

func (c telnetContext) uptime(conn io.Writer) {
	c.tmpl(conn, "uptime",
		struct {
			Uptime    time.Duration
			StartTime time.Time
		}{
			time.Since(c.startTime),
			c.startTime,
		},
	)
}

func (c telnetContext) ver(conn io.Writer) {
	c.tmpl(conn, "ver",
		struct {
			Version string
		}{
			version,
		},
	)
}

func (c telnetContext) whoami(conn net.Conn) {
	c.tmpl(conn, "whoami",
		struct {
			RemoteAddr net.Addr
		}{
			conn.RemoteAddr(),
		},
	)
}

// degToDir converts a direction to a string.
func (telnetContext) degToDir(deg int) string {
	var dirs = []string{
		"N", "NNE", "NE",
		"ENE", "E", "ESE",
		"SE", "SSE", "S", "SSW", "SW",
		"WSW", "W", "WNW", "NW", "NNW"}

	if deg < 0 {
		deg = 0
	}
	i := uint((float32(deg)/22.5)+0.5) % 16

	return dirs[i]
}

func (telnetContext) readLine(conn io.Reader) (string, error) {
	r := bufio.NewReader(conn)
	return r.ReadString('\n')
}

func (c telnetContext) tmpl(conn io.Writer, name string, data interface{}) {
	err := c.templates.ExecuteTemplate(conn, name, data)
	if err != nil {
		Error.Printf("Telnet template %s is missing", name)
		fmt.Fprintln(conn, "Content not available.")
	}
}

// telnetServer starts the telnet server.  It's blocking and should be called as
// a goroutine.
func telnetServer(bindAddress string, sc serverContext) {
	tc := telnetContext{serverContext: sc}

	// Parse templates
	fmap := template.FuncMap{
		"degToDir": tc.degToDir,
		"sunTime": func(t time.Time) string {
			return t.Format("15:04")
		},
		"archiveTime": func(t time.Time) string {
			return t.Format("01/02 15:04")
		},
		"longTime": func(t time.Time) string {
			return t.Format("Monday, January 2 2006 at 15:04:05")
		},
	}
	if t, err := template.New("").Funcs(fmap).ParseGlob("tmpl/telnet/*.tmpl"); err == nil {
		tc.templates = t
	} else {
		Error.Fatalf("Telnet template parse error: %s", err.Error())
	}

	// Launch server
	address := bindAddress + ":8023"
	Info.Printf("Telnet server started on %s", address)
	l, err := net.Listen("tcp", address)
	if err != nil {
		Error.Fatalf("Telnet server error: %s", err.Error())
	}

	for {
		conn, _ := l.Accept()
		go tc.commandPrompt(conn)
	}
}
