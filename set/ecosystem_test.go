package set_test

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func ecosystemFixture(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("../doc/observatory-ecosystem.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestEcosystemRecipeResolvesUnitsOverridesAndReferences(t *testing.T) {
	made, err := set.Read("ecosystem.yaml", []byte(ecosystemFixture(t)))
	if err != nil {
		t.Fatal(err)
	}
	if made.Renderer() != "gpu" || made.Complexity() != "quick" || made.Seed() != 11 {
		t.Error("scene defaults were lost")
	}
	obs, ok := made.Recipe("ridge telescope")
	if !ok || obs.Optical().Aperture.InMetres() != 8 || obs.Optical().Diameter.InMetres() != 24 {
		t.Errorf("recipe %+v", obs)
	}
	things, stands := made.Things(), made.Stands()
	if things[0].Kind() != "optical-observatory" || things[0].Complexity() != "demo" {
		t.Errorf("thing %+v", things[0])
	}
	if stands[0].Kind() != "coast-redwood" || stands[0].Complexity() != "clumps" {
		t.Errorf("stand %+v", stands[0])
	}
	if name, clearance := stands[0].Exclude(); name != "the observatory" || clearance != 40 {
		t.Errorf("exclusion %s %v", name, clearance)
	}
	mist, ok := made.Mist("grove mist")
	if !ok || mist.Extinction() != 0.006 || mist.Depth() != 20 {
		t.Error("mist properties were lost")
	}
}

func TestEcosystemSchemaRejectsUnknownFieldsUnitsAndAmbiguousRecipes(t *testing.T) {
	for _, change := range []struct{ from, to, keyword, domain string }{
		{"renderer: gpu", "renderer: gup", "enum", ""},
		{"complexity: clumps", "complexity: clumpps", "enum", ""},
		{"count: 5000", "count: 100001", "maximum", ""},
		{"unit: m^-1", "unit: kg", "const", ""},
		{"diameter: {value: 24, unit: m}", "diameter: {value: 24, unit: deg}", "const", ""},
		{"count: 5000", "count: 5000\n        unexpected: true", "additionalProperties", ""},
		{"recipe: redwoods", "recipe: redwoods\n        kind: spruce", "oneOf", ""},
		{"aperture: {value: 8, unit: m}", "aperture: {value: 24, unit: m}", "", "invalid optical observatory"},
		{"recipe: redwoods", "recipe: missing tree", "", "unknown recipe"},
		{"weather: grove mist", "weather: missing weather", "", "unknown weather"},
		{"around: the observatory", "around: missing building", "", "excludes unknown object"},
	} {
		raw := strings.Replace(ecosystemFixture(t), change.from, change.to, 1)
		if raw == ecosystemFixture(t) {
			t.Fatalf("ineffective change %s", change.from)
		}
		_, err := set.Read("bad-ecosystem.yaml", []byte(raw))
		if err == nil || !strings.Contains(err.Error(), "bad-ecosystem.yaml") {
			t.Fatalf("%s: %v", change.to, err)
		}
		var invalid *jsonschema.ValidationError
		isSchema := errors.As(err, &invalid)
		if change.keyword != "" {
			if !isSchema || !schemaKeyword(invalid, change.keyword) {
				t.Errorf("%s did not get schema %s: %v", change.to, change.keyword, err)
			}
		} else if isSchema || !strings.Contains(err.Error(), change.domain) {
			t.Errorf("%s did not get domain refusal %q: %v", change.to, change.domain, err)
		}
	}
}

// schemaKeyword proves which schema constraint refused the actual input.
func schemaKeyword(err *jsonschema.ValidationError, keyword string) bool {
	if slices.Contains(err.ErrorKind.KeywordPath(), keyword) {
		return true
	}
	for _, cause := range err.Causes {
		if schemaKeyword(cause, keyword) {
			return true
		}
	}
	return false
}

// plainFilm is the fixture's film as it is written; rigFilm is the same
// film shot through a rig: a frame, a lens, a shutter and glimpses,
// which decide the height, the exposure and how wide the picture sees.
const (
	plainFilm = "  film:\n    width: 1280\n    height: 800\n    rays: 2\n" +
		"    exposure: 1\n"
	rigFilm = "  film:\n    width: 1280\n    rays: 2\n" +
		"    print: {knee: 0.1, top: 100, source: eye}\n" +
		"    frame: {width: 36, height: 24}\n" +
		"    lens: {focal: 50, stop: 8, focus: the observatory}\n" +
		"    shutter: 0.01\n    glimpses: 1\n"
)

// the test: a scene that opts into the schema may say everything its
// reader reads of a film and a camera: a print, a whole rig, and the
// camera's eye.
func TestEcosystemSchemaTakesARigAndAnEye(t *testing.T) {
	raw := strings.Replace(ecosystemFixture(t), plainFilm, rigFilm, 1)
	raw = strings.Replace(raw, "    across the forest:\n",
		"    across the forest:\n      eye: human\n", 1)
	if !strings.Contains(raw, "lens:") || !strings.Contains(raw, "eye:") {
		t.Fatal("the fixture's film or camera moved")
	}
	if _, err := set.Read("rig.yaml", []byte(raw)); err != nil {
		t.Fatal(err)
	}
}

// the anti-test: a rig is whole or absent, and a film shot through one
// does not also say its height, exposure or width of view; a print says
// its knee, top and source; an eye is human.
func TestEcosystemSchemaRefusesHalfARigAndAStrangeEye(t *testing.T) {
	for _, change := range []struct{ from, to, keyword string }{
		{plainFilm, rigFilm + "    height: 800\n", "oneOf"},
		{plainFilm, strings.Replace(rigFilm, "    shutter: 0.01\n", "",
			1), "oneOf"},
		{plainFilm, plainFilm + "    lens: {focal: 50, stop: 8, " +
			"focus: the observatory}\n", "oneOf"},
		{plainFilm, strings.Replace(rigFilm, ", source: eye", "", 1),
			"required"},
		{"    across the forest:\n",
			"    across the forest:\n      eye: eagle\n", "const"},
	} {
		raw := strings.Replace(ecosystemFixture(t), change.from, change.to, 1)
		if raw == ecosystemFixture(t) {
			t.Fatalf("ineffective change to %q", change.to)
		}
		_, err := set.Read("bad-rig.yaml", []byte(raw))
		var invalid *jsonschema.ValidationError
		if !errors.As(err, &invalid) || !schemaKeyword(invalid,
			change.keyword) {
			t.Errorf("%q did not get schema %s: %v", change.to,
				change.keyword, err)
		}
	}
}

// The pieces a scene that opts into the schema may now say, as the
// reader reads them: a stand within a wedge of an object, a set's bounds
// and its texts, and a hud.
const (
	forestCount = "        count: 5000\n"
	withinWedge = forestCount + "        within: {of: the observatory, " +
		"from: 0, to: 90, near: 10, far: 500}\n"
	setThings = "  set:\n    things:\n"
	setPieces = "  set:\n" +
		"    bounded_box: {size_x: 1000, size_y: 1000, size_z: 500}\n" +
		"    texts:\n      the sign:\n        words: firang\n" +
		"        font: go-bold\n        size: 2\n        depth: 0.2\n" +
		"        colour: {red: 1, green: 0.5, blue: 0}\n" +
		"    things:\n"
	filmLine = "  film:\n"
	hudBlock = "  hud:\n    theme: themes/dusk.css\n    elements:\n" +
		"      title:\n        kind: text\n        words: firang\n" +
		"        ink: fg\n        across: 0.1\n        down: 0.1\n" +
		filmLine
)

// the test: a stand within a wedge, a set's bounds and texts, and a hud
// are read from a scene that opts into the schema.
func TestEcosystemSchemaTakesAWedgeTextsBoundsAndAHud(t *testing.T) {
	raw := strings.Replace(ecosystemFixture(t), forestCount, withinWedge, 1)
	raw = strings.Replace(raw, setThings, setPieces, 1)
	raw = strings.Replace(raw, filmLine, hudBlock, 1)
	for _, said := range []string{"within:", "texts:", "hud:"} {
		if !strings.Contains(raw, said) {
			t.Fatalf("the fixture moved; no %s", said)
		}
	}
	if _, err := set.Read("pieces.yaml", []byte(raw)); err != nil {
		t.Fatal(err)
	}
}

// the anti-test: each new piece refuses, by the schema, what its reader
// refuses: a wedge without its far edge, an asset of a kind there is
// none of, a colour without its blue, a hud without its theme, a text
// element with both words and a count, a count without its format, and
// a bar without its value.
func TestEcosystemSchemaRefusesHalfSaidPieces(t *testing.T) {
	element := "      title:\n        kind: text\n        words: firang\n"
	for _, change := range []struct{ from, to, keyword string }{
		{forestCount, strings.Replace(withinWedge, ", far: 500", "", 1),
			"required"},
		{setThings, "  set:\n    assets:\n      barn: {name: the barn}\n" +
			"    things:\n", "propertyNames"},
		{setThings, strings.Replace(setPieces, ", blue: 0", "", 1),
			"required"},
		{filmLine, strings.Replace(hudBlock,
			"    theme: themes/dusk.css\n", "", 1), "required"},
		{filmLine, strings.Replace(hudBlock, element, element+
			"        count: 3\n        format: \"%d\"\n", 1), "oneOf"},
		{filmLine, strings.Replace(hudBlock, "        words: firang\n",
			"        count: 3\n", 1), "dependentRequired"},
		{filmLine, strings.Replace(hudBlock, "kind: text", "kind: bar",
			1), "required"},
	} {
		raw := strings.Replace(ecosystemFixture(t), change.from, change.to, 1)
		if raw == ecosystemFixture(t) {
			t.Fatalf("ineffective change to %q", change.to)
		}
		_, err := set.Read("bad-pieces.yaml", []byte(raw))
		var invalid *jsonschema.ValidationError
		if !errors.As(err, &invalid) || !schemaKeyword(invalid,
			change.keyword) {
			t.Errorf("%q did not get schema %s: %v", change.to,
				change.keyword, err)
		}
	}
}

// declared is a set from sets/, as written, opting into the schema.
func declared(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile("../sets/" + name + ".yaml")
	if err != nil {
		t.Fatal(err)
	}
	return "\"$schema\": " + set.EcosystemSchema + "\n" + string(raw)
}

// the test: the sets already written read the same when they opt into
// the schema: the world sets, which are space and no terrain, and the
// ground sets, so the schema is held to what scenes really say.
func TestEcosystemSchemaTakesTheSetsAsWritten(t *testing.T) {
	for _, name := range []string{"shoebox", "widow", "widow-smoke",
		"widow-tail", "kilauea-iki", "melancholia",
		"planetary-turbidity"} {
		raw := []byte(declared(t, name))
		if _, err := set.Read(name+".yaml", raw); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// the anti-test: a world refuses, by the schema, what its reader
// refuses: a sky of a gradient and stars both, a colour both linear and
// a swatch, a body of no template there is, a frame with no place, and a
// motion with no ease.
func TestEcosystemSchemaRefusesAWorldHalfSaid(t *testing.T) {
	for _, change := range []struct{ from, to, keyword string }{
		{"      stars: {seed: 29,",
			"      horizon: {swatch: horizon}\n      stars: {seed: 29,",
			"oneOf"},
		{"glow: {swatch: tungsten}",
			"glow: {swatch: tungsten, srgb_linear: [1, 1, 1]}", "oneOf"},
		{"template: pulsar", "template: krypton", "enum"},
		{"- {percent: 0, at: [64.00, 0, 0]}", "- {percent: 0}",
			"required"},
		{"        ease: linear\n", "", "required"},
	} {
		raw := declared(t, "widow")
		changed := strings.Replace(raw, change.from, change.to, 1)
		if changed == raw {
			t.Fatalf("ineffective change to %q", change.to)
		}
		_, err := set.Read("bad-world.yaml", []byte(changed))
		var invalid *jsonschema.ValidationError
		if !errors.As(err, &invalid) || !schemaKeyword(invalid,
			change.keyword) {
			t.Errorf("%q did not get schema %s: %v", change.to,
				change.keyword, err)
		}
	}
}

// the anti-test: a planet refuses, by the schema, what its reader
// refuses: a real planet, which slayer cannot draw, a gas not built, a
// ring of something slayer does not know, and x-rays of anything but
// jupiter's.
func TestEcosystemSchemaRefusesAPlanetHalfSaid(t *testing.T) {
	for _, change := range []struct{ from, to, keyword string }{
		{"made_up: true", "made_up: false", "const"},
		{"methane: 0.49", "sodium: 0.49", "propertyNames"},
		{"ice: 0.6", "granite: 0.6", "propertyNames"},
	} {
		raw := declared(t, "planetary-turbidity")
		changed := strings.Replace(raw, change.from, change.to, 1)
		if changed == raw {
			t.Fatalf("ineffective change to %q", change.to)
		}
		_, err := set.Read("bad-planet.yaml", []byte(changed))
		var invalid *jsonschema.ValidationError
		if !errors.As(err, &invalid) || !schemaKeyword(invalid,
			change.keyword) {
			t.Errorf("%q did not get schema %s: %v", change.to,
				change.keyword, err)
		}
	}
}

func TestEcosystemSchemaRefusesASecondDocument(t *testing.T) {
	raw := ecosystemFixture(t) + "\n---\nscene: {name: hidden}\n"
	if _, err := set.Read("two.yaml", []byte(raw)); err == nil {
		t.Error("accepted a hidden second scene")
	}
}
