package main

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// inPicture is whether a point of the world falls inside the usual
// picture seen from an eye, through hillside's lens.
func inPicture(eye maths.Subspace, point maths.Vec) bool {
	seen := eye.Orientation.In(point.Sub(eye.Origin))
	if seen.Z >= 0 {
		return false
	}
	half := math.Tan(lensWide * maths.Degree / 2)
	tall := half * float64(usualTall) / float64(usualWide)
	return math.Abs(seen.X/-seen.Z) <= half &&
		math.Abs(seen.Y/-seen.Z) <= tall
}

// From above, every corner of the hillside is in the picture, so every
// place on it shows, and the eye looks at the hillside's middle on the
// ground, whatever its width.
func TestFromAboveEveryPlaceShows(t *testing.T) {
	for _, across := range []float64{10, 40, 82.5} {
		eye := aboveAt(across)
		half := across / 2
		for _, corner := range []maths.Vec{maths.V(-half, 0, -half),
			maths.V(half, 0, -half), maths.V(-half, 0, half),
			maths.V(half, 0, half)} {
			if !inPicture(eye, corner) {
				t.Errorf("a hillside %g across: its corner %v is out of "+
					"the picture from %v", across, corner, eye.Origin)
			}
		}
		toMiddle := maths.Vec{}.Sub(eye.Origin).Unit()
		if !eye.Forward().Near(toMiddle, 1e-9) {
			t.Errorf("from %v the eye looks %v, not at the middle",
				eye.Origin, eye.Forward())
		}
	}
}

// The usual eye does not see every place of a wide hillside: its back
// rows are behind its front ones, which is what -above is for.
func TestTheUsualEyeIsLowerThanAbove(t *testing.T) {
	if eyeAt(82.5).Origin.Y >= aboveAt(82.5).Origin.Y {
		t.Error("the usual eye is as high as the eye above")
	}
}
