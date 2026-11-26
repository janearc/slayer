package set_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/set"
)

// The counts are placements in the original scenes, not the number of
// individual trees in each stand. Every placement now resolves a checked book.
func TestMigratedPlantScenesReadTheirProfiles(t *testing.T) {
	for name, want := range map[string]int{
		"futian-shore-gpu-nobeam.yaml": 49, "futian-shore-gpu.yaml": 49,
		"futian-shore-nobeam.yaml": 49, "futian-shore.yaml": 49,
		"golden-hour-summit.yaml": 2, "groningen.yaml": 4,
		"honopu.yaml": 6, "kilauea-iki-forest.yaml": 3,
		"kilauea-iki-moonrise.yaml": 10, "kilauea-iki-noon.yaml": 4,
		"kilauea-iki.yaml": 4, "mangrove-shore.yaml": 13,
		"mangrove-study.yaml": 12, "melancholia.yaml": 11,
		"moonlake.yaml": 23, "moonlit-summit.yaml": 2,
		"native-flora-study.yaml": 7, "portofino.yaml": 5,
		"rarotonga.yaml": 4,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "sets", name)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			read, err := set.Read(path, raw)
			if err != nil {
				t.Fatal(err)
			}
			count := 0
			check := func(kind, recipeName string) {
				recipe, found := read.Recipe(recipeName)
				if found && recipe.IsPlant() {
					count++
					if kind != "plant-profile" || recipe.PlantSpecies() == "" {
						t.Fatalf("invalid plant recipe %q", recipeName)
					}
				}
			}
			for _, one := range read.Things() {
				check(one.Kind(), one.Recipe())
			}
			for _, one := range read.Stands() {
				check(one.Kind(), one.Recipe())
			}
			if count != want {
				t.Fatalf("%d plant placements, want %d", count, want)
			}
		})
	}
}

func TestMigratedMangroveStandKeepsItsPlacement(t *testing.T) {
	path := filepath.Join("..", "sets", "mangrove-shore.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read(path, raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range read.Stands() {
		if one.Name() != "kandelia a" {
			continue
		}
		recipe, found := read.Recipe(one.Recipe())
		count, seed := one.Count()
		above, below := one.Band()
		of, from, to, near, far, within := one.Within()
		if !found || recipe.PlantSpecies() != "kandelia-obovata" || count != 40 || seed != 101 || one.Complexity() != "clumps" || above != 0.1 || below != 1 || !within || of != "the marina" || from != 270 || to != 90 || near != 0 || far != 600 {
			t.Fatalf("mangrove placement changed: %+v", one)
		}
		return
	}
	t.Fatal("kandelia a is missing")
}
