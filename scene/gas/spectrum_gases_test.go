package gas

import (
	"math"
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/scene/thing"
)

// a spectrum passes, for each range, beer's law over one unit: e to the
// minus absorption plus scattering. worked by hand on a gas of test figures
// and a unit of 2: ultraviolet (0.1 and 0.2) is e^-0.6, and the visible,
// where there is a rule, is the rule's at 550 nm (0.25 and 0.25), e^-1, and
// not the table's (1 and 1), e^-4.
//
// a gas with a line gives nothing off, since lines are the aurora's.
func TestASpectrumPassesByBeersLaw(t *testing.T) {
	g := Gas{
		Ranges: map[radiation.Range]Optics{
			radiation.Ultraviolet(): {Absorb: 0.1, Scatter: 0.2},
			thing.Visible:           {Absorb: 1, Scatter: 1},
		},
		Visible: func(nm float64) (absorb, scatter float64) { return 0.25, 0.25 },
		Lines:   map[float64]float64{656: 3},
	}
	s := g.Spectrum(2)
	wantClose(t, "ultraviolet passes", s.Passes[radiation.Ultraviolet()], math.Exp(-0.6))
	wantClose(t, "the visible passes", s.Passes[thing.Visible], math.Exp(-1))
	if len(s.Passes) != 2 {
		t.Errorf("the spectrum passes %d ranges, want 2", len(s.Passes))
	}
	if s.Gives == nil || len(s.Gives) != 0 {
		t.Errorf("a gas with a line gives %v, want an empty table: lines are the aurora's", s.Gives)
	}
	g.Visible = nil
	wantClose(t, "the visible, with no rule, passes", g.Spectrum(2).Passes[thing.Visible], math.Exp(-4))
}

// haze, venus and mars pass each range by beer's law over ten metres: e to
// the minus ten times the table's absorption and scattering, taken from the
// table's figures (held in tables_held_test.go). the visible, for venus
// and mars, is the rule at 550 nm, worked by hand.
func TestTheOtherGasesPassByBeersLaw(t *testing.T) {
	const unit = 10.0
	// the rules at 550 nm: venus, 2e-3 scatter and 1e-6 plus 2e-4 times
	// e^-(22500/3200) absorb; mars, 2.4e-7 plus 3.5e-5 scatter and 5e-6
	// plus 2e-5 times 0.25 absorb
	visible := map[string]float64{
		"haze":  math.Exp(-(1e-5 + 1e-4) * unit),
		"venus": math.Exp(-(1e-6 + 2e-4*math.Exp(-22500.0/3200) + 2e-3) * unit),
		"mars":  math.Exp(-(5e-6 + 2e-5*0.25 + 2.4e-7 + 3.5e-5) * unit),
	}
	for _, g := range []Gas{Haze(), Venus(), Mars()} {
		s := g.Spectrum(unit)
		if len(s.Passes) != len(g.Ranges) {
			t.Errorf("%s passes %d ranges, has %d", g.Name, len(s.Passes), len(g.Ranges))
		}
		for r, o := range g.Ranges {
			want := math.Exp(-(o.Absorb + o.Scatter) * unit)
			if r == thing.Visible {
				want = visible[g.Name]
			}
			wantClose(t, g.Name+" passes "+r.String(), s.Passes[r], want)
		}
		if len(s.Gives) != 0 {
			t.Errorf("%s gives %v", g.Name, s.Gives)
		}
	}
}
