package set_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/set"
)

func TestReptileRecipesReadStudiesAndRejectUnsupportedOptions(t *testing.T) {
	_, raw := manufacture.EmbeddedReptileBook()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "reptiles.yaml"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "scene.yaml")
	for _, schema := range []string{"", "ecosystem-v1.json", "ecosystem-v2.json"} {
		for _, tc := range []struct {
			species, study string
			valid          bool
		}{
			{"pliosaur", ", study: baseline", true}, {"pliosaur", ", study: continuous", true},
			{"purussaurus", ", study: study", true}, {"hatzegopteryx", ", study: cage", true},
			{"hatzegopteryx", ", study: density", true}, {"hatzegopteryx", ", study: composite", true},
			{"hatzegopteryx", ", study: continuous", false}, {"unknown", "", false},
			{"pliosaur", ", pose: swim", false}, {"pliosaur", ", phase: 0", false},
			{"pliosaur", ", count: 0", false}, {"pliosaur", ", seconds: 0", false},
			{"pliosaur", ", representation: bones", false},
		} {
			source := squidSet(schema, tc.study)
			source = strings.Replace(source, "family: squid, file: squid.yaml, species: humboldt-squid", "family: reptile, file: reptiles.yaml, species: "+tc.species, 1)
			read, err := set.Read(path, []byte(source))
			if (err == nil) != tc.valid {
				t.Errorf("%s %s%s: %v", schema, tc.species, tc.study, err)
				continue
			}
			if err == nil {
				recipe, ok := read.Recipe("squid")
				if !ok || !recipe.IsAnimal() || recipe.AnimalSpecies() != tc.species {
					t.Errorf("lost %s identity", tc.species)
				}
			}
		}
	}
}
