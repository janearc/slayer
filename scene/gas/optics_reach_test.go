package gas

import (
	"math"
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// marked is a gas whose table and whose rule cannot be mistaken for each
// other: the table's figures are small whole numbers and the rule gives
// back the wavelength itself, so what a band was answered from reads off
// the answer. these are test figures and mean nothing.
func marked() Gas {
	return Gas{
		Name: "marked",
		Ranges: map[radiation.Range]Optics{
			radiation.Ultraviolet(): {Absorb: 1, Scatter: 2},
			thing.Visible:           {Absorb: 3, Scatter: 4},
			radiation.Infrared():    {Absorb: 5, Scatter: 6},
			radiation.Radio():       {Absorb: 7, Scatter: 8},
		},
		Visible: func(nm float64) (absorb, scatter float64) {
			return nm, 2 * nm
		},
	}
}

// the visible rule answers the visible and the near ultraviolet and near
// infrared past 300 nm and short of 1200 nm, the edges themselves left to
// the table: 300 and 1200 are the table's, and 300.5 and 1199.5 the rule's.
// below 300 nm, above 1200 nm and in the radio the table answers.
func TestTheRuleAnswersTheVisibleAndTheNearEdges(t *testing.T) {
	g := marked()
	rule := func(nm float64) Optics { return Optics{nm, 2 * nm} }
	for _, c := range []struct {
		nm   float64
		want Optics
		from string
	}{
		{250, Optics{1, 2}, "the ultraviolet table, short of 300"},
		{300, Optics{1, 2}, "the ultraviolet table, on 300"},
		{300.5, rule(300.5), "the rule, just past 300"},
		{350, rule(350), "the rule, in the near ultraviolet"},
		{550, rule(550), "the rule, in the visible"},
		{1000, rule(1000), "the rule, in the near infrared"},
		{1199.5, rule(1199.5), "the rule, just short of 1200"},
		{1200, Optics{5, 6}, "the infrared table, on 1200"},
		{1500, Optics{5, 6}, "the infrared table, past 1200"},
		{1e9, Optics{7, 8}, "the radio table"},
		{0, rule(550), "the rule at 550, for the zero band"},
	} {
		b := maths.Band{Nanometres: c.nm}
		if g.Absorb(b) != c.want.Absorb || g.Scatter(b) != c.want.Scatter {
			t.Errorf("%v nm: absorbs %v, scatters %v; %s is %v", c.nm,
				g.Absorb(b), g.Scatter(b), c.from, c.want)
		}
	}
}

// with no rule the table answers every band, the zero band from the visible;
// and a gas with a rule answers the zero band from the rule at 550 nm, never at
// zero, where a fourth power over zero is endless.
func TestTheZeroBandIsTheEyes(t *testing.T) {
	plain := Gas{Ranges: map[radiation.Range]Optics{
		thing.Visible: {Absorb: 3, Scatter: 4}}}
	zero := maths.Band{}
	if plain.Absorb(zero) != 3 || plain.Scatter(zero) != 4 {
		t.Errorf("the zero band in the table is %v and %v", plain.Absorb(zero),
			plain.Scatter(zero))
	}
	// the rule at 550 nm, worked: the rayleigh coefficient itself, since
	// 550 over 550 is one, and the hump 50 nm off its middle
	air := Air()
	wantClose(t, "air's zero band scatter", air.Scatter(zero), 1.16e-5)
	wantClose(t, "air's zero band absorb", air.Absorb(zero),
		3e-7*math.Exp(-50.0*50/7200))
}

// a band in no range the table declares is opaque: endless absorption and
// no scattering, so nothing is let through a metre of it and no light is
// scattered from it.
func TestABandNoRangeHoldsIsOpaque(t *testing.T) {
	g := Gas{Ranges: map[radiation.Range]Optics{
		thing.Visible: {Absorb: 3, Scatter: 4}}}
	for _, nm := range []float64{1000, 1e20} {
		b := maths.Band{Nanometres: nm}
		if a := g.Absorb(b); !math.IsInf(a, 1) {
			t.Errorf("%v nm absorbs %v, want endless", nm, a)
		}
		if s := g.Scatter(b); s != 0 {
			t.Errorf("%v nm scatters %v, want nought", nm, s)
		}
		if e := g.Extinction(b); !math.IsInf(e, 1) {
			t.Errorf("%v nm is extinguished by %v, want endless", nm, e)
		}
		if p := g.Transmit(b, 1); p != 0 {
			t.Errorf("%v nm passes %v of a metre, want none", nm, p)
		}
	}
}
