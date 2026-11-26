package set

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func algaScene(file, species, condition string, forest bool) []byte {
	return []byte(fmt.Sprintf(`$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: kelp
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    kelp: {alga: {family: kelp, file: %s, species: %s, condition: %s, forest: %t}}
  set:
    things:
      specimen: {recipe: kelp, seed: 1, complexity: quick, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`, file, species, condition, forest))
}

func TestAlgaRecipeFreezesSourceAndRefusesUnknownVariants(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "kelp.yaml")
	raw, err := os.ReadFile("../manufacture/algae/kelp/kelp.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, raw, 0600); err != nil {
		t.Fatal(err)
	}
	scene := algaScene("kelp.yaml", "macrocystis-pyrifera", "moderate", false)
	read, imported, err := ReadImported(filepath.Join(dir, "scene.yaml"), scene)
	if err != nil || !imported {
		t.Fatalf("alga source not read: imported=%t err=%v", imported, err)
	}
	recipe, ok := read.Recipe("kelp")
	if !ok || !recipe.IsAlga() || recipe.AlgaSpecies() != "macrocystis-pyrifera" || recipe.Kind() != "alga-profile" {
		t.Fatalf("wrong alga recipe: %+v", recipe)
	}
	before := recipe.Digest()
	if _, err := recipe.Construction(); err != nil {
		t.Fatal(err)
	}
	sources, err := read.ProfileSources()
	if err != nil || len(sources) != 1 {
		t.Fatalf("alga source closure: %v, %v", sources, err)
	}
	changed := strings.Replace(string(raw), "depth: {least: 12, most: 12", "depth: {least: 8, most: 8", 1)
	if changed == string(raw) {
		t.Fatal("depth source not found")
	}
	if err := os.WriteFile(file, []byte(changed), 0600); err != nil {
		t.Fatal(err)
	}
	again, err := Read(filepath.Join(dir, "scene.yaml"), scene)
	if err != nil {
		t.Fatal(err)
	}
	updated, _ := again.Recipe("kelp")
	if before == updated.Digest() {
		t.Fatal("changed model did not change recipe identity")
	}
	if recipe.Digest() != before {
		t.Fatal("rereading source changed the original recipe")
	}
	for _, invalid := range [][]byte{
		algaScene("kelp.yaml", "unknown", "moderate", false),
		algaScene("kelp.yaml", "macrocystis-pyrifera", "unknown", false),
	} {
		if _, err := Read(filepath.Join(dir, "scene.yaml"), invalid); err == nil {
			t.Fatal("accepted unknown alga variant")
		}
	}
}

func TestAuthoredKelpScenesUseAlgaRecipes(t *testing.T) {
	for _, name := range []string{"kelp-forest.yaml", "kelp-stand.yaml", "kelp-plant.yaml", "kelp-plant-water.yaml"} {
		t.Run(name, func(t *testing.T) {
			file := filepath.Join("..", "sets", name)
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			read, err := Read(file, raw)
			if err != nil {
				t.Fatal(err)
			}
			row, ok := read.Recipe("profile-kelp")
			if !ok || !row.IsAlga() || row.AlgaSpecies() != "macrocystis-pyrifera" {
				t.Fatalf("missing checked kelp recipe: %+v", row)
			}
			if _, err := row.Construction(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
