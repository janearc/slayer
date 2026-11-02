package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

var rings = Disc{Inner: 2, Outer: 4, Thick: 0.1,
	Profile: []float64{1, 1, 0, 1, 1}} // a gap in the middle

// A disc is a medium with a span: zero inside its hole, above its faces and
// beyond its edge; its profile's density in the ring; zero in the gap; and a
// ray straight down through the ring crosses its slab.
func TestDisc(t *testing.T) {
	var _ Spanner = Disc{}
	for _, c := range []struct {
		p    maths.Vec
		want float64
	}{
		{maths.V(3, 0, 0), 0},     // the gap, at t of a half
		{maths.V(2.2, 0, 0), 1},   // the inner ring
		{maths.V(0, 0, 3.8), 1},   // the outer ring
		{maths.V(1, 0, 0), 0},     // the hole
		{maths.V(5, 0, 0), 0},     // beyond the edge
		{maths.V(2.2, 0.2, 0), 0}, // above the face
	} {
		if got := rings.Density(c.p); math.Abs(got-c.want) > 1e-12 {
			t.Errorf("at %v: %v, want %v", c.p, got, c.want)
		}
	}
	down := maths.Ray{At: maths.V(2.5, 1, 0), Dir: maths.V(0, -1, 0)}
	enter, leave, ok := rings.Span(down, 0, math.Inf(1))
	if !ok || math.Abs(enter-0.95) > 1e-12 || math.Abs(leave-1.05) > 1e-12 {
		t.Errorf("down through the ring spans %v to %v, %v", enter, leave, ok)
	}
	miss := maths.Ray{At: maths.V(5, 1, 0), Dir: maths.V(0, -1, 0)}
	if _, _, ok := rings.Span(miss, 0, math.Inf(1)); ok {
		t.Error("a ray beyond the edge spans the disc")
	}
	along := maths.Ray{At: maths.V(-10, 0, 0), Dir: maths.V(1, 0, 0)}
	enter, leave, ok = rings.Span(along, 0, math.Inf(1))
	if !ok || math.Abs(enter-6) > 1e-12 || math.Abs(leave-14) > 1e-12 {
		t.Errorf("along the disc spans %v to %v, %v", enter, leave, ok)
	}
}

// Ringlets thin the disc from place to place and never thicken it, and a
// disc with no profile is one everywhere.
func TestRinglets(t *testing.T) {
	plain := Disc{Inner: 2, Outer: 4, Thick: 0.1}
	if got := (Disc{Inner: 2, Outer: 4, Thick: 0.1, Dense: 30}).
		Density(maths.V(3, 0, 0)); got != 30 {
		t.Fatalf("dense is %v", got)
	}
	grainy := plain
	grainy.Grain, grainy.Seed = 0.8, 5
	seen := map[float64]bool{}
	for r := 2.05; r < 3.95; r += 0.01 {
		p := maths.V(r, 0, 0)
		if plain.Density(p) != 1 {
			t.Fatalf("a plain disc is %v at %v", plain.Density(p), r)
		}
		g := grainy.Density(p)
		if g > 1 || g < 0 {
			t.Fatalf("ringlets gave %v at %v", g, r)
		}
		seen[math.Round(g*100)] = true
	}
	if len(seen) < 20 {
		t.Errorf("the ringlets hardly vary: %d values", len(seen))
	}
}
