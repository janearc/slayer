package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// frontClose is how near two numbers are taken to agree, picked.
const frontClose = 1e-12

// A normal that faces the ray from its side is left as it is.
func TestFrontedLeavesAFacingNormal(t *testing.T) {
	normal := maths.V(0.1, 1, 0).Unit()
	way := maths.V(1, -1, 0).Unit()
	if got := fronted(normal, maths.V(0, 1, 0), way); got != normal {
		t.Errorf("fronted %v, want %v", got, normal)
	}
}

// A normal the waves lean away from a grazing ray is tilted toward the
// ray, in the plane of the two, until the ray meets it from in front, so
// a mirror about it leaves the surface. The unit normal meets the ray at
// exactly minus frontLean, and no other length is taken for it.
//
// Here the normal 0.5, 1, 0 and the way 1, -0.1, 0 make a = 0.3560, and
// the tilt takes off the way times a + 0.0009345, which is frontLean
// times the root of (1 - a^2) over (1 - frontLean^2): a unit normal
// after that, at a cosine of minus a thousandth.
func TestFrontedTiltsAGrazingNormalInItsPlaneToFrontLean(t *testing.T) {
	normal := maths.V(0.5, 1, 0).Unit()
	way := maths.V(1, -0.1, 0).Unit()
	got := fronted(normal, maths.V(0, 1, 0), way)
	want := -frontLean
	if !within(got.Dot(way), want, frontClose) ||
		!within(got.Len(), 1, frontClose) {
		t.Errorf("fronted %v meets the ray at %g, want %g", got,
			got.Dot(way), want)
	}
	if out := got.Dot(normal.Cross(way)); !within(out, 0, frontClose) {
		t.Errorf("fronted %v leaves the plane of the normal and the "+
			"ray by %g", got, out)
	}
	normalPart := got.Scale(2 * way.Dot(got))
	mirror := way.Sub(normalPart)
	if !(mirror.Dot(got) > 0) {
		t.Errorf("mirror %v goes into the surface", mirror)
	}
}

// Anti: a normal on the other side from side is turned to it first.
func TestFrontedTurnsANormalToItsSide(t *testing.T) {
	got := fronted(maths.V(0, -1, 0), maths.V(0, 1, 0),
		maths.V(0, -1, 0))
	if !(got.Y > 0) {
		t.Errorf("fronted %v, want it facing up", got)
	}
}

// A clear height field is water, and mirrors the sky past its last
// bounce; anti: a clear plane, and an opaque height field, are not.
func TestWateryIsWaterOnly(t *testing.T) {
	clear := &thing.Material{Through: 1}
	for _, each := range []struct {
		shape    shape.Shape
		material *thing.Material
		water    bool
	}{
		{&shape.HeightField{}, clear, true},
		{shape.Plane{}, clear, false},
		{&shape.HeightField{}, &thing.Material{}, false},
	} {
		met := hit{thing: placedThing{thing: &thing.Thing{
			Shape: each.shape, Material: each.material}}}
		if met.watery() != each.water {
			t.Errorf("%T of %+v: watery %t", each.shape, each.material,
				!each.water)
		}
	}
}

// Roughened adds what is lost to a copy, and leaves the material a look
// is inside alone.
func TestRoughenedWidensACopy(t *testing.T) {
	made := &thing.Material{SlopeVariance: 0.01}
	got := roughened(made, 0.02)
	if got == made || math.Abs(got.SlopeVariance-0.03) > frontClose ||
		made.SlopeVariance != 0.01 {
		t.Errorf("roughened %+v from %+v", got, made)
	}
	if roughened(made, 0) != made {
		t.Error("roughened by nought is a copy")
	}
}
