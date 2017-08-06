// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkImplementedLoopBuffer(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var b loopBuffer
		for i := 0; i < loopsMax*2; i++ {
			b.add(Loop{Seq: int64(i)})
		}
	}
}

func TestLoopBuffer(t *testing.T) {
	for added := 0; added < loopsMax+2; added++ {
		var b loopBuffer
		var loops []Loop

		for i := 0; i < added; i++ {
			b.add(Loop{Seq: int64(i)})

			if len(loops) >= loopsMax {
				loops = loops[0 : len(loops)-1]
			}
			loops = append([]Loop{{Seq: int64(i)}}, loops...)
		}

		assert.Equal(t, loops, b.loops(), fmt.Sprintf("Added %d does not match slice", added))
	}
}
