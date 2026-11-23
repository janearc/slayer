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
	edge := picked(t, "audit-disc-profile-edge")
	middle := picked(t, "audit-disc-profile-middle")
	disc := Disc{
		Inner:   picked(t, "audit-disc-inner"),
		Outer:   picked(t, "audit-disc-outer"),
		Thick:   picked(t, "audit-disc-thick"),
		Profile: []float64{edge, middle, edge},
	}
	reach := picked(t, "audit-disc-reach")
	nan := math.NaN()
	for _, point := range []maths.Vec{maths.V(nan, nought, reach),
		maths.V(reach, nought, nan), maths.V(nan, nan, nan)} {
		if got := disc.Density(point); got != nought {
			t.Errorf("at %v: %v", point, got)
		}
	}
	if disc.profile(nan) != nought {
		t.Errorf("the profile at NaN is %v", disc.profile(nan))
	}
}

// pillars of no count or fewer are none: no tips, no density, no panic.
func TestPillarsOfNoCountAreNone(t *testing.T) {
	counts := []int{nought, pickedCount(t, "audit-pillars-fewer"),
		pickedCount(t, "audit-pillars-fewest")}
	probe := maths.V(nought, picked(t, "audit-pillars-probe-height"), nought)
	for _, count := range counts {
		pillars := Pillars{
			Count:  count,
			Height: picked(t, "audit-pillars-height"),
			Width:  picked(t, "audit-pillars-width"),
			Thick:  picked(t, "audit-pillars-thick"),
		}
		if tips := pillars.Tips(); len(tips) != nought {
			t.Errorf("%d pillars have %d tips", count, len(tips))
		}
		if got := pillars.Density(probe); got != nought {
			t.Errorf("%d pillars' density is %v", count, got)
		}
	}
}
