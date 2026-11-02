package trace

import (
	"math"
	"testing"
)

// within is whether got is within tol of want. A NaN is within nothing,
// so a check written "if !within(...)" fails on one, where "if
// math.Abs(got-want) > tol" passed it.
func within(got, want, tol float64) bool {
	return math.Abs(got-want) <= tol
}

// within turns a NaN away, whichever side it is on, and takes an exact
// match at no tolerance.
func TestWithinRejectsNaN(t *testing.T) {
	nan := math.NaN()
	if within(nan, 1, 1) || within(1, nan, 1) || within(1, 1, nan) {
		t.Error("a NaN was within something")
	}
	if !within(1, 1, 0) || !within(1, 1.5, 0.5) || within(1, 1.6, 0.5) {
		t.Error("within is wrong on plain numbers")
	}
}

// unfit is the name of the first of a pixel's numbers that is a NaN,
// or infinite where it may not be, or "" when the pixel is sound. Only
// the depth, the last number, may be infinite.
func unfit(numbers [5]float64) string {
	for k, number := range numbers {
		depth := k == len(numbers)-1
		if math.IsNaN(number) || (math.IsInf(number, 0) && !depth) {
			return channelName(k)
		}
	}
	return ""
}

// Every golden scene is finite: no red, green, blue or cover of any
// pixel is a NaN or infinite, and no depth is a NaN. A depth may be
// infinite, which is a ray that met nothing.
func TestScenesAreFinite(t *testing.T) {
	for _, scene := range goldens() {
		for i, pixel := range scene.draw().Pix {
			if name := unfit(channels(pixel)); name != "" {
				t.Errorf("%s: pixel %d's %s is not finite: %+v",
					scene.name, i, name, pixel)
			}
		}
	}
}
