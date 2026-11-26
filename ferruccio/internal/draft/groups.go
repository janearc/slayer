package draft

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Group identifies one authored procedural stand, not an individual member.
type Group struct {
	ID, Name string
}

// Groups lists authored stands without constructing or editing their members.
func (document *Document) Groups() ([]Group, error) {
	if document.kind != "set" && document.kind != "legacy-set" {
		return nil, fmt.Errorf("generated groups require a scene set")
	}
	tree, err := document.Tree()
	if err != nil {
		return nil, err
	}
	stands := member(member(member(tree.Content[0], "scene"), "set"), "stands")
	if stands == nil || stands.Kind != yaml.MappingNode {
		return nil, nil
	}
	groups := make([]Group, 0, len(stands.Content)/2)
	for i := 0; i+1 < len(stands.Content); i += 2 {
		name := stands.Content[i].Value
		groups = append(groups, Group{ID: "scene.set.stands/" + name, Name: name})
	}
	return groups, nil
}
