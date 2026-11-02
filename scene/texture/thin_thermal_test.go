package texture

import (
	"math"
	"testing"

	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
)

// blackBody is what thermal is meant to hand back at a temperature the table
// has a step at: libtheme's black body colour, a part below zero (out of gamut,
// as red-hot iron is short of blue) held at zero, and the three parts scaled to
// a luminance of one.
//
// the colour is libtheme's; this is only what the disc does with it.
func blackBody(k float64) maths.Light {
	cr, cg, cb := srgb.Light(swatch.Planckian(k))
	l := maths.RGB(math.Max(cr, 0), math.Max(cg, 0), math.Max(cb, 0))
	return l.Scale(1 / (0.2126*l.R + 0.7152*l.G + 0.0722*l.B))
}

// mixed is x and y together, y by the share.
func mixed(x, y maths.Light, share float64) maths.Light {
	return x.Scale(1 - share).Add(y.Scale(share))
}

// At a temperature on a step of the table the colour is the black body
// at that temperature, from cool orange to hot blue.
func TestThermalIsTheBlackBodyOnAStep(t *testing.T) {
	for _, k := range []float64{500, 1000, 2000, 5000, 10000, 40000, 59950, 60000} {
		if got, want := thermal(k), blackBody(k); !near(got, want) {
			t.Errorf("thermal at %v is %v, want %v", k, got, want)
		}
	}
}

// Every colour is at a luminance of one, on a step and between, so the
// disc's brightness is the temperature's and not the colour's.
func TestThermalIsAtUnitLuminance(t *testing.T) {
	for _, k := range []float64{500, 777, 1234.5, 3000, 5025, 6500, 9999, 22222, 45000, 59999} {
		if got := thermal(k).Luminance(); !within(got, 1, 1e-9) {
			t.Errorf("thermal at %v has a luminance of %v", k, got)
		}
	}
}

// Between two steps the colour is the two together, by how far along:
// 5025 is half way from 5000 to 5050, 5010 a fifth.
func TestThermalIsBlendedBetweenSteps(t *testing.T) {
	for _, c := range []struct{ k, from, share float64 }{
		{5025, 5000, 0.5}, {5010, 5000, 0.2}, {5040, 5000, 0.8}, {12525, 12500, 0.5},
	} {
		want := mixed(blackBody(c.from), blackBody(c.from+50), c.share)
		if got := thermal(c.k); !near(got, want) {
			t.Errorf("thermal at %v is %v, want %v", c.k, got, want)
		}
	}
}

// At zero kelvin and below a body gives no light, so the colour is black and
// not a number of nothing; just above, it comes up from black toward the first
// step: 25 is half of 50, and 1 is a fiftieth.
func TestThermalIsBlackAtNoughtAndComesUpFromIt(t *testing.T) {
	for _, k := range []float64{0, -1, -1000} {
		if got := thermal(k); got != (maths.Light{}) {
			t.Errorf("thermal at %v is %v, not black", k, got)
		}
	}
	for _, c := range []struct{ k, share float64 }{{25, 0.5}, {1, 0.02}, {49, 0.98}} {
		want := mixed(maths.Light{}, blackBody(50), c.share)
		if got := thermal(c.k); !near(got, want) {
			t.Errorf("thermal at %v is %v, want %v", c.k, got, want)
		}
	}
}

// Above sixty thousand kelvin the colour is that at sixty thousand, and
// it runs on into it without a step: just under, half a step under, and
// the top itself are all the same colour to the last of the blending.
func TestThermalHoldsItsTopAboveTheTable(t *testing.T) {
	top := blackBody(60000)
	for _, k := range []float64{60000, 60001, 1e6, 1e12} {
		if got := thermal(k); !near(got, top) {
			t.Errorf("thermal at %v is %v, want the top, %v", k, got, top)
		}
	}
	if got := thermal(59999.999999); !near(got, top) {
		t.Errorf("a hair under the top is %v, want %v", got, top)
	}
	if got, want := thermal(59975), mixed(blackBody(59950), top, 0.5); !near(got, want) {
		t.Errorf("half a step under the top is %v, want %v", got, want)
	}
}
