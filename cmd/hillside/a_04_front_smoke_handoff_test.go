package main

import (
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/plume"
	"github.com/janearc/slayer/scene/thing"
)

// One two-second burn ends exactly when the next starts. A particle tail
// belongs in the spark budget; adding it to fuel occupancy would halve
// this front's smoke strength throughout both burns.
func TestFrontKeepsFlamesAndFullStrengthSmokeAcrossAnExactFuelHandoff(t *testing.T) {
	given := frontOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "speed: {value: 1000, unit: m/s}",
		"speed: {value: 1, unit: m/s}", 1)
	text = strings.Replace(text, "kind: fern, count: 4", "kind: fern, count: 2", 1)
	text = strings.Replace(text, "fade-time: {value: 0.2, unit: s}",
		"fade-time: {value: 0, unit: s}", 1)
	if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	read := readBoundarySet(t, given.set)
	given.exposure = 1
	stage := staging{given: given, recipes: read, world: thing.NewWorld(),
		places: map[string]maths.Vec{"the tree": maths.V(10, 0, 5)},
		fuels: map[string][]groveTree{"front fuel": {
			{at: maths.V(10, 0, 5)}, {at: maths.V(12, 0, 5)},
		}}}
	if err := stage.prepareFronts(read); err != nil {
		t.Fatal(err)
	}
	if len(stage.fronts) != 1 {
		t.Fatalf("prepared %d fronts, want one", len(stage.fronts))
	}
	front := stage.fronts[0]
	row, has := read.Recipe("wood fire")
	if !has {
		t.Fatal("fixture lost wood fire")
	}
	spec, has := row.Fire()
	if !has || spec.Smoke == nil {
		t.Fatal("fixture lost its smoke")
	}
	// The checked YAML asks for an eight-metre radius. A single active
	// target at zero fade retains the recipe's full yield at that width.
	smokeSpec := *spec.Smoke
	smokeSpec.Base = 16
	reference, err := plume.New(smokeSpec)
	if err != nil {
		t.Fatal(err)
	}
	point := maths.V(0, 5, 0)
	want := reference.Density(point)
	if !(want > 0) {
		t.Fatal("reference sample missed the plume")
	}
	for _, sample := range []struct {
		at     time.Duration
		origin maths.Vec
	}{
		{2 * time.Second, maths.V(10, 0, 5)},
		{3 * time.Second, maths.V(10, 0, 5)},
		{4 * time.Second, maths.V(12, 0, 5)},
		{5 * time.Second, maths.V(12, 0, 5)},
	} {
		at := sample.at
		if len(front.plan.Active(at)) != 1 {
			t.Fatalf("handoff fixture does not have one active fuel target at %s", at)
		}
		world := thing.NewWorld()
		if err := front.addAt(world, &tally{}, at); err != nil {
			t.Fatal(err)
		}
		flames := 0
		for _, volume := range world.Volumes {
			field, has := volume.Medium.(*flame.Field)
			if !has {
				continue
			}
			for _, piece := range field.Tick(at).(*flame.Field).Pieces() {
				flames++
				if position := volume.At(at).Out(piece.At); position != sample.origin {
					t.Fatalf("flame at %s moved to %v, want %v", at, position, sample.origin)
				}
			}
		}
		if flames != 1 || len(world.Lamps) == 0 {
			t.Fatalf("handoff at %s lost flame or light: %d flames, %d lamps",
				at, flames, len(world.Lamps))
		}
		count := 0
		for _, volume := range world.Volumes {
			if volume.Material.Name != "front smoke" {
				continue
			}
			count++
			if gap := volume.At(at).Origin.Sub(sample.origin).Len(); !(gap < 1e-12) {
				t.Fatalf("smoke at %s lost its active target: %v, want %v",
					at, volume.At(at).Origin, sample.origin)
			}
			if volume.Material.Colour != maths.Grey(.3) {
				t.Fatalf("shared smoke lost the authored .3 albedo: %v", volume.Material.Colour)
			}
			got := volume.Medium.Density(point)

			if !(math.Abs(got-want) <= want*1e-12) {
				t.Fatalf("smoke at %s has density %.17g, want full-yield %.17g", at, got, want)
			}
		}
		if count != 1 {
			t.Fatalf("got %d shared smoke columns at %s, want one", count, at)
		}
	}
}
