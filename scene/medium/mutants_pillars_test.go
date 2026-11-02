package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// the wall along the bottom is Thick, a fifth of the height up and a little
// ragged: on the middle of the base the noise that ragged it is zero (it is
// read at the middle of space there, whatever the height), and the wall is
// whole up to 0.12 of the height, where it begins to ease.
//
// no pillar is above one, so the density there is Thick. pillars of no count,
// height or width are none; of one of any of them, they are not.
func TestPillarsAreAWallAtTheFoot(t *testing.T) {
	for _, p := range []Pillars{
		{Count: 3, Height: 10, Width: 20, Thick: 2, Seed: 4},
		{Count: 1, Height: 10, Width: 20, Thick: 2, Seed: 4},
		{Count: 3, Height: 1, Width: 20, Thick: 2, Seed: 4},
		{Count: 3, Height: 10, Width: 1, Thick: 2, Seed: 4},
	} {
		if d := p.Density(maths.V(0, 0.1*p.Height, 0)); d != 2 {
			t.Errorf("%+v: the wall is %v, want 2", p, d)
		}
	}
	for _, p := range []Pillars{
		{Count: 3, Height: 0, Width: 20, Thick: 2, Seed: 4},
		{Count: 3, Height: 10, Width: 0, Thick: 2, Seed: 4},
		{Count: 0, Height: 10, Width: 20, Thick: 2, Seed: 4},
	} {
		if d := p.Density(maths.V(0, 0, 0)); d != 0 {
			t.Errorf("%+v: %v at the foot, want none", p, d)
		}
		if d := p.Density(maths.V(0, 0.1*p.Height, 0)); d != 0 {
			t.Errorf("%+v: %v in the wall, want none", p, d)
		}
	}
}

// the box the pillars stand in is closed: on the floor of it is the wall,
// Thick, and at its top, and at its sides, the fog, a three-thousandth of
// Thick, where no column reaches (a column's height is under the whole).
func TestThePillarsBoxIsClosed(t *testing.T) {
	p := Pillars{Count: 3, Height: 10, Width: 20, Thick: 2, Seed: 4}
	if d := p.Density(maths.Vec{}); d != 2 {
		t.Errorf("on the floor the wall is %v, want 2", d)
	}
	for _, at := range []maths.Vec{maths.V(0, 10, 0), maths.V(10, 10, 0),
		maths.V(-10, 10, 0), maths.V(0, 10, 10), maths.V(0, 10, -10)} {
		if d := p.Density(at); !closeTo(d, 2*0.003, 1e-15) {
			t.Errorf("at %v the fog is %v, want %v", at, d, 2*0.003)
		}
	}
}

// at the very height of a column's tip its dome has closed to a point,
// so the edge there is the noise's shift alone: the distance from the
// middle moves by (n - 0.5) * 0.9 * base, and the column is 1 - Smooth(
// -0.3 base, 0, that) where the point is at its middle.
//
// for a single pillar the tip is Thick times the larger of that and the
// fog. n is four octaves of maths/noise at the tip scaled by 1.5 over the
// base, for the seed plus ten. over fifty seeds the tip is the fog for
// some and well above it for others.
func TestAColumnsTipIsItsNoise(t *testing.T) {
	above := 0
	for seed := range uint64(50) {
		p := Pillars{Count: 1, Height: 10, Width: 20, Thick: 1, Seed: seed}
		tip := p.Tips()[0]
		base := p.Width / 3.2
		f := 1.5 / base
		n := noise.Fractal(seed+10, tip.X*f, tip.Y*f, tip.Z*f, 4)
		shift := (n - 0.5) * 0.9 * base
		want := math.Max(0.003, 1-maths.Smooth(-0.3*base, 0, shift))
		if want > 0.01 {
			above++
		}
		if d := p.Density(tip); !closeTo(d, want, 1e-6) {
			t.Errorf("seed %d: the tip is %v, want %v", seed, d, want)
		}
	}
	if above < 10 {
		t.Fatalf("only %d of fifty tips were above the fog", above)
	}
}

// a column of gas is thinned by its filaments and never thickened, so a
// negative Filaments is none: over a spread of points through the columns,
// a Filaments of minus one is the density of none.
func TestANegativeFilamentsIsNone(t *testing.T) {
	plain := Pillars{Count: 3, Height: 10, Width: 20, Thick: 1, Seed: 6}
	negative := plain
	negative.Filaments = -1
	base := plain.Width / (3 * 3.2)
	for _, c := range plain.columns() {
		for i := range int64(4000) {
			at := maths.V(c.x+(noise.Hash(1, i)-0.5)*base,
				noise.Hash(2, i)*c.height,
				c.z+(noise.Hash(3, i)-0.5)*base)
			if n, p := negative.Density(at), plain.Density(at); n != p {
				t.Fatalf("at %v a Filaments of minus one is %v, none %v", at, n, p)
			}
		}
	}
}

// the filaments take Filaments of the gas where the streak is deepest and
// no more: a Filaments of a half leaves half of a column's whole gas
// there.
//
// the streak noise is below minus a half at these two points in a single
// pillar's body, where the streak is deepest. they were found by search
// and the test checks that they still are, and that the body is whole.
func TestFilamentsTakeNoMoreThanTheirShare(t *testing.T) {
	plain := Pillars{Count: 1, Height: 10, Width: 20, Thick: 1, Seed: 6}
	half := plain
	half.Filaments = 0.5
	base := plain.columns()[0].base
	g := 3 / base
	for _, at := range []maths.Vec{
		maths.V(-3.5034349961134796, 3.2501261627607625, -2.2284598309482697),
		maths.V(-3.5239823779911186, 3.0350469601625956, -2.2265621113066976),
	} {
		m := noise.Fractal(plain.Seed+10+50, at.X*g, at.Y*g/3, at.Z*g, 4)
		if m >= -0.5 {
			t.Fatalf("at %v the streak noise is %v, not below minus a half", at, m)
		}
		if d := plain.Density(at); d != 1 {
			t.Fatalf("at %v the column is %v, not whole", at, d)
		}
		if d := half.Density(at); !closeTo(d, 0.5, 1e-12) {
			t.Errorf("at %v half the filaments leave %v, want 0.5", at, d)
		}
	}
}
