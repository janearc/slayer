package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// closeTo is whether two numbers are within tolerance of each other. a NaN is
// close to nothing, so a test that wants a number fails on one.
func closeTo(first, second, tolerance float64) bool {
	return math.Abs(first-second) <= tolerance
}

// mutantsOrigin is the middle of space.
func mutantsOrigin() maths.Vec { return maths.V(nought, nought, nought) }

// mutantsAlong is the point on the x axis at a distance from the origin.
func mutantsAlong(distance float64) maths.Vec {
	return maths.V(distance, nought, nought)
}

// mutantsUp is the point on the y axis at a height above the origin.
func mutantsUp(height float64) maths.Vec {
	return maths.V(nought, height, nought)
}

// an air of no height has no air to thin: it reaches Reach heights, which is
// zero, and past that it is taken as nothing. at its surface that is zero, not
// the zero over zero of exp(-up/height).
func TestAirOfNoHeightIsNone(t *testing.T) {
	radius := picked(t, "mutants-air-radius")
	thick := picked(t, "mutants-air-thick")
	air := Air{Radius: radius, Height: nought, Thick: thick}
	past := picked(t, "mutants-air-past-surface")
	inside := picked(t, "mutants-air-inside")
	for _, at := range []maths.Vec{mutantsAlong(radius), mutantsAlong(past),
		mutantsAlong(inside)} {
		if density := air.Density(at); density != nought {
			t.Errorf("at %v an air of no height is %v", at, density)
		}
	}
}

// the air is counted out to Reach heights and past that it is nothing,
// so at exactly Reach heights it is still there: Thick times exp(-Reach).
// with a height of a half and a radius of ten, the top is 13 and 3 up is
// 3 over 0.5 = 6 heights, so the density is 2e^-6.
func TestAirIsStillThereAtItsTop(t *testing.T) {
	thick := picked(t, "mutants-air-thick")
	air := Air{Radius: picked(t, "mutants-air-radius"), Height: halfway,
		Thick: thick}
	want := thick * math.Exp(-picked(t, "mutants-air-top-heights"))
	top := mutantsAlong(picked(t, "mutants-air-top"))
	fine := picked(t, "mutants-tolerance-fine")
	if got := air.Density(top); !closeTo(got, want, fine) {
		t.Errorf("at the top the air is %v, want %v", got, want)
	}
}

// a puff of no radius has nothing: it thins to nothing at its edge, and
// its edge is its middle. a puff of radius one is Thick at the middle, as
// any radius is, and at half its radius is Thick times (1 - Smooth(0, 1,
// 0.5)) = Thick times 0.5.
func TestPuffAtItsRadii(t *testing.T) {
	thick := picked(t, "mutants-puff-thick")
	middle := mutantsOrigin()
	none := Puff{Radius: nought, Thick: thick}
	if got := none.Density(middle); got != nought {
		t.Errorf("a puff of no radius is %v at its middle", got)
	}
	unit := Puff{Radius: unity, Thick: thick}
	if got := unit.Density(middle); got != thick {
		t.Errorf("a puff of radius one is %v at its middle", got)
	}
	wide := Puff{Radius: picked(t, "mutants-puff-wide-radius"), Thick: thick}
	want := thick * halfway
	fine := picked(t, "mutants-tolerance-fine")
	if got := wide.Density(mutantsUp(unity)); !closeTo(got, want, fine) {
		t.Errorf("half way out a puff is %v, want %v", got, want)
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
	amount := picked(t, "mutants-wispy-amount")
	radius := picked(t, "mutants-puff-wide-radius")
	fine := picked(t, "mutants-tolerance-fine")
	scales := []float64{nought, halfway, picked(t, "mutants-wispy-scale")}
	thicks := []float64{unity, picked(t, "mutants-wispy-thick-between")}
	for _, thick := range thicks {
		for seed := range uint64(pickedCount(t, "mutants-wispy-seeds")) {
			for _, scale := range scales {
				wispy := Wispy{Of: Puff{Radius: radius, Thick: thick},
					Seed: seed, Scale: scale, Amount: amount}
				want := thick * (unity - amount*halfway)
				got := wispy.Density(mutantsOrigin())
				if !closeTo(got, want, fine) {
					t.Errorf("seed %d scale %v thick %v: %v, want %v", seed,
						scale, thick, got, want)
				}
			}
		}
	}
}

// a wispy nothing has the bounds of nothing, not a panic.
func TestWispyOfNothingHasNoBounds(t *testing.T) {
	if bounds := (Wispy{}).Bounds(); bounds != (maths.Box{}) {
		t.Errorf("a wispy nothing is bounded by %v", bounds)
	}
}

// a giant of no depth has no rise to thicken over, so it is clear; and a
// giant of radius one is Thick at its centre, a depth below the top:
// Smooth(0, 0.5, 1) is one.
func TestGiantOfNoDepthAndOfRadiusOne(t *testing.T) {
	thick := picked(t, "mutants-giant-thick")
	flat := Giant{Radius: picked(t, "mutants-giant-radius"), Depth: nought,
		Thick: thick}
	nearRim := mutantsUp(picked(t, "mutants-giant-near-rim"))
	nearerRim := mutantsUp(picked(t, "mutants-giant-nearer-rim"))
	for _, at := range []maths.Vec{mutantsOrigin(), nearRim, nearerRim} {
		if density := flat.Density(at); density != nought {
			t.Errorf("a giant of no depth is %v at %v", density, at)
		}
	}
	small := Giant{Radius: unity, Depth: halfway, Thick: thick}
	if density := small.Density(mutantsOrigin()); density != thick {
		t.Errorf("a giant of radius one is %v at its centre", density)
	}
}

// a plume of no height or no width has no column to be dense in, so it is clear
// at the foot of where its column would stand, not the zero over zero of a
// ratio to nothing.
func TestPlumeOfNoSizeIsNone(t *testing.T) {
	height := picked(t, "mutants-plume-height")
	width := picked(t, "mutants-plume-width")
	flat := Plume{Height: nought, Width: width, Thick: unity}
	if density := flat.Density(mutantsOrigin()); density != nought {
		t.Errorf("a plume of no height is %v at its foot", density)
	}
	thin := Plume{Height: height, Width: nought, Thick: unity}
	axis := mutantsUp(height * halfway)
	if density := thin.Density(axis); density != nought {
		t.Errorf("a plume of no width is %v on its axis", density)
	}
}

// Thick is the density at the column's heart, so a plume a unit high or a
// unit wide is Thick on its axis a thousandth up from its foot, to within
// the column's thinning there, a twentieth of a thousandth of Thick
// (1 - 0.5h at h = 0.001) and the umbrella's far smaller tail.
func TestSmallPlumesAreThickOnTheAxis(t *testing.T) {
	height := picked(t, "mutants-plume-height")
	width := picked(t, "mutants-plume-width")
	rise := picked(t, "mutants-plume-foot-rise")
	coarse := picked(t, "mutants-tolerance-coarse")
	for _, plume := range []Plume{
		{Height: unity, Width: width, Thick: unity},
		{Height: height, Width: unity, Thick: unity}} {
		at := mutantsUp(rise * plume.Height)
		if got := plume.Density(at); !closeTo(got, unity, coarse) {
			t.Errorf("%+v is %v on its axis at %v", plume, got, at)
		}
	}
}

// a jet a unit long still has its density, Thick times (1 - y over Length) on
// the axis: a quarter up, three quarters of 3 is 2.25. a jet of no length has
// none. a jet that starts at From has its full density at From, not zero: at
// 1.5 of 10, 3 times 0.85.
func TestJetOfLengthOneAndFrom(t *testing.T) {
	base := picked(t, "mutants-stream-base")
	thick := picked(t, "mutants-stream-thick")
	fine := picked(t, "mutants-tolerance-fine")
	short := Jet{Length: unity, Base: base, Thick: thick}
	quarter := mutantsUp(picked(t, "mutants-stream-quarter"))
	want := picked(t, "mutants-stream-quarter-density")
	if got := short.Density(quarter); !closeTo(got, want, fine) {
		t.Errorf("a jet a unit long is %v a quarter up, want %v", got, want)
	}
	none := Jet{Length: nought, Base: base, Thick: thick}
	if got := none.Density(mutantsOrigin()); got != nought {
		t.Errorf("a jet of no length is %v", got)
	}
	from := Jet{Length: picked(t, "mutants-stream-length"), Base: base,
		Thick: thick, From: picked(t, "mutants-jet-from")}
	wantFrom := picked(t, "mutants-jet-from-density")
	looser := picked(t, "mutants-tolerance-looser")
	got := from.Density(mutantsUp(from.From))
	if !closeTo(got, wantFrom, looser) {
		t.Errorf("at From the jet is %v, want %v", got, wantFrom)
	}
}

// a jet of no base is no width at its origin: zero there, not the 0/0 of a
// ratio to nothing.
func TestJetOfNoBaseIsNoneAtItsOrigin(t *testing.T) {
	jet := Jet{Length: picked(t, "mutants-stream-length"), Base: nought,
		Spread: picked(t, "mutants-stream-spread"), Thick: unity}
	if got := jet.Density(mutantsOrigin()); got != nought {
		t.Errorf("a jet of no base is %v at its origin", got)
	}
}

// past two and a half widths there is nothing and at exactly two and a
// half there is still the gaussian's exp(-2 * 2.5^2) = exp(-12.5), of the
// jet and the tail both: a width of one at the origin, a point 2.5 out.
func TestTheGaussianReachesTwoAndAHalfWidths(t *testing.T) {
	want := math.Exp(-picked(t, "mutants-gaussian-exponent"))
	at := mutantsAlong(picked(t, "mutants-gaussian-reach"))
	length := picked(t, "mutants-stream-length")
	base := picked(t, "mutants-stream-base")
	finest := picked(t, "mutants-tolerance-finest")
	jet := Jet{Length: length, Base: base, Thick: unity}
	if got := jet.Density(at); !closeTo(got, want, finest) {
		t.Errorf("the jet at 2.5 widths is %v, want %v", got, want)
	}
	tail := Tail{Length: length, Base: base, Thick: unity}
	if got := tail.Density(at); !closeTo(got, want, finest) {
		t.Errorf("the tail at 2.5 widths is %v, want %v", got, want)
	}
}

// a jet's knots are each side's own and the upper side is the one at y of zero:
// there the knot noise is read at the origin, where it is zero for any seed, so
// a knotted jet on its axis at its origin is its plain Thick.
func TestAKnottedJetIsPlainAtItsOrigin(t *testing.T) {
	length := picked(t, "mutants-stream-length")
	thick := picked(t, "mutants-knotted-thick")
	for seed := range uint64(pickedCount(t, "mutants-knotted-seeds")) {
		jet := Jet{Length: length, Base: unity, Thick: thick, Knots: unity,
			Seed: seed}
		if got := jet.Density(mutantsOrigin()); got != thick {
			t.Errorf("seed %d: the knotted jet is %v at its origin, want %v",
				seed, got, thick)
		}
	}
}

// a tail is Thick at its head, y of zero; a tail a unit long has its density, a
// quarter up three quarters of 3; a tail of no base is no width at its head, so
// zero there and not 0/0.
func TestTailAtItsHeadAndOfLengthOne(t *testing.T) {
	length := picked(t, "mutants-stream-length")
	base := picked(t, "mutants-stream-base")
	thick := picked(t, "mutants-stream-thick")
	fine := picked(t, "mutants-tolerance-fine")
	head := Tail{Length: length, Base: base, Thick: thick}
	if got := head.Density(mutantsOrigin()); got != thick {
		t.Errorf("the head is %v, want %v", got, thick)
	}
	short := Tail{Length: unity, Base: base, Thick: thick}
	quarter := mutantsUp(picked(t, "mutants-stream-quarter"))
	want := picked(t, "mutants-stream-quarter-density")
	if got := short.Density(quarter); !closeTo(got, want, fine) {
		t.Errorf("a tail a unit long is %v a quarter along, want %v", got, want)
	}
	bare := Tail{Length: length, Base: nought,
		Spread: picked(t, "mutants-stream-spread"), Thick: unity}
	if got := bare.Density(mutantsOrigin()); got != nought {
		t.Errorf("a tail of no base is %v at its head", got)
	}
}
