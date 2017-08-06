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
func (b *loopBuffer) add(l Loop) {
	b.Lock()
	defer b.Unlock()

	b.cur = (b.cur + 1) % loopsMax
	b.buf[b.cur] = l
	if b.size < loopsMax {
		b.size++
	}
}

// loops returns the loop buffer as a slice.
func (b *loopBuffer) loops() (l []Loop) {
	b.RLock()
	defer b.RUnlock()

	j := b.cur
	for i := 0; i < b.size; i++ {
		l = append(l, b.buf[j])
		j = (j - 1 + loopsMax) % loopsMax
	}

	return
}
