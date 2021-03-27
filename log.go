// Copyright (c) 2016 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"sync"
)

// logger wraps the standard Logger with a MultiWriter.  Telnet
// connections use this to temporarily subscribe to debug or trace.
type logger struct {
	*log.Logger              // Logger
	writers      []io.Writer // Slice of writers that need to be written to
	sync.RWMutex             // Mutex to coordinate writers changes
}

// Loggers
var (
	Trace = multiLog(io.Discard, "[TRCE]", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	Debug = multiLog(io.Discard, "[DBUG]", log.LstdFlags|log.Lshortfile)
	Info  = multiLog(os.Stdout, "[INFO]", log.LstdFlags)
	Warn  = multiLog(os.Stderr, "[WARN]", log.LstdFlags|log.Lshortfile)
	Error = multiLog(os.Stderr, "[ERRO]", log.LstdFlags|log.Lshortfile)
)

func multiLog(w io.Writer, prefix string, flag int) *logger {
	l := &logger{}
	l.Logger = log.New(l, prefix, flag)

	if w != io.Discard {
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

func (l *logger) Write(p []byte) (int, error) {
	l.RLock()
	defer l.RUnlock()

	// The logger ends with newlines but for telnet we need carriage
	// returns too.
	return io.MultiWriter(l.writers...).Write(bytes.ReplaceAll(p, []byte{lf}, []byte{cr, lf}))
}
