//go:build darwin

package fuelbed_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/fuelbed"
)

func TestMetalAdvectionMatchesTheIndependentTransportWitnesses(test *testing.T) {
	data := advectionData(test)
	for name, row := range data.Cases {
		test.Run(name, func(test *testing.T) {
			device, err := fuelbed.NewMetalAdvected(data.spec(row), row.transport())
			requireFuelDevice(test, err)
			defer device.Close()
			checkAdvectedWitness(test, device, data, row, data.MetalHeld)
		})
	}
}

func TestMetalAdvectionKeepsTheDeclaredReactingFuelBudget(test *testing.T) {
	data := advectionData(test)
	spec := fixtureWithHot()
	velocity := fuelbed.Transport{East: data.ReactingVelocity[0], South: data.ReactingVelocity[1]}
	cpu, err := fuelbed.NewAdvected(spec, velocity)
	if err != nil {
		test.Fatal(err)
	}
	device, err := fuelbed.NewMetalAdvected(spec, velocity)
	requireFuelDevice(test, err)
	defer device.Close()
	for _, at := range data.Times {
		want, err := cpu.At(at)
		if err != nil {
			test.Fatal(err)
		}
		got, err := device.At(at)
		if err != nil || len(got) != len(want) {
			test.Fatal("Metal state missing", at, err)
		}
		for index, cell := range got {
			if !(math.Abs(cell.Excess-want[index].Excess) <= data.ReactingHeatHeld) ||
				!(math.Abs(cell.Fuel-want[index].Fuel) <= data.ReactingFuelHeld) {
				test.Fatal("declared CPU/Metal budget exceeded", at, index, cell, want[index])
			}
		}
	}
}

func TestMetalTransportRefusesUnrepresentableCoefficientsBeforeOpeningADevice(test *testing.T) {
	data := advectionData(test)
	row := data.Cases["east"]
	spec := data.spec(row)
	// This finite binary64 activation exceeds float32's finite range.
	spec.Activation = float64(math.MaxFloat32) * 2
	if _, err := fuelbed.NewAdvected(spec, row.transport()); err != nil {
		test.Fatal("fixture should be valid on the CPU", err)
	}
	if device, err := fuelbed.NewMetalAdvected(spec, row.transport()); err == nil || device != nil {
		if device != nil {
			device.Close()
		}
		test.Fatal("unrepresentable activation reached a Metal session", err)
	}
}
