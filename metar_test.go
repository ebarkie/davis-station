// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"time"
)

func Examplemetar() {
	l := Loop{Timestamp: time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)}
	fmt.Println(metar(l))

	// Output:
	// METAR 021504Z AUTO 00000KT M18/M18 A0000 RMK AO1 SLP000 T11781178
}
