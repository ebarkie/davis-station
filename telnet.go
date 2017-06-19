// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"bufio"
	"fmt"
	"net"
	"text/template"
	"time"
)

// Telnet server for accessing weather station data old-school style
// or debugging.

// telnetCtx is the telnet context.  It includes the serverCtx,
// the parsed telnet templates, and a reference to the open
// telnet Conn.
type telnetCtx struct {
	serverCtx
	templates *template.Template
	conn      net.Conn
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

// readLine reads a line of data from the telnet Conn.
//
// The telnet server does not run in RFC 1184 linemode so
// data is not received until the remote sends a CR.
func (c telnetCtx) readLine() (string, error) {
	r := bufio.NewReader(c.conn)
	return r.ReadString('\n')
}

// tmpl executes the named template with the specified data
// and sends the output to the telnet Conn.
func (c telnetCtx) tmpl(name string, data interface{}) {
	err := c.templates.ExecuteTemplate(c.conn, name, data)
	if err != nil {
		Error.Printf("Template %s error: %s", name, err.Error())
		fmt.Fprintln(c.conn, "Content not available.")
	}
}

// telnetServer starts the telnet server.  It's blocking and should be called as
// a goroutine.
func telnetServer(bindAddress string, sc serverCtx) {
	// Parse templates.
	fmap := template.FuncMap{
		"degToDir": telnetCtx{}.degToDir,
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
	t, err := template.New("").Funcs(fmap).ParseGlob("tmpl/telnet/*.tmpl")
	if err != nil {
		Error.Fatalf("Telnet template parse error: %s", err.Error())
	}

	// Start listening.
	address := bindAddress + ":8023"
	Info.Printf("Telnet server started on %s", address)
	l, err := net.Listen("tcp", address)
	if err != nil {
		Error.Fatalf("Telnet server error: %s", err.Error())
	}

	// Accept new connections.
	for {
		c := telnetCtx{
			serverCtx: sc,
			templates: t,
		}
		c.conn, _ = l.Accept()
		go c.commandPrompt()
	}
}
