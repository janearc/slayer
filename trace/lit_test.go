package trace

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// A sun leaves its highlight on a black satin floor. The anti-test: a
// lamp that lights from everywhere leaves none, so the floor under a
// flood alone is black, and under a flood and a sun it shows the sun's
// highlight and nothing more.
func TestAFloodLeavesNoHighlight(t *testing.T) {
	flooded := thing.NewWorld()
	flooded.Add(thing.Thing{Shape: shape.Plane{}, Material: lacquer(0.3)})
	flooded.Lamp(film.Flood{Strength: maths.Grey(lamplit)})
	if got := lone(flooded, downAtFloor, 0).Light(); got != (maths.Light{}) {
		t.Errorf("a flood alone leaves a highlight of %v", got)
	}
	both := floored(lacquer(0.3), mirrored)
	both.Lamp(film.Flood{Strength: maths.Grey(lamplit)})
	sun := lone(floored(lacquer(0.3), mirrored), downAtFloor, 0).Light()
	if got := lone(both, downAtFloor, 0).Light(); got != sun || !(sun.R > 0) {
		t.Errorf("a flood and a sun leave %v, the sun alone %v", got, sun)
	}
}

// Under more lamps than a point asks, a highlight is still the sum of
// theirs: more suns than lampsAsked, lying one way and sharing one
// sun's light among them, leave the highlight the one sun leaves. The
// anti-test is the same suns under the floor, which leave none.
func TestAHighlightUnderManyLampsIsTheSumOfTheirs(t *testing.T) {
	const many = lampsAsked + 4
	crowded := func(toward maths.Vec) *thing.World {
		world := thing.NewWorld()
		world.Add(thing.Thing{Shape: shape.Plane{}, Material: lacquer(0.3)})
		for range many {
			world.Lamp(thing.Sun(toward, maths.Grey(lamplit/many)))
		}
		return world
	}
	one := lone(floored(lacquer(0.3), mirrored), downAtFloor, 0).Light().R
	got := lone(crowded(mirrored), downAtFloor, 0).Light().R
	if !(one > 0) || !within(got, one, one*1e-5) {
		t.Errorf("%d suns leave a highlight of %v, one sun of their "+
			"light together %v", many, got, one)
	}
	if got := bright(lone(crowded(maths.V(1, -1, 0)), downAtFloor, 0)); got != 0 {
		t.Errorf("%d suns under the floor leave a highlight of %v", many, got)
	}
}

// A Satin so fine its square is nought is no roughness at all: it
// leaves no highlight, and nothing that is not a number. The anti-test
// is a Satin fine and still squarable, a thousandth, which leaves one.
func TestASatinTooFineToSquareLeavesNoHighlight(t *testing.T) {
	const finest = 1e-170 // its square, 1e-340, is below the least float
	if got := lone(floored(lacquer(finest), mirrored), downAtFloor, 0).Light(); got != (maths.Light{}) {
		t.Errorf("a Satin of %v leaves %v", finest, got)
	}
	if got := bright(lone(floored(lacquer(1e-3), mirrored), downAtFloor, 0)); !(got > 0) {
		t.Errorf("a Satin of a thousandth leaves a highlight of %v", got)
	}
}

// A sun drawn as a disc is seen in the sky where a look meets nothing:
// along a way within its disc, its strength over its disc's solid
// angle, two pi times one less the cosine of half its width. The
// anti-tests: along a way outside its disc, nothing; and a sun not
// drawn as a disc is seen nowhere.
func TestAWideSunIsSeenInTheSkyAndAPointSunIsNot(t *testing.T) {
	toward := maths.V(0, 1, 0)
	const size, strength = 0.2, 2.0
	wide := thing.NewWorld()
	wide.Lamp(film.Sun{Toward: toward, Strength: maths.Grey(strength),
		Size: size, Disc: true})
	want := strength / (2 * math.Pi * (1 - math.Cos(size/2)))
	at := maths.Ray{Dir: toward}
	if got := lone(wide, at, 0).Light(); !within(got.R, want, want*1e-6) {
		t.Errorf("the sun's disc shows %v, want %v", got, want)
	}
	inside := maths.Ray{Dir: maths.V(math.Sin(0.09), math.Cos(0.09), 0)}
	if got := lone(wide, inside, 0).Light(); !within(got.R, want, want*1e-6) {
		t.Errorf("0.09 radians from its middle the disc shows %v, want %v",
			got, want)
	}
	outside := maths.Ray{Dir: maths.V(math.Sin(0.11), math.Cos(0.11), 0)}
	if got := lone(wide, outside, 0).Light(); got != (maths.Light{}) {
		t.Errorf("0.11 radians from the sun's middle the sky shows %v", got)
	}
	point := thing.NewWorld()
	point.Lamp(thing.Sun(toward, maths.Grey(strength)))
	if got := lone(point, at, 0).Light(); got != (maths.Light{}) {
		t.Errorf("a sun not drawn as a disc shows %v in the sky", got)
	}
}

// A crinkled floor is lit unevenly: its face tilts from place to place,
// so under one sun its brightness differs from place to place, never
// below nought; the anti-test is the same floor flat, which is one
// brightness everywhere.
func TestACrinkledFloorIsLitUnevenly(t *testing.T) {
	spread := func(crinkle *thing.Crinkle) (least, most float64) {
		world := thing.NewWorld()
		world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
			Colour: maths.Grey(1), Crinkle: crinkle}})
		world.Lamp(thing.Sun(maths.V(1, 2, 0), maths.Grey(math.Pi)))
		least, most = math.Inf(1), math.Inf(-1)
		for place := range 40 {
			x := float64(place) * 0.037
			got := bright(lone(world, maths.Ray{At: maths.V(x, 1, 0),
				Dir: maths.V(0, -1, 0)}, 0))
			least, most = math.Min(least, got), math.Max(most, got)
		}
		return least, most
	}
	if least, most := spread(nil); least != most || !(least > 0) {
		t.Errorf("a flat floor is lit from %v to %v", least, most)
	}
	least, most := spread(&thing.Crinkle{Size: 0.05, Slope: 0.4, Seed: 7})
	if !(least >= 0) || !(most-least > 0.01) {
		t.Errorf("a crinkled floor is lit from %v to %v", least, most)
	}
}

// retold is a look's pixel with its steps told, as a replay tells them.
func retold(world *thing.World, ray maths.Ray) (frame.Pixel, []Step) {
	scene := stage(world, 0)
	walked := &walk{}
	scene.walked = walked
	return shade(scene, look{ray: ray}), walked.steps
}

// says is whether any step is the one named, saying the words given.
func says(steps []Step, what, words string) bool {
	for _, step := range steps {
		if step.What == what && strings.Contains(step.Said, words) {
			return true
		}
	}
	return false
}

// A replay tells what a look did and does not change what it saw: the
// bend at a clear surface, the lamps' highlights, a lamp that lights
// from everywhere, and the face of slayer's mesh a look met. In each
// world, none of which has a sky, the pixel told is the pixel untold.
// The anti-test: a look is told nothing it did not do, so no world's
// replay holds another world's step.
func TestAReplayTellsWhatALookDidAndSeesTheSame(t *testing.T) {
	down := maths.Ray{At: maths.V(0, 1, 0), Dir: maths.V(0, -1, 0)}
	flooded := thing.NewWorld()
	flooded.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Colour: maths.Grey(1)}})
	flooded.Lamp(film.Flood{Strength: maths.Grey(1)})
	cube := thing.NewWorld()
	cube.Add(thing.Thing{Shape: aCube(t),
		Material: &thing.Material{Name: "clay", Colour: maths.Grey(1)}})
	cube.Lamp(thing.Sun(maths.V(0, 1, 0), maths.Grey(math.Pi)))
	looks := map[string]struct {
		world       *thing.World
		ray         maths.Ray
		what, words string
	}{
		"a pool": {pooled(&thing.Material{Glow: maths.Grey(1)}), down,
			"a clear surface, bent by snell", "through"},
		"a satin floor": {floored(lacquer(0.3), mirrored), downAtFloor,
			"the lamps' highlights", "glint"},
		"a flooded floor": {flooded, down,
			"a lamp lighting from everywhere", "light"},
	}
	for name, each := range looks {
		pixel, steps := retold(each.world, each.ray)
		if !says(steps, each.what, each.words) {
			t.Errorf("%s: no step %q saying %q among %d", name, each.what,
				each.words, len(steps))
		}
		if untold := lone(each.world, each.ray, 0); pixel != untold ||
			!(bright(pixel) > 0) {
			t.Errorf("%s: told it sees %v, untold %v", name, pixel.Light(),
				untold.Light())
		}
		for other, theirs := range looks {
			if other != name && says(steps, theirs.what, "") {
				t.Errorf("%s: told %q, which is %s's", name, theirs.what, other)
			}
		}
	}
	// the cube's top is lit square by the sun overhead: one, as white
	// clay is under a sun of pi; and the replay names the face it met
	pixel, steps := retold(cube, down)
	if !within(bright(pixel), 1, 1e-6) {
		t.Errorf("the cube's top is lit by %v, want one", bright(pixel))
	}
	var named bool
	for _, step := range steps {
		named = named || strings.Contains(step.Said, "clay") &&
			strings.Contains(step.Said, " face ")
	}
	if !named {
		t.Errorf("no step names the clay face the look met: %+v", steps)
	}
}
