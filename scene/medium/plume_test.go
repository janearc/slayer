package medium

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// plumeTested is the plume the test below holds, read from the chosen
// numbers.
func plumeTested(t *testing.T) Plume {
	t.Helper()
	return Plume{Height: picked(t, "plume-height"),
		Width: picked(t, "plume-width"),
		Thick: unity}
}

// plumeAt is the point at a distance out along x and a height up.
func plumeAt(out, height float64) maths.Vec {
	return maths.V(out, height, nought)
}

// A plume is dense at the column's foot, has its umbrella's dome
// halfway up at most of the width, is empty under the dome, and stops
// at its height.
func TestPlumeIsAColumnUnderAnUmbrella(t *testing.T) {
	plume := plumeTested(t)
	footHeight := picked(t, "plume-foot-height")
	foot := plume.Density(plumeAt(nought, footHeight))
	if foot < picked(t, "plume-foot-density-least") {
		t.Errorf("the column's foot is %v", foot)
	}
	domeOut, domeHeight := picked(t, "plume-dome-out"), plume.Height*halfway
	dome := plume.Density(plumeAt(domeOut, domeHeight))
	if dome < picked(t, "plume-dome-density-least") {
		t.Errorf("the dome halfway up is %v", dome)
	}
	underOut := picked(t, "plume-under-out")
	under := plume.Density(plumeAt(underOut, domeHeight))
	if under > picked(t, "plume-under-density-most") {
		t.Errorf("inside the umbrella is %v", under)
	}
	overHeight := picked(t, "plume-over-height")
	over := plume.Density(plumeAt(nought, overHeight))
	depth := picked(t, "plume-below-depth")
	below := plume.Density(plumeAt(nought, -depth))
	if over != nought || below != nought {
		t.Error("the plume goes beyond its height")
	}
	plumeBoundsHold(t, plume)
}

// plumeBoundsHold checks the plume's box: the umbrella's shell is 10 out at
// the ground and its skirt is cut three skins, of 1.5 each, past it: 14.5.
func plumeBoundsHold(t *testing.T, plume Plume) {
	t.Helper()
	box := plume.Bounds()
	edge := picked(t, "plume-bounds-edge")
	tolerance := picked(t, "plume-tolerance")
	if !closeTo(box.Max.X, edge, tolerance) ||
		!closeTo(box.Min.X, -edge, tolerance) ||
		!closeTo(box.Max.Z, edge, tolerance) ||
		!closeTo(box.Min.Z, -edge, tolerance) ||
		box.Min.Y != nought || box.Max.Y != plume.Height {
		t.Errorf("bounds %v", box)
	}
}
