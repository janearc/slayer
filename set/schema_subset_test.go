package set

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"slices"
	"strconv"
	"strings"
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

// v2Also is, for a block of the reader named by its file, method and
// anchor, the other literal maps of steps that hold the keys the reader
// takes in a file that names schema version 2 and version 1 does not
// have, each as file/method/anchor.
var v2Also = map[string][]string{
	"scene.go/scene/renderer":            {"space.go/spaceScene/setting"},
	"world.go/film/rays":                 {"space.go/spaceFilm/fps"},
	"scene.go/cameraKeys/eye":            {"space_cameras.go/spaceCamera/duration"},
	"scene.go/locationKeys/loc_relative": {"space_cameras.go/spaceLocation/relative"},
	"world.go/things/north": {"space_things.go/placing/place",
		"space_things.go/starring/star", "space_things.go/lanterning/panels",
		"space_things.go/orbiting/radius", "space_things.go/nebulous/thickness",
		"space_world.go/spacecraft/orbit"},
}

// v2Unread is, for a block of the reader named the same way, the keys
// schema version 2 has that the reader does not take yet.
var v2Unread = map[string][]string{}

// A schema version the reader reads against: its name and its text.
type schemaVersion struct {
	name string
	raw  []byte
	two  bool
}

func TestEcosystemSubsetAndReaderKeySetsStayInStep(t *testing.T) {
	for _, version := range []schemaVersion{{"version 1", ecosystemRaw, false},
		{"version 2", ecosystemV2Raw, true}} {
		t.Run(version.name, func(t *testing.T) {
			keySetsStayInStep(t, version)
		})
	}
}

// keySetsStayInStep requires each block of keys the reader reads to take
// the keys the schema's block has, as far as the version goes: version
// 1's without the keys only version 2 has, and version 2's but for the
// ones the reader does not take yet.
func keySetsStayInStep(t *testing.T, version schemaVersion) {
	var schema map[string]any
	if err := json.Unmarshal(version.raw, &schema); err != nil {
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
		{"shapes.go", "fizzes", "rise", []string{"$defs", "world", "properties", "fizzes", "additionalProperties"}, nil},
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
		{"scene.go", "cameraKeys", "eye", []string{"$defs", "camera"}, nil},
		{"scene.go", "locationKeys", "loc_relative", []string{"$defs", "camera", "properties", "location"}, nil},
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
		name := test.file + "/" + test.method + "/" + test.anchor
		for _, also := range v2Also[name] {
			parts := strings.Split(also, "/")
			keys := readerKeySet(t, parts[0], parts[1], parts[2])
			if version.two {
				readerKeys = append(readerKeys, keys...)
			}
		}
		for _, key := range v2Unread[name] {
			if slices.Contains(readerKeys, key) {
				t.Fatal("stale unread key", test.path, key)
			}
			if version.two {
				wanted = append(wanted, key)
			}
		}
		slices.Sort(readerKeys)
		readerKeys = slices.Compact(readerKeys)
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

// v2Blocks are the blocks of keys that only schema version 2 has, each by
// the reader's file, method and anchor and the path of its schema block.
var v2Blocks = []keyBlock{
	{"space.go", "frameStep", "at", []string{"$defs", "space-frame"}},
	{"space_things.go", "placeStep", "at", []string{"$defs", "thing", "properties", "place"}},
	{"space_things.go", "coronaStep", "reach", []string{"$defs", "corona"}},
	{"space_things.go", "leanStep", "value", []string{"$defs", "thing", "properties", "lean"}},
	{"space_things.go", "lightsStep", "kelvin", []string{"$defs", "thing", "properties", "lights"}},
	{"space_things.go", "kelvinStep", "from", []string{"$defs", "thing", "properties", "lights", "properties", "kelvin"}},
	{"space_things.go", "lengthStep", "unit", []string{"$defs", "space-length"}},
	{"space_cameras.go", "moveStep", "ease", []string{"$defs", "move"}},
	{"space_cameras.go", "relativeStep", "above_plane", []string{"$defs", "relative-place"}},
	{"space_cameras.go", "degreesStep", "unit", []string{"$defs", "degrees"}},
	{"space_cameras.go", "skyStep", "ra", []string{"$defs", "sky-direction"}},
	{"space_cameras.go", "lookStep", "offset", []string{"$defs", "look"}},
	{"space_cameras.go", "fieldStep", "from", []string{"$defs", "move", "properties", "field"}},
	{"space_world.go", "anchorStep", "datum", []string{"$defs", "anchor"}},
	{"space_world.go", "elevationStep", "value", []string{"$defs", "anchor", "properties", "elevation"}},
	{"space_world.go", "globeStep", "planet", []string{"$defs", "globe"}},
	{"space_world.go", "globeTerrainStep", "near", []string{"$defs", "globe", "properties", "terrain"}},
	{"space_world.go", "nearStep", "file", []string{"$defs", "globe", "properties", "terrain", "properties", "near"}},
	{"space_world.go", "beyondStep", "from", []string{"$defs", "globe", "properties", "terrain", "properties", "beyond"}},
	{"space_world.go", "starSkyStep", "stars", []string{"$defs", "sky"}},
	{"space_world.go", "starsStep", "faintest", []string{"$defs", "sky", "properties", "stars"}},
	{"space_world.go", "orbitStep", "inclination", []string{"$defs", "orbit"}},
	{"space_world.go", "attitudeStep", "shield", []string{"$defs", "attitude"}},
	{"space_world.go", "instrumentStep", "pupil", []string{"$defs", "instrument-eye"}},
}

// keyBlock is one block of keys the reader reads: where in the reader its
// literal map is, and where in the schema its properties are.
type keyBlock struct {
	file, method, anchor string
	path                 []string
}

// The blocks that only version 2 has take exactly the keys its schema
// gives them.
func TestVersionTwoBlocksStayInStep(t *testing.T) {
	var schema map[string]any
	if err := json.Unmarshal(ecosystemV2Raw, &schema); err != nil {
		t.Fatal(err)
	}
	for _, test := range v2Blocks {
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
		var actual []string
		for key := range block["properties"].(map[string]any) {
			actual = append(actual, key)
		}
		slices.Sort(actual)
		wanted := readerKeySet(t, test.file, test.method, test.anchor)
		if !reflect.DeepEqual(actual, wanted) {
			t.Errorf("%v schema keys %v; reader keys %v", test.path, actual, wanted)
		}
	}
}
