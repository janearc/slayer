package fuelbed_test

import (
	"math"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/fuelbed"
)

type advectionCase struct {
	Width, Depth int
	Velocity     [2]float64 `yaml:"velocity-m-s"`
	Initial      []float64  `yaml:"initial-excess-k"`
	One          []float64  `yaml:"after-one-step-k"`
	Two          []float64  `yaml:"after-two-steps-k"`
	Diffusion    *float64   `yaml:"diffusion-m2-s"`
}

type advectionContract struct {
	Cell             float64 `yaml:"cell-m"`
	Step             float64 `yaml:"step-s"`
	Diffusion        float64 `yaml:"diffusion-m2-s"`
	Cooling          float64 `yaml:"cooling-per-k"`
	Heating          float64 `yaml:"heating-k-s"`
	Activation       float64 `yaml:"activation-k"`
	Consumption      float64 `yaml:"consumption-per-s"`
	Ambient          float64 `yaml:"ambient-k"`
	Fuel             float64
	Cases            map[string]advectionCase
	CPUHeld          float64    `yaml:"cpu-tolerance-k"`
	MetalHeld        float64    `yaml:"metal-tolerance-k"`
	ReactingHeatHeld float64    `yaml:"reacting-tolerance-k"`
	ReactingFuelHeld float64    `yaml:"reacting-tolerance-fuel"`
	ReactingVelocity [2]float64 `yaml:"reacting-velocity-m-s"`
	Times            []float64  `yaml:"reacting-times-s"`
}

func advectionData(test *testing.T) advectionContract {
	test.Helper()
	const path = "testdata/advection-contract.yaml"
	raw, err := os.ReadFile(path)
	if err != nil {
		test.Fatal(err)
	}
	var data advectionContract
	if err := yamlfile.Read(path, raw, &data); err != nil {
		test.Fatal(err)
	}
	return data
}

func (data advectionContract) spec(row advectionCase) fuelbed.Spec {
	spec := fuelbed.Spec{Width: row.Width, Depth: row.Depth, Cell: data.Cell,
		Step: data.Step, Diffusion: data.Diffusion, Heating: data.Heating,
		Activation: data.Activation, Cooling: data.Cooling,
		Consumption: data.Consumption, Ambient: data.Ambient}
	if row.Diffusion != nil {
		spec.Diffusion = *row.Diffusion
	}
	for _, heat := range row.Initial {
		spec.Initial = append(spec.Initial, fuelbed.Cell{Excess: heat, Fuel: data.Fuel})
	}
	return spec
}

func (row advectionCase) transport() fuelbed.Transport {
	return fuelbed.Transport{East: row.Velocity[0], South: row.Velocity[1]}
}

func checkAdvectedWitness(test *testing.T, evolution fuelbed.Evolution,
	data advectionContract, row advectionCase, held float64) {
	test.Helper()
	for index, expected := range [][]float64{row.One, row.Two} {
		if len(expected) == 0 {
			continue
		}
		got, err := evolution.At(float64(index+1) * data.Step)
		if err != nil || len(got) != len(expected) {
			test.Fatal("witness state missing", got, err)
		}
		for cell, want := range expected {
			if !(math.Abs(got[cell].Excess-want) <= held) || got[cell].Fuel != data.Fuel {
				test.Fatal("hand-worked transport changed", index, cell, got[cell], want)
			}
		}
	}
}

func TestAdvectionMatchesIndependentDirectionalAndEdgeSteps(test *testing.T) {
	data := advectionData(test)
	for name, row := range data.Cases {
		test.Run(name, func(test *testing.T) {
			bed, err := fuelbed.NewAdvected(data.spec(row), row.transport())
			if err != nil {
				test.Fatal(err)
			}
			checkAdvectedWitness(test, bed, data, row, data.CPUHeld)
		})
	}
}

func TestZeroTransportRetainsTheLegacyReferenceExactly(test *testing.T) {
	spec := fixtureWithHot()
	legacy, err := fuelbed.New(spec)
	if err != nil {
		test.Fatal(err)
	}
	zero, err := fuelbed.NewAdvected(spec, fuelbed.Transport{})
	if err != nil {
		test.Fatal(err)
	}
	for _, at := range advectionData(test).Times {
		want, firstErr := legacy.At(at)
		got, secondErr := zero.At(at)
		if firstErr != nil || secondErr != nil || !slices.Equal(got, want) {
			test.Fatal("zero flow changed legacy arithmetic or rewind", at, firstErr, secondErr)
		}
	}
}

func TestAdvectionJointLimitRefusesTheFirstVelocityPastItsBound(test *testing.T) {
	data := advectionData(test)
	spec := data.spec(data.Cases["ambient-inflow"])
	limit := spec.Cell / spec.Step
	if _, err := fuelbed.NewAdvected(spec, fuelbed.Transport{East: limit}); err != nil {
		test.Fatal("exact Courant bound refused", err)
	}
	_, err := fuelbed.NewAdvected(spec,
		fuelbed.Transport{East: math.Nextafter(limit, math.Inf(1))})
	if err == nil || !strings.Contains(err.Error(), "must be <= 1") {
		test.Fatal("first representable excessive velocity was accepted", err)
	}
}

func TestAdvectionReleasesHeatOnlyForTheFuelActuallyConsumed(test *testing.T) {
	data := advectionData(test)
	row := data.Cases["ambient-inflow"]
	spec := data.spec(row)
	spec.Initial[0].Fuel = 1
	bed, err := fuelbed.NewAdvected(spec, row.transport())
	if err != nil {
		test.Fatal(err)
	}
	got, err := bed.At(spec.Step)
	if err != nil {
		test.Fatal(err)
	}
	// Independent equation: dt*Cs*F*exp(-B/u); half the old heat remains
	// in this one-cell example, with zero-excess inflow on both axes.
	burned := spec.Step * spec.Consumption * math.Exp(-spec.Activation/spec.Initial[0].Excess)
	wantFuel := 1 - burned
	wantHeat := spec.Initial[0].Excess/2 + spec.Heating/spec.Consumption*burned
	if !(math.Abs(got[0].Excess-wantHeat) <= data.CPUHeld) ||
		!(math.Abs(got[0].Fuel-wantFuel) <= data.CPUHeld) {
		test.Fatal("reaction no longer couples consumed fuel to heat release", got[0])
	}
}
