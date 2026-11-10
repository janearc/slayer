package set

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"slices"
	"strconv"
	"testing"
)

// stepKeys is the keys of a literal map of steps, the map a reader block
// reads by, and whether the node is one.
func stepKeys(t *testing.T, node ast.Node) ([]string, bool) {
	t.Helper()
	literal, ok := node.(*ast.CompositeLit)
	if !ok {
		return nil, false
	}
	mapping, ok := literal.Type.(*ast.MapType)
	if !ok {
		return nil, false
	}
	value, ok := mapping.Value.(*ast.Ident)
	if !ok || value.Name != "step" {
		return nil, false
	}
	var keys []string
	for _, entry := range literal.Elts {
		pair, ok := entry.(*ast.KeyValueExpr)
		if !ok {
			t.Fatal("reader key is not literal")
		}
		key, ok := pair.Key.(*ast.BasicLit)
		if !ok {
			t.Fatal("reader key name is not literal")
		}
		name, err := strconv.Unquote(key.Value)
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, name)
	}
	return keys, true
}

// readerMethod is each body of a reader method in a file.
func readerMethod(t *testing.T, file, method string) []*ast.BlockStmt {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var bodies []*ast.BlockStmt
	for _, decl := range parsed.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == method {
			bodies = append(bodies, fn.Body)
		}
	}
	return bodies
}

// readerKeptSet is the keys a reader block keeps without reading them:
// the list of strings given beside its key map in the same call, as the
// assets block keeps autogen. A block given none keeps nothing.
func readerKeptSet(t *testing.T, file, method, anchor string) []string {
	t.Helper()
	var kept []string
	for _, body := range readerMethod(t, file, method) {
		ast.Inspect(body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || !callReads(t, call, anchor) {
				return true
			}
			for _, arg := range call.Args {
				kept = append(kept, stringList(t, arg)...)
			}
			return true
		})
	}
	slices.Sort(kept)
	return kept
}

// callReads is whether a call is given the key map that holds the anchor.
func callReads(t *testing.T, call *ast.CallExpr, anchor string) bool {
	t.Helper()
	for _, arg := range call.Args {
		if keys, ok := stepKeys(t, arg); ok && slices.Contains(keys, anchor) {
			return true
		}
	}
	return false
}

// stringList is the strings of a literal []string, or none for any other
// node.
func stringList(t *testing.T, node ast.Node) []string {
	t.Helper()
	literal, ok := node.(*ast.CompositeLit)
	if !ok {
		return nil
	}
	array, ok := literal.Type.(*ast.ArrayType)
	if !ok {
		return nil
	}
	if element, ok := array.Elt.(*ast.Ident); !ok || element.Name != "string" {
		return nil
	}
	var names []string
	for _, entry := range literal.Elts {
		text, ok := entry.(*ast.BasicLit)
		if !ok {
			t.Fatal("a kept key is not literal")
		}
		name, err := strconv.Unquote(text.Value)
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	return names
}

// readerKeySet reads the literal key map used by a particular reader block.
// The anchor selects nested blocks without depending on source line numbers.
func readerKeySet(t *testing.T, file, method, anchor string) []string {
	t.Helper()
	var found []string
	for _, body := range readerMethod(t, file, method) {
		ast.Inspect(body, func(node ast.Node) bool {
			keys, ok := stepKeys(t, node)
			if !ok {
				return true
			}
			if slices.Contains(keys, anchor) {
				if found != nil {
					t.Fatal("ambiguous reader block", file, method, anchor)
				}
				found = keys
			}
			return true
		})
	}
	if found == nil {
		t.Fatal("reader block missing", file, method, anchor)
	}
	slices.Sort(found)
	return found
}

func TestEcosystemSubsetAndReaderKeySetsStayInStep(t *testing.T) {
	var schema map[string]any
	if err := json.Unmarshal(ecosystemRaw, &schema); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		file, method, anchor string
		path                 []string
		omitted              []string
	}{
		{"scene.go", "scene", "renderer", []string{"$defs", "scene"}, nil},
		{"planet.go", "planetKeys", "airglow", []string{"$defs", "planet"}, nil},
		{"planet.go", "planetKeys", "altitude", []string{"$defs", "planet", "properties", "pole"}, nil},
		{"planet.go", "zones", "column", []string{"$defs", "planet", "properties", "zones", "items"}, nil},
		{"planet.go", "sky", "distance", []string{"$defs", "planet", "properties", "sky"}, nil},
		{"planet.go", "rays", "times", []string{"$defs", "planet", "properties", "xrays"}, nil},
		{"weather.go", "weathered", "cells", []string{"$defs", "planet", "properties", "weather"}, nil},
		{"weather.go", "weathered", "rate", []string{"$defs", "planet", "properties", "weather", "properties", "storms"}, nil},
		{"weather.go", "heated", "tops", []string{"$defs", "planet", "properties", "heat"}, nil},
		{"weather.go", "cooked", "time", []string{"$defs", "planet", "properties", "cooking"}, nil},
		{"weather.go", "glowing", "radiance", []string{"$defs", "planet", "properties", "airglow"}, nil},
		{"planet.go", "layer", "bottom", []string{"$defs", "planet-layer"}, nil},
		{"planet.go", "layer", "lean", []string{"$defs", "planet-layer", "properties", "haze"}, nil},
		{"planet.go", "ring", "made_of", []string{"$defs", "planet-ring"}, nil},
		{"planet.go", "grained", "clumping", []string{"$defs", "planet-ring", "properties", "grains"}, nil},
		{"planet.go", "moons", "opens", []string{"$defs", "planet-ring", "properties", "moonlets"}, nil},
		{"planet.go", "storms", "turbulence", []string{"$defs", "planet-storms"}, nil},
		{"planet.go", "storms", "clearing", []string{"$defs", "planet-storms", "properties", "marks"}, nil},
		{"planet.go", "storms", "strength", []string{"$defs", "planet-storms", "properties", "churn"}, nil},
		{"planet.go", "jets", "every", []string{"$defs", "planet-storms", "properties", "jets"}, nil},
		{"planet.go", "swarm", "slope", []string{"$defs", "planet-storms", "properties", "swarm"}, nil},
		{"planet.go", "spots", "turn", []string{"$defs", "planet-storms", "properties", "spots", "items"}, nil},
		{"pieces.go", "world", "motions", []string{"$defs", "world"}, nil},
		{"pieces.go", "materials", "glow", []string{"$defs", "world", "properties", "materials", "additionalProperties"}, nil},
		{"pieces.go", "colour", "swatch", []string{"$defs", "colour"}, nil},
		{"pieces.go", "boxes", "size", []string{"$defs", "world", "properties", "boxes", "additionalProperties"}, nil},
		{"shapes.go", "spheres", "motion", []string{"$defs", "world", "properties", "spheres", "additionalProperties"}, nil},
		{"pieces.go", "lamps", "radius", []string{"$defs", "world", "properties", "lamps", "additionalProperties"}, nil},
		{"spots.go", "spotlights", "cone", []string{"$defs", "world", "properties", "spots", "additionalProperties"}, nil},
		{"pieces.go", "floods", "strength", []string{"$defs", "world", "properties", "floods", "additionalProperties"}, nil},
		{"pieces.go", "backdrop", "horizon", []string{"$defs", "world", "properties", "sky"}, nil},
		{"pieces.go", "backdrop", "share", []string{"$defs", "world", "properties", "sky", "properties", "stars"}, nil},
		{"pieces.go", "bodies", "aurora", []string{"$defs", "world", "properties", "bodies", "additionalProperties"}, nil},
		{"pieces.go", "suns", "toward", []string{"$defs", "world", "properties", "suns", "additionalProperties"}, nil},
		{"shapes.go", "wisps", "thick", []string{"$defs", "world", "properties", "wisps", "additionalProperties"}, nil},
		{"motion.go", "motions", "ease", []string{"$defs", "world", "properties", "motions", "additionalProperties"}, nil},
		{"motion.go", "framed", "percent", []string{"$defs", "world", "properties", "motions", "additionalProperties", "properties", "frames", "items"}, nil},
		{"hud.go", "hud", "theme", []string{"$defs", "hud"}, nil},
		{"hud.go", "hudKeys", "occluded", []string{"$defs", "hud-element"}, nil},
		{"scene.go", "scene", "assets", []string{"$defs", "scene", "properties", "set"}, nil},
		{"scene.go", "scene", "size_x", []string{"$defs", "scene", "properties", "set", "properties", "bounded_box"}, nil},
		{"scene.go", "assets", "name", []string{"$defs", "scene", "properties", "set", "properties", "assets", "additionalProperties"}, nil},
		{"text.go", "texts", "words", []string{"$defs", "text"}, nil},
		{"world.go", "film", "rays", []string{"$defs", "film"}, nil},
		{"world.go", "film", "focal", []string{"$defs", "film", "properties", "lens"}, nil},
		{"world.go", "printed", "knee", []string{"$defs", "film", "properties", "print"}, nil},
		{"scene.go", "cameras", "eye", []string{"$defs", "camera"}, nil},
		{"world.go", "stands", "within", []string{"$defs", "stand"}, nil},
		{"world.go", "wedge", "near", []string{"$defs", "stand", "properties", "within"}, nil},
		{"world.go", "things", "north", []string{"$defs", "thing"}, nil},
		{"world.go", "terrain", "from", []string{"$defs", "terrain"}, nil},
	} {
		block := schema
		for _, key := range test.path {
			next, ok := block[key].(map[string]any)
			if !ok {
				t.Fatal("schema object missing", test.path)
			}
			block = next
		}
		if block["additionalProperties"] != false {
			t.Fatal("subset must be explicit", test.path)
		}
		properties, ok := block["properties"].(map[string]any)
		if !ok {
			t.Fatal("schema properties missing", test.path)
		}
		var actual []string
		for key := range properties {
			actual = append(actual, key)
		}
		slices.Sort(actual)
		var wanted []string
		readerKeys := readerKeySet(t, test.file, test.method, test.anchor)
		for _, key := range test.omitted {
			if !slices.Contains(readerKeys, key) {
				t.Fatal("stale subset omission", test.path, key)
			}
		}
		for _, key := range readerKeys {
			if !slices.Contains(test.omitted, key) {
				wanted = append(wanted, key)
			}
		}
		// what the reader keeps unread, the schema takes too: a scene
		// that holds it is read, so the schema must not refuse it
		wanted = append(wanted, readerKeptSet(t, test.file, test.method,
			test.anchor)...)
		slices.Sort(wanted)
		if !reflect.DeepEqual(actual, wanted) {
			t.Errorf("%v schema keys %v; reader subset %v", test.path, actual, wanted)
		}
	}
}
