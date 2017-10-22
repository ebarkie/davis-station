// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"math"
	"time"

	"github.com/ebarkie/weatherlink/packet"
	"github.com/ebarkie/weatherlink/units"
)

// metar generates a report string for a given Loop struct.
func metar(l Loop) string {
	// Type
	s := "METAR"

	// Date/Time
	s += fmt.Sprintf(" %sZ", l.Timestamp.In(time.UTC).Format("021504"))

	// Report Modifier
	s += " AUTO" // Indicates a fully automated report with no human intervention

	// Wind
	s += fmt.Sprintf(" %03d%02.f", l.Wind.Cur.Dir, units.Kn(float64(l.Wind.Cur.Speed)))
	if units.Kn(l.Wind.Gust.Last10MinSpeed) >= 0.50 {
		s += fmt.Sprintf("G%02.f", units.Kn(l.Wind.Gust.Last10MinSpeed))
	}
	s += "KT"

	// Weather Phenomena
	if l.Rain.Rate >= 1.0 { // Heavy
		s += " +RA"
	} else if l.Rain.Rate >= 0.5 { // Moderate
		s += " RA"
	} else if l.Rain.Rate > 0.0 { // Light
		s += " -RA"
	}

	// Temperature/Dew Point
	if t := units.C(l.OutTemp); t < 0.0 {
		s += fmt.Sprintf(" M%02.f", t*-1)
	} else {
		s += fmt.Sprintf(" %02.f", t)
	}
	if t := units.C(l.DewPoint); t < 0.0 {
		s += fmt.Sprintf("/M%02.f", t*-1)
	} else {
		s += fmt.Sprintf("/%02.f", t)
	}

	// Altimeter (in inches)
	s += fmt.Sprintf(" A%04.f", l.Bar.Altimeter*100)

	// Remarks
	s += " RMK AO1" // Automated station without a precipitation descriminator

	// Pressure Rising or Falling Rapidly
	if l.Bar.Trend == packet.RisingRapid {
		s += " PRESRR"
	} else if l.Bar.Trend == packet.FallingRapid {
		s += " PRESFR"
	}

	// Sea Level Pressure
	_, d := math.Modf(l.Bar.SeaLevel * 33.8637526 / 100)
	s += fmt.Sprintf(" SLP%03.f", d*1000)

	// Hourly Precipitation Amount
	if l.Rain.Accum.LastHour > 0.0 {
		s += fmt.Sprintf(" P%04.f", l.Rain.Accum.LastHour*100)
	}

	// 24-Hour Precipitation Amount
	if l.Rain.Accum.Last24Hours > 0.0 {
		s += fmt.Sprintf(" 7%04.f", l.Rain.Accum.Last24Hours*100)
	}

	// Hourly Temperature and Dew Point
	if t := units.C(l.OutTemp); t < 0.0 {
		s += fmt.Sprintf(" T1%03.f", t*-10)
	} else {
		s += fmt.Sprintf(" T0%03.f", t*10)
	}
	if t := units.C(l.DewPoint); t < 0.0 {
		s += fmt.Sprintf("1%03.f", t*-10)
	} else {
		s += fmt.Sprintf("0%03.f", t*10)
	}

	return s
}
