package workspace

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/set"
	"gopkg.in/yaml.v3"
)

// builtinExampleProfiles replaces only bundled book files whose bytes match
// the renderer's embedded book. Other profile dependencies remain unchanged.
func builtinExampleProfiles(root, sourcePath string, source []byte) ([]byte, error) {
	var tree yaml.Node
	if err := yamlfile.Read(sourcePath, source, &tree); err != nil || len(tree.Content) != 1 {
		return source, nil
	}
	recipes := exampleField(exampleField(tree.Content[0], "scene"), "recipes")
	if recipes == nil || recipes.Kind != yaml.MappingNode {
		return source, nil
	}
	changed := false
	for i := 1; i < len(recipes.Content); i += 2 {
		for _, kind := range []string{"animal", "plant"} {
			profile := exampleField(recipes.Content[i], kind)
			family, file := exampleField(profile, "family"), exampleField(profile, "file")
			if file == nil || file.Kind != yaml.ScalarNode || file.Value == "" || filepath.IsAbs(file.Value) {
				continue
			}
			familyName := "seasonal"
			if family != nil {
				familyName = family.Value
			} else if kind == "animal" {
				continue
			}
			var embedded []byte
			var err error
			if kind == "animal" {
				_, embedded, err = manufacture.EmbeddedAnimalBook(familyName)
			} else {
				_, embedded, err = manufacture.EmbeddedPlantBook(familyName)
			}
			if err != nil {
				continue
			}
			path := filepath.Clean(filepath.Join(filepath.Dir(sourcePath), file.Value))
			relative, err := filepath.Rel(root, path)
			if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				continue
			}
			raw, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(raw, embedded) {
				continue
			}
			file.Value, file.Tag = set.BuiltinProfileBook, "!!str"
			changed = true
		}
	}
	if !changed {
		return source, nil
	}
	var output bytes.Buffer
	encoder := yaml.NewEncoder(&output)
	encoder.SetIndent(2)
	if err := encoder.Encode(&tree); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func exampleField(node *yaml.Node, name string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == name {
			return node.Content[i+1]
		}
	}
	return nil
}
