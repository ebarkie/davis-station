// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import "sync"

// loopBuffer is a circular buffer that holds up to loopsMax of
// historical loops.  It's designed to be efficient at adding
// new items since it happens very frequently.
type loopBuffer struct {
	buf  [loopsMax]WrappedLoop
	cur  int
	size int
	sync.RWMutex
}

// add adds a wrapped loop packet to the loop buffer.  Once the
// buffer is full each new packet overwrites the oldest.
func (l *loopBuffer) add(w WrappedLoop) {
	l.Lock()
	defer l.Unlock()

	l.cur = (l.cur + 1) % loopsMax
	l.buf[l.cur] = w
	if l.size < loopsMax {
		l.size++
	}
}

// loops returns the loop buffer as a slice.
func (l *loopBuffer) loops() (w []WrappedLoop) {
	l.RLock()
	defer l.RUnlock()

	j := l.cur
	for i := 0; i < l.size; i++ {
		w = append(w, l.buf[j])
		j--
		if j < 0 {
			j = loopsMax - 1
		}
	}

	return
}
