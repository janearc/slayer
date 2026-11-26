package session

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// undrawnHead is the start of a small set that names the version 2 schema,
// before its film, its cameras and what the test adds.
const undrawnHead = `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: t
  seed: 1
  light: ambient
`

// undrawnFilm and undrawnCameras are the film and the camera a small set
// says, unless a row says its own.
const (
	undrawnFilm    = "  film: {width: 64, height: 36, rays: 1, exposure: 1}\n"
	undrawnCameras = "  cameras:\n    only: {title: only, orientation: {look_at: [0, 0, 2]}, location: {at: [0, -5, 2]}}\n"
)

// undrawnInstrument is the lines of a thing's eye, which a camera may see
// through.
const undrawnInstrument = "        eye:\n          pupil: {value: 1, unit: m}\n" +
	"          focal: {value: 2, unit: m}\n          models: diffraction-limited pupil\n"

// undrawnThing is one thing under set, of a kind, with the lines after its seed.
func undrawnThing(name, kind, more string) string {
	return "      " + name + ":\n        kind: " + kind + "\n        seed: 1\n" + more
}

// undrawnSet is a space scene's things, each as undrawnThing writes it.
func undrawnSet(things ...string) string {
	return "  set:\n    things:\n" + strings.Join(things, "")
}

// undrawnRow is one feature the reader keeps and no stage draws: the set
// that says it, and the key to look for in that set to say where.
type undrawnRow struct {
	feature, key  string
	film, cameras string
	scene         string
}

// undrawnRows are every feature, each with a small set that holds it. The
// set of an attribute of a kind says the kind, which is a feature too, so
// a row says its own and the test counts every other as drawn.
func undrawnRows() []undrawnRow {
	origin := "        place: {at: origin}\n"
	orbital := func(more string) string {
		return undrawnSet(undrawnThing("ring", "culture-orbital", origin+more))
	}
	nebula := func(more string) string {
		return undrawnSet(undrawnThing("gas", "nebula", origin+more))
	}
	moving := "  cameras:\n    cam:\n      title: c\n      moves:\n        - look: {object: sun}\n          duration: {value: 1, unit: s}\n"
	sun := undrawnThing("sun", "star", origin)
	return []undrawnRow{
		{"scene.setting", "setting:", "", "", "  setting: space\n"},
		{"scene.date", "date:", "", "", "  setting: space\n  date: \"2026-06-21\"\n"},
		{"scene.frame", "frame:", "", "", "  frame: {at: sun-earth-l2}\n"},
		{"scene.anchor", "anchor:", "", "", "  anchor:\n    lat: 1\n    lon: 2\n    elevation: {value: 3, unit: m}\n    datum: wgs84\n"},
		{"scene.globe", "globe:", "", "", "  globe:\n    planet: earth\n    terrain:\n      near: {from: a}\n      beyond: {from: b}\n    detail: by-size\n"},
		{"scene.sky", "sky:", "", "", "  sky:\n    stars:\n      from: gaia\n      toward: {ra: 1, dec: 1}\n      radius: {value: 1, unit: deg}\n      faintest: 1\n"},
		{"film.fps", "fps:", "  film: {width: 64, height: 36, rays: 1, exposure: 1, fps: 24}\n", "", ""},
		{"thing.kind", "kind: point", "", "", undrawnSet(undrawnThing("a", "point", "        east: 1\n        north: 2\n"))},
		{"thing.place", "place:", "", "", undrawnSet(undrawnThing("a", "cobble", origin))},
		{"thing.around", "around:", "", "", undrawnSet(undrawnThing("a", "cobble", "        east: 1\n        north: 2\n"), undrawnThing("b", "cobble", "        around: a\n"))},
		{"thing.star", "star: sun", "", "", undrawnSet(undrawnThing("a", "star", origin+"        star: sun\n"))},
		{"thing.limb", "limb:", "", "", undrawnSet(undrawnThing("a", "star", origin+"        limb: darkened\n"))},
		{"thing.corona", "corona:", "", "", undrawnSet(undrawnThing("a", "star", origin+"        corona: {reach: 4, streamers: 7, seed: 3}\n"))},
		{"thing.panels", "panels:", "", "", undrawnSet(undrawnThing("a", "lantern", origin+"        panels: baseline\n"))},
		{"thing.turning", "turning:", "", "", undrawnSet(undrawnThing("a", "lantern", origin+"        turning: true\n"))},
		{"thing.radius", "radius:", "", "", orbital("        radius: {value: 5, unit: km}\n")},
		{"thing.width", "width: {value", "", "", orbital("        width: {value: 5, unit: km}\n")},
		{"thing.walls", "walls:", "", "", orbital("        walls: {value: 5, unit: km}\n")},
		{"thing.lean", "lean:", "", "", orbital("        lean: {value: 45, unit: deg}\n")},
		{"thing.sense", "sense:", "", "", orbital("        sense: -1\n")},
		{"thing.lights", "lights:", "", "", orbital("        lights: {kelvin: {from: 1900, to: 3200}}\n")},
		{"thing.thickness", "thickness:", "", "", nebula("        thickness: 4.5e-5\n")},
		{"thing.neon", "neon:", "", "", nebula("        neon: 0\n")},
		{"thing.argon", "argon:", "", "", nebula("        argon: 0\n")},
		{"thing.glow", "glow:", "", "", nebula("        glow: 0.2\n")},
		{"thing.orbit", "orbit:", "", "", undrawnSet(undrawnThing("base", "cobble", "        east: 1\n        north: 2\n"), undrawnThing("a", "hubble", "        orbit:\n          height: {value: 500, unit: km}\n          inclination: 97.4\n          over: base\n"))},
		{"thing.attitude", "attitude:", "", "", undrawnSet(undrawnThing("a", "hubble", origin+"        attitude: {point: nadir}\n"))},
		{"thing.eye", "eye:", "", "", undrawnSet(undrawnThing("a", "hubble", origin+undrawnInstrument))},
		{"camera.duration", "duration:", "", "  cameras:\n    cam:\n      title: c\n      orientation: {look_at: [0, 0, 2]}\n      location: {at: [0, -5, 2]}\n      duration: {value: 6, unit: s}\n", ""},
		{"camera.moves", "moves:", "", moving, undrawnSet(sun)},
		{"camera.eye", "eye: {instrument", "", "  cameras:\n    cam:\n      title: c\n      eye: {instrument: a}\n      moves:\n        - look: {object: a}\n          duration: {value: 1, unit: s}\n", undrawnSet(undrawnThing("a", "cobble", "        east: 1\n        north: 2\n"+undrawnInstrument))},
		{"camera.location.relative", "relative:", "", "  cameras:\n    cam:\n      title: c\n      orientation: {point_to: a}\n      location:\n        relative: {object: a, distance: {value: 1, unit: km}}\n", undrawnSet(undrawnThing("a", "cobble", "        east: 1\n        north: 2\n"))},
		{"atmosphere.extent", "extent: world", "", "", "  globe:\n    planet: earth\n    terrain:\n      near: {from: a}\n      beyond: {from: b}\n    detail: by-size\n  weather:\n    low:\n      kind: prescribed-mist\n      extinction: {value: 0.018, unit: m^-1}\n      depth: {value: 130, unit: m}\n      coverage: 0.7\n      seed: 23\n  atmosphere:\n    layers:\n      deck:\n        weather: low\n        extent: world\n        albedo: 0.95\n"},
	}
}

// The table holds every feature the reader can keep: no more and no
// fewer than the names the reader's source gives to versionTwo and note.
func TestTheTableHoldsEveryFeatureTheReaderKeeps(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "set", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	kept := map[string]bool{}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			name, ok := call.Fun.(*ast.SelectorExpr)
			text, literal := call.Args[0].(*ast.BasicLit)
			if ok && literal && (name.Sel.Name == "versionTwo" ||
				name.Sel.Name == "note") {
				feature, _ := strconv.Unquote(text.Value)
				kept[feature] = true
			}
			return true
		})
	}
	var listed []string
	for _, row := range undrawnRows() {
		listed = append(listed, row.feature)
	}
	var found []string
	for feature := range kept {
		found = append(found, feature)
	}
	slices.Sort(listed)
	slices.Sort(found)
	if !slices.Equal(listed, found) {
		t.Errorf("the table holds %v; the reader keeps %v", listed, found)
	}
}
