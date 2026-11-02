package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// a point that is not a number is no point: a disc of several rings has
// no density there, on any machine, rather than NaN or an index past its
// profile.
func TestADiscHasNothingAtNoPoint(t *testing.T) {
	d := Disc{Inner: 1, Outer: 2, Thick: 1, Profile: []float64{1, 0.5, 1}}
	nan := math.NaN()
	for _, p := range []maths.Vec{maths.V(nan, 0, 1.5), maths.V(1.5, 0, nan),
		maths.V(nan, nan, nan)} {
		if got := d.Density(p); got != 0 {
			t.Errorf("at %v: %v", p, got)
		}
	}
	if d.profile(nan) != 0 {
		t.Errorf("the profile at NaN is %v", d.profile(nan))
	}
}

// pillars of no count or fewer are none: no tips, no density, no panic.
func TestPillarsOfNoCountAreNone(t *testing.T) {
	for _, count := range []int{0, -1, -50} {
		p := Pillars{Count: count, Height: 10, Width: 20, Thick: 1}
		if tips := p.Tips(); len(tips) != 0 {
			t.Errorf("%d pillars have %d tips", count, len(tips))
		}
		if got := p.Density(maths.V(0, 5, 0)); got != 0 {
			t.Errorf("%d pillars' density is %v", count, got)
		}
	}
}
