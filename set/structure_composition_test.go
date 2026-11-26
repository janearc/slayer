package set

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	yaml "gopkg.in/yaml.v3"
)

const structureFixture = `version: 1
name: footing
parts:
  - id: footing
    type: leg
    at: [0, 0, 0]
    size: [2, 3, 2]
    ground: true
`

func TestStructureCompositionFromFileAndInline(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "parts.yaml"), []byte(structureFixture), 0600); err != nil {
		t.Fatal(err)
	}
	fileNode := structureNode(t, "file: parts.yaml\n")
	inlineNode := structureNode(t, "inline:\n"+indentStructure(structureFixture))
	file, digest, err := readStructureComposition(filepath.Join(dir, "scene.yaml"), fileNode)
	if err != nil {
		t.Fatal(err)
	}
	inline, _, err := readStructureComposition(filepath.Join(dir, "scene.yaml"), inlineNode)
	if err != nil {
		t.Fatal(err)
	}
	if file.Name != "footing" || inline.Name != file.Name || len(file.Parts) != 1 {
		t.Fatalf("file=%+v inline=%+v", file, inline)
	}
	changed := strings.Replace(structureFixture, "footing\nparts:", "foundation\nparts:", 1)
	if err := os.WriteFile(filepath.Join(dir, "parts.yaml"), []byte(changed), 0600); err != nil {
		t.Fatal(err)
	}
	_, next, err := readStructureComposition(filepath.Join(dir, "scene.yaml"), fileNode)
	if err != nil || digest == next {
		t.Fatalf("changed source digest unchanged: %x %x, %v", digest, next, err)
	}
}

func TestStructureCompositionRefusesMalformedInput(t *testing.T) {
	for _, value := range []string{
		"{}", "file: absent.yaml\ninline: {}", "file: absent.yaml\n", "other: true\n", "file: []\n",
		"inline:\n" + indentStructure(structureFixture+"bogus: 1\n"),
		"inline:\n" + indentStructure(strings.Replace(structureFixture, "version: 1", "version: 1\nversion: 1", 1)),
		"inline:\n" + indentStructure(strings.Replace(structureFixture, "size: [2, 3, 2]", "size: [2, -3, 2]", 1)),
	} {
		t.Run(value, func(t *testing.T) {
			if _, _, err := readStructureComposition(filepath.Join(t.TempDir(), "set.yaml"), structureNode(t, value)); err == nil {
				t.Fatal("accepted invalid composition")
			}
		})
	}
}

func structureNode(t *testing.T, source string) *yaml.Node {
	t.Helper()
	var n yaml.Node
	if err := yaml.Unmarshal([]byte(source), &n); err != nil {
		t.Fatal(err)
	}
	return n.Content[0]
}
func indentStructure(s string) string {
	return "  " + strings.ReplaceAll(strings.TrimSuffix(s, "\n"), "\n", "\n  ") + "\n"
}

func TestStructureRecipeReadsFromSetAndTracksFile(t *testing.T) {
	dir := t.TempDir()
	asset := filepath.Join(dir, "structure.yaml")
	if err := os.WriteFile(asset, []byte(structureFixture), 0600); err != nil {
		t.Fatal(err)
	}
	scene := `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: structure
  setting: space
  light: none
  film: {width: 64, height: 48, rays: 1, wide: 40, exposure: 1}
  cameras:
    far:
      location: {at: [0, -300, 2]}
      orientation: {look_at: [0, 0, 60]}
  recipes:
    landmark:
      composition: {file: structure.yaml}
  set:
    things:
      structure:
        recipe: landmark
        seed: 1
        complexity: hull
        east: 0
        north: 0
        height: {value: 0, unit: m}
  world:
    unit: 1
`
	load := func() Recipe {
		t.Helper()
		got, err := Read(filepath.Join(dir, "scene.yaml"), []byte(scene))
		if err != nil {
			t.Fatal(err)
		}
		recipe, ok := got.Recipe("landmark")
		if !ok {
			t.Fatal("recipe missing")
		}
		if _, err = recipe.Construction(); err != nil {
			t.Fatal(err)
		}
		return recipe
	}
	before := load()
	changed := strings.Replace(structureFixture, "size: [2, 3, 2]", "size: [2, 4, 2]", 1)
	if err := os.WriteFile(asset, []byte(changed), 0600); err != nil {
		t.Fatal(err)
	}
	if after := load(); before.Digest() == after.Digest() {
		t.Fatal("external composition did not change recipe digest")
	}
}

func TestStructureParametersResolveAndRefuseBadReferences(t *testing.T) {
	source := strings.Replace(structureFixture, "name: footing", "name: footing\nparameters: {height: 3}", 1)
	source = strings.Replace(source, "size: [2, 3, 2]", "size: [2, {parameter: height}, 2]", 1)
	read := func(text string) error {
		_, _, err := readStructureComposition("scene.yaml", structureNode(t, "inline:\n"+indentStructure(text)))
		return err
	}
	if err := read(source); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		strings.Replace(source, "parameter: height", "parameter: missing", 1),
		strings.Replace(source, "height: 3", "height: .nan", 1),
		strings.Replace(source, "height: 3", "height: {parameter: height}", 1),
		strings.Replace(source, "parameter: height", "parameter: height, extra: 1", 1),
	} {
		if err := read(bad); err == nil {
			t.Errorf("accepted invalid parameters: %s", bad)
		}
	}
}

func TestStructureTopDerivesHeightFromResolvedBase(t *testing.T) {
	source := strings.Replace(structureFixture, "name: footing", "name: footing\nparameters: {base: 0}", 1)
	source = strings.Replace(source, "type: leg", "type: body", 1)
	source = strings.Replace(source, "at: [0, 0, 0]", "at: [0, {parameter: base}, 0]\n    top: 3", 1)
	source = strings.Replace(source, "size: [2, 3, 2]", "size: [2, 0, 2]", 1)
	read := func(text string) (float64, error) {
		got, _, err := readStructureComposition("scene.yaml", structureNode(t, "inline:\n"+indentStructure(text)))
		if err != nil {
			return 0, err
		}
		return got.Parts[0].Size[1], nil
	}
	if height, err := read(source); err != nil || height != 3 {
		t.Fatalf("height %v, error %v", height, err)
	}
	for _, bad := range []string{strings.Replace(source, "top: 3", "top: -1", 1), strings.Replace(source, "size: [2, 0, 2]", "size: [2, 2, 2]", 1)} {
		if _, err := read(bad); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}

func TestGateCompositionVoidChangesByEditingOneParameter(t *testing.T) {
	path := filepath.Join("testdata", "gate-composition.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parse := func(data []byte) (float64, float64) {
		t.Helper()
		node := structureNode(t, "inline:\n"+indentStructure(string(data)))
		c, _, err := readStructureComposition("gate.yaml", node)
		if err != nil {
			t.Fatal(err)
		}
		for _, part := range c.Parts {
			if part.ID == "tower" {
				return part.At[1], part.Size[1]
			}
		}
		t.Fatal("tower missing")
		return 0, 0
	}
	base, height := parse(raw)
	if base != 220 || height != 530 {
		t.Fatalf("base %v height %v", base, height)
	}
	changed := strings.Replace(string(raw), "parameters: {void: 220}", "parameters: {void: 300}", 1)
	if changed == string(raw) {
		t.Fatal("void parameter missing")
	}
	base, height = parse([]byte(changed))
	if base != 300 || height != 450 {
		t.Fatalf("base %v height %v", base, height)
	}
}

func TestGateSetReadsItsExternalComposition(t *testing.T) {
	path := filepath.Join("testdata", "gate-composition-set.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	scene, err := Read(path, raw)
	if err != nil {
		t.Fatal(err)
	}
	recipe, ok := scene.Recipe("gate")
	if !ok {
		t.Fatal("gate recipe missing")
	}
	if _, err := recipe.Construction(); err != nil {
		t.Fatal(err)
	}
}
