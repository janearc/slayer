package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestTreeProfileRecipesStageAfterSourceRemoval(t *testing.T) {
	raw, err := os.ReadFile("../manufacture/plants/seasonal/profiles.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"tower", "city-clustered-green-10000"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "trees.yaml")
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			pins := ""
			if kind == "tower" {
				pins = ", pins: {skin: concrete, planted: true, storeys: 30}"
			}
			source := `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: tree models
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras: {view: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}}
  recipes:
    tree scene: {kind: ` + kind + `, tree-profiles: {file: trees.yaml}` + pins + `}
  set:
    things:
      object: {recipe: tree scene, seed: 3, complexity: quick, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`
			read, has, err := set.ReadImported(filepath.Join(dir, "scene.yaml"), []byte(source))
			if err != nil || !has {
				t.Fatalf("set snapshot: %v", err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			stage := testStaging()
			stage.recipes = read
			if err := stage.afloat(read); err != nil {
				t.Fatalf("staging frozen tree recipe: %v", err)
			}
			if len(stage.world.Things) == 0 {
				t.Fatal("tree recipe staged no geometry")
			}
		})
	}
}
