// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func loopBufferAdd(lb *loopBuffer, l int) {
	for i := 0; i < l; i++ {
		lb.add(Loop{Seq: int64(i)})
	}
}

func benchmarkLoopBufferAdd(b *testing.B, l int) {
	for n := 0; n < b.N; n++ {
		lb := loopBuffer{}
		loopBufferAdd(&lb, l)
	}
}

func BenchmarkLoopBufferAddHalf(b *testing.B) { benchmarkLoopBufferAdd(b, loopsMax/2) }
func BenchmarkLoopBufferAddOne(b *testing.B)  { benchmarkLoopBufferAdd(b, loopsMax) }
func BenchmarkLoopBufferAddTwo(b *testing.B)  { benchmarkLoopBufferAdd(b, loopsMax*2) }

func benchmarkLoopBufferLast(b *testing.B, l int) {
	var lb loopBuffer
	loopBufferAdd(&lb, l)
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		lb.last()
	}
}

func BenchmarkLoopBufferLastHalf(b *testing.B) { benchmarkLoopBufferLast(b, loopsMax/2) }
func BenchmarkLoopBufferLastOne(b *testing.B)  { benchmarkLoopBufferLast(b, loopsMax) }

func benchmarkLoopBufferLoops(b *testing.B, l int) {
	var lb loopBuffer
	loopBufferAdd(&lb, l)
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		lb.loops()
	}
}

func BenchmarkLoopBufferLoopsHalf(b *testing.B) { benchmarkLoopBufferLoops(b, loopsMax/2) }
func BenchmarkLoopBufferLoopsOne(b *testing.B)  { benchmarkLoopBufferLoops(b, loopsMax) }

func TestLoopBuffer(t *testing.T) {
	for added := 0; added < loopsMax+2; added++ {
		var lb loopBuffer
		loops := make([]Loop, 0)

		for i := 0; i < added; i++ {
			lb.add(Loop{Seq: int64(i)})

			if len(loops) >= loopsMax {
				loops = loops[0 : len(loops)-1]
			}
			loops = append([]Loop{{Seq: int64(i)}}, loops...)
		}

		assert.Equal(t, loops, lb.loops(), fmt.Sprintf("Added %d does not match slice", added))
	}
}
