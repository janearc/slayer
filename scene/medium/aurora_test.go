package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// An aurora is a glow on an oval round each magnetic pole, in its
// shell, brightest at the bottom, and nothing at the equator or beyond
// the shell; curtains vary it along the oval.
func TestAuroraIsAnOvalRoundEachPole(t *testing.T) {
	a := Aurora{Radius: 10, Low: 0.2, High: 0.6, Lat: 67 * maths.Degree,
		Width: 4 * maths.Degree, Thick: 1, Seed: 3}
	at := func(lat, lon, up float64) maths.Vec {
		return maths.V(math.Cos(lat)*math.Cos(lon), math.Sin(lat), math.Cos(lat)*math.Sin(lon)).Scale(10 + up)
	}
	north, south := a.Density(at(67*maths.Degree, 0.5, 0.26)), a.Density(at(-67*maths.Degree, 0.5, 0.26))
	if north < 0.6 || math.Abs(north-south) > 1e-9 {
		t.Errorf("the ovals are %v north and %v south", north, south)
	}
	if d := a.Density(at(0, 0.5, 0.25)); d != 0 {
		t.Errorf("the equator glows %v", d)
	}
	if d := a.Density(at(67*maths.Degree, 0.5, 0.1)); d != 0 {
		t.Errorf("below the shell glows %v", d)
	}
	if lo, hi := a.Density(at(67*maths.Degree, 0.5, 0.26)), a.Density(at(67*maths.Degree, 0.5, 0.55)); !(hi < 0.3*lo) {
		t.Errorf("the bottom is %v and the top %v", lo, hi)
	}
	a.Curtains = 0.8
	var low, high float64 = 10, 0
	for lon := 0.0; lon < 6.28; lon += 0.05 {
		d := a.Density(at(67*maths.Degree, lon, 0.26))
		low, high = math.Min(low, d), math.Max(high, d)
	}
	if !(high > 2*low) {
		t.Errorf("curtains vary the oval only from %v to %v", low, high)
	}
}
