package main

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// Fresh fuel and charcoal have authored reflectances. Check the actual
// surfaces against the real solver's remaining fuel, not just a darkening
// trend: an out-of-gamut fresh colour can also get darker as it burns.
func TestFuelBedDrawsAndCountsTheActualFuelSurfaces(t *testing.T) {
	given := fuelOnFixture(t)
	given.onGPU = false
	read := readBoundarySet(t, given.set)
	stage := staging{given: given, recipes: read, world: thing.NewWorld()}
	if err := stage.terrain(read); err != nil {
		t.Fatal(err)
	}
	if err := stage.prepareBeds(read); err != nil {
		t.Fatal(err)
	}
	defer (&liveScene{beds: stage.beds}).closeBeds()
	if len(stage.beds) != 1 || len(stage.beds[0].tiles) != 64 {
		t.Fatal("fixture did not retain its 64 real fuel surfaces")
	}
	bed := &stage.beds[0]
	consumed := false
	for _, at := range []time.Duration{0, 10 * time.Second, 30 * time.Second} {
		state, err := bed.model.At(at.Seconds())
		if err != nil {
			t.Fatal(err)
		}
		world, made := thing.NewWorld(), tally{}
		if err := bed.addAt(world, &made, at); err != nil {
			t.Fatal(err)
		}
		if len(world.Things) != 64 || made.things != len(world.Things) {
			t.Fatalf("at %s: %d actual fuel surfaces, reported %d", at, len(world.Things), made.things)
		}
		for i, item := range world.Things {
			if item.Shape != bed.tiles[i] || item.Material == nil || item.Material.Name != "surface fuel" {
				t.Fatalf("at %s: cell %d lost its original surface", at, i)
			}
			fuel := state[i].Fuel
			if at == 0 && fuel != 1 {
				t.Fatal("fresh control lost its full fuel fraction")
			}
			if at > 0 && fuel < .99 {
				consumed = true
			}
			// Linear interpolation between the authored charcoal and fresh
			// endpoints, written as intercepts and channel slopes. This
			// verifies the solver-to-material bridge, not fuel chemistry.
			want := maths.RGB(.012+.238*fuel, .01+.16*fuel, .008+.047*fuel)
			actual := item.Material.Colour
			if !(math.Abs(actual.R-want.R) <= 1e-12 &&
				math.Abs(actual.G-want.G) <= 1e-12 && math.Abs(actual.B-want.B) <= 1e-12) {
				t.Fatalf("at %s cell %d: reflectance %v, want %v for fuel %.17g", at, i, actual, want, fuel)
			}
			for _, channel := range []float64{actual.R, actual.G, actual.B} {
				if !(channel >= 0 && channel <= 1) {
					t.Fatalf("at %s cell %d: nonphysical reflectance %v", at, i, actual)
				}
			}
		}
		active := 0
		for _, volume := range world.Volumes {
			if field, ok := volume.Medium.(*flame.Field); ok {
				active += len(field.Pieces())
			}
		}
		if at == 0 && active == 0 {
			t.Fatal("fresh hot control drew no flame patches")
		}
		if made.burning != active {
			t.Fatalf("at %s: %d actual active patches, reported %d", at, active, made.burning)
		}
	}
	if !consumed {
		t.Fatal("late controls did not consume enough fuel to exercise charcoal reflectance")
	}
}
