// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

// Weather station data Quality Control checks.

import (
	"fmt"

	"github.com/ebarkie/weatherlink"
)

// qualityControl stores the QC results.  A dedicated struct is
// overkill for the limited validity checks currently implemented
// but as other checks like temporal consistency are introduced it
// will be useful.
type qualityControl struct {
	errs   []error
	passed bool
}

// assertRange implements simple min/max range checks.
func (qc *qualityControl) assertRange(f string, v float64, min float64, max float64) {
	if (v < min) || (v > max) {
		qc.errs = append(qc.errs, fmt.Errorf("Range check, %f < (%s) < %f, failed for value: %f", min, f, max, v))
	}

	return
}

// validityCheck takes a loop packet and performs a validity check using NOAA
// criteria.  A qualityControl struct is returned indicating if it passed or not.
// If it failed a slice of error descriptions are included.
func validityCheck(l weatherlink.Loop) (qc qualityControl) {
	// National Set of Validity Check Tolerances, Internal Consistency
	// Algorithms and Temporal Check Tolerances by Physical Element and
	// Observation System.
	//
	// AWIPS Document Number TSP-032-1992R2

	// Altimeter: 6.8in - 32.5in
	qc.assertRange("Barometer (altimeter)", l.Bar.Altimeter, 6.8, 32.5)
	qc.assertRange("Barometer (station)", l.Bar.Station, 6.8, 32.5)

	// Pressure (sea-level): 25.0in - 32.5in
	qc.assertRange("Barometer (sea-level)", l.Bar.SeaLevel, 25.0, 32.5)

	// Dew point: -80.0F - 90.0F
	qc.assertRange("Dew point", l.DewPoint, -80.0, 90.0)

	// Relative humidity: 0% - 100%
	qc.assertRange("Inside humidity", float64(l.InHumidity), 0, 100)
	qc.assertRange("Outside humidity", float64(l.OutHumidity), 0, 100)

	// Air temperature: -60.0F - 130.0F
	qc.assertRange("Inside air temperature", l.InTemp, -60.0, 130.0)
	qc.assertRange("Outside air temperature", l.OutTemp, -60.0, 130.0)

	// Accumulated precipitation: 0in - 44in
	qc.assertRange("Rain accumulation (last 15m)", l.Rain.Accum.Last15Min, 0, 44)
	qc.assertRange("Rain accumulation (last 1h)", l.Rain.Accum.LastHour, 0, 44)
	qc.assertRange("Rain accumulation (last 24h)", l.Rain.Accum.Last24Hours, 0, 44)
	qc.assertRange("Rain accumulation (today)", l.Rain.Accum.Today, 0, 44)

	// Sol.temperature: -40.0F - 150.0F
	for i, v := range l.SoilTemp {
		if v != nil {
			qc.assertRange(fmt.Sprintf("Soil temperature #%d", i), float64(*v), -40, 150)
		}
	}

	// Wind direction: 0deg - 360deg
	qc.assertRange("Wind direction (current)", float64(l.Wind.Cur.Dir), 0, 360)

	// Wind speed: 0mph - 287.695mph
	qc.assertRange("Wind speed (current)", float64(l.Wind.Cur.Speed), 0, 287.695)

	if qc.errs != nil {
		qc.passed = false
	} else {
		qc.passed = true
	}

	return
}
