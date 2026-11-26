package job_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/manufacture"
)

func TestBuiltinProfilesAreConstructedAndIncludedInJobInputs(t *testing.T) {
	source := `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Built-in profiles
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras:
    view: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    fish: {animal: {family: fish, file: "@builtin", species: pacific-bluefin-tuna}}
    tree: {plant: {family: seasonal, file: "@builtin", species: valley-oak}}
  set:
    things:
      tuna: {recipe: fish, seed: 1, east: 0, north: 0, height: {value: 0, unit: m}}
      oak: {recipe: tree, seed: 1, east: 5, north: 0, height: {value: 0, unit: m}}
`
	path := filepath.Join(t.TempDir(), "scene.yaml")
	still, err := job.StillFrom(path, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	scene, ok := still.ImportedSet()
	if !ok {
		t.Fatal("profile scene was not frozen")
	}
	for _, name := range []string{"fish", "tree"} {
		recipe, ok := scene.Recipe(name)
		if !ok {
			t.Fatalf("missing %s", name)
		}
		if _, err := recipe.Construction(); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	external, err := scene.ProfileSources()
	if err != nil || len(external) != 0 {
		t.Fatalf("unexpected external books: %v, %v", external, err)
	}
	_, fish, err := manufacture.EmbeddedAnimalBook("fish")
	if err != nil {
		t.Fatal(err)
	}
	_, plant, err := manufacture.EmbeddedPlantBook("seasonal")
	if err != nil {
		t.Fatal(err)
	}
	inputs := inputDigests(t, still.Canonical())
	for key, raw := range map[string][]byte{"profile:@animal:fish": fish, "profile:@plant:seasonal": plant} {
		digest := sha256.Sum256(raw)
		if got := inputs[key]; got != hex.EncodeToString(digest[:]) {
			t.Fatalf("%s digest = %q", key, got)
		}
		if _, exists := still.Inputs()[key]; exists {
			t.Fatalf("built-in book %s was exposed as a file", key)
		}
	}
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	fromFile, err := job.StillOf(path)
	if err != nil {
		t.Fatal(err)
	}
	fileInputs := inputDigests(t, fromFile.Canonical())
	for _, key := range []string{"profile:@animal:fish", "profile:@plant:seasonal"} {
		if fileInputs[key] != inputs[key] {
			t.Fatalf("file input %s = %v, want %v", key, fileInputs[key], inputs[key])
		}
	}
}
