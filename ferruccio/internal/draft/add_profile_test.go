package draft

import (
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestAddProfileThingAddsRecipeAndPlacementAtomically(t *testing.T) {
	document := poseDraft(t, addRecipeSource)
	spec := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "family"}, {Kind: yaml.ScalarNode, Tag: "!!str", Value: "fish"},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "file"}, {Kind: yaml.ScalarNode, Tag: "!!str", Value: "old.yaml"},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "species"}, {Kind: yaml.ScalarNode, Tag: "!!str", Value: "pacific-bluefin-tuna"},
	}}
	book := filepath.Join(t.TempDir(), "fish.yaml")
	if err := document.AddProfileThing("tuna", "tuna-recipe", "animal", spec, book, 2, [3]float64{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	if got, _ := document.Value("scene", "recipes", "tuna-recipe", "animal", "file"); got != book {
		t.Fatalf("book = %q", got)
	}
	objects, err := document.Objects()
	if err != nil || len(objects) != 2 || objects[1].Recipe() != "tuna-recipe" || objects[1].Position() != [3]float64{1, 2, 3} {
		t.Fatalf("profile placement = %+v, %v", objects, err)
	}
	if spec.Content[3].Value != "old.yaml" || !strings.Contains(document.Text(), "Keep this note.") {
		t.Fatal("transaction mutated the supplied spec or lost a comment")
	}
	before := document.Text()
	if err := document.AddProfileThing("second", "tuna-recipe", "animal", spec, book, 2, [3]float64{}); err == nil || document.Text() != before {
		t.Fatal("recipe collision changed source")
	}
}

func TestAddPlantProfileMovesCatalogueSeasonToRecipe(t *testing.T) {
	document := poseDraft(t, addRecipeSource)
	spec := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "family"}, {Kind: yaml.ScalarNode, Tag: "!!str", Value: "seasonal"},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "file"}, {Kind: yaml.ScalarNode, Tag: "!!str", Value: "old.yaml"},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "species"}, {Kind: yaml.ScalarNode, Tag: "!!str", Value: "valley-oak"},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "season"}, {Kind: yaml.ScalarNode, Tag: "!!str", Value: "autumn"},
	}}
	if err := document.AddProfileThing("oak", "oak-recipe", "plant", spec, filepath.Join(t.TempDir(), "plants.yaml"), 2, [3]float64{}); err != nil {
		t.Fatal(err)
	}
	if got, _ := document.Value("scene", "recipes", "oak-recipe", "season"); got != "autumn" || len(spec.Content) != 8 {
		t.Fatalf("season = %q or original catalogue entry changed", got)
	}
}

func TestRebaseProfileBookKeepsTheSameFile(t *testing.T) {
	document := poseDraft(t, addRecipeSource)
	spec := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "family"}, {Kind: yaml.ScalarNode, Tag: "!!str", Value: "fish"},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "file"}, {Kind: yaml.ScalarNode, Tag: "!!str", Value: "old.yaml"},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "species"}, {Kind: yaml.ScalarNode, Tag: "!!str", Value: "pacific-bluefin-tuna"},
	}}
	if err := document.AddProfileThing("tuna", "tuna-recipe", "animal", spec, "books/fish.yaml", 2, [3]float64{}); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	from, to := filepath.Join(root, "first", "scene.yaml"), filepath.Join(root, "second", "scene.yaml")
	if err := document.RebaseProfileBooks(from, to); err != nil {
		t.Fatal(err)
	}
	got, err := document.Value("scene", "recipes", "tuna-recipe", "animal", "file")
	if err != nil || filepath.Join(filepath.Dir(to), got) != filepath.Join(filepath.Dir(from), "books/fish.yaml") {
		t.Fatalf("rebased path %q: %v", got, err)
	}
}
