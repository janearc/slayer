package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/set"
)

func TestBundledExampleUsesMatchingEmbeddedProfile(t *testing.T) {
	root := t.TempDir()
	library := filepath.Join(root, "library")
	books := filepath.Join(root, "manufacture", "plants", "seasonal")
	for _, dir := range []string{library, books} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	_, raw, err := manufacture.EmbeddedPlantBook("seasonal")
	if err != nil {
		t.Fatal(err)
	}
	book := filepath.Join(books, "profiles.yaml")
	if err := os.WriteFile(book, raw, 0600); err != nil {
		t.Fatal(err)
	}
	source := `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Profile example
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras:
    view: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    tree: {plant: {family: seasonal, file: ../manufacture/plants/seasonal/profiles.yaml, species: valley-oak}}
  set:
    things:
      oak: {recipe: tree, seed: 1, east: 0, north: 0, height: {value: 0, unit: m}}
`
	converted, err := builtinExampleProfiles(root, filepath.Join(library, "scene.yaml"), []byte(source))
	if err != nil || !strings.Contains(string(converted), set.BuiltinProfileBook) {
		t.Fatalf("profile was not converted: %v", err)
	}
	if _, err := set.Read(filepath.Join(t.TempDir(), "scene.yaml"), converted); err != nil {
		t.Fatalf("example could not load without the original book: %v", err)
	}
	if err := os.WriteFile(book, append(raw, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	untouched, err := builtinExampleProfiles(root, filepath.Join(library, "scene.yaml"), []byte(source))
	if err != nil || string(untouched) != source {
		t.Fatal("a modified custom book was replaced")
	}
}
