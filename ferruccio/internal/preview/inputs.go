package preview

import (
	"os"
	"path/filepath"

	"github.com/janearc/puzzlemouth/yamlfile"
	"gopkg.in/yaml.v3"
)

// declaredInputs binds known direct source references without claiming closure discovery.
func (plan *Plan) declaredInputs() ([]string, error) {
	inputs := []string{plan.Input, filepath.Join(plan.Directory, "scene.yaml")}
	raw, err := os.ReadFile(plan.Input)
	if err != nil {
		return nil, err
	}
	var tree yaml.Node
	if err := yamlfile.Read(plan.Input, raw, &tree); err != nil {
		return nil, err
	}
	if len(tree.Content) != 1 {
		return nil, os.ErrInvalid
	}
	scene := field(tree.Content[0], "scene")
	base := filepath.Dir(plan.Input)
	if plan.relocated {
		base = filepath.Dir(plan.Source)
	}
	add := func(node *yaml.Node) {
		if node == nil || node.Kind != yaml.ScalarNode || node.Tag != "!!str" || node.Value == "" {
			return
		}
		path := node.Value
		if !filepath.IsAbs(path) {
			path = filepath.Join(base, path)
		}
		path = filepath.Clean(path)
		for _, known := range inputs {
			if known == path {
				return
			}
		}
		inputs = append(inputs, path)
	}
	if plan.Kind == "job" {
		add(field(scene, "set"))
	}
	if plan.Kind == "set" || plan.Kind == "legacy-set" {
		imports := field(field(scene, "world"), "imports")
		if imports != nil && imports.Kind == yaml.MappingNode {
			for index := 1; index < len(imports.Content); index += 2 {
				add(field(imports.Content[index], "file"))
			}
		}
	}
	return inputs, nil
}
