package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// Pillars have a dense wall along the bottom, columns above it that end
// below the top, and thin fog between; nothing beyond the box.
func TestPillarsStandOnAWall(t *testing.T) {
	p := Pillars{Count: 3, Height: 10, Width: 10, Thick: 1, Seed: 4}
	if d := p.Density(maths.V(2, 0.5, 2)); d < 0.9 {
		t.Errorf("the wall is %v", d)
	}
	// somewhere at half height a column is dense, and most of the row is fog
	dense, thin := 0, 0
	for x := -4.9; x < 5; x += 0.1 {
		switch d := p.Density(maths.V(x, 5, 0)); {
		case d > 0.9:
			dense++
		case d < 0.02:
			thin++
		}
	}
	if dense < 5 || thin < 40 {
		t.Errorf("at half height %d dense and %d thin of 100", dense, thin)
	}
	if d := p.Density(maths.V(0, 9.95, 0)); d > 0.02 {
		t.Errorf("the top is %v", d)
	}
	if p.Density(maths.V(0, -1, 0)) != 0 || p.Density(maths.V(6, 5, 0)) != 0 {
		t.Error("gas beyond the box")
	}
}

// The tips are the tops of the columns, one each, inside the box and
// above half height; filaments thin the gas inside a column in
// streaks without emptying it.
func TestPillarsHaveTipsAndFilaments(t *testing.T) {
	p := Pillars{Count: 3, Height: 10, Width: 10, Thick: 1, Seed: 4}
	tips := p.Tips()
	if len(tips) != 3 {
		t.Fatalf("%d tips", len(tips))
	}
	for _, tip := range tips {
		if tip.Y < 5 || tip.Y > 10 || math.Abs(tip.X) > 5 || math.Abs(tip.Z) > 5 {
			t.Errorf("a tip at %v", tip)
		}
		if d := p.Density(tip.Add(maths.V(0, -1.5, 0))); d < 0.3 {
			t.Errorf("under the tip at %v the gas is %v", tip, d)
		}
	}
	streaked := p
	streaked.Filaments = 0.8
	c := tips[0]
	low, high, n := 1.0, 0.0, 0
	for y := 1.0; y < c.Y-2; y += 0.2 {
		d := streaked.Density(maths.V(c.X, y, c.Z))
		low, high = math.Min(low, d), math.Max(high, d)
		if p.Density(maths.V(c.X, y, c.Z)) > 0.9 {
			n++
		}
	}
	if n < 10 || !(high > 0.8 && low < 0.7*high) {
		t.Errorf("filaments run the column from %.2f to %.2f over %d dense points", low, high, n)
	}
}
