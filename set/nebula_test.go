package set_test

import (
	"math"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// nebula is the nebula's set file as written, changed by each pair of
// was and now.
func nebula(t *testing.T, changes ...string) string {
	t.Helper()
	source, err := os.ReadFile("testdata/nebula.yaml")
	if err != nil {
		t.Fatal(err)
	}
	written := string(source)
	for i := 0; i+1 < len(changes); i += 2 {
		if !strings.Contains(written, changes[i]) {
			t.Fatalf("%q is not in the nebula", changes[i])
		}
		written = strings.Replace(written, changes[i], changes[i+1], 1)
	}
	return written
}

// nebulaRead is the nebula's world, read as written or changed.
func nebulaRead(t *testing.T, changes ...string) set.World {
	t.Helper()
	made, err := set.Read("nebula.yaml", []byte(nebula(t, changes...)))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := made.World()
	return world
}

// a world keeps the order its keys are written in, and its unit; a world
// that says no unit is in metres.
func TestAWorldKeepsItsOrderAndItsUnit(t *testing.T) {
	world := nebulaRead(t)
	if order := world.Order(); !slices.Equal(order, []string{"unit", "sky",
		"materials", "bodies", "spheres", "motions", "wisps", "spots",
		"floods"}) || world.Unit() != 1e7 {
		t.Errorf("the order is %v, the unit %v", order, world.Unit())
	}
	world.Order()[0] = "changed"
	if world.Order()[0] != "unit" {
		t.Error("changing the order handed back changed the world's")
	}
	if metres := builtRead(t); metres.Unit() != 1 {
		t.Errorf("a world with no unit is in units of %v m", metres.Unit())
	}
}

// a sky of stars is its seed and its look, and no gradient.
func TestASkyOfStars(t *testing.T) {
	sky, _ := nebulaRead(t).Sky()
	stars, starry := sky.Starry()
	share, size, bright := stars.Look()
	horizon, _, _ := sky.Light()
	if !starry || stars.Seed() != 29 || share != 0.08 || size != 0.5 ||
		bright != 0.6 || horizon != (set.Colour{}) {
		t.Errorf("the stars are %+v, %v", stars, starry)
	}
}

// a colour may be a swatch by its name, which is then all it is; and a
// material may glow with a colour rather than a black body's light.
func TestAColourMayBeASwatch(t *testing.T) {
	world := nebulaRead(t)
	glows, glowing := world.Materials()[0].Glowing()
	named, isSwatch := glows.Swatch()
	if !glowing || named != "tungsten" || !isSwatch ||
		linear(glows) != [3]float64{} {
		t.Errorf("the cool star glows %+v, %v", glows, glowing)
	}
	if heat, luminance, black := world.Materials()[0].Glow(); black ||
		heat != 0 || luminance != 0 {
		t.Errorf("the cool star glows as a black body at %v K", heat)
	}
	if _, glowing := builtRead(t).Materials()[1].Glowing(); glowing {
		t.Error("a black body's glow is a colour")
	}
	if named, isSwatch := builtRead(t).Floods()[0].Colour().Swatch(); isSwatch ||
		named != "" {
		t.Errorf("a linear colour is the swatch %q", named)
	}
}

// a body may say the radius it is drawn at, and leave its tilt and its
// aurora to its template.
func TestABodyLeavesWhatItDoesNotSayToItsTemplate(t *testing.T) {
	pulsar := nebulaRead(t).Bodies()[0]
	radius, sized := pulsar.Radius()
	tilt, tilted := pulsar.Tilt()
	shown, saysAurora := pulsar.Aurora()
	if pulsar.Template() != "pulsar" || radius != 0.001 || !sized ||
		tilted || tilt != 0 || saysAurora || shown || pulsar.Detail() != 4 {
		t.Errorf("the pulsar is %+v", pulsar)
	}
}

// a sphere is its material, its place, its size and the motion that
// carries it, if one does; a nought in a place is nought, never minus
// nought.
func TestASphereIsWhatItSays(t *testing.T) {
	spheres := nebulaRead(t).Spheres()
	moving, carried := spheres[0].Motion()
	if len(spheres) != 2 || spheres[0].Name() != "companion" ||
		spheres[0].Material() != "cool-star" ||
		spheres[0].Middle() != (maths.Vec{}) || spheres[0].Radius() != 10.5 ||
		moving != "orbit" || !carried {
		t.Errorf("spheres %+v", spheres)
	}
	still, carried := spheres[1].Motion()
	middle := spheres[1].Middle()
	if still != "" || carried || middle != maths.V(5, 1, 0) ||
		math.Signbit(middle.Z) {
		t.Errorf("the still sphere is %+v at %v", spheres[1], middle)
	}
}

// a motion is its run and its frames, each frame's share its percent of
// a hundred and its place in the world's space.
func TestAMotionIsItsRunAndItsFrames(t *testing.T) {
	orbit := nebulaRead(t).Motions()[0]
	length, ease, hold := orbit.Run()
	frames := orbit.Frames()
	// divided as the program divides, and not folded by the compiler,
	// which works the constant exactly and rounds it once
	percent := 4.1667
	if orbit.Name() != "orbit" || length != 7200*time.Second ||
		ease != "linear" || !hold || len(frames) != 3 ||
		frames[0] != (set.Frame{Share: 0, At: maths.V(64, 0, 0)}) ||
		frames[1] != (set.Frame{Share: percent / 100,
			At: maths.V(61.82, 0, 16.56)}) || frames[2].Share != 1 {
		t.Errorf("the orbit is %v, %v, %v, %+v", length, ease, hold, frames)
	}
}

// a wisp is its colour, its place, its ball and its raggedness.
func TestAWispIsWhatItSays(t *testing.T) {
	wisp := nebulaRead(t).Wisps()[0]
	radius, thick := wisp.Ball()
	seed, scale, amount := wisp.Ragged()
	named, _ := wisp.Colour().Swatch()
	if wisp.Name() != "nebula" || named != "red" ||
		wisp.Middle() != (maths.Vec{}) || radius != 600 || thick != 0.002 ||
		seed != 7 || scale != 200 || amount != 0.6 {
		t.Errorf("the wisp is %+v", wisp)
	}
}

// a spot is its place, its aim, its cone and its light.
func TestASpotIsWhatItSays(t *testing.T) {
	wind := nebulaRead(t).Spots()[0]
	cone, soft := wind.Cone()
	heat, strength := wind.Shine()
	if wind.Name() != "wind" || wind.At() != (maths.Vec{}) ||
		wind.Aim() != maths.V(1, 0, 0) || cone != 12 || soft != 3 ||
		heat != 9000 || strength != 9000 {
		t.Errorf("the wind is %+v", wind)
	}
}

// what a world hands back of its new lists is its own copy.
func TestAWorldsNewListsAreCopies(t *testing.T) {
	world := nebulaRead(t)
	world.Spheres()[0] = set.Sphere{}
	world.Spots()[0] = set.Spotlight{}
	world.Wisps()[0] = set.Wisp{}
	world.Motions()[0] = set.Motion{}
	orbit := world.Motions()[0]
	orbit.Frames()[0] = set.Frame{}
	if world.Spheres()[0].Name() != "companion" ||
		world.Spots()[0].Name() != "wind" ||
		world.Wisps()[0].Name() != "nebula" ||
		world.Motions()[0].Name() != "orbit" ||
		orbit.Frames()[0].Share != 0 || orbit.Frames()[0].At.X != 64 {
		t.Error("changing a list handed back changed the world's")
	}
}

// the nebula's mistakes are refused where they are written.
func TestANebulasMistakesAreRefused(t *testing.T) {
	for _, mistake := range []struct{ was, now, said string }{
		{"motion: orbit", "motion: spin",
			`sphere companion is carried by "spin", and the world holds ` +
				"no motion by that name"},
		{"material: cool-star\n        middle: [0, 0, 0]\n        radius: 10.5",
			"material: hot-star\n        middle: [0, 0, 0]\n        radius: 10.5",
			`sphere companion is made of "hot-star", and the world holds ` +
				"no material by that name"},
		{"        radius: 10.5\n", "",
			"a sphere says [material middle radius source], all of them"},
		{"radius: 10.5", "radius: 0", "radius takes a number from 5e-324"},
		{"{percent: 4.1667,", "{percent: 104.1667,",
			"percent takes a number from 0 to 100"},
		{"{percent: 100, at", "{percent: 3, at",
			"a frame at 3% after one at 4.1667%: frames go in order"},
		{"{percent: 0, at: [64, 0, 0]}", "{percent: 0}",
			"a frame says [percent at], all of them"},
		{"          - {percent: 0, at: [64, 0, 0]}\n" +
			"          - {percent: 4.1667, at: [61.82, -16.56, 0]}\n" +
			"          - {percent: 100, at: [64, 0, 0]}\n",
			"          []\n", "frames takes a list of frames"},
		{"seconds: 7200", "seconds: 0", "seconds takes a number from 1e-09"},
		{"ease: linear", "ease: swoop", "ease takes [linear ease"},
		{"hold: true", "hold: yes", `hold takes true or false, not "yes"`},
		{"        hold: true\n", "",
			"a motion says [seconds ease hold frames], all of them"},
		{"amount: 0.6", "amount: 1.1", "amount takes a number from 5e-324 to 1"},
		{"        scale: 200\n", "",
			"a wisp says [colour middle radius thick seed scale amount " +
				"source], all of them, and this has no scale"},
		{"seed: 7", "seed: 7.5", "seed takes a whole number"},
		{"aim: [1, 0, 0]", "aim: [0, 0, 0]",
			"spot wind aims at nought, nought and nought, which is no way"},
		{"cone: 12", "cone: 0", "cone takes a number from 5e-324 to 180"},
		{"soft: 3", "soft: 181", "soft takes a number from 0 to 180"},
		{"        soft: 3\n", "",
			"a spot says [at aim cone soft temperature strength source], " +
				"all of them, and this has no soft"},
		{"      stars: {seed: 29,", "      horizon: {swatch: zenith}\n" +
			"      stars: {seed: 29,",
			"a sky is a gradient or stars, not both"},
		{"stars: {seed: 29, share: 0.08,", "stars: {share: 0.08,",
			"stars says [seed share size bright], all of them, and this " +
				"has no seed"},
		{"share: 0.08", "share: 1.08", "share takes a number from 0 to 1"},
		{"      stars: {seed: 29, share: 0.08, size: 0.5, bright: 0.6}\n" +
			"      source: eye\n", "      stars: {seed: 29, share: 0.08, " +
			"size: 0.5, bright: 0.6}\n",
			"a sky of stars says [stars source], all of them"},
		{"radius: 0.001", "radius: 0", "radius takes a number from 5e-324"},
		{"unit: 10000000", "unit: 0", "unit takes a number from 5e-324"},
		{"colour: {swatch: rust}", "colour: {swatch: rust, strength: 1}",
			"strength is not a key here"},
	} {
		_, err := set.Read("nebula.yaml", []byte(nebula(t, mistake.was,
			mistake.now)))
		if err == nil || !strings.Contains(err.Error(), mistake.said) {
			t.Errorf("%q: refused with %v, wanted %q", mistake.now, err,
				mistake.said)
		}
	}
}

// a motion's numbers are taken to their edges: a run of a nanosecond
// and of 292 years, frames at nought and a hundred percent, and a wisp
// whose noise may take all of its density.
func TestANebulasNumbersAreTakenToTheirEdges(t *testing.T) {
	longest := "9214819200"
	for _, edge := range []struct{ was, at, past string }{
		{"seconds: 7200", "seconds: 1e-9", "seconds: 0.9e-9"},
		{"seconds: 7200", "seconds: " + longest, "seconds: 9214819201"},
		{"{percent: 100, at", "{percent: 100, at", "{percent: 100.0001, at"},
		{"amount: 0.6", "amount: 1", "amount: 1.0000001"},
		{"cone: 12", "cone: 180", "cone: 180.0001"},
		{"temperature: 9000", "temperature: 1", "temperature: 0.999"},
	} {
		if _, err := set.Read("nebula.yaml", []byte(nebula(t, edge.was,
			edge.at))); err != nil {
			t.Errorf("%s: %v", edge.at, err)
		}
		if _, err := set.Read("nebula.yaml", []byte(nebula(t, edge.was,
			edge.past))); err == nil {
			t.Errorf("%s was taken", edge.past)
		}
	}
	world := nebulaRead(t, "seconds: 7200", "seconds: "+longest)
	if length, _, _ := world.Motions()[0].Run(); length <= 0 ||
		length != 9214819200*time.Second {
		t.Errorf("292 years is a run of %v", length)
	}
}
