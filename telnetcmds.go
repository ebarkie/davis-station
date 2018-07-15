// Copyright (c) 2016 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/ebarkie/davis-station/internal/textcmd"
	"github.com/ebarkie/weatherlink"
	"github.com/ebarkie/weatherlink/data"
)

func (t telnetCtx) archive(e textcmd.Env) (err error) {
	// Default archive period is 2 hours
	h := 2
	if a := e.Arg(1); a != "" {
		h, err = strconv.Atoi(a)
		if err != nil {
			return
		}
	}

	d := time.Duration(h) * time.Hour
	ac := t.ar.NewGet(time.Now().Add(-d), time.Now())
	t.template(e, "archive", ac)

	return
}

func (t telnetCtx) health(e textcmd.Env) error {
	_, lastLoop := t.lb.last()

	t.template(e, "health",
		struct {
			Bat data.LoopBat
		}{lastLoop.Bat},
	)

	return nil
}

func (t telnetCtx) help(e textcmd.Env) error {
	t.template(e, "help", nil)

	return nil
}

func (t telnetCtx) lamps(e textcmd.Env) error {
	fmt.Fprintf(e, "Setting lamps %s..", e.Arg(1))
	if e.Arg(1) == "on" {
		t.wl.Q <- weatherlink.LampsOn
	} else {
		t.wl.Q <- weatherlink.LampsOff
	}
	fmt.Fprintf(e, "done.\r\n")

	return nil
}

func (t telnetCtx) log(e textcmd.Env) error {
	fmt.Fprintf(e, "Watching log at %s level.  Press any key to end.\r\n\r\n", e.Arg(1))
	debugLoggers := []*logger{Error, Warn, Info, Debug}
	if e.Arg(1) == "trace" {
		debugLoggers = append(debugLoggers, Trace)
	}

	for _, l := range debugLoggers {
		l.addOutput(&e)
	}
	defer func() {
		for _, l := range debugLoggers {
			l.removeOutput(&e)
		}
	}()

	t.readOne(e)
	fmt.Fprintf(e, "Watch log ended.\r\n")

	return nil
}

func (t telnetCtx) loop(e textcmd.Env) error {
	var watch bool
	if a := e.Arg(1); a == "watch" {
		watch = true
	}

	printLoop := func(l loop) {
		t.template(e, "loop", l)
		if watch {
			fmt.Fprintf(e, "\r\nWatching conditions.  Press any key to end.")
		}
	}

	numLoops, lastLoop := t.lb.last()
	if numLoops < loopsMin {
		return errLoopsMin
	}
	printLoop(lastLoop)

	if watch {
		events := t.eb.Subscribe(e.RemoteAddr().String())
		defer t.eb.Unsubscribe(events)
		go func() {
			for ev := range events {
				if lastLoop, ok := ev.Data.(loop); ok {
					printLoop(lastLoop)
				}
			}
		}()

		t.readOne(e)
		fmt.Fprintf(e, "\r\n")
	}

	return nil
}

func (t telnetCtx) quit(e textcmd.Env) error {
	t.template(e, "quit", nil)

	return textcmd.ErrCmdQuit
}

func (t telnetCtx) time(e textcmd.Env) error {
	t.template(e, "time",
		struct {
			Time time.Time
		}{time.Now()},
	)

	return nil
}

func (t telnetCtx) uname(e textcmd.Env) error {
	t.template(e, "uname",
		struct {
			Banner    string
			LocalAddr net.Addr
		}{banner, e.LocalAddr()},
	)

	return nil
}

func (t telnetCtx) uptime(e textcmd.Env) error {
	// Round uptime down to nearest second
	uptime := time.Since(t.startTime)
	uptime = uptime - (uptime % time.Second)

	t.template(e, "uptime",
		struct {
			Uptime    time.Duration
			StartTime time.Time
		}{uptime, t.startTime},
	)

	return nil
}

func (t telnetCtx) ver(e textcmd.Env) error {
	t.template(e, "ver",
		struct {
			Ver string
		}{version},
	)

	return nil
}

func (t telnetCtx) whoami(e textcmd.Env) error {
	t.template(e, "whoami",
		struct {
			RemoteAddr net.Addr
		}{
			e.RemoteAddr(),
		},
	)

	return nil
}
