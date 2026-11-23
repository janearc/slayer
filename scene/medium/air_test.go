package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// airTested is the air the test below holds: its radius, its scale height
// and its density at the surface, each read from the chosen numbers.
func airTested(t *testing.T) Air {
	t.Helper()
	return Air{Radius: picked(t, "air-radius"),
		Height: picked(t, "air-scale-height"),
		Thick:  picked(t, "air-surface-density")}
}

// Air is thickest at the surface, a factor of e thinner a scale height up, zero
// below the ground and past its reach, and spans a ray by its top.
func TestAir(t *testing.T) {
	var _ Spanner = Air{}
	air := airTested(t)
	tolerance := picked(t, "air-tolerance")
	surface := maths.V(air.Radius, nought, nought)
	if density := air.Density(surface); density != air.Thick {
		t.Errorf("at the surface, %v", density)
	}
	scaleUp := maths.V(picked(t, "air-scale-height-up-x"), nought, nought)
	density := air.Density(scaleUp)
	if !(math.Abs(density-air.Thick/math.E) <= tolerance) {
		t.Errorf("a scale height up, %v", density)
	}
	ground := maths.V(picked(t, "air-underground-x"), nought, nought)
	if density := air.Density(ground); density != nought {
		t.Errorf("underground, %v", density)
	}
	margin := picked(t, "air-past-top-margin")
	past := maths.V(nought, air.Radius+airReach*air.Height+margin, nought)
	if density := air.Density(past); density != nought {
		t.Errorf("past the top, %v", density)
	}
	airSpansDown(t, air, tolerance)
}

// airSpansDown checks that a ray straight down meets the air at its top.
func airSpansDown(t *testing.T, air Air, tolerance float64) {
	t.Helper()
	start := picked(t, "air-ray-start")
	top := picked(t, "air-top-radius")
	down := maths.Ray{At: maths.V(nought, start, nought),
		Dir: maths.V(nought, -unity, nought)}
	enter, _, ok := air.Span(down, nought, math.Inf(unity))
	if !ok || !(math.Abs(enter-(start-top)) <= tolerance) {
		t.Errorf("the air's top is met at %v, %v", enter, ok)
	}
}
