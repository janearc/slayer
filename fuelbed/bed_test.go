package fuelbed_test

import (
	"github.com/janearc/slayer/fuelbed"
	"math"
	"testing"
)

func fixture() fuelbed.Spec {
	return fuelbed.Spec{Width: 3, Depth: 3, Cell: 1, Step: .01, Diffusion: .2,
		Heating: 180, Activation: 550, Cooling: .00005, Consumption: .16, Ambient: 300,
		Initial: make([]fuelbed.Cell, 9)}
}
func must(t *testing.T, s fuelbed.Spec, at float64) []fuelbed.Cell {
	t.Helper()
	b, e := fuelbed.New(s)
	if e != nil {
		t.Fatal(e)
	}
	c, e := b.At(at)
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func TestColdFuelCannotIgniteItself(t *testing.T) {
	s := fixture()
	for i := range s.Initial {
		s.Initial[i].Fuel = 1
	}
	for _, c := range must(t, s, 20) {
		if c.Excess != 0 || c.Fuel != 1 {
			t.Fatal(c)
		}
	}
	if fuelbed.Reaction(fuelbed.Cell{Excess: 0, Fuel: 1}, 550) != 0 {
		t.Fatal("cold reaction")
	}
}
func TestReactionConsumesFuelAndProducesCorrespondingHeat(t *testing.T) {
	s := fixture()
	s.Diffusion = 0
	s.Cooling = 0
	for i := range s.Initial {
		s.Initial[i] = fuelbed.Cell{Excess: 900, Fuel: 1}
	}
	cells := must(t, s, 4)
	for _, c := range cells {
		if c.Fuel >= .9 || c.Fuel <= 0 || c.Excess <= 900 {
			t.Fatal(c)
		}
		if math.Abs(c.Excess-900-s.Heating/s.Consumption*(1-c.Fuel)) > 1e-8 {
			t.Fatal("heat/fuel balance", c)
		}
	}
}
func TestEmptyHotFuelOnlyCools(t *testing.T) {
	s := fixture()
	s.Diffusion = 0
	for i := range s.Initial {
		s.Initial[i].Excess = 900
	}
	for _, c := range must(t, s, 10) {
		want := 900 * math.Exp(-s.Heating*s.Cooling*10)
		if c.Fuel != 0 || math.Abs(c.Excess-want) > .01 {
			t.Fatal(c, want)
		}
	}
	if fuelbed.Reaction(fuelbed.Cell{Excess: 900, Fuel: 0}, 550) != 0 {
		t.Fatal("empty reaction")
	}
}
func TestHeatTransferReachesNeighboursWithoutRefuellingThem(t *testing.T) {
	s := fixture()
	s.Initial[4] = fuelbed.Cell{Excess: 900, Fuel: 1}
	s.Initial[5].Fuel = 1
	cells := must(t, s, 2)
	if cells[5].Excess < 100 || cells[5].Fuel >= 1 {
		t.Fatal("neighbour did not preheat/burn", cells[5])
	}
	for i, c := range cells {
		if i != 4 && i != 5 && c.Fuel != 0 {
			t.Fatal("created fuel", c)
		}
	}
	s.Diffusion = 0
	if c := must(t, s, 2)[5]; c.Excess != 0 || c.Fuel != 1 {
		t.Fatal("spread without heat transport", c)
	}
}
func TestSnapshotsRewindAndCannotMutateTheBed(t *testing.T) {
	s := fixture()
	s.Initial[4] = fuelbed.Cell{Excess: 900, Fuel: 1}
	b, e := fuelbed.New(s)
	if e != nil {
		t.Fatal(e)
	}
	want, _ := b.At(1)
	s.Initial[4].Excess = 0
	want[0].Fuel = 999
	b.At(2)
	got, e := b.At(1)
	if e != nil {
		t.Fatal(e)
	}
	ref := must(t, fixtureWithHot(), 1)
	for i := range got {
		if got[i] != ref[i] {
			t.Fatal("mutable/history dependent", got[i], ref[i])
		}
	}
}
func fixtureWithHot() fuelbed.Spec {
	s := fixture()
	s.Initial[4] = fuelbed.Cell{Excess: 900, Fuel: 1}
	return s
}
func TestStepConvergesAndFuelNeverGrows(t *testing.T) {
	s := fixtureWithHot()
	coarse := must(t, s, 5)
	s.Step /= 2
	fine := must(t, s, 5)
	for i, c := range coarse {
		if c.Excess < 0 || c.Fuel < 0 || c.Fuel > s.Initial[i].Fuel {
			t.Fatal(c)
		}
		if math.Abs(c.Excess-fine[i].Excess) > .6 || math.Abs(c.Fuel-fine[i].Fuel) > .001 {
			t.Fatal("step did not converge", c, fine[i])
		}
	}
}
func TestInvalidGridsAndTimesAreRefused(t *testing.T) {
	for _, change := range []func(*fuelbed.Spec){
		func(s *fuelbed.Spec) { s.Width = 0 }, func(s *fuelbed.Spec) { s.Width = 10000 },
		func(s *fuelbed.Spec) { s.Cell = 0 }, func(s *fuelbed.Spec) { s.Step = 0 },
		func(s *fuelbed.Spec) { s.Step = 100 }, func(s *fuelbed.Spec) { s.Diffusion = -1 },
		func(s *fuelbed.Spec) { s.Activation = math.NaN() }, func(s *fuelbed.Spec) { s.Heating = math.Inf(1) },
		func(s *fuelbed.Spec) { s.Consumption = 0 }, func(s *fuelbed.Spec) { s.Ambient = -1 },
		func(s *fuelbed.Spec) { s.Initial[0].Fuel = 1.1 }, func(s *fuelbed.Spec) { s.Initial[0].Excess = -1 },
		func(s *fuelbed.Spec) { s.Initial = s.Initial[:1] },
	} {
		s := fixture()
		change(&s)
		if _, e := fuelbed.New(s); e == nil {
			t.Fatal("invalid spec accepted", s)
		}
	}
	b, e := fuelbed.New(fixture())
	if e != nil {
		t.Fatal(e)
	}
	for _, at := range []float64{-1, math.NaN(), math.Inf(1), 86401} {
		if _, e := b.At(at); e == nil {
			t.Fatal("invalid time", at)
		}
	}
}

func TestGrassProxyCoefficientsHoldTheirPublishedPowers(t *testing.T) {
	s := fuelbed.Grass(fixture())
	for i, pair := range [][2]float64{{s.Diffusion, .21360}, {s.Heating, 187.93}, {s.Activation, 558.49}, {s.Cooling, .000048372}, {s.Consumption, .16250}} {
		if math.Abs(pair[0]-pair[1]) > 1e-12 {
			t.Fatal("wrong coefficient power", i, pair)
		}
	}
}
func TestWideMissingFuelBreakStopsTheCombustionWave(t *testing.T) {
	for _, gap := range []bool{false, true} {
		s := fuelbed.Grass(fuelbed.Spec{Width: 80, Depth: 1, Cell: .5, Step: .01, Ambient: 300, Initial: make([]fuelbed.Cell, 80)})
		for i := range s.Initial {
			x := (float64(i) - 8) * .5
			f := 1.
			if gap && i >= 24 && i < 40 {
				f = 0
			}
			s.Initial[i] = fuelbed.Cell{Excess: 900 * math.Exp(-x*x/4), Fuel: f}
		}
		cells := must(t, s, 180)
		if gap {
			if cells[48].Fuel < .99 {
				t.Fatal("front crossed an eight metre break", cells[48])
			}
			for i := 24; i < 40; i++ {
				if cells[i].Fuel != 0 {
					t.Fatal("break refuelled")
				}
			}
		} else if cells[48].Fuel > .05 {
			t.Fatal("unbroken fuel did not carry the wave", cells[48])
		}
	}
}

func TestInsulatedDiffusionConservesHeatWithoutWrappingEdges(t *testing.T) {
	s := fixture()
	s.Heating = 1
	s.Cooling = 0
	s.Initial[0].Excess = 900
	first := must(t, s, s.Step)
	if first[0].Excess >= 900 || first[1].Excess <= 0 || first[2].Excess != 0 || first[6].Excess != 0 {
		t.Fatal("wrong boundary stencil", first)
	}
	cells := must(t, s, 10)
	sum := 0.0
	for _, c := range cells {
		sum += c.Excess
		if c.Fuel != 0 {
			t.Fatal("diffusion created fuel")
		}
	}
	if math.Abs(sum-900) > 1e-8 {
		t.Fatal("insulated layer lost/gained heat", sum)
	}
}
