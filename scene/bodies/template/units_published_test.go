package template

import (
	"math"
	"testing"

	"github.com/janearc/slayer/scene/bodies"
)

// physics, worked by hand: a body's day is a Duration, turned from
// hours, and the published sidereal rotation is held in hours as
// remembered.
//
// the sun 609.1 (25.38 days), mercury 1407.5 (58.646 days), venus
// 5832.5 (243.025 days), the earth 23.934, mars 24.623, jupiter 9.925,
// saturn 10.656, uranus 17.24, neptune 16.11, pluto and charon 153.29
// (6.387 days).
//
// each template's Day, read back in hours, is held to 1 per cent of
// it, so a day taken for minutes or seconds shows.
func TestEachDayIsThePublishedRotationInHours(t *testing.T) {
	for name, want := range map[string]float64{
		"sol": 609.1, "mercury": 1407.5, "venus": 5832.5, "earth": 23.934,
		"mars": 24.623, "jupiter": 9.925, "saturn": 10.656, "uranus": 17.24,
		"neptune": 16.11, "pluto": 153.29, "charon": 153.29,
	} {
		b := byName(t, name)
		if got := b.Day.Hours(); !(math.Abs(got-want) <= 0.01*want) {
			t.Errorf("%s: day is %v hours, published about %v", name, got, want)
		}
	}
}

// physics, worked by hand: a body's Tilt is in radians, turned from
// degrees, and the published axial tilt is held in degrees as
// remembered.
//
// mercury 0.034, venus 177.36, the earth 23.44, mars 25.19, jupiter
// 3.13, saturn 26.73, uranus 97.77, neptune 28.32, pluto 122.53.
//
// each template's Tilt, read back in degrees (a radian is 180 / pi of
// them), is held to a tenth of a degree of it, so a tilt left in
// degrees shows.
func TestEachTiltIsThePublishedAxialTiltInRadians(t *testing.T) {
	for name, want := range map[string]float64{
		"mercury": 0.034, "venus": 177.36, "earth": 23.44, "mars": 25.19,
		"jupiter": 3.13, "saturn": 26.73, "uranus": 97.77, "neptune": 28.32,
		"pluto": 122.53,
	} {
		b := byName(t, name)
		if got := b.Tilt * 180 / math.Pi; !(math.Abs(got-want) <= 0.1) {
			t.Errorf("%s: tilt is %v degrees, published about %v", name, got, want)
		}
	}
}

// physics, worked by hand: lengths are kilometres. the moon is some
// sixty earth radii away, 384,400 over 6371 being 60.3; the sun's
// radius is 109 of the earth's, 696,000 over 6371 being 109.2;
// jupiter's is 11.0 of the earth's, 69,911 over 6371.
//
// each is held to a tenth of a radius or so.
func TestLengthsAreInKilometresByTheirRatios(t *testing.T) {
	earth := Earth()
	if got := earth.Moons[0].Orbit / earth.Radius; math.Abs(got-60.3) > 0.1 {
		t.Errorf("the moon is %v earth radii away, want 60.3", got)
	}
	if got := Sol().Radius / earth.Radius; math.Abs(got-109.2) > 0.1 {
		t.Errorf("the sun is %v earth radii across, want 109.2", got)
	}
	if got := Jupiter().Radius / earth.Radius; math.Abs(got-11.0) > 0.1 {
		t.Errorf("jupiter is %v earth radii across, want 11.0", got)
	}
}

// physics, worked by hand: the earth's radius in kilometres is 6371,
// which is 6.371e6 m, 40,030 km round at the equator being a circle of
// 2 pi 6371.
//
// the earth's mass is in kilograms: 5.972e24 is 81.3 moons, 5.972e24
// over 7.342e22.
func TestTheEarthIsInKilometresAndKilograms(t *testing.T) {
	e := Earth()
	if got := 2 * math.Pi * e.Radius; math.Abs(got-40030) > 10 {
		t.Errorf("the earth is %v km round, want about 40,030", got)
	}
	if got := e.Mass / e.Moons[0].Body.Mass; math.Abs(got-81.3) > 0.1 {
		t.Errorf("the earth is %v moons of mass, want 81.3", got)
	}
}

// byName is the template of that name, or the test fails.
func byName(t *testing.T, name string) bodies.Body {
	t.Helper()
	for _, b := range everyBody() {
		if b.Name == name {
			return b
		}
	}
	t.Fatalf("no body named %s", name)
	return bodies.Body{}
}
