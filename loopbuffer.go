// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import "sync"

// loopBuffer is a circular buffer that holds up to loopsMax of
// historical loops.  It's designed to be efficient at adding
// new items since it happens very frequently.
type loopBuffer struct {
	buf  [loopsMax]Loop
	cur  int
	size int
	sync.RWMutex
}

// add adds a wrapped loop packet to the loop buffer.  Once the
// buffer is full each new packet overwrites the oldest.
func (lb *loopBuffer) add(l Loop) {
	lb.Lock()
	defer lb.Unlock()

	lb.cur = (lb.cur + 1) % loopsMax
	lb.buf[lb.cur] = l
	if lb.size < loopsMax {
		lb.size++
	}
}

// loops returns the loop buffer as a slice.
func (lb *loopBuffer) loops() (l []Loop) {
	lb.RLock()
	defer lb.RUnlock()

	j := lb.cur
	for i := 0; i < lb.size; i++ {
		l = append(l, lb.buf[j])
		j = (j - 1 + loopsMax) % loopsMax
	}

	return
}
