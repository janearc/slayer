package main

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/set"
)

func TestExplicitZeroSceneSeedOverridesTheCaller(t *testing.T) {
	raw := strings.Replace(fixtureScene, "scene:", "scene:\n  seed: 0", 1)
	read, err := set.Read("zero.yaml", []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	given := mustParse(t)
	given.seed = 17
	got, err := filmed(read, given)
	if err != nil {
		t.Fatal(err)
	}
	if got.seed != 0 {
		t.Errorf("explicit zero seed became %d", got.seed)
	}
}

func TestEcosystemStagesOneSharedFieldAndKeepsItsClearing(t *testing.T) {
	given := mustParse(t)
	given.set = onFixture(t, "", "")
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	scene := strings.ReplaceAll(string(raw), "alight: true", "alight: false")
	scene = strings.Replace(scene, "kind: spruce", "recipe: redwoods\n        complexity: hull", 1)
	scene = strings.Replace(scene, "kind: fern", "recipe: redwoods\n        complexity: clumps\n        exclude:\n          around: the stone\n          clearance: {value: 10, unit: m}\n        atmosphere:\n          weather: grove\n          extent: canopy-and-ground\n          fade-at-boundary: true", 1)
	scene = strings.Replace(scene, "  terrain:", `  seed: 17
  renderer: cpu
  recipes:
    redwoods:
      kind: coast-redwood
  weather:
    grove:
      kind: prescribed-mist
      extinction: {value: 0.006, unit: m^-1}
      depth: {value: 20, unit: m}
      coverage: 1
      seed: 23
  terrain:`, 1)
	if err := os.WriteFile(given.set, []byte(scene), 0o644); err != nil {
		t.Fatal(err)
	}
	world, made, got, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if made.things != 14 || len(world.Volumes) != 1 || got.onGPU || got.seed != 17 {
		t.Fatalf("%v, %d fields, gpu %v, seed %d", made, len(world.Volumes), got.onGPU, got.seed)
	}
	if _, ok := world.Volumes[0].Medium.(*medium.Grove); !ok {
		t.Fatal("not a shared grove")
	}
	// Check the actual placements, not only the exclusion helper.
	given.placed = given.set + ".csv"
	if _, _, _, err := staged(given); err != nil {
		t.Fatal(err)
	}
	positions, err := os.ReadFile(given.placed)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, line := range strings.Split(string(positions), "\n") {
		if !strings.HasPrefix(line, `"ferns",`) {
			continue
		}
		count++
		var east, north, height float64
		if _, err := fmt.Sscanf(strings.TrimPrefix(line, `"ferns",coast-redwood,`), "%f,%f,%f", &east, &north, &height); err != nil {
			t.Fatal(err)
		}
		if math.Hypot(east-1050, north-8990) < 10 {
			t.Error("tree inside clearing")
		}
	}
	if count != 12 {
		t.Errorf("checked %d stand placements, want 12", count)
	}
}
