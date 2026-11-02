// Package set reads YAML scene descriptions and delegates each part to the
// package responsible for building it. A set specifies scene content and
// settings; individual packages retain their construction and validation rules.
package set

import (
	"fmt"

	yaml "go.yaml.in/yaml/v3"
)

// parse is a set file's top: the keys it is made of, every node keeping
// the line and column it was written at, so a refusal can point there.
//
// it refuses a file that is not YAML, one whose top is not keys, and a
// block that says a key twice, which YAML forbids and its reader lets
// through when it keeps the file as written.
func parse(file string, source []byte) (*yaml.Node, error) {
	var document yaml.Node
	if err := yaml.Unmarshal(source, &document); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	if len(document.Content) != 1 ||
		document.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s: a set file is keys, and this is not",
			file)
	}
	return document.Content[0], once(file, document.Content[0])
}

// once refuses a block anywhere under a node that says a key twice.
func once(file string, node *yaml.Node) error {
	seen := map[string]bool{}
	for index, each := range node.Content {
		if node.Kind == yaml.MappingNode && index%2 == 0 {
			if seen[each.Value] {
				return fmt.Errorf("%s:%d:%d: %s is said twice in one "+
					"block", file, each.Line, each.Column, each.Value)
			}
			seen[each.Value] = true
		}
		if err := once(file, each); err != nil {
			return err
		}
	}
	return nil
}
