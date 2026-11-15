package shape

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// flat is a level field at a height, three by three over two across.
func flat(t *testing.T, level float64) *HeightField {
	t.Helper()
	heights := make([]float64, 9)
	for i := range heights {
		heights[i] = level
	}
	field, err := NewHeightField(heights, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	return field
}

// A level field reads its level everywhere over the square, faces up,
// and its distance is the height above it.
func TestALevelFieldIsItsLevel(t *testing.T) {
	field := flat(t, 0.25)
	for _, at := range []maths.Vec{maths.V(0, 1, 0), maths.V(-0.9, 1, 0.7)} {
		if d := field.Distance(at); !closeTo(d, at.Y-0.25) {
			t.Errorf("over %v the distance is %v, want %v", at, d, at.Y-0.25)
		}
		if n := field.Normal(at); !closeTo(n.Y, 1) {
			t.Errorf("over %v the field faces %v", at, n)
		}
	}
	// beside the square the field is at least the sideways gap away
	if d := field.Distance(maths.V(2, 0.25, 0)); d < 1-1e-9 {
		t.Errorf("a metre beside the square the distance is %v", d)
	}
}

// A ramp faces square to its slope, and its distance never says more
// than the truth, so a march cannot step through it.
func TestARampFacesItsSlopeAndNeverOverstates(t *testing.T) {
	// rising one in one along x: columns at -1, 0, 1
	heights := []float64{-1, 0, 1, -1, 0, 1, -1, 0, 1}
	field, err := NewHeightField(heights, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := maths.V(-1, 1, 0).Unit()
	if n := field.Normal(maths.V(0.2, 5, -0.3)); !closeTo(n.Sub(want).Len(), 0) {
		t.Errorf("the ramp faces %v, want %v", n, want)
	}
	// the bound: |Distance| is at most the true distance to the plane
	truth := func(p maths.Vec) float64 {
		return math.Abs(p.Dot(want)) // the ramp is the plane x = y
	}
	for _, at := range []maths.Vec{maths.V(0, 0.5, 0), maths.V(0.3, 0.1, 0.2),
		maths.V(-0.5, -0.2, 0.4), maths.V(0.1, 2, -0.1)} {
		if d := math.Abs(field.Distance(at)); d > truth(at)+1e-9 {
			t.Errorf("at %v the distance %v overstates %v", at, d, truth(at))
		}
	}
}

// The checks: too few samples, a wrong count, no extent, a hole.
func TestAHeightFieldRefusesWhatItCannotBe(t *testing.T) {
	if _, err := NewHeightField([]float64{1}, 1, 1); err == nil {
		t.Error("one sample was not refused")
	}
	if _, err := NewHeightField(make([]float64, 8), 3, 1); err == nil {
		t.Error("eight samples for nine were not refused")
	}
	if _, err := NewHeightField(make([]float64, 9), 3, 0); err == nil {
		t.Error("no extent was not refused")
	}
	holed := make([]float64, 9)
	holed[4] = math.NaN()
	if _, err := NewHeightField(holed, 3, 1); err == nil {
		t.Error("a hole was not refused")
	}
}

// closeTo is within a billionth, absolutely.
func closeTo(got, want float64) bool { return math.Abs(got-want) < 1e-9 }
