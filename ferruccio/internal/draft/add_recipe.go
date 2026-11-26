package draft

import (
	"bytes"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// RecipeNames lists recipes in their authored order from a checked set.
func (document *Document) RecipeNames() ([]string, error) {
	if document.kind != "set" {
		return nil, fmt.Errorf("recipe browsing requires a set")
	}
	tree, err := document.Tree()
	if err != nil {
		return nil, err
	}
	scene := member(tree.Content[0], "scene")
	if !document.addSetting(scene) {
		return nil, fmt.Errorf("recipe placement requires a space or v2 ground set")
	}
	recipes := member(scene, "recipes")
	var names []string
	if recipes != nil && recipes.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(recipes.Content); i += 2 {
			names = append(names, recipes.Content[i].Value)
		}
	}
	return names, nil
}

// RebaseProfileBooks keeps relative profile books pointed at the same files
// when a set is saved into another directory. Invalid drafts remain editable.
func (document *Document) RebaseProfileBooks(from, to string) error {
	if document.kind != "set" || !filepath.IsAbs(from) || !filepath.IsAbs(to) || filepath.Dir(from) == filepath.Dir(to) {
		return nil
	}
	tree, err := document.Tree()
	if err != nil {
		return nil
	}
	recipes := member(member(tree.Content[0], "scene"), "recipes")
	if recipes == nil || recipes.Kind != yaml.MappingNode {
		return nil
	}
	changed := false
	for i := 1; i < len(recipes.Content); i += 2 {
		for _, kind := range []string{"animal", "plant"} {
			book := member(member(recipes.Content[i], kind), "file")
			if book == nil || book.Kind != yaml.ScalarNode || book.Value == "" || book.Value == "@builtin" || filepath.IsAbs(book.Value) {
				continue
			}
			absolute := filepath.Join(filepath.Dir(from), book.Value)
			relative, err := filepath.Rel(filepath.Dir(to), absolute)
			if err != nil {
				return err
			}
			book.Value, changed = relative, true
		}
	}
	if !changed {
		return nil
	}
	var encoded bytes.Buffer
	encoder := yaml.NewEncoder(&encoded)
	encoder.SetIndent(2)
	if err := encoder.Encode(tree); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	if _, err := document.read(encoded.Bytes()); err != nil {
		return err
	}
	document.source = bytes.Clone(encoded.Bytes())
	return nil
}

// AddRecipeThing places an existing recipe in a supported set at metres east,
// north and height. It atomically refuses an invalid or ambiguous placement.
func (document *Document) AddRecipeThing(name, recipe string, seed uint64, at [3]float64) error {
	return document.addThing(name, "recipe", recipe, seed, at, nil)
}

// AddKindThing places a constructor kind in a supported set. Callers must check
// the kind against Slayer's constructor registry before submitting it.
func (document *Document) AddKindThing(name, kind string, seed uint64, at [3]float64) error {
	return document.addThing(name, "kind", kind, seed, at, nil)
}

type profileInsertion struct {
	kind string
	spec *yaml.Node
}

// AddProfileThing adds a profile-backed recipe and its placement together.
// The caller must check the chosen book and constructed candidate with Slayer.
func (document *Document) AddProfileThing(name, recipe, kind string, spec *yaml.Node, book string, seed uint64, at [3]float64) error {
	if (kind != "animal" && kind != "plant") || spec == nil || spec.Kind != yaml.MappingNode || strings.TrimSpace(book) == "" ||
		strings.TrimSpace(recipe) == "" || strings.TrimSpace(recipe) != recipe {
		return fmt.Errorf("a profile placement needs a recipe, profile and book path")
	}
	copy := copyNode(spec)
	file := member(copy, "file")
	if file == nil || file.Kind != yaml.ScalarNode {
		return fmt.Errorf("profile has no book field")
	}
	file.Tag, file.Value = "!!str", book
	return document.addThing(name, "recipe", recipe, seed, at, &profileInsertion{kind: kind, spec: copy})
}

// copyNode detaches a checked catalogue mapping before changing its book.
func copyNode(node *yaml.Node) *yaml.Node {
	copy := *node
	copy.Content = make([]*yaml.Node, len(node.Content))
	for index, child := range node.Content {
		copy.Content[index] = copyNode(child)
	}
	return &copy
}

// addThing validates and commits one explicit recipe or kind placement.
func (document *Document) addThing(name, field, value string, seed uint64, at [3]float64, profile *profileInsertion) error {
	if document.kind != "set" || strings.TrimSpace(name) == "" || name != strings.TrimSpace(name) || value == "" || seed > 1<<53 {
		return fmt.Errorf("a named placement requires a set, name and valid seed")
	}
	for _, coordinate := range at {
		if math.IsNaN(coordinate) || math.IsInf(coordinate, 0) {
			return fmt.Errorf("recipe placement coordinates must be finite")
		}
	}
	tree, err := document.Tree()
	if err != nil {
		return err
	}
	scene := member(tree.Content[0], "scene")
	if !document.addSetting(scene) {
		return fmt.Errorf("recipe placement requires a space or v2 ground set")
	}
	if document.groundSetting(scene) && (at[0] < 0 || at[1] < 0 || at[2] < 0) {
		return fmt.Errorf("ground placement coordinates and height must be nonnegative")
	}
	if field == "recipe" {
		recipes := member(scene, "recipes")
		if profile == nil && member(recipes, value) == nil {
			return fmt.Errorf("recipe %q is not defined in this set", value)
		}
		if profile != nil {
			if member(recipes, value) != nil {
				return fmt.Errorf("recipe %q already exists", value)
			}
			recipes, err = scalar(scene, []string{"recipes"})
			if err != nil || recipes.Kind != yaml.MappingNode {
				return fmt.Errorf("scene recipes must be a mapping")
			}
			recipe := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			recipe.Content = append(recipe.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: profile.kind}, profile.spec)
			if profile.kind == "plant" {
				for index := 0; index+1 < len(profile.spec.Content); index += 2 {
					if profile.spec.Content[index].Value == "season" {
						recipe.Content = append(recipe.Content, profile.spec.Content[index:index+2]...)
						profile.spec.Content = append(profile.spec.Content[:index], profile.spec.Content[index+2:]...)
						break
					}
				}
			}
			recipes.Content = append(recipes.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}, recipe)
		}
	}
	things, err := scalar(scene, []string{"set", "things"})
	if err != nil || things.Kind != yaml.MappingNode {
		return fmt.Errorf("scene things must be a mapping")
	}
	if member(things, name) != nil {
		return fmt.Errorf("thing %q already exists", name)
	}
	stringNode := func(value string) *yaml.Node { return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value} }
	numberNode := func(value float64) *yaml.Node {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: strconv.FormatFloat(value, 'f', -1, 64)}
	}
	block := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	block.Content = []*yaml.Node{
		stringNode(field), stringNode(value),
		stringNode("seed"), {Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.FormatUint(seed, 10)},
		stringNode("east"), numberNode(at[0]),
		stringNode("north"), numberNode(at[1]),
	}
	setNumber(quantityNode(block, "height", "m"), at[2])
	if document.groundSetting(scene) {
		setNumber(quantityNode(block, "heading", "deg"), 0)
	}
	things.Content = append(things.Content, stringNode(name), block)
	var encoded bytes.Buffer
	encoder := yaml.NewEncoder(&encoded)
	encoder.SetIndent(2)
	if err := encoder.Encode(tree); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	if _, err := document.read(encoded.Bytes()); err != nil {
		return fmt.Errorf("recipe placement would make invalid set YAML: %w", err)
	}
	document.source = bytes.Clone(encoded.Bytes())
	return nil
}

// addSetting identifies scene coordinate semantics this editor can author.
func (document *Document) addSetting(scene *yaml.Node) bool {
	setting := member(scene, "setting")
	return document.groundSetting(scene) || setting != nil && setting.Value == "space"
}

// groundSetting follows Slayer’s v2 default when no setting is authored.
func (document *Document) groundSetting(scene *yaml.Node) bool {
	setting := member(scene, "setting")
	return document.kind == "set" && document.schemaName == "https://github.com/janearc/slayer/schemas/ecosystem-v2.json" && (setting == nil || setting.Value == "ground")
}

// GroundSetting reports whether a v2 set uses explicit or default terrain.
func (document *Document) GroundSetting() bool {
	tree, err := document.Tree()
	return err == nil && document.groundSetting(member(tree.Content[0], "scene"))
}
