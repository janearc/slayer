package gas

import (
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// scaled is a gas for Per, in figures that are exact in binary so that a
// product or a quotient is exact too. test figures, and mean nothing.
func scaled() Gas {
	return Gas{
		Name:    "scaled",
		Phase:   0.5,
		Sources: []string{"a test"},
		Ranges: map[radiation.Range]Optics{
			thing.Visible:        {Absorb: 0.25, Scatter: 0.5},
			radiation.Infrared(): {Absorb: 1, Scatter: 2},
		},
		Visible: func(nm float64) (absorb, scatter float64) {
			return nm / 1000, 0.125
		},
		Lines: map[float64]float64{656: 0.5, 500: 1.5},
	}
}

// per four metres a unit is four times the gas: each range's absorption
// and scattering, the visible rule's both parts at any wavelength, and each
// line's strength, and not a quarter of any. what the gas is, its name,
// asymmetry and sources, is kept.
func TestPerScalesEveryNumberByTheUnit(t *testing.T) {
	g := scaled().Per(4)
	if got := g.Ranges[thing.Visible]; got != (Optics{Absorb: 1, Scatter: 2}) {
		t.Errorf("the visible range is %v, want 1 and 2", got)
	}
	if got := g.Ranges[radiation.Infrared()]; got != (Optics{Absorb: 4, Scatter: 8}) {
		t.Errorf("the infrared range is %v, want 4 and 8", got)
	}
	if a, s := g.Visible(500); a != 2 || s != 0.5 {
		t.Errorf("the rule at 500 nm is %v and %v, want 2 and 0.5", a, s)
	}
	if a, s := g.Visible(250); a != 1 || s != 0.5 {
		t.Errorf("the rule at 250 nm is %v and %v, want 1 and 0.5", a, s)
	}
	if len(g.Lines) != 2 || g.Lines[656] != 2 || g.Lines[500] != 6 {
		t.Errorf("the lines are %v, want 656 at 2 and 500 at 6", g.Lines)
	}
	if g.Name != "scaled" || g.Phase != 0.5 || len(g.Sources) != 1 {
		t.Errorf("per lost what the gas is: %q, %v, %v", g.Name, g.Phase, g.Sources)
	}
	// and the optics read the scaled rule and the scaled table
	if a := g.Absorb(maths.Band{Nanometres: 500}); a != 2 {
		t.Errorf("the scaled gas absorbs %v at 500 nm, want 2", a)
	}
	if s := g.Scatter(maths.Band{Nanometres: 5000}); s != 8 {
		t.Errorf("the scaled gas scatters %v at 5000 nm, want 8", s)
	}
}

// the gas Per was made from is as it was: its ranges, its rule and its
// lines are not the scaled gas's, and are not shared with it.
func TestPerLeavesTheGasItCameFrom(t *testing.T) {
	was := scaled()
	was.Per(4)
	if got := was.Ranges[radiation.Infrared()]; got != (Optics{Absorb: 1, Scatter: 2}) {
		t.Errorf("the infrared range became %v", got)
	}
	if a, s := was.Visible(500); a != 0.5 || s != 0.125 {
		t.Errorf("the rule at 500 nm became %v and %v", a, s)
	}
	if was.Lines[656] != 0.5 || was.Lines[500] != 1.5 {
		t.Errorf("the lines became %v", was.Lines)
	}
}

// a gas with no rule has none after, and one with no lines has none: not an
// empty rule that answers zero, nor an empty map. an empty map of lines is kept
// as a map.
func TestPerAddsNeitherARuleNorLines(t *testing.T) {
	bare := Gas{Ranges: map[radiation.Range]Optics{
		thing.Visible: {Absorb: 0.25}}}
	got := bare.Per(4)
	if got.Visible != nil {
		t.Error("per gave a gas with no rule a rule")
	}
	if got.Lines != nil {
		t.Errorf("per gave a gas with no lines the lines %v", got.Lines)
	}
	if got.Absorb(maths.Band{Nanometres: 550}) != 1 {
		t.Error("the table was not scaled")
	}
	empty := Gas{Lines: map[float64]float64{}}.Per(4)
	if empty.Lines == nil || len(empty.Lines) != 0 {
		t.Errorf("a gas of no lines became %v", empty.Lines)
	}
}
