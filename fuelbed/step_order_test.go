package fuelbed_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/fuelbed"
)

func TestFixedTickBoundaryDoesNotLoseTheThirdStep(t *testing.T) {
	s := fixture()
	s.Step, s.Heating, s.Cooling, s.Diffusion = 0.1, 1, 1, 0
	s.Initial[4].Excess = 900
	got := must(t, s, 0.3)
	// Three Euler cooling steps have the exact decimal factor 0.9^3.
	if math.Abs(got[4].Excess-656.1) > 1e-9 {
		t.Fatal("0.3 seconds missed a completed tick", got[4])
	}
}

func TestCoupledHeatAndFuelHaveFirstOrderTimeConvergence(t *testing.T) {
	s := fixtureWithHot()
	coarse := must(t, s, 5)
	s.Step /= 2
	fine := must(t, s, 5)
	s.Step /= 2
	finer := must(t, s, 5)
	var temperature, fuel [2]float64
	for i := range coarse {
		for k, pair := range [][2]fuelbed.Cell{{coarse[i], fine[i]}, {fine[i], finer[i]}} {
			temperature[k] += math.Pow(pair[0].Excess-pair[1].Excess, 2)
			fuel[k] += math.Pow(pair[0].Fuel-pair[1].Fuel, 2)
		}
	}
	for name, errors := range map[string][2]float64{"temperature": temperature, "fuel": fuel} {
		ratio := math.Sqrt(errors[0] / errors[1])
		if errors[1] <= 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) || ratio < 1.8 || ratio > 2.2 {
			t.Fatal("halving the step did not halve the error", name, ratio, errors)
		}
		t.Logf("%s step-difference ratio %.8f", name, ratio)
	}
}
