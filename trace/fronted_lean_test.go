package trace

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// A normal the waves lean away from a grazing ray is tilted until the
// ray meets it at frontLean: that is what fronted says of itself, and
// what frontLean says it is, the cosine between a fronted normal and the
// ray. So the unit normal fronted gives back meets the ray at a cosine
// of minus frontLean, a thousandth, whatever the lean it began with.
//
// fronted sets that cosine and then makes the normal a unit long, which
// divides it by the square root of 1 - a^2 + frontLean^2, a the cosine
// the normal leant away by: 0.9345 for a normal of 0.5, 1, 0 and a way
// of 1, -0.1, 0, where a is 0.3560. Each normal here leans away from
// the ray, by 0.356, 0.852 and 0.094.
func TestFrontedMeetsTheRayAtFrontLean(t *testing.T) {
	way := maths.V(1, -0.1, 0).Unit()
	for _, normal := range []maths.Vec{maths.V(0.5, 1, 0).Unit(),
		maths.V(0.9, 0.4359, 0).Unit(), maths.V(0.2, 1, 0.3).Unit()} {
		got := fronted(normal, maths.V(0, 1, 0), way)
		if !within(got.Dot(way), -frontLean, frontClose) ||
			!within(got.Len(), 1, frontClose) {
			t.Errorf("fronted %v from %v meets the ray at %g, want %g",
				got, normal, got.Dot(way), -frontLean)
		}
	}
}

// Anti: a normal that already faces the ray by more than frontLean is
// not brought back to it: fronted tilts toward the ray and never away.
func TestFrontedDoesNotTiltAFacingNormalBackToFrontLean(t *testing.T) {
	way := maths.V(1, -1, 0).Unit()
	normal := maths.V(0, 1, 0)
	got := fronted(normal, normal, way)
	if got != normal || within(got.Dot(way), -frontLean, frontClose) {
		t.Errorf("fronted %v meets the ray at %g, want it left at %g",
			got, got.Dot(way), normal.Dot(way))
	}
}
