// Copyright (c) 2016 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import "sync"

// loopBuffer is a circular buffer that holds up to loopsMax of
// historical loops.  It's designed to be efficient at adding
// new items since it happens very frequently.
type loopBuffer struct {
	buf [loopsMax]loop
	cur int
	len int
	sync.RWMutex
}

// add adds a Loop packet to the loop buffer.  Once the
// buffer is full each new packet overwrites the oldest.
func (lb *loopBuffer) add(l loop) {
	lb.Lock()
	defer lb.Unlock()

	lb.cur = (lb.cur + 1) % loopsMax
	lb.buf[lb.cur] = l
	if lb.len < loopsMax {
		lb.len++
	}
}

// last returns the number of items in the loop buffer and the
// last one added.
func (lb *loopBuffer) last() (int, loop) {
	lb.RLock()
	defer lb.RUnlock()

	return lb.len, lb.buf[lb.cur]
}

// loops returns the loop buffer as a slice.
func (lb *loopBuffer) loops() []loop {
	lb.RLock()
	defer lb.RUnlock()

	ls := make([]loop, lb.len)
	j := lb.cur
	for i := 0; i < lb.len; i++ {
		ls[i] = lb.buf[j]
		j = (j - 1 + loopsMax) % loopsMax
	}

	return ls
}
