// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package units implements very simple and lightweight unit
// conversion functions.
package units

// C converts Celsius to Fahrenheit.
func C(f float64) float64 {
	return (f - 32) * 0.5556
}

// Kn converts Miles Per Hour (MPH) to Knots.
func Kn(mph float64) float64 {
	return mph * 0.8688
}
