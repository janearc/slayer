// Package profilecheck validates shared profile-book YAML boundaries.
package profilecheck

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

// Authored checks the shared provenance required of named profiles.
func Authored(kind, status, by, date, source string) bool {
	return kind != "" && by != "" && date != "" &&
		(status == "picked" || ((status == "read" || status == "derived") && source != ""))
}

// checkedProfileDocument enforces one document with bounded alias traversal,
// rejecting null and merge keys before they can disappear in value fields.
func checkedProfileDocument(file string, raw []byte) (*yaml.Node, error) {
	var document yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return nil, fmt.Errorf("%s: trailing YAML: %w", file, err)
		}
		return nil, fmt.Errorf("%s: multiple YAML documents", file)
	}
	// yaml.v3's decoder returns EOF for an empty stream; a decoded document
	// always contains its single root node.
	visits := 0
	if err := checkProfileNodes(document.Content[0], "", 0, &visits); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	return document.Content[0], nil
}

// NumbersPresent checks numeric presence in sequence-based books.
// Optional pointer blocks can be omitted, but not supplied as null.
func NumbersPresent(file string, raw []byte, profileType reflect.Type, optional func(*yaml.Node, string) bool) error {
	document, err := checkedProfileDocument(file, raw)
	if err != nil {
		return err
	}
	profiles := Alias(Field(document, "profiles"))
	if profiles == nil || profiles.Kind != yaml.SequenceNode {
		return fmt.Errorf("%s: profiles must be a sequence", file)
	}
	for i, node := range profiles.Content {
		if err := checkProfileNumbers(node, profileType, node, "", optional, 0); err != nil {
			return fmt.Errorf("%s: profile %d: %w", file, i+1, err)
		}
	}
	return nil
}

// Section names a mapping whose values have the stated numeric shape.
type Section struct {
	Name string
	Kind reflect.Type
}

// SpeciesNumbersPresent checks the figure and appearance maps shared
// by the bird and fish books; family-specific bounds remain in their readers.
func SpeciesNumbersPresent(file string, raw []byte, sections []Section) error {
	document, err := checkedProfileDocument(file, raw)
	if err != nil {
		return err
	}
	species := Alias(Field(document, "species"))
	if species == nil || species.Kind != yaml.MappingNode {
		return fmt.Errorf("%s: species must be a mapping", file)
	}
	for i := 0; i+1 < len(species.Content); i += 2 {
		name, row := species.Content[i].Value, species.Content[i+1]
		for _, section := range sections {
			items := Alias(Field(row, section.Name))
			if items == nil || items.Kind != yaml.MappingNode {
				return fmt.Errorf("%s: %s needs %s", file, name, section.Name)
			}
			for j := 0; j+1 < len(items.Content); j += 2 {
				path := name + "." + section.Name + "." + items.Content[j].Value
				if err := checkProfileNumbers(items.Content[j+1], section.Kind, row, path, nil, 0); err != nil {
					return fmt.Errorf("%s: %w", file, err)
				}
			}
		}
	}
	return nil
}

// Field returns one named member of a YAML mapping, following aliases.
func Field(node *yaml.Node, key string) *yaml.Node {
	node = Alias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// Null may silently disappear even in optional strings or arrays of strings.
// Merge keys are refused so presence checks cannot disagree with the decoder.
func checkProfileNodes(node *yaml.Node, path string, depth int, visits *int) error {
	*visits++
	if *visits > 100000 {
		return fmt.Errorf("profile YAML exceeds its node budget")
	}
	if depth > 100 {
		return fmt.Errorf("profile nesting too deep at %s", path)
	}
	if node.Kind == yaml.AliasNode {
		return checkProfileNodes(node.Alias, path, depth+1, visits)
	}
	if node.Tag == "!!null" {
		return fmt.Errorf("null field %s", path)
	}
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Tag == "!!merge" {
				return fmt.Errorf("YAML merge keys are not supported at %s", path)
			}
			child := key.Value
			if path != "" {
				child = path + "." + child
			}
			if err := checkProfileNodes(node.Content[i+1], child, depth+1, visits); err != nil {
				return err
			}
		}
	case yaml.SequenceNode:
		for i, child := range node.Content {
			if err := checkProfileNodes(child, fmt.Sprintf("%s[%d]", path, i), depth+1, visits); err != nil {
				return err
			}
		}
	}
	return nil
}

// Alias follows a bounded chain of YAML aliases.
func Alias(node *yaml.Node) *yaml.Node {
	for i := 0; node != nil && node.Kind == yaml.AliasNode && i < 100; i++ {
		node = node.Alias
	}
	return node
}

func profileHasCheckedValues(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Float64:
		return true
	case reflect.Ptr, reflect.Array, reflect.Slice:
		return profileHasCheckedValues(t.Elem())
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.IsExported() && profileHasCheckedValues(f.Type) {
				return true
			}
		}
	}
	return false
}

func checkProfileNumbers(node *yaml.Node, t reflect.Type, profile *yaml.Node, path string, optional func(*yaml.Node, string) bool, depth int) error {
	if !profileHasCheckedValues(t) {
		return nil
	}
	if node == nil {
		if t.Kind() == reflect.Ptr || t.Kind() == reflect.Bool || (optional != nil && optional(profile, path)) {
			return nil
		}
		return fmt.Errorf("missing numeric field %s", path)
	}
	if depth > 100 {
		return fmt.Errorf("profile nesting too deep at %s", path)
	}
	if node.Kind == yaml.AliasNode {
		return checkProfileNumbers(node.Alias, t, profile, path, optional, depth+1)
	}
	if node.Tag == "!!null" {
		return fmt.Errorf("null field %s", path)
	}
	if t.Kind() == reflect.Ptr {
		return checkProfileNumbers(node, t.Elem(), profile, path, optional, depth+1)
	}
	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Float64:
		return nil // The strict decoder checks the scalar type.
	case reflect.Array, reflect.Slice:
		if node.Kind != yaml.SequenceNode || (t.Kind() == reflect.Array && len(node.Content) != t.Len()) {
			return fmt.Errorf("invalid numeric sequence %s", path)
		}
		for i, child := range node.Content {
			if err := checkProfileNumbers(child, t.Elem(), profile, fmt.Sprintf("%s[%d]", path, i), optional, depth+1); err != nil {
				return err
			}
		}
	case reflect.Struct:
		if node.Kind != yaml.MappingNode {
			return fmt.Errorf("invalid numeric block %s", path)
		}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() || !profileHasCheckedValues(f.Type) {
				continue
			}
			key := strings.Split(f.Tag.Get("yaml"), ",")[0]
			if key == "" {
				key = strings.ToLower(f.Name)
			}
			childPath := key
			if path != "" {
				childPath = path + "." + key
			}
			if err := checkProfileNumbers(Field(node, key), f.Type, profile, childPath, optional, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}
