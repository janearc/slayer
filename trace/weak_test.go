package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// weakSide is how many pixels a weak-field picture is each way.
const weakSide = 8

// weakly is one scene drawn with and without a field too weak to bend
// or shift anything: its name, its world made fresh each time, and
// where it is seen from.
type weakly struct {
	name  string
	world func() *thing.World
	eye   maths.Subspace
}

// weakScenes is one scene for each thing a field's march handles apart
// from the straight tracer: a surface, a painted volume, a gas, a
// beamed volume, haze, and a mirror.
func weakScenes() []weakly {
	near := lookingAt(maths.V(0, 2, 6), centre)
	return []weakly{
		{name: "a solid ball under a sun", world: smoking, eye: near},
		{name: "a painted puff", world: weakPuff, eye: near},
		{name: "gas air", world: earthlike,
			eye: lookingAt(maths.V(0, 0, 3), maths.Vec{})},
		{name: "a beamed jet", world: weakJet,
			eye: lookingAt(maths.V(0, 12, 0.01), maths.Vec{})},
		{name: "haze", world: weakHaze, eye: near},
		{name: "a mirror", world: weakMirror, eye: near},
	}
}

// weakPuff is lit smoke over a glowing floor.
func weakPuff() *thing.World {
	world := puffed(2)
	world.Lamp(thing.Sun(sunwards, maths.Grey(1)))
	return world
}

// weakJet is a glowing jet at nine tenths of the speed of light.
func weakJet() *thing.World {
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{
		Medium:   medium.Jet{Length: 4, Base: 1, Spread: 0.1, Thick: 1},
		Material: &thing.Material{Name: "jet", Glow: maths.Grey(0.2)},
		Beam:     &thing.Beam{Speed: 0.9}})
	return world
}

// weakHaze is the solid scene in haze of its own colour.
func weakHaze() *thing.World {
	world := smoking()
	world.Haze = thing.Haze{Density: 0.1, Colour: maths.Grey(0.25)}
	return world
}

// weakMirror is the solid scene on a glossy floor under a white sky.
func weakMirror() *thing.World {
	world := smoking()
	world.Things[0].Material = &thing.Material{Name: "glass",
		Colour: maths.Grey(0.1), Gloss: 1}
	world.Sky = even(maths.Grey(1))
	return world
}

// drawn is a weak scene's picture, with the weak field round the
// world's middle or without it.
func (scene weakly) drawn(field bool) *frame.Frame {
	world := scene.world()
	if field {
		world.AddField(thing.Field{Mass: 1e-6, Reach: 40})
	}
	lens := film.Pinhole{Wide: 40 * maths.Degree}
	return draw(world, scene.eye, lens, 0, weakSide, weakSide)
}

// stillWrong is the weak-field scenes the tracer gets wrong today, and
// what is wrong with each. A scene named here has to fail: the day it
// passes, the test says to take its line out.
//
// The numbers are pixels of sixty-four, measured when each line was
// written.
var stillWrong = map[string]wrong{
	"gas air": {pixels: 34, share: 0.25, why: "the air's light is " +
		"off, the blue most, measured at 34 pixels and 23 percent: " +
		"a field's step is cut into 4 where a crossing with no " +
		"field is cut into 48. measured, cutting both into 48 " +
		"leaves 12 pixels off by 1.2 percent, and only both at 800 " +
		"pass: the plain march is itself no better than a percent"},
	"a beamed jet": {pixels: 4, share: 0.4, why: "measured at 4 " +
		"pixels, 36 percent dim: a cube of the doppler factor " +
		"outside a field, a fourth power inside"},
}

// wrong is a scene known to be wrong, why, and how wrong it may be
// before the test fails: the most pixels a weak field may move, and
// the most any channel may move by, as a share of its value.
type wrong struct {
	why    string
	pixels int
	share  float64
}

// offBy is how many pixels of a picture have a channel more than a
// thousandth from the value wanted, and the largest share of its
// value any channel is off by.
func offBy(picture *frame.Frame, pixels [][5]float64) (int, float64) {
	count, most := 0, 0.0
	for i, pixel := range picture.Pix {
		got, off := channels(pixel), false
		for k, want := range pixels[i] {
			gap := apart(got[k], want, 1e-3)
			if gap == 0 {
				continue
			}
			off = true
			most = math.Max(most, gap/math.Abs(want))
		}
		if off {
			count++
		}
	}
	return count, most
}

// A field a millionth of a unit in mass bends and shifts nothing, so
// every scene is the picture it is with no field at all, each channel
// within a thousandth of itself.
func TestAWeakFieldChangesNothing(t *testing.T) {
	for _, scene := range weakScenes() {
		t.Run(scene.name, func(t *testing.T) { unmoved(t, scene) })
	}
}

// unmoved holds one scene to its own picture with no field, unless it
// is in stillWrong, when it holds the scene to being wrong still.
func unmoved(t *testing.T, scene weakly) {
	plain, curved := scene.drawn(false), scene.drawn(true)
	pixels := make([][5]float64, len(plain.Pix))
	for i, pixel := range plain.Pix {
		pixels[i] = channels(pixel)
	}
	said, moved := worstBy(curved, pixels, 1e-3)
	listed, known := stillWrong[scene.name]
	switch {
	case moved && known:
		count, share := offBy(curved, pixels)
		t.Logf("wrong, as listed: %d pixels, the worst by %.3f of "+
			"its value; %s", count, share, listed.why)
		if count > listed.pixels || !(share <= listed.share) {
			t.Errorf("worse than listed: %d pixels and %.3f, over "+
				"%d and %.3f; furthest: %s", count, share,
				listed.pixels, listed.share, said)
		}
	case moved:
		t.Errorf("the weak field moved the picture; furthest: %s", said)
	case known:
		t.Errorf("the scene is right now; take it out of stillWrong, "+
			"which says: %s", listed.why)
	}
}

// A ray that passes a ball and goes on to the floor behind it meets
// that floor with a weak field round the ball as it does with none.
// The ray's nearest point to the field's middle is short of the floor,
// so it has turned outward and left the field's march before it lands.
func TestARayLeavingAFieldStillMeetsTheFloor(t *testing.T) {
	past := maths.Ray{At: maths.V(3, 4, 6),
		Dir: maths.V(-0.2, -0.5, -1).Unit()}
	plain := lone(smoking(), past, 0)
	// lit: past half of a white matte surface's own, which is one over pi
	if plain.Cover != 1 || !(bright(plain) > 0.5/math.Pi) {
		t.Fatalf("with no field the ray should land on lit floor: %+v",
			plain)
	}
	world := smoking()
	world.AddField(thing.Field{Mass: 1e-6, Reach: 40})
	curved := lone(world, past, 0)
	if curved.Cover != 1 || !within(bright(curved), bright(plain), 1e-6) {
		t.Errorf("in a weak field the ray brings back %+v, with "+
			"none %+v", curved, plain)
	}
}

// Through a weak field a ball and the floor behind it are as far away
// as they are with no field: the march counts every piece of its way,
// the last piece onto a surface and the straight run out to the reach.
func TestDepthThroughAWeakFieldIsTheDistance(t *testing.T) {
	at := maths.Ray{At: maths.V(0, 1, 9), Dir: maths.V(0, 0, -1)}
	past := maths.Ray{At: maths.V(3, 4, 6),
		Dir: maths.V(-0.2, -0.5, -1).Unit()}
	beyond := maths.Ray{At: maths.V(0, 60, 0),
		Dir: maths.V(0, -1, 0.3).Unit()}
	world := smoking()
	world.AddField(thing.Field{Mass: 1e-6, Reach: 40})
	for name, aim := range map[string]maths.Ray{"at the ball": at,
		"past the ball": past, "from beyond the reach": beyond} {
		want := float64(lone(smoking(), aim, 0).Depth)
		got := float64(lone(world, aim, 0).Depth)
		if !(want > 1) || !within(got, want, 1e-3*want) {
			t.Errorf("%s the depth is %v in the field and %v with "+
				"none", name, got, want)
		}
	}
}

// A volume in front of a surface is gathered on the march's last step
// too. Through a thin glowing slab lying on a glowing floor, thin
// enough to sit inside that one step, a weak field gives what no field
// gives: the floor through the slab, and the slab's own glow.
func TestAFieldGathersTheVolumeBeforeASurface(t *testing.T) {
	const density, deep, glow = 70, 0.01, 0.3
	over := slab{density: density, box: maths.Box{
		Min: maths.V(-10, 0, -10), Max: maths.V(10, deep, 10)}}
	build := func(field bool) *thing.World {
		world := glowing()
		world.AddVolume(thing.Volume{Medium: over,
			Material: &thing.Material{Glow: maths.Grey(glow)}})
		if field {
			world.AddField(thing.Field{Mass: 1e-6, Reach: 40})
		}
		return world
	}
	aim := maths.Ray{At: maths.V(0.5, 3, 0), Dir: maths.V(0, -1, 0)}
	kept := math.Exp(-density * deep)
	want := kept + glow*(1-kept)
	// a pixel is kept to seven figures, and the two marches add the
	// slab up in different pieces, so five places is asked
	plain := float64(lone(build(false), aim, 0).R)
	if !within(plain, want, 1e-5) {
		t.Fatalf("with no field the floor is %v, want %v", plain, want)
	}
	curved := float64(lone(build(true), aim, 0).R)
	if !within(curved, want, 1e-5) {
		t.Errorf("in a weak field the floor is %v, want %v", curved,
			want)
	}
}

// A gain is capped at shiftTop so one hot pixel does not bloom a
// frame, and a gain that is not a number is no light at all: a broken
// geodesic gives a dark step, never a NaN that an average spreads.
func TestAGainIsCappedAndNeverANaN(t *testing.T) {
	if got := capped(math.NaN()); got != 0 {
		t.Errorf("a NaN gain is capped to %v, want nought", got)
	}
	if got := capped(16); got != 16 {
		t.Errorf("a gain of 16 is capped to %v", got)
	}
	if got := capped(1e9); got != shiftTop {
		t.Errorf("a gain of a billion is capped to %v, want %v", got,
			float64(shiftTop))
	}
}

// A look keeps its pixel's width through a field's march: a ball
// painted by the width it is asked at is asked at the same width
// inside a weak field as outside one.
func TestALookKeepsItsPixelThroughAField(t *testing.T) {
	weak := thing.Field{Mass: 1e-6, Reach: 40}
	painted := func(field bool) float64 {
		world := thing.NewWorld()
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
			Material: &thing.Material{Texture: footprint{}}})
		world.Lamp(thing.Sun(maths.V(0, 0, 1), maths.Grey(1)))
		if field {
			world.AddField(weak)
		}
		eye := lookingAt(maths.V(0, 0, 3), maths.Vec{})
		lens := film.Pinhole{Wide: 60 * maths.Degree}
		return float64(draw(world, eye, lens, 0, 65, 65).At(32, 32).R)
	}
	plain, curved := painted(false), painted(true)
	if !(plain > 0) || !within(curved, plain, 1e-6*plain) {
		t.Errorf("the ball is asked at a width of %v in a weak field "+
			"and %v with none", curved, plain)
	}
}

// A look keeps its count of bounces through a field's march, so a
// mirror seen in a mirror is no more followed inside a weak field than
// outside one. The ray lands a unit from the field's middle, where the
// march is an ordinary one, and mirrors up toward the ball.
//
// With no ball the floor mirrors the sky by fresnel's share, the same
// with the field as without: so the check below cannot pass by the
// ray simply coming back black.
func TestAMirrorInAFieldIsStillNotFollowed(t *testing.T) {
	steep := maths.Ray{At: maths.V(1.3, 1, 0),
		Dir: maths.V(-0.3, -1, 0).Unit()}
	mirrored := func(ball, field bool) float64 {
		world := mirrors(ball)
		if field {
			world.AddField(thing.Field{Mass: 1e-6, Reach: 40})
		}
		return bright(lone(world, steep, 0))
	}
	open, openInField := mirrored(false, false), mirrored(false, true)
	if !(open > 0.03) || !within(openInField, open, 1e-6) {
		t.Fatalf("with no ball the floor mirrors %v in a weak field "+
			"and %v with none", openInField, open)
	}
	under, underInField := mirrored(true, false), mirrored(true, true)
	if under != 0 || underInField != 0 {
		t.Errorf("the ball in the floor is %v in a weak field and %v "+
			"with none: a second bounce was taken", underInField,
			under)
	}
}
