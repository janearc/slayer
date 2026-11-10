package set_test

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// built is the small built world's set file, as written.
func built(t *testing.T) string {
	t.Helper()
	source, err := os.ReadFile("testdata/built.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return string(source)
}

// builtRead is the small built world, read.
func builtRead(t *testing.T) set.World {
	t.Helper()
	made, err := set.Read("built.yaml", []byte(built(t)))
	if err != nil {
		t.Fatal(err)
	}
	world, has := made.World()
	if !has {
		t.Fatal("the built set says no world")
	}
	return world
}

// linear is a colour's three parts, for comparing.
func linear(colour set.Colour) [3]float64 {
	red, green, blue := colour.Linear()
	return [3]float64{red, green, blue}
}

// a world's materials are each as written: a colour, a glow, or both,
// and which it has not, it says it has not.
func TestAWorldsMaterialsAreWhatTheySay(t *testing.T) {
	materials := builtRead(t).Materials()
	if len(materials) != 3 || materials[0].Name() != "grass" ||
		materials[1].Name() != "lit-window" ||
		materials[2].Name() != "ember" {
		t.Fatalf("materials %+v", materials)
	}
	grass, has := materials[0].Colour()
	if !has || linear(grass) != [3]float64{0.05, 0.06, 0.04} {
		t.Errorf("grass is %v, %v", grass, has)
	}
	if heat, luminance, glows := materials[0].Glow(); glows || heat != 0 ||
		luminance != 0 {
		t.Errorf("grass glows %v at %v K, %v", luminance, heat, glows)
	}
	if _, has := materials[1].Colour(); has {
		t.Error("the lit window has a colour")
	}
	heat, luminance, glows := materials[1].Glow()
	if !glows || heat != 2700 || luminance != 0.45 {
		t.Errorf("the lit window glows %v at %v K, %v", luminance, heat,
			glows)
	}
	ember, has := materials[2].Colour()
	heat, luminance, glows = materials[2].Glow()
	if !has || linear(ember) != [3]float64{0.3, 0.1, 0} || !glows ||
		heat != 1200 || luminance != 0.02 {
		t.Errorf("the ember is %v and glows %v at %v K", ember, luminance,
			heat)
	}
}

// a world's places come back in the world's space: a map's east is the
// world's x, its north the world's minus z, its up the world's y. a
// size is turned the same way and is never negative.
func TestAWorldsPlacesAreTheWorlds(t *testing.T) {
	world := builtRead(t)
	boxes := world.Boxes()
	if len(boxes) != 2 || boxes[0].Name() != "lawn" ||
		boxes[0].Material() != "grass" ||
		boxes[0].Middle() != maths.V(1, 3, -2) ||
		boxes[0].Size() != maths.V(4, 6, 5) ||
		boxes[1].Size() != maths.V(1, 0.8, 0.06) {
		t.Errorf("boxes %+v", boxes)
	}
	lamps := world.Lamps()
	heat, strength := lamps[0].Shine()
	if len(lamps) != 1 || lamps[0].Name() != "porch" ||
		lamps[0].At() != maths.V(-2, 7.5, -3) || heat != 2700 ||
		strength != 0.35 || lamps[0].Radius() != 0.05 {
		t.Errorf("lamps %+v", lamps)
	}
	bodies := world.Bodies()
	tilt, tilted := bodies[0].Tilt()
	shown, saysAurora := bodies[0].Aurora()
	radius, sized := bodies[0].Radius()
	if len(bodies) != 1 || bodies[0].Name() != "saturn" ||
		bodies[0].Template() != "saturn" ||
		bodies[0].Middle() != maths.V(-94576000, 6.4, -133982700) ||
		tilt != 50 || !tilted || shown || !saysAurora || sized ||
		radius != 0 || bodies[0].Detail() != 24 {
		t.Errorf("bodies %+v", bodies)
	}
	suns := world.Suns()
	if len(suns) != 1 || suns[0].Name() != "sun" ||
		suns[0].Toward() != maths.V(0.5762, -0.04, 0.8163) ||
		linear(suns[0].Colour()) != [3]float64{0.35, 0.34, 0.32} {
		t.Errorf("suns %+v", suns)
	}
}

// a world's light from everywhere is as written: its flood, and its sky
// at the horizon, the zenith and the ground.
func TestAWorldsFloodAndSkyAreWhatTheySay(t *testing.T) {
	world := builtRead(t)
	floods := world.Floods()
	if len(floods) != 1 || floods[0].Name() != "night" ||
		linear(floods[0].Colour()) != [3]float64{0.01, 0.013, 0.025} ||
		floods[0].Strength() != 2 {
		t.Errorf("floods %+v", floods)
	}
	sky, has := world.Sky()
	if stars, starry := sky.Starry(); starry || stars != (set.Stars{}) {
		t.Error("a gradient sky is stars")
	}
	horizon, zenith, ground := sky.Light()
	if !has || linear(horizon) != [3]float64{0.05, 0.07, 0.12} ||
		linear(zenith) != [3]float64{0.006, 0.01, 0.03} ||
		linear(ground) != [3]float64{0.01, 0.01, 0.012} {
		t.Errorf("sky %+v, %v", sky, has)
	}
}

// what a world hands back is its own copy: changing a list leaves the
// world's as it was.
func TestAWorldsListsAreCopies(t *testing.T) {
	world := builtRead(t)
	world.Materials()[0] = set.Material{}
	world.Boxes()[0] = set.Box{}
	world.Lamps()[0] = set.Lamp{}
	world.Floods()[0] = set.Flood{}
	world.Bodies()[0] = set.Body{}
	world.Suns()[0] = set.Sunlight{}
	if world.Materials()[0].Name() != "grass" ||
		world.Boxes()[0].Name() != "lawn" ||
		world.Lamps()[0].Name() != "porch" ||
		world.Floods()[0].Name() != "night" ||
		world.Bodies()[0].Name() != "saturn" ||
		world.Suns()[0].Name() != "sun" {
		t.Error("changing a list handed back changed the world's")
	}
}

// a set that says no world has none, a world with no sky says so, and a
// film with no print says that.
func TestWhatIsNotSaidIsNotThere(t *testing.T) {
	made, err := set.Read("shoebox.yaml", []byte(shoebox(t)))
	if err != nil {
		t.Fatal(err)
	}
	if _, has := made.World(); has {
		t.Error("the shoebox says a world")
	}
	film, _ := made.Film()
	if knee, top, has := film.Print(); has || knee != 0 || top != 0 {
		t.Errorf("the shoebox's film says a print, %v and %v", knee, top)
	}
	written := built(t)
	cut := strings.Index(written, "    sky:\n")
	end := strings.Index(written, "    bodies:")
	world, err := set.Read("built.yaml", []byte(written[:cut]+
		written[end:]))
	if err != nil {
		t.Fatal(err)
	}
	inside, _ := world.World()
	if _, has := inside.Sky(); has {
		t.Error("a world with no sky says one")
	}
}

// a film's print is its knee and its top, as written.
func TestAFilmsPrintIsWhatItSays(t *testing.T) {
	made, err := set.Read("built.yaml", []byte(built(t)))
	if err != nil {
		t.Fatal(err)
	}
	film, _ := made.Film()
	knee, top, has := film.Print()
	if !has || knee != 0.7 || top != 8 || film.Wide() != 60 ||
		film.Exposure() != 2 {
		t.Errorf("print %v %v %v", knee, top, has)
	}
}

// a camera that stands at a place stands there, in the world's space,
// facing the place it looks at with the world's up for its up.
func TestAStandingCameraLooksWhereItSays(t *testing.T) {
	made, err := set.Read("built.yaml", []byte(built(t)))
	if err != nil {
		t.Fatal(err)
	}
	nowhere := func(string) (maths.Vec, bool) { return maths.Vec{}, false }
	shot, err := made.Shot("front", nowhere)
	if err != nil {
		t.Fatal(err)
	}
	eye, look := maths.V(10, 2, 20), maths.V(0, 3, 0)
	want := maths.Subspace{Origin: eye,
		Orientation: maths.LookAtUp(eye, look, maths.V(0, 1, 0))}
	if shot.Name() != "front" || shot.Title() != "the front" ||
		shot.Eye() != want {
		t.Errorf("shot %+v, wanted the eye %+v", shot, want)
	}
	shots, err := made.Shots(nowhere)
	if err != nil || len(shots) != 1 || shots[0] != shot {
		t.Errorf("shots %+v, %v", shots, err)
	}
}

// facing is how the built set's front camera faces, looking at a place
// in the set's metres.
func facing(t *testing.T, lookAt string) maths.Orientation {
	t.Helper()
	written := strings.Replace(built(t), "look_at: [0, 0, 3]",
		"look_at: "+lookAt, 1)
	made, err := set.Read("built.yaml", []byte(written))
	if err != nil {
		t.Fatal(err)
	}
	shot, err := made.Shot("front", nil)
	if err != nil {
		t.Fatal(err)
	}
	return shot.Eye().Orientation
}

// a camera looking steeply up, but not within a hair of straight up,
// keeps the world's up for its up, and its picture the right way up.
func TestACameraLookingSteeplyUpKeepsTheWorldsUp(t *testing.T) {
	eye, look := maths.V(10, 2, 20), maths.V(10, 12, 10)
	want := maths.LookAtUp(eye, look, maths.V(0, 1, 0))
	heading := maths.LookAtUp(eye, look, maths.V(0, 0, -1))
	if got := facing(t, "[10, -10, 12]"); got != want || got == heading {
		t.Errorf("faces %+v, wanted %+v", got, want)
	}
}

// a camera that looks nearly straight down takes its heading for its
// up, as a camera standing off an asset does, and not the turn the
// world's up would give it there.
func TestACameraLookingNearlyDownTakesItsHeadingForUp(t *testing.T) {
	eye, look := maths.V(10, 2, 20), maths.V(10.00001, -50, 20)
	want := maths.LookAtUp(eye, look, maths.V(1, 0, 0))
	worlds := maths.LookAtUp(eye, look, maths.V(0, 1, 0))
	if got := facing(t, "[10.00001, -20, -50]"); got != want ||
		got == worlds {
		t.Errorf("faces %+v, wanted %+v, not %+v", got, want, worlds)
	}
}

// a camera is found by its name, and one the set has not is refused; a
// camera that cannot be made says which.
func TestAShotIsFoundByItsName(t *testing.T) {
	made, err := set.Read("shoebox.yaml", []byte(shoebox(t)))
	if err != nil {
		t.Fatal(err)
	}
	box := func(asset string) (maths.Vec, bool) {
		return maths.V(-1, 0, 0), asset == "the box"
	}
	shot, err := made.Shot("shoot_1", box)
	if err != nil || shot.Name() != "shoot_1" {
		t.Errorf("shoot_1: %+v, %v", shot, err)
	}
	if cameras := made.Cameras(); !slices.Equal(cameras,
		[]string{"shoot_1"}) {
		t.Errorf("the cameras are %v", cameras)
	}
	if none, _ := set.Read("none.yaml", []byte("scene:\n  name: x\n")); len(none.Cameras()) != 0 {
		t.Errorf("a set with no cameras has %v", none.Cameras())
	}
	if _, err := made.Shot("shoot_2", box); err == nil ||
		err.Error() != "the set has no camera shoot_2" {
		t.Errorf("a camera the set has not: %v", err)
	}
	nowhere := func(string) (maths.Vec, bool) { return maths.Vec{}, false }
	if _, err := made.Shot("shoot_1", nowhere); err == nil ||
		!strings.HasPrefix(err.Error(), "camera shoot_1: stands off") {
		t.Errorf("a camera off an asset nowhere: %v", err)
	}
	for _, look := range []string{"[10, -20, 2]", "[10, -20, 40]"} {
		written := strings.Replace(built(t), "look_at: [0, 0, 3]",
			"look_at: "+look, 1)
		made, err := set.Read("built.yaml", []byte(written))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := made.Shot("front", nowhere); err == nil ||
			!strings.Contains(err.Error(), "camera front: looks "+
				"straight up or down, or at where it stands") {
			t.Errorf("looking at %s: %v", look, err)
		}
	}
}

// a built world's mistakes are refused where they are written.
func TestABuiltWorldsMistakesAreRefused(t *testing.T) {
	for _, mistake := range []struct{ was, now, said string }{
		{"material: grass", "material: moss",
			`box lawn is made of "moss", and the world holds no material`},
		{"size: [4, 5, 6]", "size: [4, 0, 6]",
			"built.yaml:39:19: size takes a number from 5e-324"},
		{"size: [4, 5, 6]", "size: [4, 5]",
			"built.yaml:39:15: size takes three numbers, as [x, y, z]"},
		{"middle: [1, 2, 3]", "middle: 1",
			"middle takes three numbers"},
		{"middle: [1, 2, 3]", "middle: [1, 2, 1e14]",
			"middle takes a number from -1e+13 to 1e+13"},
		{"        size: [4, 5, 6]\n", "",
			"built.yaml:37:9: a box says [material middle size source], " +
				"all of them, and this has no size"},
		{"glow: {temperature: 2700, luminance: 0.45}",
			"glow: {temperature: 2700}",
			"a glow says [temperature luminance], all of them, and this " +
				"has no luminance"},
		{"glow: {temperature: 2700, luminance: 0.45}",
			"glow: {temperature: 0, luminance: 0.45}",
			"temperature takes a number from 1 to 1e+06"},
		{"glow: {temperature: 2700, luminance: 0.45}",
			"lit: {temperature: 2700, luminance: 0.45}",
			"lit is not a key here; this block takes [colour glow source]"},
		{"        glow: {temperature: 2700, luminance: 0.45}\n", "",
			"material lit-window has no colour and no glow"},
		{"        source: \"a thermocouple, 2026-10-05\"\n", "",
			"a material says [source], all of them, and this has no source"},
		{"source: \"a thermocouple, 2026-10-05\"", "source: guess",
			`built.yaml:34:17: source takes eye or a quoted measurement, ` +
				`not "guess"`},
		{"source: \"a thermocouple, 2026-10-05\"", "source: \"\"",
			`source takes eye or a quoted measurement, not ""`},
		{"source: \"a thermocouple, 2026-10-05\"", "source: [eye]",
			`source takes eye or a quoted measurement, not ""`},
		{"colour: {srgb_linear: [0.05, 0.06, 0.04]}",
			"colour: {srgb_linear: [0.05, -0.06, 0.04]}",
			"srgb_linear takes a number from 0 to 1e+12"},
		{"colour: {srgb_linear: [0.05, 0.06, 0.04]}", "colour: {}",
			"a colour says srgb_linear or swatch, one of the two"},
		{"colour: {srgb_linear: [0.05, 0.06, 0.04]}",
			"colour: {srgb_linear: [0.05, 0.06, 0.04], swatch: moss}",
			"a colour says srgb_linear or swatch, one of the two"},
		{"colour: {srgb_linear: [0.05, 0.06, 0.04]}",
			"colour: {swatch: mauve}", "swatch takes [blue chalk"},
		{"colour: {srgb_linear: [0.05, 0.06, 0.04]}",
			"colour: {oklab: [0.5, 0, 0]}",
			"oklab is not a key here; this block takes [srgb_linear " +
				"swatch]"},
		{"glow: {temperature: 2700, luminance: 0.45}",
			"glow: {swatch: tungsten, temperature: 2700}",
			"temperature is not a key here; this block takes [srgb_linear " +
				"swatch]"},
		{"glow: {temperature: 2700, luminance: 0.45}",
			"glow: {swatch: tungsten, srgb_linear: [1, 1, 1]}",
			"a colour says srgb_linear or swatch, one of the two"},
		{"glow: {temperature: 2700, luminance: 0.45}", "glow: 2700",
			"this takes keys under it"},
		{"radius: 0.05", "radius: -0.05",
			"radius takes a number from 0 to 1e+13"},
		{"        radius: 0.05\n", "",
			"a lamp says [at temperature strength radius source], all of " +
				"them, and this has no radius"},
		{"strength: 0.35", "strength: -1",
			"strength takes a number from 0 to 1e+12"},
		{"      source: eye\n    bodies:", "    bodies:",
			"a sky says [horizon zenith ground source], all of them"},
		{"      night:\n        colour: {srgb_linear: [0.01, 0.013, 0.025]}\n",
			"      night:\n",
			"a flood says [colour strength source], all of them, and this " +
				"has no colour"},
		{"        strength: 2\n", "",
			"a flood says [colour strength source], all of them, and this " +
				"has no strength"},
		{"template: saturn", "template: vulcan",
			"template takes [charon earth"},
		{"tilt: 50", "tilt: 181", "tilt takes a number from -180 to 180"},
		{"aurora: false", "aurora: off",
			`aurora takes true or false, not "off"`},
		{"detail: 24", "detail: 0", "detail takes a number from 1 to"},
		{"detail: 24", "detail: 2.5", "detail takes a whole number"},
		{"        detail: 24\n", "",
			"a body says [template middle detail], all of them"},
		{"toward: [0.5762, -0.8163, -0.04]", "toward: [0, 0, 0]",
			"sun sun is toward nought, nought and nought, which is no way"},
		{"toward: [0.5762, -0.8163, -0.04]", "toward: [0, 0, 1e14]",
			"toward takes a number from -1e+13 to 1e+13"},
		{"        toward: [0.5762, -0.8163, -0.04]\n", "",
			"a sun says [toward colour source], all of them"},
		{"    floods:", "    flood:",
			"flood is not a key here; this block takes [bodies boxes " +
				"floods lamps materials motions sky spheres spots suns unit " +
				"wisps]"},
		{"      top: 8", "      top: 0.5",
			"a top of 0.5 at or under a knee of 0.7"},
		{"      knee: 0.7", "      knee: -0.7", "knee takes a number from 0"},
		{"      source: eye\n  cameras:", "  cameras:",
			"a print says [knee top source], all of them"},
		{"      lit-window:", "      grass:",
			"built.yaml:28:7: grass is said twice in one block"},
		{"look_at: [0, 0, 3]", "look_at: [0, 0, 3]\n        point_to: x",
			"camera front stands at a place and looks at one, both"},
		{"look_at: [0, 0, 3]", "look_at: [0, 0, 3]\n        alpha: 3",
			"camera front stands at a place and looks at one, both"},
		{"at: [10, -20, 2]", "at: [10, -20, 2]\n        loc_relative: " +
			"{object: x}",
			"camera front stands at a place and looks at one, both"},
		{"look_at: [0, 0, 3]", "point_to: the box",
			"camera front stands at a place and looks at one, both, or " +
				"stands off an asset"},
		{"        look_at: [0, 0, 3]", "        alpha: 3",
			"camera front stands at a place and looks at one, both"},
		{"        at: [10, -20, 2]", "        loc_relative: {object: x}",
			"camera front stands at a place and looks at one, both"},
		{"look_at: [0, 0, 3]", "look_at: [0, 0]",
			"look_at takes three numbers"},
		{"at: [10, -20, 2]", "at: [10, -20, 1e14]",
			"at takes a number from -1e+13 to 1e+13"},
	} {
		if !strings.Contains(built(t), mistake.was) {
			t.Fatalf("%q is not in the file", mistake.was)
		}
		written := strings.Replace(built(t), mistake.was, mistake.now, 1)
		_, err := set.Read("built.yaml", []byte(written))
		if err == nil || !strings.Contains(err.Error(), mistake.said) {
			t.Errorf("%q: refused with %v, wanted %q", mistake.now, err,
				mistake.said)
		}
	}
}

// a source in single quotes is a measurement as one in double quotes is.
func TestASourceMayBeSingleQuoted(t *testing.T) {
	written := strings.Replace(built(t), "source: \"a thermocouple, "+
		"2026-10-05\"", "source: 'a ruler'", 1)
	if _, err := set.Read("built.yaml", []byte(written)); err != nil {
		t.Error(err)
	}
}

// a camera that says only where it stands, or only where it looks, is
// refused, as one that says one of them beside an asset is.
func TestACameraSaysBothPlacesOrNeither(t *testing.T) {
	for _, mistake := range []struct{ was, now string }{
		{"      location:\n        at: [10, -20, 2]\n", ""},
		{"      orientation:\n        look_at: [0, 0, 3]\n", ""},
	} {
		written := strings.Replace(built(t), mistake.was, mistake.now, 1)
		_, err := set.Read("built.yaml", []byte(written))
		if err == nil || !strings.Contains(err.Error(), "camera front "+
			"stands at a place and looks at one, both") {
			t.Errorf("without %q: %v", mistake.was, err)
		}
	}
}

// a world's numbers are taken to their edges and refused past them.
func TestAWorldsNumbersAreTakenToTheirEdges(t *testing.T) {
	for _, edge := range []struct{ was, at, past string }{
		{"temperature: 2700, luminance", "temperature: 1, luminance",
			"temperature: 0.999, luminance"},
		{"temperature: 2700\n", "temperature: 1\n", "temperature: 0.999\n"},
		{"temperature: 2700\n", "temperature: 1000000\n",
			"temperature: 1000001\n"},
		{"temperature: 2700, luminance", "temperature: 1000000, luminance",
			"temperature: 1000001, luminance"},
		{"luminance: 0.45", "luminance: 0", "luminance: -1e-300"},
		{"luminance: 0.45", "luminance: 1e12", "luminance: 1.0000001e12"},
		{"size: [4, 5, 6]", "size: [4, 5e-324, 6]", "size: [4, 0, 6]"},
		{"size: [4, 5, 6]", "size: [4, 1e13, 6]", "size: [4, 1.0000001e13, 6]"},
		{"middle: [1, 2, 3]", "middle: [-1e13, 2, 3]",
			"middle: [-1.0000001e13, 2, 3]"},
		{"radius: 0.05", "radius: 0", "radius: -1e-300"},
		{"tilt: 50", "tilt: -180", "tilt: -180.0001"},
		{"detail: 24", "detail: 1", "detail: 0"},
		{"detail: 24", "detail: 2147483647", "detail: 2147483648"},
		{"knee: 0.7", "knee: 0", "knee: -1e-300"},
		{"top: 8", "top: 1e12", "top: 1.0000001e12"},
	} {
		at := strings.Replace(built(t), edge.was, edge.at, 1)
		if _, err := set.Read("built.yaml", []byte(at)); err != nil {
			t.Errorf("%s: %v", edge.at, err)
		}
		past := strings.Replace(built(t), edge.was, edge.past, 1)
		if _, err := set.Read("built.yaml", []byte(past)); err == nil {
			t.Errorf("%s was taken", edge.past)
		}
	}
}

// a file that is not YAML is refused, saying the file, and so is a
// block that takes keys and is given a word.
func TestWhatIsNotKeysIsRefused(t *testing.T) {
	for written, said := range map[string]string{
		"scene: [":               "broken.yaml:\n  line 1:",
		"- one\n- two\n":         "broken.yaml: a set file is keys, and this is not",
		"":                       "broken.yaml: a set file is keys, and this is not",
		"scene: 3\n":             "broken.yaml:1:8: this takes keys under it",
		"scene:\n  world: [1]\n": "broken.yaml:2:10: this takes keys under it",
	} {
		_, err := set.Read("broken.yaml", []byte(written))
		if err == nil || !strings.HasPrefix(err.Error(), said) {
			t.Errorf("%q: %v, wanted %q", written, err, said)
		}
	}
}

// a key said twice is refused wherever it is: at the top, and in a list
// inside a block.
func TestAKeySaidTwiceIsRefusedAnywhere(t *testing.T) {
	for _, written := range []string{
		"scene:\n  name: one\nscene:\n  name: two\n",
		"scene:\n  hud:\n    - {a: 1, a: 2}\n",
	} {
		_, err := set.Read("twice.yaml", []byte(written))
		if err == nil || !strings.Contains(err.Error(), "is said twice") {
			t.Errorf("%q: %v", written, err)
		}
	}
}
