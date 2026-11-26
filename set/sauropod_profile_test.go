package set

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/dinosaurs/sauropods"
)

func TestSauropodRecipeAndOptionRefusalsInBothSchemas(t *testing.T) {
	dir := t.TempDir()
	file, raw, err := manufacture.EmbeddedAnimalBook("sauropod")
	if err != nil {
		t.Fatal(err)
	}
	book, err := sauropods.Read(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	names := book.Species()
	if len(names) != 4 {
		t.Fatalf("got %d profiles", len(names))
	}
	if err := os.WriteFile(filepath.Join(dir, file), raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, schema := range []string{"", "ecosystem-v1.json", "ecosystem-v2.json"} {
		for _, tc := range []struct {
			name, species, options string
			valid                  bool
		}{
			{"default", names[0], "", true},
			{"missing", "no-such-sauropod", "", false},
			{"pose", names[0], ", pose: swim", false},
			{"count", names[0], ", count: 0", false},
			{"time", names[0], ", seconds: 0", false},
			{"phase", names[0], ", phase: 0", false},
			{"representation", names[0], ", representation: skin", false},
			{"study", names[0], ", study: study", false},
		} {
			t.Run(schema+"/"+tc.name, func(t *testing.T) {
				prefix := ""
				if schema != "" {
					prefix = "$schema: https://github.com/janearc/slayer/schemas/" + schema + "\n"
				}
				source := prefix + fmt.Sprintf(`scene:
  name: sauropod
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    specimen: {animal: {family: sauropod, file: %s, species: %s%s}}
  set:
    things:
      one: {recipe: specimen, seed: 1, complexity: quick, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`, file, tc.species, tc.options)
				if schema != "ecosystem-v2.json" {
					source = strings.Replace(source, "  setting: space\n", "", 1)
				}
				read, err := Read(filepath.Join(dir, "scene.yaml"), []byte(source))
				if (err == nil) != tc.valid {
					t.Fatalf("valid=%t: %v", tc.valid, err)
				}
				if err == nil {
					recipe, ok := read.Recipe("specimen")
					if !ok || !recipe.IsAnimal() || recipe.AnimalSpecies() != tc.species {
						t.Fatalf("wrong recipe: %#v", recipe)
					}
					if _, err := recipe.Construction(); err != nil {
						t.Fatal(err)
					}
				}
			})
		}
	}
}
