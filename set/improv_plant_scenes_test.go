package set_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestImprovSummitScenesReadConiferProfiles(t *testing.T) {
	for _, name := range []string{
		"../improv/cloudsea/cloudsea.yaml",
		"../improv/searching/studies/summit-interior.yaml",
		"../improv/searching/studies/summit-reference.yaml",
	} {
		t.Run(filepath.Base(filepath.Dir(name))+"/"+filepath.Base(name), func(t *testing.T) {
			raw, err := os.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			scene, has, err := set.ReadImported(name, raw)
			if err != nil || !has {
				t.Fatalf("profile snapshot: %v", err)
			}
			for id, species := range map[string]string{"redwoods": "coast-redwood", "sequoias": "giant-sequoia"} {
				recipe, ok := scene.Recipe(id)
				if !ok || !recipe.IsPlant() || recipe.PlantFamily() != "conifer" || recipe.PlantSpecies() != species {
					t.Fatalf("%s did not retain the %s profile", id, species)
				}
				if _, err := recipe.Construction(); err != nil {
					t.Fatalf("%s: %v", id, err)
				}
			}
			sources, err := scene.ProfileSources()
			if err != nil || len(sources) != 1 {
				t.Fatalf("shared conifer book: %v, %v", sources, err)
			}
		})
	}
}
