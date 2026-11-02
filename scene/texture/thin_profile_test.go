package texture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// peakByHand is the profile r^-3/4 (1 - r^-1/2)^1/4 at its peak, worked
// from the derivative, not from the code. the log of the profile is
// -3/4 ln r + 1/4 ln(1 - r^-1/2), and it is flat where 3/r = (1/2)
// r^-3/2 / (1 - r^-1/2), which is 3 = 3.5 r^-1/2, so r^-1/2 is 6/7.
//
// so r is 49/36. there r^-3/4 is (6/7)^1.5 and the second factor is
// (1 - 6/7)^1/4 = 7^-1/4, so the peak is (6/7)^1.5 * 7^-1/4, which bc
// gives as 0.48787133923227617.
var peakByHand = math.Pow(6.0/7, 1.5) * math.Pow(7, -0.25)

// shares are the profile over its peak at some radii, from bc:
// e(-0.75*l(r)) * e(0.25*l(1 - sqrt(1/r))) / 0.48787133923227617. each is
// below one, since the peak is the largest.
var shares = []struct{ r, share float64 }{
	{1.2, 0.9712913712267185},
	{1.45, 0.9953826544437553},
	{2, 0.8966016358667303},
	{4, 0.6093856204353388},
	{10, 0.3314534621914452},
	{25, 0.1733852642550332},
}

// The profile is zero at the inner edge and inside it, and not a number there
// would be: the square root of a share above one leaves a negative to take the
// fourth root of.
func TestThinProfileIsNothingAtAndInsideTheEdge(t *testing.T) {
	for _, r := range []float64{1, 0.999, 0.5, 0, -3} {
		if got := profile(r); got != 0 {
			t.Errorf("the profile at %v is %v", r, got)
		}
	}
}

// The profile peaks at 49/36 of the inner radius at the value worked
// from its derivative, and the shares of the peak at other radii are
// bc's. the peak is the largest: either side of it, a hundredth, is
// less.
func TestThinProfilePeaksWhereTheDerivativeSays(t *testing.T) {
	if got := profile(49.0 / 36); !within(got, peakByHand, 1e-15) {
		t.Errorf("the profile at 49/36 is %v, worked by hand %v", got, peakByHand)
	}
	if !within(peak, peakByHand, 1e-15) {
		t.Errorf("the peak is %v, worked by hand %v", peak, peakByHand)
	}
	for _, r := range []float64{49.0/36 - 0.01, 49.0/36 + 0.01} {
		if got := profile(r); !(got < peakByHand) {
			t.Errorf("the profile at %v is %v, not below the peak", r, got)
		}
	}
	for _, c := range shares {
		if got := profile(c.r) / peakByHand; !within(got, c.share, 1e-12) {
			t.Errorf("the profile at %v is %v of the peak, bc says %v", c.r, got, c.share)
		}
	}
}

// The temperature is Hot times the share of the peak at the radius in inner
// radii, so with an inner radius of 2 the radius 2r gives the share at r. it is
// zero outside the disc either side, zero with no inner radius, and never not a
// number.
func TestThinTemperatureIsTheShareOfHot(t *testing.T) {
	d := Thin{Inner: 2, Outer: 60, Hot: 25000, Bright: 1}
	for _, c := range shares {
		if got, want := d.Temperature(2*c.r), 25000*c.share; !within(got, want, 1e-8) {
			t.Errorf("at %v inner radii the temperature is %v, want %v", c.r, got, want)
		}
	}
	if got := d.Temperature(2 * 49.0 / 36); got != 25000 {
		t.Errorf("at the peak the temperature is %v, not the hottest", got)
	}
	for _, r := range []float64{0, 1, 2, 60.5, 100} {
		if got := d.Temperature(r); got != 0 {
			t.Errorf("at %v the temperature is %v, not nought", r, got)
		}
	}
	for _, inner := range []float64{0, -1} {
		bad := Thin{Inner: inner, Outer: 60, Hot: 25000, Bright: 1}
		for _, r := range []float64{0, 0.5, 3} {
			if got := bad.Temperature(r); got != 0 {
				t.Errorf("an inner radius of %v at %v is %v kelvin", inner, r, got)
			}
		}
	}
}

// Below 500 kelvin the disc is dark, and at 500 and above it is lit. the
// peak is Hot to the last digit, so Hot sets the temperature there: 499
// is dark, 500 and 500.5 are lit, each to a luminance of one (the share
// of Hot is one, so one to the fourth).
func TestThinDiscIsDarkBelowFiveHundredKelvin(t *testing.T) {
	p := maths.V(49.0/36, 0, 0)
	for _, c := range []struct {
		hot  float64
		lit  bool
		what string
	}{{499, false, "499"}, {500, true, "500"}, {500.5, true, "500.5"}, {501, true, "501"}} {
		d := Thin{Inner: 1, Outer: 10, Hot: c.hot, Bright: 1}
		if k := d.Temperature(49.0 / 36); k != c.hot {
			t.Fatalf("the peak of a disc of %v is %v kelvin", c.what, k)
		}
		got := d.At(p)
		if !c.lit && got != (maths.Light{}) {
			t.Errorf("a disc of %v kelvin is lit: %v", c.what, got)
		}
		if c.lit && !within(got.Luminance(), 1, 1e-9) {
			t.Errorf("a disc of %v kelvin has a luminance of %v, not 1", c.what, got.Luminance())
		}
	}
}

// A smooth disc is as bright as the fourth power of its temperature over
// the hottest, whatever its colour: the colour is at unit luminance.
//
// with Bright of 3 the luminance at r is 3 times the share to the
// fourth, by bc: 0.6462463565813805 at 2, 0.1379014422046468 at 4,
// 0.8900166314577877 at 1.2.
func TestThinDiscIsBrightAsTheFourthPowerOfItsShare(t *testing.T) {
	d := Thin{Inner: 1, Outer: 25, Hot: 25000, Bright: 3}
	for _, c := range []struct{ r, fourth float64 }{
		{2, 0.6462463565813805}, {4, 0.1379014422046468}, {1.2, 0.8900166314577877},
	} {
		got := d.At(maths.V(0, 0, c.r)).Luminance()
		if want := 3 * c.fourth; !within(got, want, 1e-9*want) {
			t.Errorf("at %v the luminance is %v, want %v", c.r, got, want)
		}
	}
}
