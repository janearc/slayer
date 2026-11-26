package set

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture"
)

func TestPlantRecipeReadsExplicitFamiliesAndRefusesWrongBooks(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte("profiles: ["), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, family, file, species, season string
		valid                               bool
	}{
		{"default seasonal", "", "plant.yaml", "valley-oak", "autumn", true},
		{"explicit seasonal", "seasonal", "plant.yaml", "valley-oak", "", true},
		{"conifer", "conifer", "conifer.yaml", "spruce", "", true},
		{"fern", "fern", "fern.yaml", "fern", "", true},
		{"cactus", "cactus", "cactus.yaml", "saguaro", "", true},
		{"wrong family", "fern", "conifer.yaml", "spruce", "", false},
		{"unknown family", "flower", "plant.yaml", "valley-oak", "", false},
		{"missing file", "fern", "missing.yaml", "fern", "", false},
		{"malformed book", "fern", "bad.yaml", "fern", "", false},
		{"missing species", "fern", "fern.yaml", "no-such-fern", "", false},
		{"unsupported season", "conifer", "conifer.yaml", "spruce", "autumn", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			family, season := "", ""
			if tc.family != "" {
				family = ", family: " + tc.family
			}
			if tc.season != "" {
				season = ", season: " + tc.season
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
    a: {plant: {file: %s, species: %s%s}%s}
  set:
    things:
      specimen: {recipe: a, seed: 1, complexity: hull, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`, tc.file, tc.species, family, season)
			read, err := Read(filepath.Join(dir, "scene.yaml"), []byte(source))
			if !tc.valid {
				if err == nil {
					t.Fatal("accepted invalid plant recipe")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			recipe, ok := read.Recipe("a")
			if !ok || !recipe.IsPlant() || recipe.Kind() != "plant-profile" || recipe.PlantSpecies() != tc.species {
				t.Fatalf("incorrect checked plant recipe: %+v, found %t", recipe, ok)
			}
			expectedFamily := tc.family
			if expectedFamily == "" {
				expectedFamily = "seasonal"
			}
			if recipe.PlantFamily() != expectedFamily {
				t.Fatalf("family = %q, want %q", recipe.PlantFamily(), expectedFamily)
			}
			if _, err := recipe.Construction(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPlantRecipeRefusesMixedConstructionKinds(t *testing.T) {
	dir := t.TempDir()
	raw, err := os.ReadFile(filepath.Join("..", "manufacture", "plants", "ferns", "fern-profiles.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "f.yaml"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, block, field string }{
		{"kind", "plant: {family: fern, file: f.yaml, species: fern}, kind: tower", "kind"},
		{"composition", "plant: {family: fern, file: f.yaml, species: fern}, composition: {file: c.yaml}", "composition"},
		{"unknown plant field", "plant: {family: fern, file: f.yaml, species: fern, extra: value}", "extra"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := fmt.Sprintf(`$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: family
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    a: {%s}
  set:
    things:
      specimen: {recipe: a, seed: 1, complexity: quick, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`, tc.block)
			if _, err := Read(filepath.Join(dir, "scene.yaml"), []byte(source)); err == nil || !strings.Contains(err.Error(), tc.field) {
				t.Fatalf("invalid recipe field %q: %v", tc.field, err)
			}
		})
	}
}
