// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import "sync"

// loopBuffer is a circular buffer that holds up to loopsMax of
// historical loops.  It's designed to be efficient at adding
// new items since it happens very frequently.
type loopBuffer struct {
	buf [loopsMax]Loop
	cur int
	len int
	sync.RWMutex
}

// Add adds a Loop packet to the loop buffer.  Once the
// buffer is full each new packet overwrites the oldest.
func (lb *loopBuffer) Add(l Loop) {
	lb.Lock()
	defer lb.Unlock()

	lb.cur = (lb.cur + 1) % loopsMax
	lb.buf[lb.cur] = l
	if lb.len < loopsMax {
		lb.len++
	}
}

// Last returns the number of items in the loop buffer and the
// last one added.
func (lb *loopBuffer) Last() (n int, l Loop) {
	// This is a bit ugly to read but it's about 30% faster
	// without defer and multi-assignment.
	lb.RLock()
	n = lb.len
	l = lb.buf[lb.cur]
	lb.RUnlock()
	return
}

// Loops returns the loop buffer as a slice.
func (lb *loopBuffer) Loops() []Loop {
	lb.RLock()
	defer lb.RUnlock()

	ls := make([]Loop, lb.len)
	j := lb.cur
	for i := 0; i < lb.len; i++ {
		ls[i] = lb.buf[j]
		j = (j - 1 + loopsMax) % loopsMax
	}

	return ls
}
