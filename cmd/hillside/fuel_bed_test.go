package main

import (
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"os"
	"strings"
	"testing"
	"time"
)

func fuelOnFixture(t *testing.T) asked {
	t.Helper()
	given := fireOnFixture(t)
	raw, e := os.ReadFile(given.set)
	if e != nil {
		t.Fatal(e)
	}
	config := `  fuel-beds:
    litter:
      recipe: wood fire
      model: mandel-grass
      east: 1050
      north: 8960
      width: 8
      depth: 8
      cell: {value: 0.5, unit: m}
      step: {value: 0.01, unit: s}
      ignition-radius: {value: 1.6, unit: m}
      ignition-temperature: {value: 1200, unit: K}
      ambient-temperature: {value: 300, unit: K}
      fuel: 1
`
	text := string(raw)
	start, end := strings.Index(text, "      smoke:"), strings.Index(text, "  terrain:")
	if start < 0 || end < start {
		t.Fatal("smoke fixture changed")
	}
	text = text[:start] + text[end:]
	text = strings.Replace(text, "  film:", config+"  film:", 1)
	if e = os.WriteFile(given.set, []byte(text), 0644); e != nil {
		t.Fatal(e)
	}
	return given
}
func TestFuelBedStagesChangingFuelOnBothBackends(t *testing.T) {
	for _, backend := range []string{"cpu", "gpu-native"} {
		t.Run(backend, func(t *testing.T) { checkFuelBedStaging(t, backend) })
	}
}

func checkFuelBedStaging(t *testing.T, backend string) {
	given := fuelOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), "renderer: gpu-native", "renderer: "+backend, 1))
	if err := os.WriteFile(given.set, raw, 0600); err != nil {
		t.Fatal(err)
	}

	_, _, ready, e := staged(given)
	if e != nil {
		t.Fatal(e)
	}
	if len(ready.live.beds) != 1 {
		t.Fatal("bed missing")
	}
	defer ready.live.closeBeds()
	first, _, e := ready.live.frameAt(0)
	if e != nil {
		t.Fatal(e)
	}
	later, _, e := ready.live.frameAt(10 * time.Second)
	if e != nil {
		t.Fatal(e)
	}
	if len(later.Volumes) < 2 || len(later.Lamps) == 0 {
		t.Fatal("missing bed flames/light")
	}
	initialTiles := map[shape.Shape]maths.Light{}
	for _, item := range first.Things {
		if item.Material != nil && item.Material.Name == "surface fuel" {
			initialTiles[item.Shape] = item.Material.Colour
		}
	}
	darkened, tiles := 0, 0
	for _, item := range later.Things {
		if item.Material == nil || item.Material.Name != "surface fuel" {
			continue
		}
		tiles++
		before, ok := initialTiles[item.Shape]
		now := item.Material.Colour
		if !ok || !(now.R <= before.R && now.G <= before.G && now.B <= before.B) {
			t.Fatalf("fuel brightened or geometry changed: before %v, now %v", before, now)
		}
		if now != before {
			darkened++
		}
	}
	if len(initialTiles) != 64 || tiles != 64 || darkened == 0 {
		t.Fatalf("fuel tiles: initial %d, later %d, darkened %d", len(initialTiles), tiles, darkened)
	}

	absorption := func(world *thing.World) float64 {
		total := 0.0
		for _, v := range world.Volumes {
			if field, ok := v.Medium.(*flame.Field); ok {
				for _, p := range field.Pieces() {
					f, _, _, _ := p.Body.Samples()
					total += f.Absorption
				}
			}
		}
		return total
	}
	early, late := absorption(first), absorption(later)
	if !(early > 0 && late > 0) || early == late {
		t.Fatal("reaction did not change canonical GPU flames", early, late)
	}

}
func TestFuelBedCannotStandBeyondTheTerrain(t *testing.T) {
	given := fuelOnFixture(t)
	raw, e := os.ReadFile(given.set)
	if e != nil {
		t.Fatal(e)
	}
	raw = []byte(strings.Replace(string(raw), "east: 1050", "east: 100000", 1))
	// Change the bed specifically; leave the scene's named things alone.
	if _, e = set.Read("check.yaml", raw); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(given.set, raw, 0644); e != nil {
		t.Fatal(e)
	}
	if _, _, _, e = staged(given); e == nil {
		t.Fatal("off-ground bed accepted")
	}
}

func TestTerrainSurfaceStillConsumesFuelAndEmitsFlames(t *testing.T) {
	given := fuelOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "      fuel: 1", "      fuel: 1\n      surface: terrain", 1)
	if err = os.WriteFile(given.set, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
	_, _, ready, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	defer ready.live.closeBeds()
	initial, err := ready.live.beds[0].model.At(0)
	if err != nil {
		t.Fatal(err)
	}
	later, err := ready.live.beds[0].model.At(20)
	if err != nil {
		t.Fatal(err)
	}
	if later[27].Fuel >= initial[27].Fuel {
		t.Fatal("hiding fuel display disabled consumption")
	}
	world, _, err := ready.live.frameAt(20 * time.Second)
	if err != nil {
		t.Fatal(err)
	}
	ground := 0
	for _, item := range world.Things {
		if item.Material != nil && item.Material.Name == "surface fuel" {
			t.Fatal("diagnostic surface covers the terrain")
		}
		if item.Material != nil && item.Material.Name == "basalt" {
			ground++
		}
	}
	if ground != 1 || len(world.Volumes) < 2 || len(world.Lamps) == 0 {
		t.Fatal("terrain/fire lost", ground, len(world.Volumes), len(world.Lamps))
	}
}
