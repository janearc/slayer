package session

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestExternalPlantRecipeStagesWithoutRegistry(t *testing.T) {
	dir := t.TempDir()
	raw, err := os.ReadFile("../manufacture/plants/seasonal/profiles.yaml")
	if err != nil {
		t.Fatal(err)
	}
	raw = bytes.Replace(raw, []byte("kind: valley-oak"), []byte("kind: meadow-oak"), 1)
	if err := os.WriteFile(filepath.Join(dir, "plants.yaml"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	source := `scene:
  name: meadow
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far:
      location: {at: [0, -20, 2]}
      orientation: {look_at: [0, 0, 2]}
  recipes:
    oak: {plant: {file: plants.yaml, species: meadow-oak}}
  set:
    things:
      tree:
        recipe: oak
        seed: 1
        complexity: hull
        position:
          east: {value: 0, unit: m}
          north: {value: 0, unit: m}
          up: {value: 0, unit: m}
`
	read, err := set.Read(filepath.Join(dir, "scene.yaml"), []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	stage := testStaging()
	stage.given = mustParse(t)
	stage.recipes = read
	if err := stage.stand(read); err != nil {
		t.Fatal(err)
	}
	if len(stage.world.Things) == 0 {
		t.Fatal("external plant staged no surfaces")
	}
}

func TestExternalPlantRecipeStagesInSpace(t *testing.T) {
	dir := t.TempDir()
	raw, err := os.ReadFile("../manufacture/plants/seasonal/profiles.yaml")
	if err != nil {
		t.Fatal(err)
	}
	raw = bytes.Replace(raw, []byte("kind: valley-oak"), []byte("kind: fire"), 1)
	if err := os.WriteFile(filepath.Join(dir, "plants.yaml"), raw, 0600); err != nil {
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
    oak: {plant: {file: plants.yaml, species: fire}}
  set:
    things:
      tree: {recipe: oak, seed: 1, complexity: hull, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`
	read, err := set.Read(filepath.Join(dir, "scene.yaml"), []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if read.Things()[0].Kind() != "plant-profile" {
		t.Fatal("species name was used as the behavior category")
	}
	if recipe, ok := read.Recipe("oak"); !ok || recipe.PlantSpecies() != "fire" {
		t.Fatal("species identity was lost")
	}
	stage := testStaging()
	stage.recipes = read
	if err := stage.afloat(read); err != nil {
		t.Fatal(err)
	}
	if len(stage.world.Things) == 0 {
		t.Fatal("external plant staged no surfaces")
	}
}

// Profile identity belongs in placement reports without changing the kind
// used to build a stand or the identity of unrelated recipes.
func TestPlantStandReportsSpecies(t *testing.T) {
	dir := t.TempDir()
	raw, err := os.ReadFile("../manufacture/plants/seasonal/profiles.yaml")
	if err != nil {
		t.Fatal(err)
	}
	raw = bytes.Replace(raw, []byte("kind: valley-oak"), []byte("kind: meadow-oak"), 1)
	if err := os.WriteFile(filepath.Join(dir, "plants.yaml"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	source := strings.Replace(standSetText(""), "  set:\n", `  recipes:
    oak: {plant: {file: plants.yaml, species: meadow-oak}}
    dwelling:
      composition:
        inline:
          version: 1
          name: post
          parts:
            - {id: foot, type: leg, at: [0, 0, 0], size: [2, 3, 2], ground: true}
  set:
`, 1)
	source = strings.Replace(source, "        kind: yurt\n        count: 20", "        recipe: oak\n        count: 20", 1)
	source = strings.Replace(source, "        complexity: quick\n        within:", "        complexity: hull\n        within:", 1)
	read, err := set.Read(filepath.Join(dir, "scene.yaml"), []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	stage := testStaging()
	stage.recipes = read
	if err := stage.afloat(read); err != nil {
		t.Fatal(err)
	}
	if got := read.Stands()[0].Kind(); got != "plant-profile" {
		t.Fatalf("construction kind changed to %q", got)
	}
	if got := stage.reportedKind("composition", "dwelling"); got != "composition" {
		t.Fatalf("unrelated recipe reported as %q", got)
	}
	if len(stage.stood) != 21 || stage.stood[0].kind != "yurt" {
		t.Fatalf("unrelated recipe reporting changed: %+v", stage.stood)
	}
	for _, one := range stage.stood[1:] {
		if one.kind != "meadow-oak" {
			t.Fatalf("stand member reported as %q, want meadow-oak", one.kind)
		}
	}
	path := filepath.Join(dir, "places.csv")
	if err := stage.written(path); err != nil {
		t.Fatal(err)
	}
	csv, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(csv), `"the district",meadow-oak,`); got != 20 {
		t.Fatalf("placement report has %d species-labelled rows, want 20: %s", got, csv)
	}
}
