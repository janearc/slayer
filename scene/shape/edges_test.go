package shape

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// the boxes, exactly: a sphere's cube, and a tube's, to the outside of its
// wall and the ends of its length.
func TestTheBoxesAreExact(t *testing.T) {
	if got, want := (Sphere{Radius: 2}).Bounds(), (maths.Box{Min: maths.V(-2, -2, -2),
		Max: maths.V(2, 2, 2)}); got != want {
		t.Errorf("a sphere of 2 is in %v", got)
	}
	if got, want := (Tube{Radius: 10, Thick: 2, Height: 4}).Bounds(),
		(maths.Box{Min: maths.V(-11, -2, -11), Max: maths.V(11, 2, 11)}); got != want {
		t.Errorf("a tube of 10, 2 thick and 4 long, is in %v", got)
	}
}

// along the x axis from five out, a unit sphere is met at 4 and left at
// 6. a near of exactly 4 is past the first, so the second is met; a far
// of exactly 4 is short of both. a ray at height one only touches it, at
// 5: met, but it crosses nothing.
func TestASphereAtTheEdgesOfItsRange(t *testing.T) {
	s, along := Sphere{Radius: 1}, maths.Ray{At: maths.V(-5, 0, 0), Dir: maths.V(1, 0, 0)}
	if got, ok := s.Hit(along, 4, 100); !ok || got != 6 {
		t.Errorf("from a near of 4: %v %v", got, ok)
	}
	if got, ok := s.Hit(along, 0, 4); ok {
		t.Errorf("to a far of 4: %v", got)
	}
	touch := maths.Ray{At: maths.V(-5, 1, 0), Dir: maths.V(1, 0, 0)}
	if got, ok := s.Hit(touch, 0, 100); !ok || got != 5 {
		t.Errorf("a ray that touches: %v %v", got, ok)
	}
	if enter, leave, ok := s.Span(touch, 0, 100); ok {
		t.Errorf("a ray that touches spans %v to %v", enter, leave)
	}
	// at height a half, the roots are 5 less and more than the root of 0.75
	half := maths.Ray{At: maths.V(-5, 0.5, 0), Dir: maths.V(1, 0, 0)}
	enter, leave, ok := s.Span(half, 0, 100)
	if r := math.Sqrt(0.75); !ok || enter != 5-r || leave != 5+r {
		t.Errorf("at a half: %v to %v, %v", enter, leave, ok)
	}
}

// a ray slanting down from four high at four in five reaches the ground at
// five along it; one straight up from two below, at two; and only between
// near and far, exclusive of both.
func TestThePlaneAtTheEdgesOfItsRange(t *testing.T) {
	var p Plane
	slant := maths.Ray{At: maths.V(0, 4, 0), Dir: maths.V(0.6, -0.8, 0)}
	if got, ok := p.Hit(slant, 0, 100); !ok || math.Abs(got-5) > 1e-15 {
		t.Errorf("slanting: %v %v", got, ok)
	}
	up := maths.Ray{At: maths.V(0, -2, 0), Dir: maths.V(0, 1, 0)}
	if got, ok := p.Hit(up, 0, 100); !ok || got != 2 {
		t.Errorf("straight up: %v %v", got, ok)
	}
	for _, nf := range [][2]float64{{2, 100}, {0, 2}, {3, 100}, {0, 1}} {
		if got, ok := p.Hit(up, nf[0], nf[1]); ok {
			t.Errorf("between %v and %v: met at %v", nf[0], nf[1], got)
		}
	}
}

// marched, a tube's wall a tenth from its end faces straight out: the step
// is small enough not to reach round the corner.
func TestAMarchedNormalStaysOnItsFace(t *testing.T) {
	type marched struct{ Shape }
	got := Normal(marched{Tube{Radius: 10, Thick: 2, Height: 4}}, maths.V(11, 1.9, 0))
	if !got.Near(maths.V(1, 0, 0), 1e-6) {
		t.Errorf("the wall a tenth from its end faces %v", got)
	}
}
