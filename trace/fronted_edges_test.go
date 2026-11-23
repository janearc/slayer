package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// A ray that runs along the normal, a dot of exactly one, has no plane of
// the two to tilt the normal in, and a root of nothing. What it gets is a
// unit normal across the ray, tilted so that the ray meets it at minus
// frontLean: as good as any other, and a number and not the zero vector
// that taking the way off the normal leaves, which a mirror cannot use.
func TestFrontedGivesAUnitNormalWhereTheRayRunsAlongIt(t *testing.T) {
	normal := maths.V(0, 1, 0)
	got := fronted(normal, normal, normal)
	if math.IsNaN(got.Len()) || !within(got.Len(), 1, frontClose) ||
		!within(got.Dot(normal), -frontLean, frontClose) {
		t.Errorf("fronted %v meets the ray at %g, want a unit normal at %g",
			got, got.Dot(normal), -frontLean)
	}
}

// Anti: a ray exactly against the normal, a dot of exactly minus one,
// already faces it by more than frontLean, and the normal is left as it
// is: the same normal, not one across the ray.
func TestFrontedLeavesANormalTheRayRunsExactlyAgainst(t *testing.T) {
	normal := maths.V(0, 1, 0)
	if got := fronted(normal, normal, normal.Neg()); got != normal {
		t.Errorf("fronted %v, want the normal %v left as it is", got,
			normal)
	}
}

// A dot a hair over one, as two unit vectors can have by rounding, is
// taken as one: a unit normal, and no number that is not one.
func TestFrontedTakesADotAHairOverOneAsOne(t *testing.T) {
	way := maths.V(0.6, 0.8, 0).Unit()
	for _, normal := range []maths.Vec{way, way.Scale(1 + 1e-15)} {
		got := fronted(normal, normal, way)
		if math.IsNaN(got.Len()) || !within(got.Len(), 1, 1e-9) {
			t.Errorf("fronted %v from %v, want a unit normal", got, normal)
		}
	}
}

// Anti: a ray a hair off the normal is tilted as any other, to meet it at
// minus frontLean, and not taken for one along it. The tilt there is a
// billionth of a unit taken off a float's last place, so the cosine holds
// to a ten-millionth and not to a trillionth.
func TestFrontedTiltsANormalARayIsAHairOffToFrontLean(t *testing.T) {
	way := maths.V(0, 1, 0)
	normal := maths.V(1e-6, 1, 0).Unit()
	got := fronted(normal, normal, way)
	if !within(got.Dot(way), -frontLean, 1e-7) ||
		!within(got.Len(), 1, 1e-9) {
		t.Errorf("fronted %v meets the ray at %g, want %g", got,
			got.Dot(way), -frontLean)
	}
}
