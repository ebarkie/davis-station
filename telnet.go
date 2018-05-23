// Copyright (c) 2016 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"text/template"
	"time"

	"github.com/ebarkie/davis-station/internal/textcmd"
	"github.com/ebarkie/telnet"
	"github.com/ebarkie/telnet/option"
)

// Telnet server for accessing weather station data old-school style
// or debugging.

const (
	nul = 0x00 // Null
	bs  = 0x08 // Backspace
	lf  = 0x0a // Line Feed
	cr  = 0x0d // Carriage return
	sp  = 0x20 // Space
	del = 0x7f // Delete
)

// telnetCtx is the telnet context.  It includes the serverCtx,
// parsed telnet templates, and the command rules.
type telnetCtx struct {
	serverCtx
	t  *template.Template
	sh textcmd.Shell
}

// telnetServer starts the telnet server.
func telnetServer(sc serverCtx, cfg config) {
	// Inherit generic server context so we have access to things like
	// archive records and loop packets.
	t := telnetCtx{serverCtx: sc}

	// Parse templates
	err := t.parseTemplates(cfg.res + "/tmpl/telnet/*.tmpl")
	if err != nil {
		Error.Fatalf("Telnet template parse error: %s", err.Error())
	}

	// Register shell commands
	t.sh.Register("(?:\x04|exit|logoff|logout|quit)", t.quit)
	t.sh.Register("(?:\\?|help)", t.help)
	t.sh.Register("(?:archive|trend)(?:[[:space:]]+([[:digit:]]+))*", t.archive)
	t.sh.Register("(?:cond|loop)", t.loop)
	t.sh.Register("(?:date|time)", t.time)
	t.sh.Register("health", t.health)
	t.sh.Register("(?:lamps)[[:space:]]+(off|on)", t.lamps)
	t.sh.Register("uname", t.uname)
	t.sh.Register("up(?:time)*", t.uptime)
	t.sh.Register("ver(?:s)*", t.ver)
	t.sh.Register("watch[[:space:]]+log[[:space:]]+(debug|trace)", t.log)
	t.sh.Register("(watch)[[:space:]]+(?:cond|loop)", t.loop)
	t.sh.Register("who[[:space:]]*am[[:space:]]*i", t.whoami)

	// Listen and accept new connections
	addr := net.JoinHostPort(cfg.addr, "8023")
	Info.Printf("Telnet server started on %s", addr)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		Error.Fatalf("Telnet server error: %s", err.Error())
	}

	for {
		conn, _ := l.Accept()
		go t.start(conn)
	}
}

// telnetConn is a Conn consisting of a TCPConn and a telnet ReadWriter.
type telnetConn struct {
	io.ReadWriter
	net.Conn
}

func (c telnetConn) Read(b []byte) (n int, err error) { return c.ReadWriter.Read(b) }
func (c telnetConn) Write(b []byte) (int, error)      { return c.ReadWriter.Write(b) }

// start starts a new telnet session.  It sets up the telnet ReadWriter,
// negotiates character mode, and then passes control to the prompt for the
// duration of the connection.
func (t telnetCtx) start(conn net.Conn) {
	defer conn.Close()
	defer Debug.Printf("Telnet connection from %s closed", conn.RemoteAddr())
	Debug.Printf("Telnet connection from %s opened", conn.RemoteAddr())

	// Create telnet Conn and negotiate character mode.
	echo := &option.Echo{}
	sga := &option.SGA{}
	tn := telnet.NewReadWriter(conn, echo, sga)

	tn.AskUs(sga, true)
	tn.AskUs(echo, true)

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var err error
	for err == nil {
		_, err = tn.Read([]byte{})
		if echo.Us {
			break
		}
	}
	conn.SetReadDeadline(time.Time{})

	if !echo.Us {
		tn.Write([]byte("Protocol negotiation failed, closing connection.\r\n"))
		return
	}

	t.prompt(telnetConn{ReadWriter: tn, Conn: conn})
}

// prompt is the shell or "main menu" prompt for the telnet Conn.
func (t telnetCtx) prompt(conn net.Conn) {
	// Welcome banner
	t.sh.Exec(conn, "uname")

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

		err = t.sh.Exec(conn, s)
		if err == textcmd.ErrCmdQuit {
			return
		}
		if err != nil {
			Warn.Printf("Telnet command error %s: %s: %s", conn.RemoteAddr(), s, err.Error())
			t.template(conn, "error",
				struct {
					Cmd string
					Err string
				}{s, err.Error()},
			)
		}
	}
}

// ansiEsc is a helper function for emitting an ANSI escape sequence.
func (telnetCtx) ansiEsc(s string) string {
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

	// The original 8 color specificationis used for maximum compatibility.
	// 256 color mode also works with the default Linux and Mac OS X
	// terminals.  24-bit mode does not work with Mac OS X.
	if v >= hot {
		return t.ansiEsc("31") // Red
	} else if v >= warm {
		return t.ansiEsc("33") // Yellow
	}

	if v <= cold {
		return t.ansiEsc("34") // Blue
	} else if v <= cool {
		return t.ansiEsc("36") // Cyan
	}

	return ""
}

// highlight returns the ANSI bold code if the value is non-zero or
// an empty string if it is zero.
func (t telnetCtx) highlight(v float64) string {
	if v != 0 {
		return t.ansiEsc("1") // Bold
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
// parses the templates from the files identified by the pattern
// for later execution.
func (t *telnetCtx) parseTemplates(p string) (err error) {
	fmap := template.FuncMap{
		"archiveTime": func(t time.Time) string {
			return t.Format("01/02 15:04")
		},
		"colorScale": t.colorScale,
		"degToDir":   t.degToDir,
		"highlight":  t.highlight,
		"int":        func(i int) int { return i },
		"longTime": func(t time.Time) string {
			return t.Format("Monday, January 2 2006 at 15:04:05")
		},
		"metar": metar,
		"noColor": func() string {
			return t.ansiEsc("0")
		},
		"sunTime": func(t time.Time) string {
			return t.Format("15:04")
		},
	}
	t.t, err = template.New("").Funcs(fmap).ParseGlob(p)

	return
}

// readOne reads one byte.
//
// If the byte read is a carriage return then the enter key was pressed
// so the newline that follows will be read and discarded.
func (t telnetCtx) readOne(r io.Reader) (byte, error) {
	buf := make([]byte, 1)
	_, err := r.Read(buf)
	if buf[0] == cr {
		t.readOne(r)
	}
	return buf[0], err
}

// readLine reads a line of data and returns it as a string.  It
// handles character mode operations including echoing and backspacing.
func (telnetCtx) readLine(rw io.ReadWriter) (s string, err error) {
	buf := make([]byte, 1024)
	var n, nt int
	for {
		n, err = rw.Read(buf)
		if err != nil {
			return
		}

		for i := 0; i < n; i++ {
			switch buf[i] {
			case nul, lf:
				// A null or newline indicates end of line
				rw.Write([]byte("\r\n"))
				return
			case cr:
				// Ignore carriage returns
			case bs, del:
				// Backspace (^H or ^?)
				if nt > 0 {
					rw.Write([]byte{bs, sp, bs})
					s = s[:len(s)-1]
					nt--
				}
			default:
				rw.Write([]byte{buf[i]}) // Echo
				s += string(buf[i])
				nt++
			}
		}
	}
}

// template executes the named template with the specified data
// and sends the output to the Conn.
func (t telnetCtx) template(w io.Writer, name string, data interface{}) {
	bw := bufio.NewWriter(w)
	err := t.t.ExecuteTemplate(bw, name, data)
	if err != nil {
		Error.Printf("Template %s error: %s", name, err.Error())
		fmt.Fprintf(bw, "Content not available.\r\n")
	}
	bw.Flush()
}
