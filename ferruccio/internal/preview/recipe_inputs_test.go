package preview

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestDeclaredInputsIncludeSharedRecipeBooks(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source", "scene.yaml")
	input := filepath.Join(root, "frozen.yaml")
	absolute := filepath.Join(root, "animals.yaml")
	raw := "scene:\n  recipes:\n    first: &plant {plant: {file: ../plants.yaml}}\n    second: {plant: {file: ../plants.yaml}}\n    third: {animal: {file: " + absolute + "}}\n    fourth: *plant\n"
	if err := os.WriteFile(input, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	plan := Plan{Input: input, Source: source, Directory: root, Kind: "set", relocated: true}
	inputs, err := plan.declaredInputs()
	if err != nil {
		t.Fatal(err)
	}
	plant := filepath.Join(root, "plants.yaml")
	if !slices.Contains(inputs, plant) || !slices.Contains(inputs, absolute) || len(inputs) != 4 {
		t.Fatalf("declared inputs = %v", inputs)
	}
}

func TestJobDeclaresBooksRelativeToReferencedSet(t *testing.T) {
	root := t.TempDir()
	setDir := filepath.Join(root, "sets")
	if err := os.Mkdir(setDir, 0700); err != nil {
		t.Fatal(err)
	}
	setPath := filepath.Join(setDir, "city.yaml")
	if err := os.WriteFile(setPath, []byte("scene:\n  recipes:\n    tree: {plant: {file: profiles.yaml}}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(root, "job.yaml")
	if err := os.WriteFile(input, []byte("scene: {set: sets/city.yaml}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	plan := Plan{Input: input, Directory: root, Kind: "job"}
	inputs, err := plan.declaredInputs()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(inputs, setPath) || !slices.Contains(inputs, filepath.Join(setDir, "profiles.yaml")) {
		t.Fatalf("job's set and its book were not declared: %v", inputs)
	}
}

func TestDeclaredInputsUsesExecutableForEmbeddedRecipeBooks(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "frozen.yaml")
	const raw = `scene:
  recipes:
    tuna:
      animal: {family: fish, file: &builtin '@builtin', species: pacific-bluefin-tuna}
    fern:
      plant: {family: fern, file: *builtin, species: fern}
    oak:
      plant: {file: books/@builtin}
    tree:
      plant: {file: custom-plants.yaml}
`
	if err := os.WriteFile(input, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	plan := Plan{Input: input, Directory: root, Kind: "set"}
	inputs, err := plan.declaredInputs()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{input, filepath.Join(root, "scene.yaml"), filepath.Join(root, "books", "@builtin"), filepath.Join(root, "custom-plants.yaml")}
	if !slices.Equal(inputs, want) {
		t.Fatalf("declared inputs = %v, want %v", inputs, want)
	}
}

func TestJobOmitsEmbeddedBooksFromReferencedSet(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "frozen.yaml")
	setPath := filepath.Join(root, "subject.yaml")
	source := "scene:\n  recipes:\n    fern: {plant: &book {file: '@builtin'}}\n    alga: {alga: *book}\n"
	if err := os.WriteFile(setPath, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte("scene: {set: subject.yaml}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	plan := Plan{Input: input, Directory: root, Kind: "job"}
	inputs, err := plan.declaredInputs()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{input, filepath.Join(root, "scene.yaml"), setPath, filepath.Join(root, "@builtin")}
	if !slices.Equal(inputs, want) {
		t.Fatalf("job's set and external book inputs = %v, want %v", inputs, want)
	}
}
