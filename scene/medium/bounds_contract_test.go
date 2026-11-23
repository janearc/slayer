package medium

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// inside is whether a point is within a box, its faces included.
func inside(box maths.Box, at maths.Vec) bool {
	return at.X >= box.Min.X && at.X <= box.Max.X &&
		at.Y >= box.Min.Y && at.Y <= box.Max.Y &&
		at.Z >= box.Min.Z && at.Z <= box.Max.Z
}

// boundsContractPick is a point's place along one axis, drawn in the span
// from low to high grown reach times about its middle.
func boundsContractPick(source *draws, reach, low, high float64) float64 {
	middle, half := (low+high)/double, (high-low)/double
	return middle + half*reach*(double*source.next()-unity)
}

// growth is a point drawn in the box grown by the contract's reach about its
// middle.
func growth(t *testing.T, box maths.Box, source *draws) maths.Vec {
	t.Helper()
	reach := picked(t, "bounds-contract-outside-reach")
	return maths.V(boundsContractPick(source, reach, box.Min.X, box.Max.X),
		boundsContractPick(source, reach, box.Min.Y, box.Max.Y),
		boundsContractPick(source, reach, box.Min.Z, box.Max.Z))
}

// boundsContractWithin is a place along one axis, drawn in the span from low
// to high.
func boundsContractWithin(source *draws, low, high float64) float64 {
	return low + (high-low)*source.next()
}

// justOutside is, for each of a box's six faces, a point a hair past the face
// and within the face's extent on the other axes: where a medium cut by a
// narrow box would still have gas. How far past a face is a share of the
// box's size.
func justOutside(t *testing.T, box maths.Box, source *draws) []maths.Vec {
	t.Helper()
	size := maths.V(box.Max.X-box.Min.X, box.Max.Y-box.Min.Y,
		box.Max.Z-box.Min.Z)
	step := picked(t, "bounds-contract-outside-face-step")
	var points []maths.Vec
	for range pickedCount(t, "bounds-contract-outside-points") /
		pickedCount(t, "bounds-contract-face-divisor") {
		at := maths.V(boundsContractWithin(source, box.Min.X, box.Max.X),
			boundsContractWithin(source, box.Min.Y, box.Max.Y),
			boundsContractWithin(source, box.Min.Z, box.Max.Z))
		points = append(points,
			maths.V(box.Max.X+step*size.X, at.Y, at.Z),
			maths.V(box.Min.X-step*size.X, at.Y, at.Z),
			maths.V(at.X, box.Max.Y+step*size.Y, at.Z),
			maths.V(at.X, box.Min.Y-step*size.Y, at.Z),
			maths.V(at.X, at.Y, box.Max.Z+step*size.Z),
			maths.V(at.X, at.Y, box.Min.Z-step*size.Z))
	}
	return points
}

// leak is the first point outside a medium's box where its density is not
// zero, and whether there is one. It looks at points drawn at random about
// the box and at points a hair past each face.
func leak(t *testing.T, subject Medium) (maths.Vec, bool) {
	t.Helper()
	box := subject.Bounds()
	source := draws(unity)
	points := justOutside(t, box, &source)
	for range pickedCount(t, "bounds-contract-outside-points") {
		points = append(points, growth(t, box, &source))
	}
	for _, at := range points {
		if !inside(box, at) && subject.Density(at) != nought {
			return at, true
		}
	}
	return maths.Vec{}, false
}

// every medium the recording holds is exactly zero at every point outside its
// Bounds, as the Medium contract says, with no exception.
func TestEveryMediumIsZeroOutsideItsBounds(t *testing.T) {
	for name, subject := range recorded() {
		if at, found := leak(t, subject); found {
			t.Errorf("%s: density %v at %v, outside %v", name,
				subject.Density(at), at, subject.Bounds())
		}
	}
}

// leaky is a medium whose box is the unit cube and whose gas is everywhere.
type leaky struct{}

// Bounds is the unit cube.
func (leaky) Bounds() maths.Box {
	return maths.Box{Min: maths.V(nought, nought, nought),
		Max: maths.V(unity, unity, unity)}
}

// Density is one at every point, outside the box too.
func (leaky) Density(maths.Vec) float64 { return unity }

// the check bites: a medium that has gas outside its box is found, and one
// that has none is not, so the test over the recording is not vacuous.
func TestTheOutsideCheckFindsAMediumThatLeaks(t *testing.T) {
	if _, found := leak(t, leaky{}); !found {
		t.Error("a medium with gas everywhere was not found to leak")
	}
	puff := Puff{Radius: picked(t, "bounds-contract-puff-radius"), Thick: unity}
	if at, found := leak(t, puff); found {
		t.Errorf("a puff leaks at %v", at)
	}
}

// the box of a plume holds its gas and no more: there is gas a hair inside
// the box's wall, at the umbrella's skirt, so the box is not wider than the
// gas needs. A point is put a share of the reach either side of the edge.
func TestPlumeBoundsReachItsSkirt(t *testing.T) {
	plume := Plume{Height: picked(t, "bounds-contract-plume-height"),
		Width: picked(t, "bounds-contract-plume-width"), Thick: unity}
	reach := plume.Bounds().Max.X
	margin := picked(t, "bounds-contract-skirt-margin")
	height := picked(t, "bounds-contract-plume-skirt-height")
	inner := maths.V(reach*(unity-margin), height, nought)
	if density := plume.Density(inner); density <= nought {
		t.Errorf("no gas at the plume's skirt, density %v", density)
	}
	outer := maths.V(reach*(unity+margin), height, nought)
	if density := plume.Density(outer); density != nought {
		t.Errorf("gas %v past the plume's skirt", density)
	}
}

// boundsContractTail is the tail of the bounds test: 12 long, bending by 0.3
// and spreading by a tenth from a base of 1, read from the chosen numbers.
func boundsContractTail(t *testing.T) Tail {
	t.Helper()
	return Tail{Length: picked(t, "bounds-contract-tail-length"),
		Base:   unity,
		Spread: picked(t, "bounds-contract-tail-spread"),
		Curve:  picked(t, "bounds-contract-tail-curve"),
		Thick:  unity}
}

// the box of a tail is the bend at its end and the cut of its gaussian at its
// widest, 3.6 and 2.5 of 1.7 for a tail 12 long that bends by 0.3 and spreads
// by a tenth from a base of 1: 7.85 either side. There is gas a hair inside
// that cut halfway along, and none a hair past it.
func TestTailBoundsAreItsBendAndItsCut(t *testing.T) {
	tail := boundsContractTail(t)
	box := tail.Bounds()
	worked := picked(t, "bounds-contract-tail-edge")
	tolerance := picked(t, "bounds-contract-tail-tolerance")
	if !closeTo(box.Max.X, worked, tolerance) ||
		!closeTo(box.Min.X, -worked, tolerance) ||
		!closeTo(box.Max.Z, worked, tolerance) ||
		box.Min.Y != nought || box.Max.Y != tail.Length {
		t.Errorf("bounds %v", box)
	}
	half := tail.Length / double
	edge := tail.Curve*half*half/tail.Length +
		chosen.skirtWidths*(tail.Base/double+half*tail.Spread)
	margin := picked(t, "bounds-contract-skirt-margin")
	inner := maths.V(edge*(unity-margin), half, nought)
	if density := tail.Density(inner); density <= nought {
		t.Errorf("no gas a hair inside the tail's cut, density %v", density)
	}
	outer := maths.V(edge*(unity+margin), half, nought)
	if density := tail.Density(outer); density != nought {
		t.Errorf("gas %v a hair past the tail's cut", density)
	}
}

// a tail whose spread is negative narrows as it goes, so its box is its
// head's width and the bend, and the gas is still inside it.
func TestATailThatNarrowsStaysInItsBounds(t *testing.T) {
	narrowing := Tail{Length: picked(t, "bounds-contract-tail-length"),
		Base:   picked(t, "bounds-contract-narrow-base"),
		Spread: picked(t, "bounds-contract-narrow-spread"),
		Curve:  picked(t, "bounds-contract-narrow-curve"),
		Thick:  unity}
	if at, found := leak(t, narrowing); found {
		t.Errorf("a narrowing tail leaks at %v", at)
	}
}
