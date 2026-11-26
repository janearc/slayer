package session

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/set"
)

func TestExternalAnimalFamiliesStageOnLandAndSpace(t *testing.T) {
	for _, tc := range []struct{ family, file, species, extra string }{
		{"fish", "fish.yaml", "pacific-bluefin-tuna", "pose: glide"},
		{"bird", "birds.yaml", "grey-heron", "pose: standing"},
		{"theropod", "allosaurus-profiles.yaml", "allosaurus", "seconds: 0.5"},
		{"pinniped", "pinniped_profiles.yaml", "harbor-seal", "pose: hauled"},
		{"pinniped", "pinniped_profiles.yaml", "harbor-seal", "representation: bones"},
		{"pinniped", "pinniped_profiles.yaml", "harbor-seal", "study: detailed-wet"},
	} {
		t.Run(tc.family+"/"+tc.extra, func(t *testing.T) {
			dir := t.TempDir()
			_, raw, err := manufacture.EmbeddedAnimalBook(tc.family)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, tc.file), raw, 0600); err != nil {
				t.Fatal(err)
			}
			source := fmt.Sprintf(`$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: animal
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    specimen: {animal: {family: %s, file: %s, species: %s, %s}}
  set:
    things:
      one: {recipe: specimen, seed: 1, complexity: quick, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`, tc.family, tc.file, tc.species, tc.extra)
			read, err := set.Read(filepath.Join(dir, "scene.yaml"), []byte(source))
			if err != nil {
				t.Fatal(err)
			}
			if read.Things()[0].Kind() != "animal-profile" {
				t.Fatal("species became behavior category")
			}
			stage := testStaging()
			stage.recipes = read
			if err := stage.afloat(read); err != nil {
				t.Fatal(err)
			}
			if len(stage.world.Things) == 0 {
				t.Fatal("animal staged no surfaces")
			}
			if tc.family == "fish" {
				return // Fish land staging is covered by the reserved-name case below.
			}
			land := fmt.Sprintf(`scene:
  name: animal
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    specimen: {animal: {family: %s, file: %s, species: %s, %s}}
  set:
    things:
      one: {recipe: specimen, seed: 1, complexity: quick, position: {east: {value: 0, unit: m}, north: {value: 0, unit: m}, up: {value: 0, unit: m}}}
`, tc.family, tc.file, tc.species, tc.extra)
			landSet, err := set.Read(filepath.Join(dir, "land.yaml"), []byte(land))
			if err != nil {
				t.Fatal(err)
			}
			if landSet.Things()[0].Kind() != "animal-profile" {
				t.Fatal("land species became behavior category")
			}
			landStage := testStaging()
			landStage.given = mustParse(t)
			landStage.recipes = landSet
			if err := landStage.stand(landSet); err != nil {
				t.Fatal(err)
			}
			if len(landStage.world.Things) == 0 {
				t.Fatal("animal staged no land surfaces")
			}
		})
	}
}

func TestExternalAnimalSpeciesNamedFireStagesOnLand(t *testing.T) {
	dir := t.TempDir()
	raw, err := os.ReadFile("../manufacture/animals/vertebrates/fishes/fish.yaml")
	if err != nil {
		t.Fatal(err)
	}
	raw = bytes.Replace(raw, []byte("  pacific-bluefin-tuna:"), []byte("  fire:"), 1)
	if err := os.WriteFile(filepath.Join(dir, "fish.yaml"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	source := `scene:
  name: fish
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far:
      location: {at: [0, -20, 2]}
      orientation: {look_at: [0, 0, 2]}
  recipes:
    specimen: {animal: {family: fish, file: fish.yaml, species: fire}}
  set:
    things:
      one:
        recipe: specimen
        seed: 1
        complexity: quick
        position:
          east: {value: 0, unit: m}
          north: {value: 0, unit: m}
          up: {value: 0, unit: m}
`
	read, err := set.Read(filepath.Join(dir, "scene.yaml"), []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if read.Things()[0].Kind() != "animal-profile" {
		t.Fatal("species name became behavior class")
	}
	stage := testStaging()
	stage.given = mustParse(t)
	stage.recipes = read
	if err := stage.stand(read); err != nil {
		t.Fatal(err)
	}
	if len(stage.world.Things) == 0 {
		t.Fatal("animal staged no surfaces")
	}
}
