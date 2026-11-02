package thing

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// the hole's maths has its spin axis z where the field's is up, and it
// is a turn: x stays, y goes to z and z goes to minus y. worked by hand,
// (1, 2, 3) goes to (1, -3, 2).
func TestInHoleIsAQuarterTurnAboutX(t *testing.T) {
	f := &Field{}
	if got := f.InHole(maths.V(1, 2, 3)); got != [3]float64{1, -3, 2} {
		t.Errorf("(1, 2, 3) went to %v, want [1 -3 2]", got)
	}
	if got := f.InHole(maths.Up); got != [3]float64{0, 0, 1} {
		t.Errorf("up went to %v, want the hole's z", got)
	}
	if got := f.InHole(maths.XAxis); got != [3]float64{1, 0, 0} {
		t.Errorf("x went to %v, want the hole's x", got)
	}
	if got := f.InHole(maths.ZAxis); got != [3]float64{0, -1, 0} {
		t.Errorf("z went to %v, want the hole's minus y", got)
	}
}

// a turn keeps handedness: x cross y is z in the field, and the same
// of their images in the hole is the image of z, which is [0 -1 0].
// a mirror would give [0 1 0].
func TestInHoleKeepsItsHandedness(t *testing.T) {
	f := &Field{}
	x, y := f.InHole(maths.XAxis), f.InHole(maths.YAxis)
	cross := [3]float64{x[1]*y[2] - x[2]*y[1], x[2]*y[0] - x[0]*y[2], x[0]*y[1] - x[1]*y[0]}
	if want := f.InHole(maths.ZAxis); cross != want {
		t.Errorf("the images of x and y cross to %v, want the image of z, %v", cross, want)
	}
}

// FromHole is InHole undone, worked by hand: [1 -3 2] comes back to
// (1, 2, 3); the hole's z is up, its x is x, its minus y is z.
func TestFromHoleUndoesInHole(t *testing.T) {
	f := &Field{}
	if got := f.FromHole([3]float64{1, -3, 2}); got != maths.V(1, 2, 3) {
		t.Errorf("[1 -3 2] came back as %v, want (1, 2, 3)", got)
	}
	if got := f.FromHole([3]float64{0, 0, 1}); got != maths.Up {
		t.Errorf("the hole's z came back as %v, want up", got)
	}
	if got := f.FromHole([3]float64{0, -1, 0}); got != maths.ZAxis {
		t.Errorf("the hole's minus y came back as %v, want z", got)
	}
	if got := f.FromHole([3]float64{1, 0, 0}); got != maths.XAxis {
		t.Errorf("the hole's x came back as %v, want x", got)
	}
	for _, v := range []maths.Vec{maths.V(1, 2, 3), maths.V(-4, 0.5, 9), {}} {
		if got := f.FromHole(f.InHole(v)); got != v {
			t.Errorf("%v went there and back as %v", v, got)
		}
	}
}
