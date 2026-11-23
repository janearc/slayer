package session

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/weather"
)

// longitudes is a flow that gives back the longitude it is read at, so a
// turn can be seen.
type longitudes struct{}

// At is the longitude read, as the origin.
func (longitudes) At(_, longitude float64) (origin, spin, age float64) {
	return longitude, 0, 0
}

// The seam meets itself: a marker at longitude nothing and at a whole
// turn is read from one place, turned or not, and a turn of a whole turn
// is no turn.
func TestTheSeamMeetsItself(t *testing.T) {
	for _, by := range []float64{0, math.Pi / 3, fullTurn, -fullTurn / 4} {
		turned := turnedFlow{flow: longitudes{}, by: by}
		at, _, _ := turned.At(0, 0)
		round, _, _ := turned.At(0, fullTurn)
		if math.Abs(at-round) > 1e-12 || at < 0 || at >= fullTurn {
			t.Errorf("turned by %g: %g and %g", by, at, round)
		}
	}
	whole := turnedFlow{flow: longitudes{}, by: fullTurn}
	if at, _, _ := whole.At(0, 1); math.Abs(at-1) > 1e-12 {
		t.Errorf("a whole turn moved longitude 1 to %g", at)
	}
}

// The poles do not pinch into a star: past the weather's walls every
// longitude reads the same air, spinning none, so nothing fans out.
func TestThePolesReadOneAir(t *testing.T) {
	grid, err := weather.NewGrid(32, 16, walls*maths.Degree, 6e7)
	if err != nil {
		t.Fatal(err)
	}
	stirring, err := weather.NewStirring(3.8e-17, 1.6e6, 1.0e-5)
	if err != nil {
		t.Fatal(err)
	}
	sky, err := weather.New(grid, 18000, 1e7, 1e6, stirring, 7)
	if err != nil {
		t.Fatal(err)
	}
	sky.Run(float64(10 * time.Hour / time.Second))
	for _, pole := range []float64{-math.Pi / 2, math.Pi / 2} {
		origin, spin, age := sky.At(pole, 0)
		for _, longitude := range []float64{1, 2, 4, 6} {
			o, s, a := sky.At(pole, longitude)
			if math.Abs(o-origin) > 1e-9 || s != 0 || spin != 0 ||
				math.Abs(a-age) > 1e-6 {
				t.Errorf("pole %g at %g: %g %g %g against %g %g %g",
					pole, longitude, o, s, a, origin, spin, age)
			}
		}
	}
}
