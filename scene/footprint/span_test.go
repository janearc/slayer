package footprint_test

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/footprint"
)

// spanView is a camera at the origin looking down minus z, a quarter turn
// of field across 200 by 100 square pixels, so its focal length is 100
// pixels, with the near plane at a tenth of a metre.
func spanView(t *testing.T, eye maths.Subspace) footprint.Perspective {
	t.Helper()
	view, err := footprint.New(eye, measure.Radians(math.Pi/2), 200, 100, 1, measure.Metres(0.1))
	if err != nil {
		t.Fatal(err)
	}
	return view
}

// nearly is whether a value is within a part in a billion of the want.
func nearly(got, want float64) bool { return math.Abs(got-want) <= 1e-9*want }

// TestSpanIsTheDiagonalOfTheScreenRectangle checks hand worked rectangles:
// a flat box at depth ten, and a box that runs from depth ten to twenty,
// whose nearer end makes the rectangle wider than its centre would.
func TestSpanIsTheDiagonalOfTheScreenRectangle(t *testing.T) {
	view := spanView(t, maths.Local())
	cases := []struct {
		name string
		box  maths.Box
		want float64
	}{
		// x runs -1 to 1 over depth 10, at 100 pixels a unit: 20 across; y runs
		// -0.5 to 0.5: 10 down; the diagonal is the square root of 500.
		{"flat at depth ten", maths.Box{Min: maths.V(-1, -0.5, -10), Max: maths.V(1, 0.5, -10)}, math.Sqrt(500)},
		// x runs 1 to 3: from 1/20 to 3/10 of a unit, 25 pixels; y runs -1 to
		// 1: 10 pixels either side at the nearer depth, 20 down.
		{"depth ten to twenty", maths.Box{Min: maths.V(1, -1, -20), Max: maths.V(3, 1, -10)}, math.Sqrt(625 + 400)},
	}
	for _, each := range cases {
		got, err := view.Span(each.box)
		if err != nil || !got.Visible || got.Limited || !nearly(got.Pixels, each.want) || got.Pixels < each.want {
			t.Errorf("%s: %+v, %v; want %v pixels, visible, rounded outward", each.name, got, err, each.want)
		}
	}
}

// TestSpanKeepsWhatItCannotBound checks a box that reaches the near plane, a
// box that is behind the camera, a box too big to give finite pixels, and the
// inputs it refuses.
func TestSpanKeepsWhatItCannotBound(t *testing.T) {
	view := spanView(t, maths.Local())
	got, err := view.Span(maths.Box{Min: maths.V(-1, -1, -5), Max: maths.V(1, 1, 0.5)})
	if err != nil || !got.Visible || !got.Limited || got.Pixels != math.MaxFloat64 {
		t.Errorf("a box through the near plane: %+v, %v", got, err)
	}
	got, err = view.Span(maths.Box{Min: maths.V(-1, -1, 10), Max: maths.V(1, 1, 11)})
	if err != nil || got.Visible || got.Limited || got.Pixels != 0 {
		t.Errorf("a box behind the camera: %+v, %v", got, err)
	}
	huge := maths.Box{Min: maths.V(-1e308, -1, -1e308), Max: maths.V(1e308, 1, -1)}
	got, err = view.Span(huge)
	if err != nil || !got.Visible || !got.Limited || got.Pixels != math.MaxFloat64 {
		t.Errorf("a box too wide to measure: %+v, %v", got, err)
	}
	if _, err := (footprint.Perspective{}).Span(maths.Box{}); err == nil {
		t.Error("an unchecked view gave a span")
	}
	for name, bad := range map[string]maths.Box{
		"unordered": {Min: maths.V(1, 0, 0)},
		"NaN":       {Max: maths.V(0, math.NaN(), 0)},
	} {
		if _, err := view.Span(bad); err == nil {
			t.Errorf("%s bounds were accepted", name)
		}
	}
	reach := maths.Box{Min: maths.V(-math.MaxFloat64, -1, -1), Max: maths.V(math.MaxFloat64, 1, 1)}
	if _, err := view.Span(reach); err == nil || !strings.Contains(err.Error(), "numeric range") {
		t.Errorf("a box that overflows camera space: %v", err)
	}
}

// TestAssessRetainsWhatArithmeticCannotBound checks a box that overflows
// camera space and a displacement under an unset orientation, which counts
// as no turn.
func TestAssessRetainsWhatArithmeticCannotBound(t *testing.T) {
	box := maths.Box{Min: maths.V(-1, -1, -50), Max: maths.V(1, 1, -40)}
	reach := maths.Box{Min: maths.V(-math.MaxFloat64, -1, -1), Max: maths.V(math.MaxFloat64, 1, 1)}
	if _, err := spanView(t, maths.Local()).Assess(reach, measure.Metres(0)); err == nil {
		t.Error("a box that overflows camera space was assessed")
	}
	// A depth of 50 to 40 at 100 pixels a metre of focal length: one metre
	// of displacement moves a point at most about 100*1*(1+0/40)/(40-1)
	// pixels along each axis at the nearer depth.
	unset := maths.Subspace{Scale: 1}
	assessed, err := spanView(t, unset).Assess(box, measure.Metres(1))
	if err != nil || !assessed.Visible || assessed.Limited || assessed.Pixels < 2 || assessed.Pixels > 20 {
		t.Errorf("a displacement under an unset orientation: %+v, %v", assessed, err)
	}
}

// TestPoseOrientationMustBeAProperTurn checks the refusal of axes that are
// not unit length, and of axes that are not at right angles.
func TestPoseOrientationMustBeAProperTurn(t *testing.T) {
	for name, turn := range map[string]maths.Orientation{
		"axis of length two": {X: maths.V(2, 0, 0), Y: maths.V(0, 1, 0), Z: maths.V(0, 0, 1)},
		"axes not square":    {X: maths.V(1, 0, 0), Y: maths.V(0.6, 0.8, 0), Z: maths.V(0, 0, 1)},
		"infinite axis":      {X: maths.V(math.Inf(1), 0, 0), Y: maths.V(0, 1, 0), Z: maths.V(0, 0, 1)},
	} {
		_, err := footprint.New(maths.Subspace{Scale: 1, Orientation: turn}, measure.Radians(math.Pi/2), 200, 100, 1, measure.Metres(0.1))
		if err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

// TestAssessRetainsAPixelBoundTooLargeToCount checks a view of a hair's width
// of field, whose focal length is about 10^302 pixels, with a displacement
// that leaves a tenth of a femtometre of depth: the bound passes the largest
// number, so the geometry is retained as limited.
func TestAssessRetainsAPixelBoundTooLargeToCount(t *testing.T) {
	view, err := footprint.New(maths.Local(), measure.Radians(1e-300), 200, 100, 1, measure.Metres(1e-20))
	if err != nil {
		t.Fatal(err)
	}
	point := maths.Box{Min: maths.V(0, 0, -1), Max: maths.V(0, 0, -1)}
	got, err := view.Assess(point, measure.Metres(math.Nextafter(1, 0)))
	if err != nil || !got.Visible || !got.Limited || got.Pixels != math.MaxFloat64 {
		t.Errorf("an unbounded pixel bound: %+v, %v", got, err)
	}
}
