//go:build darwin

package fuelbed_test

import (
	"math"
	"os"
	"testing"

	"github.com/janearc/slayer/fuelbed"
	"github.com/janearc/slayer/set"
)

func TestMetalMatchesTheShippedGrassBedThroughSixtySeconds(t *testing.T) {
	path := "../doc/fire-fuel.yaml"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	scene, err := set.Read(path, raw)
	if err != nil {
		t.Fatal(err)
	}
	beds := scene.FuelBeds()
	if len(beds) != 1 {
		t.Fatal("study must contain one bed")
	}
	b := beds[0]
	spec := fuelbed.Grass(fuelbed.Spec{Width: b.Width(), Depth: b.Depth(), Cell: b.Cell(),
		Step: b.Step(), Ambient: b.Ambient(), Initial: make([]fuelbed.Cell, b.Width()*b.Depth())})
	if spec.Width != 32 || spec.Depth != 32 {
		t.Fatal("shipped grid changed", spec.Width, spec.Depth)
	}
	for z := 0; z < spec.Depth; z++ {
		for x := 0; x < spec.Width; x++ {
			dx := (float64(x) + 0.5 - float64(spec.Width)/2) * spec.Cell
			dz := (float64(z) + 0.5 - float64(spec.Depth)/2) * spec.Cell
			spec.Initial[z*spec.Width+x] = fuelbed.Cell{
				Excess: (b.Hot() - b.Ambient()) * math.Exp(-(dx*dx+dz*dz)/(b.Radius()*b.Radius())),
				Fuel:   b.Fuel()}
		}
	}
	cpu, err := fuelbed.New(spec)
	if err != nil {
		t.Fatal(err)
	}
	device, err := fuelbed.NewMetal(spec)
	requireFuelDevice(t, err)
	defer device.Close()
	for _, at := range []float64{0, 10, 30, 60, 30, 60} {
		want, err := cpu.At(at)
		if err != nil {
			t.Fatal(err)
		}
		got, err := device.At(at)
		if err != nil {
			t.Fatal(err)
		}
		temperature, fuel := 0.0, 0.0
		reacted, warmed := 0, 0
		for i, c := range got {
			if math.IsNaN(c.Excess+c.Fuel) || math.IsInf(c.Excess+c.Fuel, 0) {
				t.Fatal("nonfinite cell", at, i, c)
			}
			temperature = max(temperature, math.Abs(c.Excess-want[i].Excess))
			fuel = max(fuel, math.Abs(c.Fuel-want[i].Fuel))
			if c.Fuel < spec.Initial[i].Fuel-0.01 {
				reacted++
			}
			if spec.Initial[i].Excess < 1 && c.Excess > 1 {
				warmed++
			}
		}
		t.Logf("%gs: largest temperature error %.9g K, fuel %.9g; %d reacting, %d preheated cells", at, temperature, fuel, reacted, warmed)
		if temperature > 0.1 || fuel > 0.0001 {
			t.Fatal("shipped Grass CPU/Metal divergence", at, temperature, fuel)
		}
		if at == 60 && (reacted == 0 || warmed == 0) {
			t.Fatal("comparison did not exercise combustion and heat transfer", reacted, warmed)
		}
	}
}
