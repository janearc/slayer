package shape

import (
	"math"
	"testing"
)

// The tables the tests of this file read: points and rays, and the ranges
// a ray is asked over, each with a near and a far.
const (
	edgesVectorFile = "testdata/edges-vectors.csv"
	edgesRangeFile  = "testdata/edges-ranges.csv"
)

// edgesTube is the tube the tests of this file measure, from the rows of
// numbers.yaml that say its radius, wall and length.
func edgesTube(t *testing.T) Tube {
	t.Helper()
	return Tube{Radius: picked(t, "edges-tube-radius"),
		Thick:  picked(t, "edges-tube-wall"),
		Height: picked(t, "edges-tube-length")}
}

// the boxes, exactly: a sphere's cube, and a tube's, to the outside of its
// wall and the ends of its length.
func TestTheBoxesAreExact(t *testing.T) {
	vectors := shapeVectorsIn(t, edgesVectorFile)
	ball := Sphere{Radius: picked(t, "edges-sphere-radius")}
	if got, want := ball.Bounds(), vectors.box("sphere"); got != want {
		t.Errorf("a sphere of %v is in %v", ball.Radius, got)
	}
	tube := edgesTube(t)
	if got, want := tube.Bounds(), vectors.box("tube"); got != want {
		t.Errorf("a tube of %+v is in %v", tube, got)
	}
}

// along the x axis from five out, a unit sphere is met at 4 and left at
// 6. a near of exactly 4 is past the first, so the second is met; a far
// of exactly 4 is short of both. a ray at height one only touches it, at
// 5: met, but it crosses nothing.
func TestASphereAtTheEdgesOfItsRange(t *testing.T) {
	vectors := shapeVectorsIn(t, edgesVectorFile)
	ball := Sphere{Radius: unity}
	along := vectors.ray("along")
	meeting := picked(t, "edges-first-meeting")
	end := picked(t, "edges-range-end")
	if got, ok := ball.Hit(along, meeting, end); !ok ||
		got != picked(t, "edges-leaving") {
		t.Errorf("from a near of %v: %v %v", meeting, got, ok)
	}
	if got, ok := ball.Hit(along, nought, meeting); ok {
		t.Errorf("to a far of %v: %v", meeting, got)
	}
	touch := vectors.ray("touch")
	centre := picked(t, "edges-centre-distance")
	if got, ok := ball.Hit(touch, nought, end); !ok || got != centre {
		t.Errorf("a ray that touches: %v %v", got, ok)
	}
	if enter, leave, ok := ball.Span(touch, nought, end); ok {
		t.Errorf("a ray that touches spans %v to %v", enter, leave)
	}
	// at height a half, the roots are the centre's distance less and more
	// than the root of three quarters
	level := vectors.ray("level")
	enter, leave, ok := ball.Span(level, nought, end)
	root := math.Sqrt(picked(t, "edges-three-quarters"))
	if !ok || enter != centre-root || leave != centre+root {
		t.Errorf("at a half: %v to %v, %v", enter, leave, ok)
	}
}

// a ray slanting down from four high at four in five reaches the ground at
// five along it; one straight up from two below, at two; and only between
// near and far, exclusive of both.
func TestThePlaneAtTheEdgesOfItsRange(t *testing.T) {
	vectors := shapeVectorsIn(t, edgesVectorFile)
	var ground Plane
	end := picked(t, "edges-range-end")
	slant := vectors.ray("slant")
	got, ok := ground.Hit(slant, nought, end)
	if !ok || math.Abs(got-picked(t, "edges-slant-distance")) >
		picked(t, "edges-slant-tolerance") {
		t.Errorf("slanting: %v %v", got, ok)
	}
	up := vectors.ray("up")
	got, ok = ground.Hit(up, nought, end)
	if !ok || got != picked(t, "edges-up-distance") {
		t.Errorf("straight up: %v %v", got, ok)
	}
	for _, each := range readSheet(t, edgesRangeFile) {
		near, far := cell(t, each, "near"), cell(t, each, "far")
		if got, ok := ground.Hit(up, near, far); ok {
			t.Errorf("between %v and %v: met at %v", near, far, got)
		}
	}
}

// marched, a tube's wall a tenth from its end faces straight out: the step
// is small enough not to reach round the corner.
func TestAMarchedNormalStaysOnItsFace(t *testing.T) {
	vectors := shapeVectorsIn(t, edgesVectorFile)
	type marched struct{ Shape }
	hidden := marched{edgesTube(t)}
	got := Normal(hidden, vectors.point("near-the-end"))
	tolerance := picked(t, "edges-marched-tolerance")
	if !got.Near(vectors.point("outward"), tolerance) {
		t.Errorf("the wall a tenth from its end faces %v", got)
	}
}
