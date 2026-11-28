package generate

import (
	"math"
	"testing"
)

// The coefficient falls as a star gets hotter, stays between nought and
// one, and is held at the table's ends.
func TestHotterStarsAreDarkenedLessAtTheLimb(t *testing.T) {
	previous := 1.0
	for kelvin := 3000.0; kelvin < 100000; kelvin *= 1.1 {
		got := LimbCoefficient(kelvin)
		if !(got > 0 && got <= 1) || got > previous {
			t.Fatalf("%v K: %v after %v", kelvin, got, previous)
		}
		previous = got
	}
	if got := LimbCoefficient(9500); math.Abs(got-0.50) > 1e-12 {
		t.Errorf("an A star at 9500 K is %v, not the table's 0.50", got)
	}
}

// The anti-test: a number that is no temperature never gives a
// coefficient that darkens the centre or blacks the edge.
func TestNoTemperatureGivesACoefficientOutsideNoughtToOne(t *testing.T) {
	for _, kelvin := range []float64{-1, 0, 1e12} {
		if got := LimbCoefficient(kelvin); !(got > 0 && got < 1) {
			t.Errorf("%v K gives %v", kelvin, got)
		}
	}
}
