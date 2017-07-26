// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"text/template"
	"time"
)

// Telnet server for accessing weather station data old-school style.
// Or it's useful for debugging.

// telnetCtx is the telnet context.  It includes the serverCtx,
// parsed telnet templates, and the command rules.
type telnetCtx struct {
	serverCtx
	t  *template.Template
	tc TextCmds
}

// commandPrompt is the "main menu" for the telnet Conn.
func (t telnetCtx) commandPrompt(conn net.Conn) {
	defer conn.Close()
	defer Debug.Printf("Telnet connection from %s closed", conn.RemoteAddr())

	Debug.Printf("Telnet connection from %s opened", conn.RemoteAddr())
	// Welcome banner
	t.tc.Exec(conn, "uname")

	// Loop forever until connection is closed or a command returns
	// an ErrCmdQuit error.
	for {
		t.template(conn, "prompt", nil)
		s, err := t.readLine(conn)
		if err != nil {
			// Client closed the connection
			return
		}
		Debug.Printf("Telnet command from %s: %s", conn.RemoteAddr(), s)

		err = t.tc.Exec(conn, s)
		if err == ErrCmdQuit {
			return
		}
		if err != nil {
			Warn.Printf("Telnet command error %s: %s: %s", conn.RemoteAddr(), s, err.Error())
			fmt.Fprintf(conn, "%s: %s.\n", s, err.Error())
		}
	}
}

// ansi is a helper function for emitting an ANSI escape sequence.
func (telnetCtx) ansi(s string) string {
	return "\x1b[" + s + "m"
}

// colorScale returns an ANSI color code based on the passed value and
// scale.  The mapping looks like:
//
// Cold<= Cooler<=     -     >=Warm   >=Hot
// Blue   Cyan     <default> Yellow   Red
func (t telnetCtx) colorScale(i interface{}, cold, cool, warm, hot float64) string {
	var v float64
	switch i.(type) {
	case int:
		v = float64(i.(int))
	case float64:
		v = i.(float64)
	default:
		return ""
	}

	// The original 8 color specification code is used for maximum
	// compatibility.  256 color mode also works under Linux and Mac
	// OS X Terminal.  24-bit mode does not work under Mac OS X Terminal.
	if v >= hot {
		return t.ansi("31") // Red
	} else if v >= warm {
		return t.ansi("33") // Yellow
	}

	if v <= cold {
		return t.ansi("34") // Blue
	} else if v <= cool {
		return t.ansi("36") // Cyan
	}

	return ""
}

// highlight returns the ANSI bold code if the value is non-zero or
// an empty string if it is zero.
func (t telnetCtx) highlight(v float64) string {
	if v != 0 {
		return t.ansi("1") // Bold
	}

	return ""
}

// degToDir takes a direction in degrees and returns a human friendly
// direction as a string.
func (telnetCtx) degToDir(deg int) string {
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

// parseTemplates sets up all the telnet template functions and
// parses the templates for later execution.
func (t *telnetCtx) parseTemplates() (err error) {
	fmap := template.FuncMap{
		"degToDir":   t.degToDir,
		"colorScale": t.colorScale,
		"highlight":  t.highlight,
		"noColor": func() string {
			return t.ansi("0")
		},
		"archiveTime": func(t time.Time) string {
			return t.Format("01/02 15:04")
		},
		"longTime": func(t time.Time) string {
			return t.Format("Monday, January 2 2006 at 15:04:05")
		},
		"sunTime": func(t time.Time) string {
			return t.Format("15:04")
		},
	}
	t.t, err = template.New("").Funcs(fmap).ParseGlob("tmpl/telnet/*.tmpl")

	return
}

// readLine reads a line of data from the Conn and returns it
// space trimmed.
//
// This telnet server does not run in RFC 1184 linemode so
// data is not received until the remote sends a CR.
func (telnetCtx) readLine(r io.Reader) (s string, err error) {
	br := bufio.NewReader(r)
	s, err = br.ReadString('\n')
	if err == nil {
		s = strings.TrimSpace(s)
	}

	return
}

// template executes the named template with the specified data
// and sends the output to the Conn.
func (t telnetCtx) template(w io.Writer, name string, data interface{}) {
	err := t.t.ExecuteTemplate(w, name, data)
	if err != nil {
		Error.Printf("Template %s error: %s", name, err.Error())
		fmt.Fprintln(w, "Content not available.")
	}
}

// telnetServer starts the telnet server.  It's blocking and should be called as
// a goroutine.
func telnetServer(bindAddress string, sc serverCtx) {
	// Inherit generic server context so we have access to things like
	// archive records and loop packets.
	t := telnetCtx{serverCtx: sc}

	// Parse templates
	err := t.parseTemplates()
	if err != nil {
		Error.Fatalf("Telnet template parse error: %s", err.Error())
	}

	// Register commands
	t.tc.Register("(?:archive|trend)(?:[[:space:]]+([[:digit:]]+))*", t.archive)
	t.tc.Register("(?:cond|loop)", t.loop)
	t.tc.Register("(?:\\?|help)", t.help)
	t.tc.Register("(?:\x04|exit|logoff|logout|quit)", t.quit)
	t.tc.Register("(?:date|time)", t.time)
	t.tc.Register("uname", t.uname)
	t.tc.Register("up(?:time)*", t.uptime)
	t.tc.Register("ver(?:s)*", t.ver)
	t.tc.Register("watch[[:space:]]+debug", t.debug)
	t.tc.Register("(watch)[[:space:]]+(?:cond|loop)", t.loop)
	t.tc.Register("who[[:space:]]*am[[:space:]]*i", t.whoami)

	// Listen and accept new connections
	address := bindAddress + ":8023"
	Info.Printf("Telnet server started on %s", address)
	l, err := net.Listen("tcp", address)
	if err != nil {
		Error.Fatalf("Telnet server error: %s", err.Error())
	}

	for {
		conn, _ := l.Accept()
		go t.commandPrompt(conn)
	}
}
