package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestConiferShootRecipeStagesFromFrozenBook(t *testing.T) {
	dir := t.TempDir()
	raw, err := os.ReadFile("../manufacture/plants/conifers/conifer-profiles.yaml")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "conifers.yaml")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	source := `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: shoot
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    specimen: {plant: {family: conifer, file: conifers.yaml, species: coast-redwood, representation: shoot}}
  set:
    things:
      shoot: {recipe: specimen, seed: 1, complexity: hull, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`
	read, err := set.Read(filepath.Join(dir, "scene.yaml"), []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	stage := testStaging()
	stage.recipes = read
	if err := stage.afloat(read); err != nil {
		t.Fatal(err)
	}
	if len(stage.world.Things) == 0 {
		t.Fatal("frozen shoot staged no geometry")
	}
}
