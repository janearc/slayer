package gas

import (
	"math"
	"testing"
)

// fourth is (550/nm)^4 for the wavelengths the rules are worked at, as the
// fractions they come to: 440 nm is 5/4, so 625/256; 660 nm is 5/6, so
// 625/1296; 700 nm is 11/14, so 14641/38416.
var fourth = map[float64]float64{
	440: 625.0 / 256,
	660: 625.0 / 1296,
	700: 14641.0 / 38416,
}

// air's visible rule scatters by the fourth power of the wavelength from
// 1.16e-5 a metre at 550 nm, the rayleigh slope, and absorbs in a hump
// centred on 600 nm, 60 nm wide (sigma), 3e-7 at the top: the exponent
// is minus the distance from 600 over two sigma squared, 7200.
//
// the figures are the table's, held so nothing moves, not measured values.
func TestAirsRuleIsRayleighAndAHump(t *testing.T) {
	a := Air().Visible
	for _, nm := range []float64{440, 660, 700} {
		_, s := a(nm)
		wantClose(t, "air's scatter", s, 1.16e-5*fourth[nm])
	}
	// 600 is the top; 660 and 540 are one sigma off, e^-0.5 of it;
	// 440 is 160 off, e^-(25600/7200)
	for _, c := range []struct{ nm, away float64 }{
		{600, 0}, {660, 0.5}, {540, 0.5}, {440, 25600.0 / 7200},
	} {
		got, _ := a(c.nm)
		wantClose(t, "air's absorb", got, 3e-7*math.Exp(-c.away))
	}
	if _, s := a(550); s != 1.16e-5 {
		t.Errorf("air scatters %v at 550 nm, want its coefficient", s)
	}
}

// below 300 nm air's rule is the ultraviolet's figures, 1e-2 and 6e-5, and
// from 300 nm up it is the rule: the table's edge, held so nothing moves.
// 299.5 is the table's; 300 and 300.5 are the rule's, scattering by the
// fourth power.
func TestAirsRuleChangesAt300(t *testing.T) {
	a := Air().Visible
	for _, nm := range []float64{250, 299.5} {
		if ab, s := a(nm); ab != 1e-2 || s != 6e-5 {
			t.Errorf("air at %v nm gives %v and %v, want 1e-2 and 6e-5", nm, ab, s)
		}
	}
	for _, nm := range []float64{300, 300.5} {
		_, s := a(nm)
		r := 550 / nm
		wantClose(t, "air's scatter past the edge", s, 1.16e-5*r*r*r*r)
	}
}

// venus's rule scatters 2e-3 a metre at 550 nm by the fourth power, and
// absorbs 1e-6 with a hump of 2e-4 centred on 400 nm, 40 nm wide: the
// unknown absorber's tail into the blue. one sigma off the middle, at 360
// and 440 nm, it is e^-0.5 of the top; at 550 nm, 150 off, e^-(22500/3200).
//
// the figures are the table's, held so nothing moves, not measured values.
func TestVenusRuleIsRayleighAndABlueHump(t *testing.T) {
	v := Venus().Visible
	for _, nm := range []float64{440, 660, 700} {
		_, s := v(nm)
		wantClose(t, "venus's scatter", s, 2e-3*fourth[nm])
	}
	for _, c := range []struct{ nm, away float64 }{
		{400, 0}, {360, 0.5}, {440, 0.5}, {550, 22500.0 / 3200},
	} {
		got, _ := v(c.nm)
		wantClose(t, "venus's absorb", got, 1e-6+2e-4*math.Exp(-c.away))
	}
}

// below 320 nm venus's rule is the ultraviolet's 1e-2 and 1e-2, and from
// 320 nm up it is the rule. 319.5 is the first; 320 and 320.5 the rule's.
func TestVenusRuleChangesAt320(t *testing.T) {
	v := Venus().Visible
	for _, nm := range []float64{250, 319.5} {
		if ab, s := v(nm); ab != 1e-2 || s != 1e-2 {
			t.Errorf("venus at %v nm gives %v and %v, want 1e-2 and 1e-2", nm, ab, s)
		}
	}
	for _, nm := range []float64{320, 320.5} {
		ab, s := v(nm)
		r := 550 / nm
		wantClose(t, "venus's scatter past the edge", s, 2e-3*r*r*r*r)
		wantClose(t, "venus's absorb past the edge", ab,
			1e-6+2e-4*math.Exp(-(nm-400)*(nm-400)/3200))
	}
}

// mars's rule is the gas, 2.4e-7 a metre at 550 nm by the fourth power, and the
// dust, 3.5e-5 flat, for the scatter; and for the absorption 5e-6 and a ramp of
// 2e-5 times (600-nm)/200 that falls to zero at 600 nm and stays there, red of
// it. at 440 nm the ramp is 0.8, at 500 nm 0.5, at 550 nm 0.25.
//
// the figures are the table's, held so nothing moves, not measured values.
func TestMarsRuleIsGasDustAndARamp(t *testing.T) {
	m := Mars().Visible
	for _, nm := range []float64{440, 660, 700} {
		_, s := m(nm)
		wantClose(t, "mars's scatter", s, 2.4e-7*fourth[nm]+3.5e-5)
	}
	for _, c := range []struct{ nm, ramp float64 }{
		{440, 0.8}, {500, 0.5}, {550, 0.25}, {600, 0}, {660, 0}, {700, 0},
	} {
		got, _ := m(c.nm)
		wantClose(t, "mars's absorb", got, 5e-6+2e-5*c.ramp)
	}
}
