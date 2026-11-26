package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/set"
)

func TestSquidProfilesStageOnLandAndInSpace(t *testing.T) {
	_, raw, err := manufacture.EmbeddedAnimalBook("squid")
	if err != nil {
		t.Fatal(err)
	}
	for _, space := range []bool{false, true} {
		dir := t.TempDir()
		asset := filepath.Join(dir, "squid.yaml")
		if err := os.WriteFile(asset, raw, 0600); err != nil {
			t.Fatal(err)
		}
		setting := ""
		if space {
			setting = "  setting: space\n"
		}
		source := `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: squid
` + setting + `  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras: {view: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}}
  recipes:
    squid: {animal: {family: squid, file: squid.yaml, species: humboldt-squid, phase: 1.1}}
  set:
    things:
      squid: {recipe: squid, seed: 3, complexity: quick, position: {east: {value: 0, unit: m}, north: {value: 0, unit: m}, up: {value: 0, unit: m}}}
  world: {unit: 1}
`
		read, has, err := set.ReadImported(filepath.Join(dir, "scene.yaml"), []byte(source))
		if err != nil || !has {
			t.Fatalf("space %v: snapshot: %v", space, err)
		}
		if err := os.Remove(asset); err != nil {
			t.Fatal(err)
		}
		stage := testStaging()
		stage.recipes = read
		if space {
			err = stage.afloat(read)
		} else {
			err = stage.stand(read)
		}
		if err != nil {
			t.Fatalf("space %v: stage: %v", space, err)
		}
		if len(stage.world.Things) != 292 {
			t.Fatalf("space %v: staged %d things", space, len(stage.world.Things))
		}
	}
}
