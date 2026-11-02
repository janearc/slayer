package film

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// every ray a pinhole or a lens makes is one unit long, across pictures
// of many shapes and lenses of many widths.
func TestEveryRayIsOneUnitLong(t *testing.T) {
	for _, s := range []Sensor{Pinhole{}, Pinhole{Wide: 1.2, Pixel: 0.75},
		Pinhole{Wide: 0.05}, Lens{Focal: 24}, Lens{Focal: 200}} {
		for _, size := range [][2]int{{1, 1}, {96, 40}, {7, 300}} {
			for y := range size[1] {
				for x := range size[0] {
					r := s.Ray(x, y, size[0], size[1], maths.Band{})
					if l := r.Dir.Len(); math.Abs(l-1) > 1e-12 {
						t.Fatalf("%+v at %d, %d of %v: %v long", s, x, y,
							size, l)
					}
				}
			}
		}
	}
}

// a picture of no pixels, and a field that is no field, still give a ray
// one unit long: straight ahead, or the 60 degree default.
func TestADegeneratePictureLooksAhead(t *testing.T) {
	ahead := maths.V(0, 0, -1)
	for _, size := range [][2]int{{0, 5}, {5, 0}, {0, 0}, {-3, 4}} {
		if r := (Pinhole{}).Ray(0, 0, size[0], size[1], maths.Band{}); r.Dir !=
			ahead {
			t.Errorf("a picture %v looks along %v", size, r.Dir)
		}
	}
	plain := Pinhole{}.Ray(3, 1, 8, 4, maths.Band{})
	for _, wide := range []float64{-1, math.NaN(), math.Pi, 4} {
		if r := (Pinhole{Wide: wide}).Ray(3, 1, 8, 4, maths.Band{}); r.Dir !=
			plain.Dir {
			t.Errorf("a field of %v looks along %v, not the default's %v",
				wide, r.Dir, plain.Dir)
		}
	}
	square := Pinhole{}.Ray(3, 1, 8, 4, maths.Band{})
	for _, shape := range []float64{-0.75, math.NaN()} {
		if r := (Pinhole{Pixel: shape}).Ray(3, 1, 8, 4, maths.Band{}); r.Dir !=
			square.Dir {
			t.Errorf("a pixel %v wide looks along %v, not square's %v",
				shape, r.Dir, square.Dir)
		}
	}
}

// the default pinhole sees 60 degrees across: its edge pixels look out
// at nearly 30 degrees either side.
func TestThePinholeSeesSixtyDegrees(t *testing.T) {
	const w = 2000
	r := Pinhole{}.Ray(w-1, 0, w, 1, maths.Band{})
	if got := math.Atan2(r.Dir.X, -r.Dir.Z) / maths.Degree; math.Abs(got-
		30) > 0.05 {
		t.Errorf("the rightmost pixel looks %v degrees out", got)
	}
}

// a strip of no width, or one seen from no height, is the gauge's own
// lens, not one of no angle or of a bent one.
func TestASwathOfNothingIsAPlainLens(t *testing.T) {
	for _, want := range [][2]float64{{0, 400}, {40, 0}, {40, -5},
		{-1, 400}, {0, 0}} {
		l := Swath(want[0], want[1])
		if l != (Lens{Gauge: Super35}) {
			t.Errorf("a swath %v across from %v is %+v", want[0], want[1], l)
		}
	}
	// forty kilometres from four hundred is a lens of 5.725 degrees
	if got := Swath(40, 400).Wide() / maths.Degree; math.Abs(got-
		5.7248) > 0.001 {
		t.Errorf("forty from four hundred sees %v degrees", got)
	}
}
