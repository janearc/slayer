package shape

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// the exact answer and the marched answer have to agree, or one of them is
// lying about the shape.
func TestSphereHitAgreesWithItsDistance(t *testing.T) {
	s := Sphere{Radius: 2}
	r := maths.Ray{At: maths.V(0, 0, 10), Dir: maths.V(0, 0, -1)}
	got, ok := s.Hit(r, 0, math.Inf(1))
	if !ok || math.Abs(got-8) > 1e-9 {
		t.Fatalf("hit at %v, %v; want 8", got, ok)
	}
	if d := s.Distance(r.Point(got)); math.Abs(d) > 1e-9 {
		t.Fatalf("the point it hit is %v from the surface", d)
	}
}

func TestARayFromInsideMeetsTheFarSide(t *testing.T) {
	s := Sphere{Radius: 1}
	got, ok := s.Hit(maths.Ray{Dir: maths.XAxis}, 1e-9, math.Inf(1))
	if !ok || math.Abs(got-1) > 1e-9 {
		t.Fatalf("hit at %v, %v; want 1", got, ok)
	}
}

func TestAMissIsAMiss(t *testing.T) {
	s := Sphere{Radius: 1}
	if _, ok := s.Hit(maths.Ray{At: maths.V(0, 5, 10), Dir: maths.V(0, 0, -1)}, 0, math.Inf(1)); ok {
		t.Fatal("a ray passing four units above the sphere hit it")
	}
}

func TestThePlaneFacesUpAndHasNoEdge(t *testing.T) {
	p := Plane{}
	r := maths.Ray{At: maths.V(1e6, 3, -1e6), Dir: maths.V(0, -1, 0)}
	if got, ok := p.Hit(r, 0, math.Inf(1)); !ok || got != 3 {
		t.Fatalf("hit at %v, %v; want 3", got, ok)
	}
	if _, _, ok := p.Bounds().Hit(r, 0, math.Inf(1)); !ok {
		t.Fatal("the plane's box missed a ray that meets the plane")
	}
}

func TestTheMarchedNormalMatchesTheExactOne(t *testing.T) {
	s := Sphere{Radius: 1}
	p := maths.V(1, 1, 1).Unit()
	marched := Normal(struct{ Shape }{s}, p) // hides Normaler
	if !marched.Near(s.Normal(p), 1e-6) {
		t.Fatalf("marched %v, exact %v", marched, s.Normal(p))
	}
}

// A sphere spans a ray from where it enters to where it leaves, and a
// ray that misses spans nothing.
func TestSphereSpan(t *testing.T) {
	s := Sphere{Radius: 2}
	r := maths.Ray{At: maths.V(0, 0, 10), Dir: maths.V(0, 0, -1)}
	enter, leave, ok := s.Span(r, 0, math.Inf(1))
	if !ok || math.Abs(enter-8) > 1e-9 || math.Abs(leave-12) > 1e-9 {
		t.Fatalf("spans %v to %v, %v; want 8 to 12", enter, leave, ok)
	}
	if _, _, ok := s.Span(maths.Ray{At: maths.V(0, 5, 10), Dir: maths.V(0, 0, -1)},
		0, math.Inf(1)); ok {
		t.Fatal("a miss spans the sphere")
	}
}
