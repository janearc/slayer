package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// at the oval's heart the density thins by e over every third of the
// shell, and the shell is counted to its top: a shell from 1 to 2 above
// a body of radius 10, the pole at latitude pi over two, and a point on
// the axis at the top, 12 out, is Thick times e^-3 whole.
func TestAnAuroraIsStillThereAtTheTopOfItsShell(t *testing.T) {
	a := Aurora{Radius: 10, Low: 1, High: 2, Lat: math.Pi / 2, Width: 0.5, Thick: 1}
	if d := a.Density(maths.V(0, 12, 0)); !closeTo(d, math.Exp(-3), 1e-15) {
		t.Errorf("at the top of the shell it is %v, want %v", d, math.Exp(-3))
	}
}

// a shell of no thickness or an oval of no width has nothing in it, at the very
// place where its ratio would be zero over zero.
func TestAnAuroraOfNoShellOrNoWidthIsNone(t *testing.T) {
	flat := Aurora{Radius: 10, Low: 1, High: 1, Lat: math.Pi / 2, Width: 0.5, Thick: 1}
	if d := flat.Density(maths.V(0, 11, 0)); d != 0 {
		t.Errorf("a shell of no thickness is %v", d)
	}
	thin := Aurora{Radius: 10, Low: 1, High: 2, Lat: math.Pi / 2, Width: 0, Thick: 1}
	if d := thin.Density(maths.V(0, 11.5, 0)); d != 0 {
		t.Errorf("an oval of no width is %v on its line", d)
	}
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
	a := Aurora{Radius: 10, Low: 1, High: 2, Lat: math.Pi / 2, Width: 0.5, Thick: 1}
	want := math.Exp(-1.5)
	over, under := 0, 0
	for i := range int64(20000) {
		axis := maths.V(noise.Hash(1, i)-0.5, noise.Hash(2, i)-0.5,
			noise.Hash(3, i)-0.5)
		a.Axis = axis
		unit := axis.Unit()
		for _, sign := range []float64{1, -1} {
			at := unit.Scale(sign * 11.5)
			switch dot := at.Unit().Dot(unit); {
			case dot > 1:
				over++
			case dot < -1:
				under++
			}
			if d := a.Density(at); !closeTo(d, want, 1e-9) {
				t.Fatalf("axis %v, %v out: %v, want %v", axis, sign*11.5, d, want)
			}
		}
	}
	if over == 0 || under == 0 {
		t.Fatalf("no axis rounded past one (%d) or past minus one (%d)", over, under)
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
	x := math.Log(1000)
	for range 2000 {
		x = math.Nextafter(x, 0)
	}
	for range 4000 {
		if math.Exp(-x) == 1e-3 {
			t.Fatalf("exp(-%v) is exactly 1e-3", x)
		}
		x = math.Nextafter(x, 100)
	}
}
