// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/ebarkie/davis-station/internal/textcmd"
)

func (t telnetCtx) archive(c textcmd.Ctx) (err error) {
	// Default archive period is 2 hours
	h := 2
	if a := c.Arg(1); a != "" {
		h, err = strconv.Atoi(a)
		if err != nil {
			return
		}
	}

	d := time.Duration(h) * time.Hour
	ac := t.ad.NewGet(time.Now().Add(-d), time.Now())
	t.template(c.Writer(), "archive", ac)

	return
}

func (t telnetCtx) debug(c textcmd.Ctx) error {
	fmt.Fprintf(c.Writer(), "Watching log at debug level.  Press <ENTER> to end.\n\n")
	debugLoggers := []*logger{Debug, Info, Warn, Error}
	for _, l := range debugLoggers {
		l.addOutput(c.Writer())
	}
	defer func() {
		for _, l := range debugLoggers {
			l.removeOutput(c.Writer())
		}
	}()

	t.readLine(c.Reader())
	fmt.Fprintln(c.Writer(), "Watch log ended.")

	return nil
}

func (t telnetCtx) help(c textcmd.Ctx) error {
	t.template(c.Writer(), "help", nil)

	return nil
}

func (t telnetCtx) loop(c textcmd.Ctx) error {
	watch := false
	if a := c.Arg(1); a == "watch" {
		watch = true
	}

	l := t.lb.loops()
	if len(l) > 0 {
		t.template(c.Writer(), "loop", l[0])
	}

	if watch {
		inEvents := t.eb.Subscribe(c.RemoteAddr().String())
		defer t.eb.Unsubscribe(inEvents)

		go func() {
			for e := range inEvents {
				if e.Event == "loop" {
					t.template(c.Writer(), "loop", e.Data)
					fmt.Fprintf(c.Writer(), "\nWatching conditions.  Press <ENTER> to end.")
				}
			}
		}()

		t.readLine(c.Reader())
	}

	return nil
}

func (t telnetCtx) quit(c textcmd.Ctx) error {
	t.template(c.Writer(), "quit", nil)

	return textcmd.ErrCmdQuit
}

func (t telnetCtx) time(c textcmd.Ctx) error {
	t.template(c.Writer(), "time",
		struct {
			Time time.Time
		}{time.Now()},
	)

	return nil
}

func (t telnetCtx) uname(c textcmd.Ctx) error {
	t.template(c.Writer(), "uname",
		struct {
			Banner    string
			LocalAddr net.Addr
		}{banner, c.LocalAddr()},
	)

	return nil
}

func (t telnetCtx) uptime(c textcmd.Ctx) error {
	// Round uptime down to nearest second
	uptime := time.Since(t.startTime)
	uptime = uptime - (uptime % time.Second)

	t.template(c.Writer(), "uptime",
		struct {
			Uptime    time.Duration
			StartTime time.Time
		}{uptime, t.startTime},
	)

	return nil
}

func (t telnetCtx) ver(c textcmd.Ctx) error {
	t.template(c.Writer(), "ver",
		struct {
			Version string
		}{version},
	)

	return nil
}

func (t telnetCtx) whoami(c textcmd.Ctx) error {
	t.template(c.Writer(), "whoami",
		struct {
			RemoteAddr net.Addr
		}{
			c.RemoteAddr(),
		},
	)

	return nil
}
