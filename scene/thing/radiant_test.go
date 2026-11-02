package thing

import (
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
)

// A band falls in the range libtheme holds it in: a ten-thousandth of a
// nanometre is gamma, a nanometre x-ray, a metre radio; the zero band
// and green are visible; a spectrum answers by the range and is opaque
// and dark where it declares none.
func TestSpectrumByRange(t *testing.T) {
	for _, c := range []struct {
		nm   float64
		want radiation.Range
	}{{1e-4, radiation.Gamma()}, {1, radiation.XRay()},
		{50, radiation.ExtremeUltraviolet()}, {5000, radiation.Infrared()},
		{1e9, radiation.Radio()}} {
		if got := Range(maths.Band{Nanometres: c.nm}); got != c.want {
			t.Errorf("%v nm is %v, want %v", c.nm, got, c.want)
		}
	}
	if !Seen(maths.Band{}) || !Seen(maths.Band{Nanometres: 550}) ||
		Seen(maths.Band{Nanometres: 1e-4}) {
		t.Error("visible is wrong")
	}
	s := Spectrum{Passes: map[radiation.Range]float64{radiation.Gamma(): 0.6},
		Gives: map[radiation.Range]float64{radiation.XRay(): 3}}
	gamma, xray := maths.Band{Nanometres: 2e-4}, maths.Band{Nanometres: 0.5}
	if s.Through(gamma) != 0.6 || s.Through(xray) != 0 {
		t.Errorf("through: gamma %v, x-ray %v", s.Through(gamma), s.Through(xray))
	}
	if s.Emits(xray) != 3 || s.Emits(gamma) != 0 {
		t.Errorf("emits: x-ray %v, gamma %v", s.Emits(xray), s.Emits(gamma))
	}
}

// Excited gas gives off its lines at their wavelengths and little
// between, passes what it passes, returns no radar, and says it is lit.
func TestExcitedGasGlowsInLines(t *testing.T) {
	e := Excited{Lines: map[float64]float64{656: 1, 501: 0.5}, Width: 2, Passes: 0.3}
	if h := e.Emits(maths.Band{Nanometres: 656}); h < 0.99 {
		t.Errorf("h-alpha is %v", h)
	}
	if o := e.Emits(maths.Band{Nanometres: 501}); o < 0.49 || o > 0.51 {
		t.Errorf("o iii is %v", o)
	}
	if between := e.Emits(maths.Band{Nanometres: 600}); between > 1e-6 {
		t.Errorf("between the lines is %v", between)
	}
	if e.Through(maths.Band{Nanometres: 1}) != 0.3 || e.Scatters(maths.Band{}) != 0 {
		t.Error("the rest of the spectrum is wrong")
	}
	e.Excitations = map[float64]float64{501: 2}
	if e.Excitation(maths.Band{Nanometres: 502}) != 2 || e.Excitation(maths.Band{Nanometres: 656}) != 1 {
		t.Error("the excitations are not the nearest line's")
	}
}

// A visible band's range is Visible, which libtheme names as a range
// and not as a profile; the zero band, every band at once, is Visible
// too; and no band is ever fermi's, an instrument's and not a range.
func TestTheVisibleIsARange(t *testing.T) {
	if r := Range(maths.Band{Nanometres: 656}); r != Visible {
		t.Errorf("h alpha is in %v", r)
	}
	if r := Range(maths.Band{}); r != Visible {
		t.Errorf("the zero band is in %v", r)
	}
	if r := Range(maths.Band{Nanometres: 1}); r != radiation.XRay() {
		t.Errorf("a nanometre is in %v", r)
	}
	for nm := 1e-6; nm < 1e11; nm *= 10 {
		if Range(maths.Band{Nanometres: nm}) == radiation.Fermi() {
			t.Errorf("%g nm is fermi's", nm)
		}
	}
}

// Where libtheme's ranges overlap, a thing may declare either and the
// narrowest it declares answers: 110 nm is both extreme ultraviolet and
// ultraviolet; a thing that declares the ultraviolet alone passes it by
// that, one that declares both passes it by the extreme.
func TestOverlappingRangesTakeTheNarrowestDeclared(t *testing.T) {
	band := maths.Band{Nanometres: 110}
	wide := Spectrum{Passes: map[radiation.Range]float64{
		radiation.Ultraviolet(): 0.5}}
	both := Spectrum{Passes: map[radiation.Range]float64{
		radiation.Ultraviolet(): 0.5, radiation.ExtremeUltraviolet(): 0.1}}
	if wide.Through(band) != 0.5 || both.Through(band) != 0.1 {
		t.Errorf("110 nm passes %v wide and %v with both",
			wide.Through(band), both.Through(band))
	}
	radio := maths.Band{Nanometres: 5e6}
	r := Spectrum{Passes: map[radiation.Range]float64{radiation.Radio(): 0.9}}
	if r.Through(radio) != 0.9 {
		t.Errorf("a microwave band through a thing that declares radio: %v",
			r.Through(radio))
	}
}
