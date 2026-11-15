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
