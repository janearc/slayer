package session

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/set"
)

func TestMigratedAuthoredPlantScenesReadProfiles(t *testing.T) {
	for _, name := range []string{
		"improv/fire-support/tree-fire.yaml", "improv/fire-support/burning-spruce.yaml",
		"improv/moonlake/rig-heights.yaml", "doc/guide-star-deck.yaml",
		"doc/guide-star-mist.yaml", "doc/observatory-ecosystem.yaml",
		"doc/fire-woodland.yaml", "doc/fire-night.yaml",
		"set/testdata/kilauea-iki.yaml", "set/testdata/laser.yaml", "set/testdata/observatory-4k.yaml",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", name)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			read, err := set.Read(path, raw)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, one := range read.Things() {
				if row, ok := read.Recipe(one.Recipe()); ok && row.IsPlant() {
					found = true
				}
			}
			for _, one := range read.Stands() {
				if row, ok := read.Recipe(one.Recipe()); ok && row.IsPlant() {
					found = true
				}
			}
			if !found {
				t.Fatal("scene has no checked plant placement")
			}
		})
	}
}

func TestMigratedSpruceStillBurns(t *testing.T) {
	path := "../improv/fire-support/burning-spruce.yaml"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Absolute position avoids needing the study's external terrain raster;
	// the authored recipe, seed and alight setting remain unchanged.
	old := []byte("        east: 263812\n        north: 2148748\n")
	position := []byte("        position: {east: {value: 0, unit: m}, north: {value: 0, unit: m}, up: {value: 0, unit: m}}\n")
	if bytes.Count(raw, old) != 1 {
		t.Fatal("tree position fixture changed")
	}
	raw = bytes.Replace(raw, old, position, 1)
	read, err := set.Read(path, raw)
	if err != nil {
		t.Fatal(err)
	}
	tree := read.Things()[0]
	row, found := read.Recipe(tree.Recipe())
	if !found || row.PlantSpecies() != "spruce" || tree.Seed() != 3 || !tree.Alight() {
		t.Fatal("burning specimen lost its species, seed or ignition")
	}
	stage := testStaging()
	stage.given = usualAsked()
	stage.recipes = read
	if err := stage.stand(read); err != nil {
		t.Fatal(err)
	}
	if stage.made.burning != 1 {
		t.Fatalf("burning tally %d", stage.made.burning)
	}
	for _, volume := range stage.world.Volumes {
		if _, ok := volume.Medium.(*flame.Body); ok {
			return
		}
	}
	t.Fatal("alight spruce made no flame")
}
