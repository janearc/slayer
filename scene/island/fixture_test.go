package island

import (
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/footprint"
	"github.com/janearc/slayer/scene/frame"
)

// heightFunc is a Heights source made from a function.
type heightFunc func(longitude, latitude float64) (float64, bool, error)

// Height calls the function.
func (source heightFunc) Height(longitude, latitude float64) (float64, bool, error) {
	return source(longitude, latitude)
}

// flat is a source with the same height everywhere.
func flat(height float64) heightFunc {
	return func(float64, float64) (float64, bool, error) { return height, true, nil }
}

// testSphere is a one-thousand-kilometre sphere anchored at zero, zero.
func testSphere(t *testing.T) frame.Sphere {
	t.Helper()
	sphere, err := frame.NewSphere(measure.Metres(1e6), frame.Location{})
	if err != nil {
		t.Fatal(err)
	}
	return sphere
}

// testRegion is a four degree square centred on the anchor.
var testRegion = Region{West: -2, South: -2, East: 2, North: 2}

// testDetail is a three by three lattice, three levels deep, with room for
// every sample the tests ask for.
func testDetail() Detail {
	return Detail{Patches: 16, Evaluations: 200, Depth: 3, Side: 3, Points: 5000,
		CellPixels: 4, MinimumHeight: -1000, MaximumHeight: 5000}
}

// testTerrain builds a terrain over the source with the given detail.
func testTerrain(t *testing.T, from Heights, detail Detail) *Terrain {
	t.Helper()
	ground, err := New(testSphere(t), from, testRegion, detail)
	if err != nil {
		t.Fatal(err)
	}
	return ground
}

// originAt is a render origin at a world point.
func originAt(t *testing.T, point maths.Vec) frame.Relative {
	t.Helper()
	origin, err := frame.NewRelative(point)
	if err != nil {
		t.Fatal(err)
	}
	return origin
}

// footprintFor is a sixty degree square camera with the given pose.
func footprintFor(eye maths.Subspace) (footprint.Perspective, error) {
	return footprint.New(eye, measure.Degrees(60), 64, 64, 1, measure.Metres(1))
}

// lookDown is a camera at the given height over longitude and latitude
// zero, looking straight down with east to its right and north up.
func lookDown(t *testing.T, height float64) footprint.Perspective {
	t.Helper()
	view, err := footprintFor(maths.Subspace{Origin: maths.V(0, height, 0), Orientation: maths.Orientation{
		X: maths.V(1, 0, 0), Y: maths.V(0, 0, -1), Z: maths.V(0, 1, 0)}})
	if err != nil {
		t.Fatal(err)
	}
	return view
}

// footprintZero is a perspective that was never checked.
func footprintZero() footprint.Perspective { return footprint.Perspective{} }
