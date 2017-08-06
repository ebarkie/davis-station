// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoopValidity(t *testing.T) {
	a := assert.New(t)

	// Invalid uninitialized loop packet
	l := Loop{}
	qc := validityCheck(l)
	a.False(qc.passed, "Uninitialized packet fails validity check")
	a.NotNil(qc.errs, "Uninitialized packet should have errors")
	for _, err := range qc.errs {
		t.Log(err)
	}
	a.Equal(3, len(qc.errs), "Uninitialized packet fails the 3 barometer checks")

	// Valid loop packet
	l.Bar.Altimeter = 6.8
	l.Bar.SeaLevel = 25.0
	l.Bar.Station = 6.8
	qc = validityCheck(l)
	a.True(qc.passed, "Valid packet passes validity check")
	a.Nil(qc.errs, "Valid packet has no errors")

	// Invalid temperature
	l.DewPoint = 65535.0
	qc = validityCheck(l)
	a.False(qc.passed, "Bad dew point fails validity check")
	a.NotNil(qc.errs, "Bad dew point has an error message")
}
