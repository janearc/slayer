package set

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"

	"github.com/janearc/slayer/manufacture/cities"
	yaml "gopkg.in/yaml.v3"
)

// A composition is bounded before it is decoded or expanded into parts.
const maxStructureBytes = 1 << 20

// readStructureComposition resolves a file relative to its set, or reads an
// inline definition. It keeps the resolved content's digest with the recipe.
func readStructureComposition(setFile string, node *yaml.Node) (cities.Composition, [32]byte, error) {
	var empty cities.Composition
	var noDigest [32]byte
	if node.Kind != yaml.MappingNode || len(node.Content) != 2 {
		return empty, noDigest, fmt.Errorf("%s: composition needs exactly one of file or inline", setFile)
	}
	key, value := node.Content[0], node.Content[1]
	var raw []byte
	source := setFile
	switch key.Value {
	case "file":
		if value.Kind != yaml.ScalarNode || value.Tag != "!!str" || value.Value == "" {
			return empty, noDigest, fmt.Errorf("%s: composition file needs a path", setFile)
		}
		source = value.Value
		if !filepath.IsAbs(source) {
			source = filepath.Join(filepath.Dir(setFile), source)
		}
		f, err := os.Open(source)
		if err != nil {
			return empty, noDigest, fmt.Errorf("%s: composition file: %w", setFile, err)
		}
		raw, err = io.ReadAll(io.LimitReader(f, maxStructureBytes+1))
		closeErr := f.Close()
		if err != nil {
			return empty, noDigest, fmt.Errorf("%s: composition: %w", source, err)
		}
		if closeErr != nil {
			return empty, noDigest, fmt.Errorf("%s: composition: %w", source, closeErr)
		}
	case "inline":
		var err error
		raw, err = yaml.Marshal(value)
		if err != nil {
			return empty, noDigest, fmt.Errorf("%s: composition: %w", source, err)
		}
	default:
		return empty, noDigest, fmt.Errorf("%s: composition needs file or inline, got %q", setFile, key.Value)
	}
	if len(raw) > maxStructureBytes {
		return empty, noDigest, fmt.Errorf("%s: composition exceeds %d bytes", source, maxStructureBytes)
	}
	var document yaml.Node
	initial := yaml.NewDecoder(bytes.NewReader(raw))
	if err := initial.Decode(&document); err != nil {
		return empty, noDigest, fmt.Errorf("%s: composition: %w", source, err)
	}
	var extra yaml.Node
	if err := initial.Decode(&extra); err != io.EOF {
		return empty, noDigest, fmt.Errorf("%s: composition must contain one document", source)
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return empty, noDigest, fmt.Errorf("%s: composition needs a mapping", source)
	}
	if err := once(source, document.Content[0]); err != nil {
		return empty, noDigest, err
	}
	if err := resolveStructureParameters(document.Content[0]); err != nil {
		return empty, noDigest, fmt.Errorf("%s: composition: %w", source, err)
	}
	if err := resolveStructureTops(document.Content[0]); err != nil {
		return empty, noDigest, fmt.Errorf("%s: composition: %w", source, err)
	}
	resolved, err := yaml.Marshal(document.Content[0])
	if err != nil {
		return empty, noDigest, fmt.Errorf("%s: composition: %w", source, err)
	}
	var result cities.Composition
	decoder := yaml.NewDecoder(bytes.NewReader(resolved))
	decoder.KnownFields(true)
	if err := decoder.Decode(&result); err != nil {
		return empty, noDigest, fmt.Errorf("%s: composition: %w", source, err)
	}
	var trailing yaml.Node
	if err := decoder.Decode(&trailing); err != io.EOF {
		return empty, noDigest, fmt.Errorf("%s: composition must contain one document", source)
	}
	if err := result.Validate(); err != nil {
		return empty, noDigest, fmt.Errorf("%s: composition %q: %w", source, result.Name, err)
	}
	return result, sha256.Sum256(raw), nil
}

// structureRecipe keeps a checked composition with the recipe, not its path.
func (reading reader) structureRecipe(name, block *yaml.Node) error {
	var spec cities.Composition
	var digest [32]byte
	var present bool
	if err := reading.keys(block, map[string]step{
		"composition": func(_ *yaml.Node, value *yaml.Node) error {
			var err error
			spec, digest, err = readStructureComposition(reading.file, value)
			present = err == nil
			return err
		},
	}, nil); err != nil {
		return err
	}
	if !present {
		return reading.refuse(block, "recipe %s needs a composition", name.Value)
	}
	path := ""
	value := valueNode(valueNode(block, "composition"), "file")
	if value != nil {
		path = value.Value
		if !filepath.IsAbs(path) {
			path = filepath.Join(filepath.Dir(reading.file), path)
		}
		var err error
		path, err = filepath.Abs(path)
		if err != nil {
			return err
		}
	}
	reading.set.recipes[name.Value] = Recipe{kind: "composition", composition: &spec, compositionDigest: digest, compositionPath: path}
	return nil
}

// resolveStructureParameters substitutes named finite numeric values before
// typed decoding. Parameters are literals, not expressions or references.
func resolveStructureParameters(root *yaml.Node) error {
	values := make(map[string]yaml.Node)
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value != "parameters" {
			continue
		}
		block := root.Content[i+1]
		if block.Kind != yaml.MappingNode {
			return fmt.Errorf("parameters must be a mapping")
		}
		for j := 0; j < len(block.Content); j += 2 {
			key, value := block.Content[j], block.Content[j+1]
			number, err := strconv.ParseFloat(value.Value, 64)
			if key.Value == "" || value.Kind != yaml.ScalarNode || (value.Tag != "!!int" && value.Tag != "!!float") || err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
				return fmt.Errorf("parameter %q needs a finite number", key.Value)
			}
			values[key.Value] = *value
		}
		root.Content = append(root.Content[:i], root.Content[i+2:]...)
		break
	}
	var replace func(*yaml.Node) error
	replace = func(n *yaml.Node) error {
		if n.Kind == yaml.AliasNode {
			return fmt.Errorf("aliases are not supported in compositions")
		}
		if n.Kind == yaml.MappingNode {
			for i := 0; i < len(n.Content); i += 2 {
				if n.Content[i].Value == "parameter" {
					if len(n.Content) != 2 || n.Content[i+1].Kind != yaml.ScalarNode {
						return fmt.Errorf("parameter reference must name one parameter")
					}
					name := n.Content[i+1].Value
					value, ok := values[name]
					if !ok {
						return fmt.Errorf("unknown parameter %q", name)
					}
					*n = value
					return nil
				}
			}
		}
		for _, child := range n.Content {
			if err := replace(child); err != nil {
				return err
			}
		}
		return nil
	}
	return replace(root)
}

// resolveStructureTops derives the height of a body from its top and base.
// A zero height is the explicit placeholder, preventing conflicting measures.
func resolveStructureTops(root *yaml.Node) error {
	parts := valueNode(root, "parts")
	if parts == nil || parts.Kind != yaml.SequenceNode {
		return nil
	}
	for _, part := range parts.Content {
		top := valueNode(part, "top")
		if top == nil {
			continue
		}
		at, size := valueNode(part, "at"), valueNode(part, "size")
		if valueOf(part, "type") != "body" || at == nil || size == nil || at.Kind != yaml.SequenceNode || size.Kind != yaml.SequenceNode || len(at.Content) != 3 || len(size.Content) != 3 {
			return fmt.Errorf("part %s: top needs a body with at and size vectors", valueOf(part, "id"))
		}
		upper, e1 := strconv.ParseFloat(top.Value, 64)
		lower, e2 := strconv.ParseFloat(at.Content[1].Value, 64)
		placeholder, e3 := strconv.ParseFloat(size.Content[1].Value, 64)
		height := upper - lower
		if e1 != nil || e2 != nil || e3 != nil || placeholder != 0 || math.IsNaN(height) || math.IsInf(height, 0) || height <= 0 {
			return fmt.Errorf("part %s: top needs a zero height and must be above the base", valueOf(part, "id"))
		}
		*size.Content[1] = yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: strconv.FormatFloat(height, 'g', -1, 64)}
		for i := 0; i < len(part.Content); i += 2 {
			if part.Content[i].Value == "top" {
				part.Content = append(part.Content[:i], part.Content[i+2:]...)
				break
			}
		}
	}
	return nil
}
