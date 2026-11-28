package draft

import (
	"bytes"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// RebasedJobText retains the input scene when a job moves to another folder.
// An unfinished job can still be saved with its exact draft text.
func (document *Document) RebasedJobText(text, from, to string) (string, error) {
	if document.kind != "job" || !filepath.IsAbs(from) || !filepath.IsAbs(to) ||
		filepath.Dir(from) == filepath.Dir(to) {
		return text, nil
	}
	tree, err := document.read([]byte(text))
	if err != nil {
		return text, nil
	}
	scene := member(tree.Content[0], "scene")
	changed := false
	for _, key := range []string{"set", "play", "frames"} {
		node := member(scene, key)
		path := scenePath(node)
		if path == "" || filepath.IsAbs(path) {
			continue
		}
		relative, err := filepath.Rel(filepath.Dir(to), filepath.Join(filepath.Dir(from), path))
		if err != nil {
			return "", err
		}
		if relative != path {
			replaceScenePath(tree, node, relative)
			changed = true
		}
	}
	if !changed {
		return text, nil
	}
	return document.encodeRelocatedJob(tree)
}

func scenePath(node *yaml.Node) string {
	if node != nil && node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	if node == nil || node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
		return ""
	}
	return node.Value
}

func replaceScenePath(tree, node *yaml.Node, value string) {
	if node.Kind == yaml.AliasNode {
		unlinkScenePath(node, node.Alias)
	} else if node.Anchor != "" {
		retainScenePathAliases(tree, node)
	}
	node.Value = value
}

// retainScenePathAliases keeps other fields that share the authored scalar
// resolved to its original value.
func retainScenePathAliases(tree, original *yaml.Node) {
	if tree.Kind == yaml.AliasNode && tree.Alias == original {
		unlinkScenePath(tree, original)
	}
	for _, child := range tree.Content {
		retainScenePathAliases(child, original)
	}
}

func unlinkScenePath(alias, original *yaml.Node) {
	alias.Kind = yaml.ScalarNode
	alias.Tag, alias.Style, alias.Value = original.Tag, original.Style, original.Value
	alias.Alias = nil
}

func (document *Document) encodeRelocatedJob(tree *yaml.Node) (string, error) {
	var encoded bytes.Buffer
	encoder := yaml.NewEncoder(&encoded)
	encoder.SetIndent(2)
	if err := encoder.Encode(tree); err != nil {
		return "", err
	}
	if err := encoder.Close(); err != nil {
		return "", err
	}
	if _, err := document.read(encoded.Bytes()); err != nil {
		return "", err
	}
	return encoded.String(), nil
}
