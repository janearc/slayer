package main

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/fuelbed"
	"github.com/janearc/slayer/scene/thing"
)

func TestLessFuelAtTheSameHeatReducesFlameAbsorption(t *testing.T) {
	fire := flame.Fire{Radius: .25, Height: 1, Absorption: 1,
		Hottest: measure.Kelvin(1600), Edge: measure.Kelvin(900), Seed: 5}
	field, err := flame.NewField([]flame.Ignition{{Fire: fire}}, 1e-5)
	if err != nil {
		t.Fatal(err)
	}
	var absorption []float64
	for _, fuel := range []float64{1, .25, 0} {
		spec := fuelbed.Grass(fuelbed.Spec{Width: 1, Depth: 1, Cell: .5,
			Step: .01, Ambient: 300,
			Initial: []fuelbed.Cell{{Excess: 500, Fuel: fuel}}})
		model, err := fuelbed.New(spec)
		if err != nil {
			t.Fatal(err)
		}
		bed := retainedBed{model: model, spec: spec, lightCells: 3,
			groups: []fuelGroup{{field: field, indices: []int{0}}}}
		world, made := &thing.World{}, tally{}
		if err := bed.addAt(world, &made, 0); err != nil {
			t.Fatal(err)
		}
		if fuel == 0 {
			if len(world.Volumes) != 0 || len(world.Lamps) != 0 || made.burning != 0 {
				t.Fatal("spent fuel still emitted flames or light")
			}
			continue
		}
		if len(world.Volumes) != 1 || len(world.Lamps) != 1 || made.burning != 1 {
			t.Fatal("occupied cell lost its display")
		}
		pieces := world.Volumes[0].Medium.(*flame.Field).Pieces()
		got, _, _, _ := pieces[0].Body.Samples()
		if got.Hottest != fire.Hottest || got.Edge != fire.Edge {
			t.Fatal("fuel fraction changed the prescribed temperature palette")
		}
		absorption = append(absorption, got.Absorption)
	}
	if !(absorption[0] > absorption[1] && absorption[1] > 0) ||
		math.Abs(absorption[1]/absorption[0]-.25) > 1e-14 {
		t.Fatalf("fixed-heat absorption did not follow fuel: %v", absorption)
	}
}
