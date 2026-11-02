package gas

import (
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// the four gases' tables, range by range, as air.go gives them. these are
// the table's figures, held so nothing moves, not measured values.
//
// some are the first tables' and known wrong, and air.go says which beside each: the
// chappuis absorption about seventeen times low, venus's 65 against 36
// times the molecules, haze carrying air's x-ray and gamma. when they are
// mended after the switch these are the lines to change with them.
//
// the visible entry is read only when a gas has no rule, which air, venus
// and mars have, so for those the rule's tests hold the visible, and this
// holds the entry all the same.
var held = []struct {
	gas    Gas
	phase  float64
	ranges map[radiation.Range]Optics
}{
	{Air(), 0, map[radiation.Range]Optics{
		radiation.Gamma():       {Absorb: 3.3e-3},
		radiation.XRay():        {Absorb: 0.6},
		radiation.Ultraviolet(): {Absorb: 1e-2, Scatter: 6e-5},
		thing.Visible:           {Absorb: 3e-7, Scatter: 1.16e-5},
		radiation.Infrared():    {Absorb: 5e-5, Scatter: 1e-7},
		radiation.Microwave():   {Absorb: 1e-6},
		radiation.Radio():       {Absorb: 1e-7},
	}},
	{Haze(), 0.7, map[radiation.Range]Optics{
		thing.Visible:           {Absorb: 1e-5, Scatter: 1e-4},
		radiation.Ultraviolet(): {Absorb: 2e-5, Scatter: 1.5e-4},
		radiation.Infrared():    {Absorb: 5e-6, Scatter: 3e-5},
		radiation.Microwave():   {},
		radiation.Radio():       {},
		radiation.XRay():        {Absorb: 0.6},
		radiation.Gamma():       {Absorb: 3.3e-3},
	}},
	{Venus(), 0, map[radiation.Range]Optics{
		radiation.Gamma():       {Absorb: 0.21},
		radiation.XRay():        {Absorb: 39},
		radiation.Ultraviolet(): {Absorb: 1e-2, Scatter: 1e-2},
		thing.Visible:           {Absorb: 1e-6, Scatter: 2e-3},
		radiation.Infrared():    {Absorb: 1e-2, Scatter: 1e-5},
		radiation.Microwave():   {Absorb: 1e-7},
		radiation.Radio():       {Absorb: 1e-7},
	}},
	{Mars(), 0.7, map[radiation.Range]Optics{
		radiation.Gamma():       {Absorb: 3e-5},
		radiation.XRay():        {Absorb: 5e-3},
		radiation.Ultraviolet(): {Absorb: 1e-6, Scatter: 6e-5},
		thing.Visible:           {Absorb: 1e-5, Scatter: 3.5e-5},
		radiation.Infrared():    {Absorb: 1e-4, Scatter: 1e-5},
		radiation.Microwave():   {Absorb: 1e-9},
		radiation.Radio():       {Absorb: 1e-9},
	}},
}

func TestTheTablesAreHeldAsTheyAre(t *testing.T) {
	for _, h := range held {
		if h.gas.Phase != h.phase {
			t.Errorf("%s's asymmetry is %v, held at %v", h.gas.Name, h.gas.Phase, h.phase)
		}
		if len(h.gas.Ranges) != len(h.ranges) {
			t.Errorf("%s has %d ranges, held at %d", h.gas.Name, len(h.gas.Ranges), len(h.ranges))
		}
		for r, want := range h.ranges {
			if got, ok := h.gas.Ranges[r]; !ok || got != want {
				t.Errorf("%s's %v is %v, held at %v", h.gas.Name, r, got, want)
			}
		}
	}
}

// a gas answers each range from its table: a band picked in the middle of
// each range, past the rule's reach, gives the table's figure through
// Absorb and Scatter. where a gas has a rule the visible, the near
// ultraviolet and the near infrared are the rule's, so those are left to
// the rule's tests.
func TestEachRangeAnswersFromItsTable(t *testing.T) {
	for _, h := range held {
		for _, c := range []struct {
			r  radiation.Range
			nm float64
		}{
			{radiation.Gamma(), 5e-4},
			{radiation.XRay(), 1},
			{radiation.Ultraviolet(), 250},
			{radiation.Infrared(), 5000},
			{radiation.Microwave(), 5e6},
			{radiation.Radio(), 1e9},
		} {
			b := maths.Band{Nanometres: c.nm}
			want := h.ranges[c.r]
			if got := h.gas.Absorb(b); got != want.Absorb {
				t.Errorf("%s at %v nm absorbs %v, held at %v", h.gas.Name, c.nm, got, want.Absorb)
			}
			if got := h.gas.Scatter(b); got != want.Scatter {
				t.Errorf("%s at %v nm scatters %v, held at %v", h.gas.Name, c.nm, got, want.Scatter)
			}
		}
	}
}
