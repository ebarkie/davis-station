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
	c.execTemplate(conn, "archive.tmpl", ac)
}

func (c telnetContext) commandPrompt(conn net.Conn) {
	defer conn.Close()

	Debug.Printf("Telnet connection from %s opened", conn.RemoteAddr())
	c.uname(conn)

commandLoop:
	for {
		fmt.Fprintf(conn, "\n> ")
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
				fmt.Fprintf(conn, "%s: %s invalid range\n", cmd, args[0])
			}
		case "COND", "LOOP":
			c.loop(conn)
		case "DATE", "TIME":
			fmt.Fprintln(conn, time.Now())
		case "DEBUG":
			c.debug(conn)
		case "LOGOFF", "LOGOUT", "QUIT", "\x04":
			fmt.Fprintln(conn, "Bye!")
			break commandLoop
		case "UNAME":
			c.uname(conn)
		case "UPTIME":
			c.uptime(conn)
		case "VER", "VERS":
			fmt.Fprintln(conn, version)
		case "WHOAMI":
			fmt.Fprintln(conn, conn.RemoteAddr())
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
	c.execTemplate(conn, "help.tmpl", nil)
}

func (c telnetContext) loop(conn io.Writer) {
	c.execTemplate(conn, "loop.tmpl", c.lb.loops())
}

func (telnetContext) readLine(conn io.Reader) (string, error) {
	r := bufio.NewReader(conn)
	return r.ReadString('\n')
}

func (c telnetContext) execTemplate(conn io.Writer, name string, data interface{}) {
	if t := c.templates.Lookup(name); t != nil {
		t.Execute(conn, data)
	} else {
		Error.Printf("Telnet template %s is missing", name)
		fmt.Fprintln(conn, "Content not available.")
	}
}

// templateTinyTime formats a time to string as 01/02 15:04.  It is useful when
// printing rows where width space is limited.
func (telnetContext) templateTinyTime(t time.Time) string {
	return fmt.Sprintf("%02d/%02d %02d:%02d", t.Month(), t.Day(), t.Hour(), t.Minute())
}

func (telnetContext) uname(conn net.Conn) {
	fmt.Fprintf(conn, "%s on %s.\n", banner, conn.LocalAddr())
}

func (c telnetContext) uptime(conn io.Writer) {
	fmt.Fprintf(conn, "%s (started %s)\n", time.Since(c.startTime), c.startTime)
}

// telnetServer starts the telnet server.  It's blocking and should be called as
// a goroutine.
func telnetServer(bindAddress string, sc serverContext) {
	tc := telnetContext{serverContext: sc}

	// Parse templates
	fmap := template.FuncMap{
		"tinyTimestamp": tc.templateTinyTime,
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
