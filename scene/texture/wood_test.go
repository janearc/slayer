package texture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"pgregory.net/rapid"
)

// pale and dark are a wood's two colours, for the tests
var (
	pale = maths.RGB(0.6, 0.45, 0.3)
	dark = maths.RGB(0.3, 0.2, 0.1)
)

// trueOak is wood with rings a unit wide and no wander, its pith along x
// through the origin: the plainest wood there is, for sums done by hand.
func trueOak() Wood {
	return Wood{Early: pale, Late: dark, Ring: 1}
}

// A ring is early wood for most of its year and late wood at the end:
// three tenths of the way through a ring the wood is pale, nine tenths
// through it is dark.
func TestWoodIsEarlyThenLateThroughARing(t *testing.T) {
	wood := trueOak()
	if got := wood.At(maths.V(0, 0.3, 0)); !near(got, pale) {
		t.Errorf("three tenths through a ring the wood is %v, not early %v",
			got, pale)
	}
	if got := wood.At(maths.V(0, 0.9, 0)); !near(got, dark) {
		t.Errorf("nine tenths through a ring the wood is %v, not late %v",
			got, dark)
	}
	// and the next year is the same again
	if got := wood.At(maths.V(0, 1.3, 0)); !near(got, pale) {
		t.Errorf("three tenths through the next ring the wood is %v", got)
	}
}

// the anti-test of the one above: the late wood is a share of the ring
// and no more. With a share of a half, six tenths through is late; with
// the quarter a wood has by default, it is still early.
func TestWoodsLateShareIsItsOwn(t *testing.T) {
	half := trueOak()
	half.LateShare = 0.5
	if got := half.At(maths.V(0, 0.7, 0)); !near(got, dark) {
		t.Errorf("with half a ring late, seven tenths through is %v, not late", got)
	}
	if got := trueOak().At(maths.V(0, 0.6, 0)); !near(got, pale) {
		t.Errorf("with a quarter of a ring late, six tenths through is %v, not early", got)
	}
}

// The rings are round the pith: at one distance from it the wood is the
// same whichever way round, and wherever along a true grain.
func TestWoodsRingsAreRoundThePith(t *testing.T) {
	wood := trueOak()
	want := wood.At(maths.V(0, 0.9, 0))
	for name, p := range map[string]maths.Vec{
		"to the side":     maths.V(0, 0, 0.9),
		"below":           maths.V(0, -0.9, 0),
		"aslant":          maths.V(0, 0.9*math.Sqrt2/2, 0.9*math.Sqrt2/2),
		"along the grain": maths.V(7.3, 0.9, 0),
	} {
		if got := wood.At(p); !near(got, want) {
			t.Errorf("%s the wood is %v; at that distance it is %v", name, got, want)
		}
	}
}

// the anti-test of the one above: a wood that wanders is not the same
// all the way along the grain, and its pith and its grain's way are its
// own. Moved, the pith takes the rings with it; turned, the grain runs
// the other way.
func TestWoodWandersAndIsPlacedByItsPith(t *testing.T) {
	wandering := trueOak()
	wandering.Wander = 0.6
	same := true
	first := wandering.At(maths.V(0, 0.72, 0))
	for run := 0.5; run < 6; run += 0.5 {
		sample := wandering.At(maths.V(run, 0.72, 0))
		if !near(sample, first) {
			same = false
		}
	}
	if same {
		t.Error("a wood that wanders is the same all along its grain")
	}
	moved := trueOak()
	moved.Pith = maths.V(0, 2, 0)
	if got := moved.At(maths.V(0, 2.9, 0)); !near(got, dark) {
		t.Errorf("nine tenths of a ring from a moved pith is %v, not late", got)
	}
	upright := trueOak()
	upright.Along = maths.YAxis
	if got := upright.At(maths.V(0.9, 5, 0)); !near(got, dark) {
		t.Errorf("with the grain upright, nine tenths out sideways is %v, not late", got)
	}
	if got := upright.At(maths.V(0, 0.9, 0)); !near(got, pale) {
		t.Errorf("with the grain upright, a point on the pith is %v, not early", got)
	}
}

// A wood told nothing of its ring, its late share or its grain's way has
// a tenth of a unit, a quarter, and x. The anti-test: a wood told a
// ring of its own keeps it and not the tenth: 0.09 out is late wood in
// rings a tenth wide and early in rings a fifth wide.
func TestWoodFallsBackOnItsDefaults(t *testing.T) {
	plain := Wood{Early: pale, Late: dark}
	told := Wood{Early: pale, Late: dark, Ring: 0.1, LateShare: 0.25,
		Along: maths.XAxis}
	for _, p := range []maths.Vec{maths.V(0.4, 0.03, 0), maths.V(2, 0.09, 0.01),
		maths.V(-1, 0.31, 0.2)} {
		if !near(plain.At(p), told.At(p)) {
			t.Errorf("at %v a wood told nothing is %v, and told its defaults %v",
				p, plain.At(p), told.At(p))
		}
	}
	wider := Wood{Early: pale, Late: dark, Ring: 0.2}
	out := maths.V(0, 0.09, 0)
	if !near(plain.At(out), dark) || !near(wider.At(out), pale) {
		t.Errorf("0.09 out a wood told nothing is %v and one told a ring "+
			"of a fifth %v", plain.At(out), wider.At(out))
	}
}

// Wherever it is asked, wood is its early wood, its late wood, or
// something between the two, in every colour. This is the anti-test of
// every sum above: whatever the ring, the share, the wander and the
// pith, no place is paler than the early wood or darker than the late.
func TestWoodIsAlwaysBetweenItsTwoWoods(t *testing.T) {
	rapid.Check(t, func(draw *rapid.T) {
		wood := Wood{Early: pale, Late: dark,
			Ring:      rapid.Float64Range(0.01, 2).Draw(draw, "ring"),
			LateShare: rapid.Float64Range(0.05, 0.9).Draw(draw, "late share"),
			Wander:    rapid.Float64Range(0, 3).Draw(draw, "wander"),
			Pith: maths.V(rapid.Float64Range(-5, 5).Draw(draw, "pith x"),
				rapid.Float64Range(-5, 5).Draw(draw, "pith y"),
				rapid.Float64Range(-5, 5).Draw(draw, "pith z")),
		}
		p := maths.V(rapid.Float64Range(-20, 20).Draw(draw, "x"),
			rapid.Float64Range(-20, 20).Draw(draw, "y"),
			rapid.Float64Range(-20, 20).Draw(draw, "z"))
		got := wood.At(p)
		for name, part := range map[string][3]float64{
			"red":   {got.R, dark.R, pale.R},
			"green": {got.G, dark.G, pale.G},
			"blue":  {got.B, dark.B, pale.B},
		} {
			if part[0] < part[1]-1e-12 || part[0] > part[2]+1e-12 {
				draw.Fatalf("at %v the wood's %s is %v, outside %v to %v",
					p, name, part[0], part[1], part[2])
			}
		}
	})
}
