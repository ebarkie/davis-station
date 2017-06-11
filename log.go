// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

// logger wraps the standard Go logger with our own io.Writer
// interface that allows for multiple writers.  This is allows
// telnet connections to view the log outputs.
type logger struct {
	*log.Logger              // Go logger
	writers      []io.Writer // Slice of writers that need to be written to
	sync.RWMutex             // Mutex to coordinate writers changes
}

// Loggers
var (
	Trace = multiLog(ioutil.Discard, "[TRCE]", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	Debug = multiLog(ioutil.Discard, "[DBUG]", log.LstdFlags|log.Lshortfile)
	Info  = multiLog(os.Stdout, "[INFO]", log.LstdFlags)
	Warn  = multiLog(os.Stderr, "[WARN]", log.LstdFlags|log.Lshortfile)
	Error = multiLog(os.Stderr, "[ERRO]", log.LstdFlags|log.Lshortfile)
)

func multiLog(w io.Writer, prefix string, flag int) *logger {
	l := &logger{}
	l.Logger = log.New(l, prefix, flag)

	if w != ioutil.Discard {
		l.writers = []io.Writer{w}
	}

	return l
}

func (l *logger) addOutput(w io.Writer) {
	l.Lock()
	defer l.Unlock()

	l.writers = append(l.writers, w)
}

func (l *logger) removeOutput(w io.Writer) {
	l.Lock()
	defer l.Unlock()

	for i := range l.writers {
		if l.writers[i] == w {
			l.writers = append(l.writers[:i], l.writers[i+1:]...)
			return
		}
	}
}

func (l *logger) Write(p []byte) (n int, err error) {
	l.RLock()
	defer l.RUnlock()

	for _, w := range l.writers {
		// Prevent hung telnet debug sessions from blocking all logging
		switch w.(type) {
		case net.Conn:
			w.(net.Conn).SetWriteDeadline(time.Now().Add(100 * time.Millisecond))
		}
		n, err = w.Write(p)
		if (err != nil) || (n != len(p)) {
			err = io.ErrShortWrite
			continue
		}
	}

	return
}
