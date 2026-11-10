package fuelbed

import (
	"math"
	"strings"
	"testing"
)

func quietBoundarySpec() Spec {
	return Spec{Width: 1, Depth: 1, Cell: 1, Step: .5,
		Heating: 2, Activation: 1, Consumption: .5, Ambient: 300,
		Initial: []Cell{{Excess: 100, Fuel: 0}}}
}

// These endpoints are the stencil's storage and positivity contracts. Check
// both sides so a broader refusal cannot make an invalid-input test pass.
func TestGridAndStateEndpoints(t *testing.T) {
	for _, dimensions := range [][2]int{{1, 1}, {256, 1}, {1, 256}, {256, 256}, {0, 3}, {3, 0}, {257, 1}, {1, 257}} {
		s := quietBoundarySpec()
		s.Width, s.Depth = dimensions[0], dimensions[1]
		s.Initial = make([]Cell, s.Width*s.Depth)
		_, err := New(s)
		valid := s.Width >= 1 && s.Depth >= 1 && s.Width <= 256 && s.Depth <= 256
		if valid && err != nil || !valid && (err == nil || !strings.Contains(err.Error(), "dimensions must be 1..256")) {
			t.Fatalf("grid %v: %v", dimensions, err)
		}
	}
	for _, excess := range []float64{0, 10000, math.Nextafter(10000, math.Inf(1))} {
		s := quietBoundarySpec()
		s.Initial[0].Excess = excess
		_, err := New(s)
		if excess <= 10000 && err != nil || excess > 10000 && (err == nil || !strings.Contains(err.Error(), "invalid initial cell")) {
			t.Fatalf("excess %.17g: %v", excess, err)
		}
	}
}

func TestExplicitStepPositivityLimits(t *testing.T) {
	base := quietBoundarySpec()
	tests := []struct {
		name   string
		change func(*Spec)
		valid  bool
	}{
		{"joint diffusion and cooling limit", func(s *Spec) { s.Diffusion, s.Cooling = .375, .25 }, true},
		{"joint diffusion and cooling overflow", func(s *Spec) { s.Diffusion, s.Cooling = .376, .25 }, false},
		{"submetre diffusion overflow", func(s *Spec) { s.Cell, s.Diffusion = .5, .13 }, false},
		{"cooling alone overflow", func(s *Spec) { s.Cooling = 1.001 }, false},
		{"consumption at limit", func(s *Spec) { s.Consumption = 2 }, true},
		{"consumption beyond limit", func(s *Spec) { s.Consumption = 2.1 }, false},
		{"one second", func(s *Spec) { s.Step = 1 }, true},
		{"above one second", func(s *Spec) { s.Step = math.Nextafter(1, 2) }, false},
		{"minimum step", func(s *Spec) { s.Step = .0001 }, true},
		{"below minimum step", func(s *Spec) { s.Step = math.Nextafter(.0001, 0) }, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := base
			test.change(&s)
			_, err := New(s)
			if test.valid && err != nil || !test.valid && (err == nil || !strings.Contains(err.Error(), "unstable or impractical fixed time step")) {
				t.Fatalf("valid=%v: %v", test.valid, err)
			}
		})
	}
}

func TestEveryCoefficientRefusesNonfiniteValues(t *testing.T) {
	for index := range 8 {
		for _, value := range []float64{math.Inf(-1), math.Inf(1), math.NaN()} {
			s := quietBoundarySpec()
			fields := []*float64{&s.Cell, &s.Step, &s.Heating, &s.Activation, &s.Consumption, &s.Ambient, &s.Diffusion, &s.Cooling}
			*fields[index] = value
			if _, err := New(s); err == nil || !strings.Contains(err.Error(), "coefficient") {
				t.Fatalf("coefficient %d=%v: %v", index, value, err)
			}
		}
	}
	for _, value := range []float64{math.Inf(-1), math.Inf(1), math.NaN()} {
		if finite(value) {
			t.Fatalf("nonfinite value admitted: %v", value)
		}
	}
}

func TestCompletedTickDomainIncludesOneDay(t *testing.T) {
	for _, at := range []float64{0, 86400} {
		got, err := ticks(at, 1)
		if err != nil || got != int64(at) {
			t.Fatalf("completed ticks at %.17g: %d %v", at, got, err)
		}
	}
}

func TestReactionStopsAtNonpositiveHeatOrFuel(t *testing.T) {
	for _, cell := range []Cell{{Excess: -1, Fuel: 1}, {Excess: 1, Fuel: -1}, {Excess: 0, Fuel: 1}, {Excess: 1, Fuel: 0}} {
		if got := Reaction(cell, 1); got != 0 {
			t.Fatalf("inactive state %v reacted: %.17g", cell, got)
		}
	}
	if got := Reaction(Cell{Excess: 1, Fuel: .5}, 1); math.Abs(got-.5/math.E) > 1e-16 {
		t.Fatalf("positive low-temperature state did not react: %.17g", got)
	}
}
