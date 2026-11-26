package set

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlantRecipeReadsExternalSpeciesAndHashesBytes(t *testing.T) {
	dir := t.TempDir()
	raw, err := os.ReadFile("../manufacture/plants/seasonal/profiles.yaml")
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), "kind: valley-oak", "kind: meadow-oak", 1))
	asset := filepath.Join(dir, "plants.yaml")
	if err := os.WriteFile(asset, raw, 0600); err != nil {
		t.Fatal(err)
	}
	source := `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: meadow
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    oak: {plant: {file: plants.yaml, species: meadow-oak}, season: autumn}
  set:
    things:
      tree: {recipe: oak, seed: 1, complexity: hull, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`
	load := func() Recipe {
		t.Helper()
		read, err := Read(filepath.Join(dir, "scene.yaml"), []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		row, ok := read.Recipe("oak")
		if !ok {
			t.Fatal("missing recipe")
		}
		if row.Kind() != "plant-profile" || row.PlantSpecies() != "meadow-oak" {
			t.Fatalf("kind %q", row.Kind())
		}
		if _, err := row.Construction(); err != nil {
			t.Fatal(err)
		}
		return row
	}
	before := load()
	if err := os.WriteFile(asset, append(raw, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if after := load(); before.Digest() == after.Digest() {
		t.Fatal("changed profile reused recipe digest")
	}
	if err := os.Remove(asset); err != nil {
		t.Fatal(err)
	}
	if _, err := before.Construction(); err != nil {
		t.Fatalf("checked recipe lost its frozen profile after source deletion: %v", err)
	}
	if _, err := Read(filepath.Join(dir, "scene.yaml"), []byte(source)); err == nil {
		t.Fatal("missing profile accepted")
	}
}
