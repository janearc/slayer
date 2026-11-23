package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// mutantsAuroraShell is the aurora of these tests, a shell from Low to High
// above a body of one radius, round the pole at latitude pi over two, read
// from the chosen numbers.
func mutantsAuroraShell(t *testing.T) Aurora {
	t.Helper()
	return Aurora{Radius: picked(t, "mutants-aurora-radius"),
		Low:   picked(t, "mutants-aurora-low"),
		High:  picked(t, "mutants-aurora-high"),
		Lat:   math.Pi / double,
		Width: picked(t, "mutants-aurora-width"),
		Thick: unity}
}

// mutantsAuroraAbove is the point on the pole's axis at a height.
func mutantsAuroraAbove(height float64) maths.Vec {
	return maths.V(nought, height, nought)
}

// at the oval's heart the density thins by e over every third of the
// shell, and the shell is counted to its top: a shell from 1 to 2 above
// a body of radius 10, the pole at latitude pi over two, and a point on
// the axis at the top, 12 out, is Thick times e^-3 whole.
func TestAnAuroraIsStillThereAtTheTopOfItsShell(t *testing.T) {
	aurora := mutantsAuroraShell(t)
	top := mutantsAuroraAbove(picked(t, "mutants-aurora-top-height"))
	want := math.Exp(-picked(t, "mutants-aurora-top-exponent"))
	density := aurora.Density(top)
	if !closeTo(density, want, picked(t, "mutants-aurora-top-tolerance")) {
		t.Errorf("at the top of the shell it is %v, want %v", density, want)
	}
}

// a shell of no thickness or an oval of no width has nothing in it, at the very
// place where its ratio would be zero over zero.
func TestAnAuroraOfNoShellOrNoWidthIsNone(t *testing.T) {
	flat := mutantsAuroraShell(t)
	flat.High = flat.Low
	flatPoint := mutantsAuroraAbove(picked(t, "mutants-aurora-flat-height"))
	if density := flat.Density(flatPoint); density != nought {
		t.Errorf("a shell of no thickness is %v", density)
	}
	thin := mutantsAuroraShell(t)
	thin.Width = nought
	thinPoint := mutantsAuroraAbove(picked(t, "mutants-aurora-thin-height"))
	if density := thin.Density(thinPoint); density != nought {
		t.Errorf("an oval of no width is %v on its line", density)
	}
}

// mutantsAuroraAxis is the axis the hashes of an index make: each of its
// three parts is a hash taken from halfway to its middle.
func mutantsAuroraAxis(t *testing.T, index int64) maths.Vec {
	t.Helper()
	seedX := uint64(pickedCount(t, "mutants-aurora-axis-seed-x"))
	seedY := uint64(pickedCount(t, "mutants-aurora-axis-seed-y"))
	seedZ := uint64(pickedCount(t, "mutants-aurora-axis-seed-z"))
	return maths.V(noise.Hash(seedX, index)-halfway,
		noise.Hash(seedY, index)-halfway, noise.Hash(seedZ, index)-halfway)
}

// a point on the pole's axis is at the pole, whatever the axis: a
// shell from 1 to 2 and a point 11.5 out, a half way up, is e^-1.5 at
// the oval's heart, at the north pole or the south.
//
// the dot product of the point's direction with the axis can round to a
// hair past one, and the latitude is then still pi over two, not the
// arcsine of a number that has none. the axes are searched for ones
// that do, as this machine rounds them.
func TestAPointOnTheAxisIsAtThePole(t *testing.T) {
	aurora := mutantsAuroraShell(t)
	want := math.Exp(-picked(t, "mutants-aurora-axis-exponent"))
	distance := picked(t, "mutants-aurora-axis-distance")
	tolerance := picked(t, "mutants-aurora-axis-tolerance")
	over, under := nought, nought
	for index := range int64(pickedCount(t, "mutants-aurora-axis-samples")) {
		axis := mutantsAuroraAxis(t, index)
		aurora.Axis = axis
		unit := axis.Unit()
		for _, sign := range []float64{unity, -unity} {
			at := unit.Scale(sign * distance)
			switch dot := at.Unit().Dot(unit); {
			case dot > unity:
				over++
			case dot < -unity:
				under++
			}
			density := aurora.Density(at)
			if !closeTo(density, want, tolerance) {
				t.Fatalf("axis %v, %v out: %v, want %v", axis, sign*distance,
					density, want)
			}
		}
	}
	if over == nought || under == nought {
		t.Fatalf("no axis rounded past one (%d) or past minus one (%d)", over,
			under)
	}
}

// the oval is cut off below a thousandth, and the cut is held to a
// reason in .mutants: no oval is exactly 1e-3, so whether the cut is at
// or just under it moves no density.
//
// the oval is exp(-off squared), and the exp of no number within two
// thousand steps of ln 1000 is that double. this holds for math.Exp as
// arm64 computes it; if it fails elsewhere, the reason is not true there.
func TestNoOvalIsExactlyAThousandth(t *testing.T) {
	x := math.Log(picked(t, "mutants-aurora-thousand"))
	for range pickedCount(t, "mutants-aurora-thousand-back") {
		x = math.Nextafter(x, nought)
	}
	thousandth := picked(t, "mutants-aurora-thousandth")
	toward := picked(t, "mutants-aurora-thousand-toward")
	for range pickedCount(t, "mutants-aurora-thousand-forward") {
		if math.Exp(-x) == thousandth {
			t.Fatalf("exp(-%v) is exactly 1e-3", x)
		}
		x = math.Nextafter(x, toward)
	}
}
