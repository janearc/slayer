package medium

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// A puff is thickest in the middle, zero at and beyond its edge, and falls
// between without a step.
func TestPuff(t *testing.T) {
	p := Puff{Radius: 2, Thick: 3}
	if d := p.Density(maths.Vec{}); d != 3 {
		t.Errorf("the middle is %v", d)
	}
	for _, r := range []float64{2, 2.5, 10} {
		if d := p.Density(maths.V(r, 0, 0)); d != 0 {
			t.Errorf("%v out is %v", r, d)
		}
	}
	last := 3.0
	for r := 0.1; r < 2; r += 0.1 {
		d := p.Density(maths.V(0, r, 0))
		if d > last || last-d > 0.4 {
			t.Fatalf("at %v the density went %v then %v", r, last, d)
		}
		last = d
	}
	if _, _, ok := p.Bounds().Hit(maths.Ray{At: maths.V(0, 0, 5),
		Dir: maths.V(0, 0, -1)}, 0, math.Inf(1)); !ok {
		t.Error("a ray through the middle misses the bounds")
	}
}

// A wispy medium is never thicker than the one it wraps, is zero where that one
// is, and differs from place to place.
func TestWispy(t *testing.T) {
	p := Puff{Radius: 2, Thick: 1}
	w := Wispy{Of: p, Seed: 3, Scale: 0.5, Amount: 0.8}
	seen := map[float64]bool{}
	for i := range 50 {
		at := maths.V(float64(i)*0.03, 0.2, -0.1)
		d, full := w.Density(at), p.Density(at)
		if d > full+1e-12 || d < 0 {
			t.Fatalf("at %v wispy is %v, the puff %v", at, d, full)
		}
		seen[math.Round(d/full*1000)] = true
	}
	if len(seen) < 10 {
		t.Errorf("the wisps hardly vary: %d values", len(seen))
	}
	if w.Density(maths.V(5, 0, 0)) != 0 {
		t.Error("wisps outside the puff")
	}
	if (Wispy{}).Density(maths.Vec{}) != 0 {
		t.Error("a wispy nothing has density")
	}
}

// A giant is clear above its radius and thick a depth below it, with
// nothing in between but a smooth rise.
func TestGiant(t *testing.T) {
	g := Giant{Radius: 10, Depth: 1, Thick: 4}
	if d := g.Density(maths.V(10.5, 0, 0)); d != 0 {
		t.Errorf("above the top, %v", d)
	}
	if d := g.Density(maths.V(0, 8, 0)); d != 4 {
		t.Errorf("well below, %v", d)
	}
	if d := g.Density(maths.V(0, 0, 9.5)); !(d > 0 && d < 4) {
		t.Errorf("halfway down the rise, %v", d)
	}
	if d := g.Density(maths.Vec{}); d != 4 {
		t.Errorf("at the centre, %v", d)
	}
}

// A puff and a giant span a ray by their ball, not their box: a ray
// through the middle enters at the radius, a ray past the edge misses
// though it crosses the box, and a wispy puff spans as its puff does.
func TestSpanIsTheBall(t *testing.T) {
	var _ Spanner = Puff{}
	var _ Spanner = Giant{}
	var _ Spanner = Wispy{}
	down := maths.Ray{At: maths.V(0, 5, 0), Dir: maths.V(0, -1, 0)}
	for _, m := range []Spanner{Puff{Radius: 2}, Giant{Radius: 2, Depth: 1},
		Wispy{Of: Puff{Radius: 2}}} {
		enter, leave, ok := m.Span(down, 0, math.Inf(1))
		if !ok || math.Abs(enter-3) > 1e-12 || math.Abs(leave-7) > 1e-12 {
			t.Errorf("%T spans %v to %v, %v", m, enter, leave, ok)
		}
		corner := maths.Ray{At: maths.V(1.9, 5, 1.9), Dir: maths.V(0, -1, 0)}
		if _, _, ok := m.Span(corner, 0, math.Inf(1)); ok {
			t.Errorf("%T spans a ray through its box's corner", m)
		}
	}
}

// A jet with a flow carries its knots along: the density a way up the
// jet after a time is the density that far back before, and a jet
// without a flow is the same at every time.
func TestAJetsKnotsFlow(t *testing.T) {
	j := Jet{Length: 100, Base: 2, Spread: 0.05, Thick: 1, Knots: 1, Seed: 5, Flow: 1}
	later := j.Tick(3 * time.Second).(Jet)
	at := maths.V(0, 20, 0)
	if j.Density(at) == later.Density(at) {
		t.Errorf("the knots did not move")
	}
	// on the axis the density is the knot pattern times the fall off
	// along the jet; the pattern three base widths back before is the
	// pattern here now
	here, there := later.Density(at), j.Density(maths.V(0, 20-3*j.Base, 0))
	if math.Abs(here/(1-20/j.Length)-there/(1-14/j.Length)) > 1e-9 {
		t.Errorf("after three seconds the knot at 20 is %v; at 14 before it was %v", here, there)
	}
	still := Jet{Length: 100, Base: 2, Spread: 0.05, Thick: 1, Knots: 1, Seed: 5}
	if still.Density(at) != still.Tick(time.Hour).(Jet).Density(at) {
		t.Errorf("a jet without a flow moved")
	}
}
