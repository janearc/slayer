package set_test

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// shoreCover is a cover of a rock ground with sand laid low and gentle:
// the lines that say it under a terrain.
const shoreCover = `    cover:
      level: 0
      layers:
        - name: rock
          reflectance: [0.19, 0.16, 0.12]
          source: a test's own
        - name: sand
          reflectance: [0.69, 0.55, 0.26]
          source: a test's own
          height: {until: 6, blend: 4}
          slope: {from: 0.1, until: 0.5, blend: 0.2}
`

// A terrain that says its cover gives the sea's level and each surface in
// order, with its reflectance, its source and its rules.
func TestTerrainSaysItsCover(t *testing.T) {
	made, err := legacyLaser(t, byMetres, byMetres+shoreCover)
	if err != nil {
		t.Fatal(err)
	}
	ground, _ := made.Ground()
	level, layers, said := ground.Cover()
	if !said || level != 0 || len(layers) != 2 {
		t.Fatalf("level %v, %d layers, said %v", level, len(layers), said)
	}
	sand := layers[1]
	from, until, blend, height := sand.Height()
	// picked: the numbers the sand's height line says
	if !height || !math.IsInf(from, -1) || until != 6 || blend != 4 { // picked
		t.Errorf("height %v %v %v, said %v", from, until, blend, height)
	}
	from, until, blend, slope := sand.Slope()
	if !slope || from != 0.1 || until != 0.5 || blend != 0.2 { // picked
		t.Errorf("slope %v %v %v, said %v", from, until, blend, slope)
	}
	if sand.Name() != "sand" || sand.Source() != "a test's own" ||
		sand.Reflectance() != [3]float64{0.69, 0.55, 0.26} { // picked
		t.Errorf("the sand is %v %v %v", sand.Name(), sand.Source(),
			sand.Reflectance())
	}
}

// Anti: the ground under the rest says no height and no slope, and the
// accessors hold every number for a rule not said.
func TestCoverGroundUnderSaysNoRules(t *testing.T) {
	made, err := legacyLaser(t, byMetres, byMetres+shoreCover)
	if err != nil {
		t.Fatal(err)
	}
	ground, _ := made.Ground()
	_, layers, _ := ground.Cover()
	from, until, _, said := layers[0].Height()
	if said || !math.IsInf(from, -1) || !math.IsInf(until, 1) {
		t.Errorf("the ground under says a height: %v %v %v", from, until,
			said)
	}
}

// Anti: a terrain that says no cover says none.
func TestTerrainWithoutCoverSaysNone(t *testing.T) {
	made, err := legacyLaser(t, byMetres, byMetres)
	if err != nil {
		t.Fatal(err)
	}
	ground, _ := made.Ground()
	if level, layers, said := ground.Cover(); said || layers != nil ||
		level != 0 {
		t.Errorf("cover %v %v %v", level, layers, said)
	}
}

// A cover that leaves out what it needs, or says what cannot be, is refused
// with the reason, and no picture is drawn from a half-said rule.
func TestTerrainRefusesACoverThatCannotBeRead(t *testing.T) {
	for name, edit := range map[string]struct{ from, to, want string }{
		"no level": {"      level: 0\n", "", "says its level"},
		"no layers": {shoreCover, "    cover:\n      level: 0\n",
			"this has no layers"},
		"empty layers": {shoreCover, "    cover:\n      level: 0\n" +
			"      layers: []\n", "list of surfaces"},
		"layers that is a word": {shoreCover, "    cover:\n      level: 0\n" +
			"      layers: rock\n", "list of surfaces"},
		"no source": {"          source: a test's own\n          height",
			"          height", "this has no source"},
		"first with a rule": {"          source: a test's own\n        - name: sand",
			"          source: a test's own\n          height: " +
				"{until: 1, blend: 1}\n        - name: sand", "no height"},
		"later with no rule": {"          height: {until: 6, blend: 4}\n" +
			"          slope: {from: 0.1, until: 0.5, blend: 0.2}\n", "",
			"hides all"},
		"no blend": {"{until: 6, blend: 4}", "{until: 6}", "blend"},
		"no end":   {"{until: 6, blend: 4}", "{blend: 4}", "from, until"},
		"ends the wrong way": {"{until: 6, blend: 4}",
			"{from: 9, until: 6, blend: 4}", "not under"},
		"reflectance over one": {"[0.69, 0.55, 0.26]", "[1.69, 0.55, 0.26]",
			"reflectance"},
		"a slope under none": {"{from: 0.1, until: 0.5, blend: 0.2}",
			"{from: -1, until: 0.5, blend: 0.2}", "from takes a number"},
		"a key it does not take": {"level: 0\n", "level: 0\n      depth: 3\n",
			"depth"},
	} {
		if !strings.Contains(shoreCover, edit.from) {
			t.Fatalf("%s: the cover has no %q to change", name, edit.from)
		}
		text := strings.Replace(shoreCover, edit.from, edit.to, 1)
		_, err := legacyLaser(t, byMetres, byMetres+text)
		mustHold(t, name, err, edit.want)
	}
}

// seaSaid is a terrain's own sea, at two metres, in place of a cover's level.
const seaSaid = "    sea: {value: 2, unit: m}\n"

// A terrain that says its sea is the owner of the level: a cover that
// leaves its level out takes the terrain's, and one that says the same
// number is read.
func TestCoverTakesItsLevelFromTheTerrainsSea(t *testing.T) {
	left := strings.Replace(shoreCover, "      level: 0\n", "", 1)
	same := strings.Replace(shoreCover, "level: 0", "level: 2", 1)
	for name, text := range map[string]string{"left out": left, "same": same} {
		made, err := legacyLaser(t, byMetres, byMetres+seaSaid+text)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		ground, _ := made.Ground()
		if level, _, said := ground.Cover(); !said || level != 2 { // picked
			t.Errorf("%s: level %v, said %v, want the terrain's 2", name,
				level, said)
		}
	}
}

// Anti: a cover that says a level other than the terrain's sea is refused,
// and the words carry both numbers.
func TestCoverRefusesALevelThatIsNotTheTerrainsSea(t *testing.T) {
	_, err := legacyLaser(t, byMetres, byMetres+seaSaid+shoreCover)
	mustHold(t, "differs", err, "cover says the sea stands at 0 m")
	mustHold(t, "differs", err, "terrain's sea says 2 m")
}

// honopuSet is a set file that says a cover and an ocean, as it is on disk.
func honopuSet(t *testing.T, change func(string) string) error {
	t.Helper()
	text, err := os.ReadFile("../sets/honopu.yaml")
	if err != nil {
		t.Fatal(err)
	}
	changed := change(string(text))
	_, err = set.Read("honopu.yaml", []byte(changed))
	return err
}

// An ocean whose middle stands at another height than the cover's level is
// refused with both numbers; the set as it is, with both at nought, is read.
func TestCoverRefusesAnOceanAtAnotherLevel(t *testing.T) {
	const wasLevel, becomes = "middle: [431000, 2451500, 0]",
		"middle: [431000, 2451500, 5]" // picked: a middle five metres up
	err := honopuSet(t, func(text string) string { return text })
	if err != nil {
		t.Fatalf("the set as it is was refused: %v", err)
	}
	err = honopuSet(t, func(text string) string {
		return strings.Replace(text, wasLevel, becomes, 1)
	})
	mustHold(t, "ocean up", err, "ocean's middle stands at 5 m")
	mustHold(t, "ocean up", err, "sea at 0 m")
}

// A terrain may say its reflectance or its cover, not both: the cover's
// first surface is the ground's colour.
func TestTerrainRefusesAReflectanceAndACover(t *testing.T) {
	_, err := legacyLaser(t, byMetres, byMetres+shoreCover+
		"    reflectance: [0.1, 0.1, 0.1]\n")
	mustHold(t, "both", err, "reflectance or its cover")
}
