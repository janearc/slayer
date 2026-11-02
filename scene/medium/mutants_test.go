package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// closeTo is whether two numbers are within e of each other. a NaN is
// close to nothing, so a test that wants a number fails on one.
func closeTo(a, b, e float64) bool { return math.Abs(a-b) <= e }

// an air of no height has no air to thin: it reaches Reach heights, which is
// zero, and past that it is taken as nothing. at its surface that is zero, not
// the zero over zero of exp(-up/height).
func TestAirOfNoHeightIsNone(t *testing.T) {
	a := Air{Radius: 10, Height: 0, Thick: 2}
	for _, at := range []maths.Vec{maths.V(10, 0, 0), maths.V(10.5, 0, 0),
		maths.V(9, 0, 0)} {
		if d := a.Density(at); d != 0 {
			t.Errorf("at %v an air of no height is %v", at, d)
		}
	}
}

// the air is counted out to Reach heights and past that it is nothing,
// so at exactly Reach heights it is still there: Thick times exp(-Reach).
// with a height of a half and a radius of ten, the top is 13 and 3 up is
// 3 over 0.5 = 6 heights, so the density is 2e^-6.
func TestAirIsStillThereAtItsTop(t *testing.T) {
	a := Air{Radius: 10, Height: 0.5, Thick: 2}
	if d := a.Density(maths.V(13, 0, 0)); !closeTo(d, 2*math.Exp(-6), 1e-15) {
		t.Errorf("at the top the air is %v, want %v", d, 2*math.Exp(-6))
	}
}

// a puff of no radius has nothing: it thins to nothing at its edge, and
// its edge is its middle. a puff of radius one is Thick at the middle, as
// any radius is, and at half its radius is Thick times (1 - Smooth(0, 1,
// 0.5)) = Thick times 0.5.
func TestPuffAtItsRadii(t *testing.T) {
	if d := (Puff{Radius: 0, Thick: 3}).Density(maths.Vec{}); d != 0 {
		t.Errorf("a puff of no radius is %v at its middle", d)
	}
	if d := (Puff{Radius: 1, Thick: 3}).Density(maths.Vec{}); d != 3 {
		t.Errorf("a puff of radius one is %v at its middle", d)
	}
	if d := (Puff{Radius: 2, Thick: 3}).Density(maths.V(0, 1, 0)); !closeTo(d, 1.5, 1e-15) {
		t.Errorf("half way out a puff is %v, want 1.5", d)
	}
}

// the noise is zero at the middle of space whatever the seed and the scale
// (gradient noise is zero on every lattice point, and the middle is the lattice
// point every octave keeps), so a wispy puff there is Thick thinned by Amount
// of its half: Thick times (1 - Amount * 0.5).
//
// a Thick of one is checked too, since one is a number the wrapped density
// could be mistaken for.
func TestWispyAtTheMiddleOfSpace(t *testing.T) {
	for _, thick := range []float64{1, 1.5} {
		for seed := range uint64(10) {
			for _, scale := range []float64{0, 0.5, 3} {
				w := Wispy{Of: Puff{Radius: 2, Thick: thick}, Seed: seed,
					Scale: scale, Amount: 0.6}
				want := thick * (1 - 0.6*0.5)
				if d := w.Density(maths.Vec{}); !closeTo(d, want, 1e-15) {
					t.Errorf("seed %d scale %v thick %v: %v, want %v", seed,
						scale, thick, d, want)
				}
			}
		}
	}
}

// a wispy nothing has the bounds of nothing, not a panic.
func TestWispyOfNothingHasNoBounds(t *testing.T) {
	if b := (Wispy{}).Bounds(); b != (maths.Box{}) {
		t.Errorf("a wispy nothing is bounded by %v", b)
	}
}

// a giant of no depth has no rise to thicken over, so it is clear; and a
// giant of radius one is Thick at its centre, a depth below the top:
// Smooth(0, 0.5, 1) is one.
func TestGiantOfNoDepthAndOfRadiusOne(t *testing.T) {
	g := Giant{Radius: 10, Depth: 0, Thick: 4}
	for _, at := range []maths.Vec{{}, maths.V(0, 9, 0), maths.V(0, 9.99, 0)} {
		if d := g.Density(at); d != 0 {
			t.Errorf("a giant of no depth is %v at %v", d, at)
		}
	}
	if d := (Giant{Radius: 1, Depth: 0.5, Thick: 4}).Density(maths.Vec{}); d != 4 {
		t.Errorf("a giant of radius one is %v at its centre", d)
	}
}

// a plume of no height or no width has no column to be dense in, so it is clear
// at the foot of where its column would stand, not the zero over zero of a
// ratio to nothing.
func TestPlumeOfNoSizeIsNone(t *testing.T) {
	flat := Plume{Height: 0, Width: 20, Thick: 1}
	if d := flat.Density(maths.Vec{}); d != 0 {
		t.Errorf("a plume of no height is %v at its foot", d)
	}
	thin := Plume{Height: 10, Width: 0, Thick: 1}
	if d := thin.Density(maths.V(0, 5, 0)); d != 0 {
		t.Errorf("a plume of no width is %v on its axis", d)
	}
}

// Thick is the density at the column's heart, so a plume a unit high or a
// unit wide is Thick on its axis a thousandth up from its foot, to within
// the column's thinning there, a twentieth of a thousandth of Thick
// (1 - 0.5h at h = 0.001) and the umbrella's far smaller tail.
func TestSmallPlumesAreThickOnTheAxis(t *testing.T) {
	for _, p := range []Plume{{Height: 1, Width: 20, Thick: 1},
		{Height: 10, Width: 1, Thick: 1}} {
		at := maths.V(0, 0.001*p.Height, 0)
		if d := p.Density(at); !closeTo(d, 1, 1e-3) {
			t.Errorf("%+v is %v on its axis at %v", p, d, at)
		}
	}
}

// a jet a unit long still has its density, Thick times (1 - y over Length) on
// the axis: a quarter up, three quarters of 3 is 2.25. a jet of no length has
// none. a jet that starts at From has its full density at From, not zero: at
// 1.5 of 10, 3 times 0.85.
func TestJetOfLengthOneAndFrom(t *testing.T) {
	short := Jet{Length: 1, Base: 2, Thick: 3}
	if d := short.Density(maths.V(0, 0.25, 0)); !closeTo(d, 2.25, 1e-15) {
		t.Errorf("a jet a unit long is %v a quarter up, want 2.25", d)
	}
	if d := (Jet{Length: 0, Base: 2, Thick: 3}).Density(maths.Vec{}); d != 0 {
		t.Errorf("a jet of no length is %v", d)
	}
	from := Jet{Length: 10, Base: 2, Thick: 3, From: 1.5}
	if d := from.Density(maths.V(0, 1.5, 0)); !closeTo(d, 2.55, 1e-12) {
		t.Errorf("at From the jet is %v, want 2.55", d)
	}
}

// a jet of no base is no width at its origin: zero there, not the 0/0 of a
// ratio to nothing.
func TestJetOfNoBaseIsNoneAtItsOrigin(t *testing.T) {
	j := Jet{Length: 10, Base: 0, Spread: 0.1, Thick: 1}
	if d := j.Density(maths.Vec{}); d != 0 {
		t.Errorf("a jet of no base is %v at its origin", d)
	}
}

// past two and a half widths there is nothing and at exactly two and a
// half there is still the gaussian's exp(-2 * 2.5^2) = exp(-12.5), of the
// jet and the tail both: a width of one at the origin, a point 2.5 out.
func TestTheGaussianReachesTwoAndAHalfWidths(t *testing.T) {
	want := math.Exp(-12.5)
	at := maths.V(2.5, 0, 0)
	if d := (Jet{Length: 10, Base: 2, Thick: 1}).Density(at); !closeTo(d, want, 1e-18) {
		t.Errorf("the jet at 2.5 widths is %v, want %v", d, want)
	}
	if d := (Tail{Length: 10, Base: 2, Thick: 1}).Density(at); !closeTo(d, want, 1e-18) {
		t.Errorf("the tail at 2.5 widths is %v, want %v", d, want)
	}
}

// a jet's knots are each side's own and the upper side is the one at y of zero:
// there the knot noise is read at the origin, where it is zero for any seed, so
// a knotted jet on its axis at its origin is its plain Thick.
func TestAKnottedJetIsPlainAtItsOrigin(t *testing.T) {
	for seed := range uint64(16) {
		j := Jet{Length: 10, Base: 1, Thick: 2, Knots: 1, Seed: seed}
		if d := j.Density(maths.Vec{}); d != 2 {
			t.Errorf("seed %d: the knotted jet is %v at its origin, want 2",
				seed, d)
		}
	}
}

// a tail is Thick at its head, y of zero; a tail a unit long has its density, a
// quarter up three quarters of 3; a tail of no base is no width at its head, so
// zero there and not 0/0.
func TestTailAtItsHeadAndOfLengthOne(t *testing.T) {
	if d := (Tail{Length: 10, Base: 2, Thick: 3}).Density(maths.Vec{}); d != 3 {
		t.Errorf("the head is %v, want 3", d)
	}
	short := Tail{Length: 1, Base: 2, Thick: 3}
	if d := short.Density(maths.V(0, 0.25, 0)); !closeTo(d, 2.25, 1e-15) {
		t.Errorf("a tail a unit long is %v a quarter along, want 2.25", d)
	}
	bare := Tail{Length: 10, Base: 0, Spread: 0.1, Thick: 1}
	if d := bare.Density(maths.Vec{}); d != 0 {
		t.Errorf("a tail of no base is %v at its head", d)
	}
}
