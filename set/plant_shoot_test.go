package set

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/manufacture"
)

func TestConiferShootRecipeAndRefusals(t *testing.T) {
	dir := t.TempDir()
	for family, bookFamily := range map[string]string{"plant": "seasonal", "conifer": "conifer", "fern": "fern", "cactus": "cactus"} {
		_, raw, err := manufacture.EmbeddedPlantBook(bookFamily)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, family+".yaml"), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, schema := range []string{"", "$schema: https://github.com/janearc/slayer/schemas/ecosystem-v1.json\n", "$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json\n"} {
		for _, tc := range []struct {
			name, plant, season string
			valid, shoot        bool
		}{
			{"coast shoot", "family: conifer, file: conifer.yaml, species: coast-redwood, representation: shoot", "", true, true},
			{"sequoia shoot", "family: conifer, file: conifer.yaml, species: giant-sequoia, representation: shoot", "", true, true},
			{"whole conifer", "family: conifer, file: conifer.yaml, species: coast-redwood", "", true, false},
			{"spruce has no shoot", "family: conifer, file: conifer.yaml, species: spruce, representation: shoot", "", false, false},
			{"unknown species", "family: conifer, file: conifer.yaml, species: absent, representation: shoot", "", false, false},
			{"seasonal shoot", "family: seasonal, file: plant.yaml, species: valley-oak, representation: shoot", "", false, false},
			{"default family shoot", "file: plant.yaml, species: valley-oak, representation: shoot", "", false, false},
			{"fern shoot", "family: fern, file: fern.yaml, species: fern, representation: shoot", "", false, false},
			{"cactus shoot", "family: cactus, file: cactus.yaml, species: saguaro, representation: shoot", "", false, false},
			{"season and shoot", "family: conifer, file: conifer.yaml, species: coast-redwood, representation: shoot", ", season: autumn", false, false},
			{"unknown representation", "family: conifer, file: conifer.yaml, species: coast-redwood, representation: trunk", "", false, false},
			{"empty representation", "family: conifer, file: conifer.yaml, species: coast-redwood, representation: ''", "", false, false},
			{"null representation", "family: conifer, file: conifer.yaml, species: coast-redwood, representation: null", "", false, false},
		} {
			t.Run(schema+tc.name, func(t *testing.T) {
				source := schema + fmt.Sprintf(`scene:
  name: shoot
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    specimen: {plant: {%s}%s}
  set:
    things:
      one: {recipe: specimen, seed: 1, complexity: hull, position: {east: {value: 0, unit: m}, north: {value: 0, unit: m}, up: {value: 0, unit: m}}}
`, tc.plant, tc.season)
				read, err := Read(filepath.Join(dir, "scene.yaml"), []byte(source))
				if !tc.valid {
					if err == nil {
						t.Fatal("accepted invalid model")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				recipe, ok := read.Recipe("specimen")
				if !ok || !recipe.IsPlant() {
					t.Fatal("not a checked plant recipe")
				}
				row, err := recipe.Construction()
				if err != nil {
					t.Fatal(err)
				}
				made, err := manufacture.MakeWithComplexity(*row, manufacture.Sow(1), "hull")
				if err != nil {
					t.Fatal(err)
				}
				if tc.shoot && made.Bounds().Max.Y >= 1 {
					t.Fatalf("expected loose shoot, got bounds %v", made.Bounds())
				}
				if !tc.shoot && made.Bounds().Max.Y <= 1 {
					t.Fatalf("expected whole tree, got bounds %v", made.Bounds())
				}
			})
		}
	}
}
