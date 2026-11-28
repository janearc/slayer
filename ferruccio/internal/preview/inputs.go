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
	add := func(node *yaml.Node, allowEmbeddedBook bool) string {
		if node != nil && node.Kind == yaml.AliasNode {
			node = node.Alias
		}
		if node == nil || node.Kind != yaml.ScalarNode || node.Tag != "!!str" || node.Value == "" {
			return ""
		}
		path := node.Value
		if allowEmbeddedBook && path == "@builtin" {
			return ""
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(base, path)
		}
		path = filepath.Clean(path)
		for _, known := range inputs {
			if known == path {
				return path
			}
		}
		inputs = append(inputs, path)
		return path
	}
	if plan.Kind == "job" {
		if path := add(field(scene, "set"), false); path != "" {
			if source, err := os.ReadFile(path); err == nil {
				var setTree yaml.Node
				if yamlfile.Read(path, source, &setTree) == nil && len(setTree.Content) == 1 {
					scene = field(setTree.Content[0], "scene")
					base = filepath.Dir(path)
				}
			}
		}
	}
	if plan.Kind == "set" || plan.Kind == "legacy-set" || plan.Kind == "job" {
		// Recipe profile and composition books are direct renderer inputs too.
		// Keep their file references in the queue record, including repeated
		// recipes that share one book.
		type recipeInput struct {
			node            *yaml.Node
			embeddedAllowed bool
		}
		seen := make(map[recipeInput]bool)
		var recipes func(*yaml.Node, bool)
		recipes = func(node *yaml.Node, embeddedAllowed bool) {
			current := recipeInput{node, embeddedAllowed}
			if node == nil || seen[current] {
				return
			}
			seen[current] = true
			if node.Kind == yaml.AliasNode {
				recipes(node.Alias, embeddedAllowed)
				return
			}
			if node.Kind == yaml.MappingNode {
				for i := 0; i+1 < len(node.Content); i += 2 {
					key := node.Content[i].Value
					if key == "file" {
						add(node.Content[i+1], embeddedAllowed)
					} else {
						recipes(node.Content[i+1], key == "animal" || key == "plant")
					}
				}
			} else if node.Kind == yaml.SequenceNode {
				for _, child := range node.Content {
					recipes(child, false)
				}
			}
		}
		recipes(field(scene, "recipes"), false)
		imports := field(field(scene, "world"), "imports")
		if imports != nil && imports.Kind == yaml.MappingNode {
			for index := 1; index < len(imports.Content); index += 2 {
				add(field(imports.Content[index], "file"), false)
			}
		}
	}
	return inputs, nil
}
