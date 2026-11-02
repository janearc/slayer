package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// wantSpan checks a span against one worked by hand: whether it is met,
// and where, to a billionth.
func wantSpan(t *testing.T, name string, enter, leave float64, ok bool,
	wantEnter, wantLeave float64, wantOK bool) {
	t.Helper()
	if ok != wantOK {
		t.Errorf("%s: met is %v, want %v", name, ok, wantOK)
		return
	}
	if wantOK && !(closeTo(enter, wantEnter, 1e-9) && closeTo(leave, wantLeave, 1e-9)) {
		t.Errorf("%s: %v to %v, want %v to %v", name, enter, leave, wantEnter,
			wantLeave)
	}
}

var (
	alongX  = maths.V(1, 0, 0)
	alongY  = maths.V(0, 1, 0)
	endless = math.Inf(1)
)

// slab is where a ray is between two planes. a ray along them is in it
// or not by its height, the planes themselves in; a ray across them
// crosses at the two heights, whichever way it goes; and a span of no
// length is no span.
func TestSlabIsBetweenTwoPlanes(t *testing.T) {
	e, l, ok := slab(2, maths.Ray{At: maths.V(0, 1, 0), Dir: alongX}, 0, 10)
	wantSpan(t, "along, inside", e, l, ok, 0, 10, true)
	_, _, ok = slab(2, maths.Ray{At: maths.V(0, 3, 0), Dir: alongX}, 0, 10)
	wantSpan(t, "along, outside", 0, 0, ok, 0, 0, false)
	e, l, ok = slab(2, maths.Ray{At: maths.V(0, 2, 0), Dir: alongX}, 0, 10)
	wantSpan(t, "along, on the plane", e, l, ok, 0, 10, true)
	e, l, ok = slab(2, maths.Ray{At: maths.V(0, -2, 0), Dir: alongX}, 0, 10)
	wantSpan(t, "along, on the lower plane", e, l, ok, 0, 10, true)
	_, _, ok = slab(2, maths.Ray{At: maths.V(0, 1, 0), Dir: alongX}, 3, 3)
	wantSpan(t, "along, no length", 0, 0, ok, 0, 0, false)

	// up from y = -5, the planes at y = -2 and 2 are met at 3 and 7
	e, l, ok = slab(2, maths.Ray{At: maths.V(0, -5, 0), Dir: alongY}, 0, endless)
	wantSpan(t, "up", e, l, ok, 3, 7, true)
	// down from y = 5, at 3 and 7 the same
	e, l, ok = slab(2, maths.Ray{At: maths.V(0, 5, 0), Dir: alongY.Neg()}, 0, endless)
	wantSpan(t, "down", e, l, ok, 3, 7, true)
	// up from the middle, the planes at y = -1 and 1 are met at -1 and 1:
	// held to 1 and 5 it leaves where it enters, at 1
	_, _, ok = slab(1, maths.Ray{At: maths.Vec{}, Dir: alongY}, 1, 5)
	wantSpan(t, "touching", 0, 0, ok, 0, 0, false)
}

// cylinder is where a ray is inside a cylinder round y. a ray along y is
// in it or not by how far it is from the axis, the wall itself in; a ray
// across it crosses the chord; a ray that only touches has no length.
//
// a ray a hair off the axis is on the quadratic, not taken for one along
// it: at 1e-6 across to 1 along, it leaves a cylinder 1e-4 in radius
// where it has gone 1e-4 over 1e-6 = 100 along.
func TestCylinderIsInsideARadius(t *testing.T) {
	upAt := func(x float64) maths.Ray { return maths.Ray{At: maths.V(x, 0, 0), Dir: alongY} }
	e, l, ok := cylinder(2, upAt(1), 0, 10)
	wantSpan(t, "along, inside", e, l, ok, 0, 10, true)
	_, _, ok = cylinder(2, upAt(3), 0, 10)
	wantSpan(t, "along, outside", 0, 0, ok, 0, 0, false)
	// 2.1 squared less 2 squared is 0.41, outside by less than one
	_, _, ok = cylinder(2, upAt(2.1), 0, 10)
	wantSpan(t, "along, just outside", 0, 0, ok, 0, 0, false)
	e, l, ok = cylinder(2, upAt(2), 0, 10)
	wantSpan(t, "along, on the wall", e, l, ok, 0, 10, true)
	e, l, ok = cylinder(2, upAt(1), 3, 4)
	wantSpan(t, "along, a unit", e, l, ok, 3, 4, true)
	_, _, ok = cylinder(2, upAt(1), 3, 3)
	wantSpan(t, "along, no length", 0, 0, ok, 0, 0, false)

	// from x = -10 along x, a cylinder of radius 6 about y is met at
	// 4 and 16 by a ray at z = 0, and only touched at 10 by one at z = 6
	e, l, ok = cylinder(6, maths.Ray{At: maths.V(-10, 0, 0), Dir: alongX}, 0, endless)
	wantSpan(t, "across", e, l, ok, 4, 16, true)
	_, _, ok = cylinder(6, maths.Ray{At: maths.V(-10, 0, 6), Dir: alongX}, 0, endless)
	wantSpan(t, "touching", 0, 0, ok, 0, 0, false)

	// the sideways part squared is 1e-12 to the last bit, which is the
	// bound under which a ray is taken to run along the axis
	side := 1e-6
	e, l, ok = cylinder(1e-4, maths.Ray{At: maths.Vec{}, Dir: maths.V(side, 1, 0)},
		0, 1000)
	wantSpan(t, "a hair off the axis", e, l, ok, 0, 100, true)
}

// gasFrom is a medium that is something past a place along x and nothing
// before it: the simplest thing to probe a ray into.
type gasFrom struct{ x float64 }

func (g gasFrom) Bounds() maths.Box { return maths.Box{} }
func (g gasFrom) Density(p maths.Vec) float64 {
	if p.X > g.x {
		return 1
	}
	return 0
}

// probed steps its range in cells of no more than finest, and no fewer
// than forty-eight, and as many as 2048, and keeps from the start of the
// first cell whose middle is in gas to the end of the last.
//
// 4096 long in steps of 1 is 2048 cells of 2, the cap; gas past 1000.9
// first has a middle in the cell from 1000 (middle 1001), so it enters
// at 1000 and leaves at the end, 4096.
func TestProbedStopsAtTwoThousandAndFortyEight(t *testing.T) {
	ray := maths.Ray{At: maths.Vec{}, Dir: alongX}
	e, l, ok := probed(gasFrom{1000.9}, ray, 0, 4096, 1)
	wantSpan(t, "capped", e, l, ok, 1000, 4096, true)
}

// with no finest to go by, forty-eight cells: 48 long is cells of 1, and
// gas past 20.9 first has a middle in the cell from 21 (middle 21.5).
//
// a machine that turns an endless count into the largest whole number
// would take 2048 cells here. arm64 does, and amd64 does not, so this
// test bites on arm64 and passes without biting elsewhere.
func TestProbedWithNoFinestIsFortyEightCells(t *testing.T) {
	ray := maths.Ray{At: maths.Vec{}, Dir: alongX}
	e, l, ok := probed(gasFrom{20.9}, ray, 0, 48, 0)
	wantSpan(t, "no finest", e, l, ok, 21, 48, true)
}

// a jet's span is probed in cells no coarser than half its base. a jet
// of base 1, 100 long, with a ray up its axis from the origin and gas
// past 30.9: 100 over 0.5 is 200 cells of 0.5.
//
// the first cell with a middle past 30.9 is the one from 31 (middle
// 31.25), so the span enters at 31 and leaves at 100.
func TestAJetsSpanIsProbedByHalfItsBase(t *testing.T) {
	j := Jet{Length: 100, Base: 1, Thick: 1, From: 30.9}
	e, l, ok := j.Span(maths.Ray{At: maths.Vec{}, Dir: alongY}, 0, endless)
	wantSpan(t, "base", e, l, ok, 31, 100, true)
}

// a jet of no base is probed by an eighth of its widest. 64 long and
// spreading by an eighth is 8 wide at the end, an eighth of 8 is 1, so
// 64 cells of 1; the gas is past 29.25, the first cell with a middle
// past it is from 29 (middle 29.5), and the last ends at 64.
func TestAJetOfNoBaseIsProbedByAnEighthOfItsSpread(t *testing.T) {
	j := Jet{Length: 64, Base: 0, Spread: 0.125, Thick: 1, From: 29.25}
	e, l, ok := j.Span(maths.Ray{At: maths.Vec{}, Dir: alongY}, 0, endless)
	wantSpan(t, "spread", e, l, ok, 29, 64, true)
}

// the same of a tail: no base, 64 long, spread an eighth, cells of 1.
// it bends by Curve times y squared over the length, and the gas goes
// while the bend is within two and a half widths: 0.8 y^2 / 64 over y / 8
// is 0.1 y, so to y = 25.
//
// the last cell with a middle within it is the one from 24 (middle
// 24.5), and it ends at 25.
func TestATailOfNoBaseIsProbedByAnEighthOfItsSpread(t *testing.T) {
	tail := Tail{Length: 64, Base: 0, Spread: 0.125, Curve: 0.8, Thick: 1}
	e, l, ok := tail.Span(maths.Ray{At: maths.Vec{}, Dir: alongY}, 0, endless)
	wantSpan(t, "spread", e, l, ok, 0, 25, true)
}

// a ray along a tail's head, at y of zero, is in its slab: the head is as dense
// as anywhere, Thick, and the tail is 2 across there, so a ray from x = -10 is
// met at 9 and 11.
func TestARayAlongATailsHeadIsInIt(t *testing.T) {
	tail := Tail{Length: 64, Base: 2, Thick: 1}
	e, l, ok := tail.Span(maths.Ray{At: maths.V(-10, 0, 0), Dir: alongX}, 0, endless)
	wantSpan(t, "along the head", e, l, ok, 9, 11, true)
}

// a ray straight up a jet is met where it crosses the slab, the jet
// being gas all the way: 2 long either side of the origin and 1 across,
// so from y = -5 it is met at 3 and 7.
func TestARayUpAJetIsInIt(t *testing.T) {
	j := Jet{Length: 2, Base: 1, Thick: 1}
	e, l, ok := j.Span(maths.Ray{At: maths.V(0, -5, 0), Dir: alongY}, 0, endless)
	wantSpan(t, "up the jet", e, l, ok, 3, 7, true)
}

// a disc's span is its slab clipped to its outer cylinder. a disc 0.4
// thick has planes at y = -0.2 and 0.2.
func TestADiscsSpan(t *testing.T) {
	d := Disc{Inner: 2, Outer: 6, Thick: 0.4}
	// straight up from y = -5, at 4.8 and 5.2, inside the cylinder or on
	// its wall; down from 5 the same; and outside it, a miss
	e, l, ok := d.Span(maths.Ray{At: maths.V(3, -5, 0), Dir: alongY}, 0, endless)
	wantSpan(t, "up", e, l, ok, 4.8, 5.2, true)
	e, l, ok = d.Span(maths.Ray{At: maths.V(3, 5, 0), Dir: alongY.Neg()}, 0, endless)
	wantSpan(t, "down", e, l, ok, 4.8, 5.2, true)
	e, l, ok = d.Span(maths.Ray{At: maths.V(0, -5, 6), Dir: alongY}, 0, endless)
	wantSpan(t, "up the wall", e, l, ok, 4.8, 5.2, true)
	_, _, ok = d.Span(maths.Ray{At: maths.V(0, -5, 8), Dir: alongY}, 0, endless)
	wantSpan(t, "up, outside in z", 0, 0, ok, 0, 0, false)
	_, _, ok = d.Span(maths.Ray{At: maths.V(8, -5, 0), Dir: alongY}, 0, endless)
	wantSpan(t, "up, outside in x", 0, 0, ok, 0, 0, false)

	// along x, above the disc, a miss; on its face, in: the chord of
	// the outer cylinder from x = -10 is 4 to 16; touching it, no length
	_, _, ok = d.Span(maths.Ray{At: maths.V(-10, 5, 0), Dir: alongX}, 0, endless)
	wantSpan(t, "along, above", 0, 0, ok, 0, 0, false)
	e, l, ok = d.Span(maths.Ray{At: maths.V(-10, 0.2, 0), Dir: alongX}, 0, endless)
	wantSpan(t, "along, on the face", e, l, ok, 4, 16, true)
	_, _, ok = d.Span(maths.Ray{At: maths.V(-10, 0, 6), Dir: alongX}, 0, endless)
	wantSpan(t, "along, touching", 0, 0, ok, 0, 0, false)
}
