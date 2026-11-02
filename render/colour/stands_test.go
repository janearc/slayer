package colour

import (
	"math"
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
)

// a band stands for a colour no lamp has to be less than dark for, as bright as
// white by BT.709 luminance (0.2126, 0.7152, 0.0722), at any wavelength of any
// range.
//
// the violet end of gamma needs a negative green from the lamps, in libtheme's
// own arithmetic, and Stands gives it zero instead.
func TestAStandingColourIsNeverBelowNoughtAndAsBrightAsWhite(t *testing.T) {
	for _, span := range shown {
		short, long := span.Edges()
		for i := 0; i < 60; i++ {
			nm := short * math.Pow(long/short, float64(i)/60)
			l := Stands(maths.Band{Nanometres: nm})
			if l.R < 0 || l.G < 0 || l.B < 0 {
				t.Errorf("%v at %g nm stands for %v, less than dark", span, nm, l)
			}
			if y := l.Luminance(); math.Abs(y-1) > 1e-9 {
				t.Errorf("%v at %g nm stands at luminance %v", span, nm, y)
			}
		}
	}
	gamma, _ := radiation.Gamma().Edges()
	violet := Stands(maths.Band{Nanometres: gamma})
	if violet.G != 0 || violet.R <= 0 || violet.B <= violet.R {
		t.Errorf("the violet end of gamma stands for %v", violet)
	}
}

// a range holds its lowest wavelength and not its highest, and a
// wavelength in no range is white.
//
// the lowest wavelength of every range lands on the observer's first
// row, so gamma's lowest and x-ray's are one violet; x-ray starts where
// gamma ends, so the end of gamma is not gamma's.
//
// 400 nm, where ultraviolet ends, is visible light and white, as is
// anything between ultraviolet and infrared. 5e-6 nm is fermi's alone,
// and fermi is an instrument and not a range.
func TestARangeHoldsItsLowEndAndNotItsHighEnd(t *testing.T) {
	gamma, _ := radiation.Gamma().Edges()
	xray, _ := radiation.XRay().Edges()
	_, ultraviolet := radiation.Ultraviolet().Edges()
	violet := Stands(maths.Band{Nanometres: gamma})
	if violet == maths.Grey(1) {
		t.Fatal("the low end of gamma is white")
	}
	if got := Stands(maths.Band{Nanometres: xray}); got != violet {
		t.Errorf("x-ray's lowest wavelength stands for %v, gamma's for %v", got, violet)
	}
	for _, nm := range []float64{ultraviolet, 550, 5e-6} {
		if got := Stands(maths.Band{Nanometres: nm}); got != maths.Grey(1) {
			t.Errorf("%g nm stands for %v, want white", nm, got)
		}
	}
}

// the arithmetic of Stands scales a colour by one over its luminance,
// and would not survive a colour with none. every whole nanometre of
// the observer's rows, which are all that a range's colours can land
// on, has some light in at least one lamp.
func TestEveryRowTheRangesLandOnHasLight(t *testing.T) {
	lo, hi := swatch.Observed()
	for n := lo; n <= hi; n++ {
		r, g, b := srgb.Light(swatch.Monochrome(n))
		y := maths.RGB(max(r, 0), max(g, 0), max(b, 0)).Luminance()
		if !(y > 0) {
			t.Errorf("row %d has no light: %v %v %v", n, r, g, b)
		}
	}
}
