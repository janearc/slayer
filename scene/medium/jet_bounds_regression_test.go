package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// A box one width across would clip nonzero Gaussian gas at a flat wall.
func TestJetBoundsContainItsRadiatingSkirt(t *testing.T) {
	jet := Jet{Length: picked(t, "jet-bounds-regression-length"),
		Base:   picked(t, "jet-bounds-regression-base"),
		Spread: picked(t, "jet-bounds-regression-spread"),
		Thick:  unity}
	across := picked(t, "jet-bounds-regression-witness-x")
	up := picked(t, "jet-bounds-regression-witness-y")
	witness := maths.V(across, up, nought)
	if jet.Density(witness) <= nought {
		t.Fatal("witness misses skirt")
	}
	box := jet.Bounds()
	if witness.X > box.Max.X || witness.X < box.Min.X {
		t.Fatal("radiating gas is clipped by its volume bounds")
	}
	back := picked(t, "jet-bounds-regression-ray-back")
	ray := maths.Ray{At: maths.V(across, up, -back), Dir: maths.ZAxis}
	far := picked(t, "jet-bounds-regression-ray-far")
	enter, leave, ok := jet.Span(ray, nought, far)
	if !ok || enter >= back || leave <= back {
		t.Fatal("ray through visible skirt is culled", enter, leave, ok)
	}
}

// jetBoundsRegressionRecorded is one recorded jet, checked by its name.
type jetBoundsRegressionRecorded struct{ name string }

// For every recorded jet, a one-width limit would exclude a positive portion
// of its Gaussian skirt. The bounds and ray span must include that gas, and
// the independently calculated density must agree at a radius inside the
// width and at one in the skirt. Knots are disabled here to isolate the
// common Gaussian envelope.
func TestJetRecordedSkirtsFollowTheGaussianSupport(t *testing.T) {
	for _, name := range []string{"jet", "jet, flowing", "jet, knotted",
		"jet, wide open"} {
		t.Run(name, jetBoundsRegressionRecorded{name}.check)
	}
}

// check holds the recorded jet's bounds to its skirt, then its density and
// span at a radius inside the width and at one in the skirt.
func (recording jetBoundsRegressionRecorded) check(t *testing.T) {
	jet := recorded()[recording.name].(Jet)
	jet.Knots = nought
	widest := jet.Base/double + jet.Length*jet.Spread
	outer := chosen.skirtWidths * widest
	box := jet.Bounds()
	if box.Min.X != -outer || box.Max.X != outer ||
		box.Min.Z != -outer || box.Max.Z != outer {
		t.Fatalf("bounds do not cover 2.5 widths: %+v", box)
	}
	height := picked(t, "jet-bounds-regression-height-share") * jet.Length
	width := jet.Base/double + height*jet.Spread
	reach := picked(t, "jet-bounds-regression-skirt-reach")
	for _, radius := range []float64{halfway * width,
		(widest + reach*width) / double} {
		jetBoundsRegressionRadius(t, jet, radius, height, widest, outer)
	}
}

// jetBoundsRegressionRadius holds a jet's density at a radius to the
// Gaussian, and, past the jet's widest, its span to the skirt there.
func jetBoundsRegressionRadius(t *testing.T, jet Jet, radius, height, widest,
	outer float64) {
	t.Helper()
	width := jet.Base/double + height*jet.Spread
	point := maths.V(radius, height, nought)
	ratio := radius / width
	want := jet.Thick * (unity - height/jet.Length) *
		math.Exp(-double*ratio*ratio)
	got := jet.Density(point)
	if !(want > nought) ||
		math.Abs(got-want) > picked(t, "jet-bounds-regression-tolerance") {
		t.Fatalf("density at radius %g = %g; Gaussian = %g", radius, got, want)
	}
	if radius <= widest {
		return
	}
	back := picked(t, "jet-bounds-regression-back-widths")
	far := picked(t, "jet-bounds-regression-far-widths")
	start := maths.V(radius, height, -back*outer)
	ray := maths.Ray{At: start, Dir: maths.ZAxis}
	enter, leave, ok := jet.Span(ray, nought, far*outer)
	if !ok || enter >= back*outer || leave <= back*outer {
		t.Fatalf("positive skirt past the one-width limit is clipped: "+
			"%g, %g, %v", enter, leave, ok)
	}
}
