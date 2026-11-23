package daylight

import (
	"math"
	"testing"
	"time"
)

func TestEquinoxAtGreenwich(t *testing.T) {
	// At the equator near the March equinox, sunrise is east, the noon sun
	// is almost overhead, and sunset is west. The approximate equation of
	// time shifts each event by several minutes.
	for _, check := range []struct {
		hour              int
		azimuth, altitude float64
		day               bool
	}{
		{6, 90, -2, false},
		{12, 90, 88, true},
		{18, 270, 2, true},
		{0, 270, -88, false},
	} {
		at := time.Date(2026, time.March, 20, check.hour, 0, 0, 0, time.UTC)
		sun, err := Sun(0, 0, at)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(sun.Altitude-check.altitude) > 2 || sun.Daylight != check.day {
			t.Errorf("%v altitude/day %g/%v", at, sun.Altitude, sun.Daylight)
		}
		if check.hour == 6 || check.hour == 18 {
			if math.Abs(sun.Azimuth-check.azimuth) > 2 {
				t.Errorf("%v azimuth %g", at, sun.Azimuth)
			}
		}
	}
}

func TestHonoluluTimeZoneAndNight(t *testing.T) {
	hawaii := time.FixedZone("HST", -10*60*60)
	noon := time.Date(2026, time.June, 21, 12, 0, 0, 0, hawaii)
	midnight := time.Date(2026, time.June, 21, 0, 0, 0, 0, hawaii)
	day, err := Sun(21.27, -157.7, noon)
	if err != nil {
		t.Fatal(err)
	}
	utc, err := Sun(21.27, -157.7, noon.UTC())
	if err != nil || day != utc {
		t.Fatalf("same instant differs by timezone: %+v, %+v, %v", day, utc, err)
	}
	night, err := Sun(21.27, -157.7, midnight)
	if err != nil {
		t.Fatal(err)
	}
	if !day.Daylight || day.Altitude < 80 || night.Daylight || night.Altitude > -40 {
		t.Fatalf("Honolulu day %+v, night %+v", day, night)
	}
}

func TestSunRejectsInvalidCoordinates(t *testing.T) {
	for _, point := range [][2]float64{{91, 0}, {-91, 0}, {0, 181}, {0, -181},
		{math.NaN(), 0}, {0, math.Inf(1)}} {
		if _, err := Sun(point[0], point[1], time.Now()); err == nil {
			t.Errorf("accepted %+v", point)
		}
	}
}
