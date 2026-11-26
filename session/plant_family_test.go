package session

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/set"
)

func TestExternalPlantFamiliesStage(t *testing.T) {
	for _, tc := range []struct{ family, species, detail string }{
		{"conifer", "spruce", "hull"},
		{"fern", "fern", "quick"},
		{"cactus", "saguaro", "hull"},
	} {
		t.Run(tc.family, func(t *testing.T) {
			dir := t.TempDir()
			_, raw, err := manufacture.EmbeddedPlantBook(tc.family)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "plants.yaml"), raw, 0600); err != nil {
				t.Fatal(err)
			}
			source := fmt.Sprintf(`$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: family
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    tree: {plant: {family: %s, file: plants.yaml, species: %s}}
  set:
    things:
      specimen: {recipe: tree, seed: 1, complexity: %s, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`, tc.family, tc.species, tc.detail)
			read, err := set.Read(filepath.Join(dir, "scene.yaml"), []byte(source))
			if err != nil {
				t.Fatal(err)
			}
			stage := testStaging()
			stage.recipes = read
			if err := stage.afloat(read); err != nil {
				t.Fatal(err)
			}
			if len(stage.world.Things) == 0 {
				t.Fatal("family staged no space surfaces")
			}
			land := fmt.Sprintf(`scene:
  name: family
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    tree: {plant: {family: %s, file: plants.yaml, species: %s}}
  set:
    things:
      specimen: {recipe: tree, seed: 1, complexity: %s, position: {east: {value: 0, unit: m}, north: {value: 0, unit: m}, up: {value: 0, unit: m}}}
`, tc.family, tc.species, tc.detail)
			landSet, err := set.Read(filepath.Join(dir, "land.yaml"), []byte(land))
			if err != nil {
				t.Fatal(err)
			}
			landStage := testStaging()
			landStage.given = mustParse(t)
			landStage.recipes = landSet
			if err := landStage.stand(landSet); err != nil {
				t.Fatal(err)
			}
			if len(landStage.world.Things) == 0 {
				t.Fatal("family staged no land surfaces")
			}
		})
	}
}
