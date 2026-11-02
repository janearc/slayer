package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// unlitLamp is a lamp with a direction and no light: one that does not
// reach where it is asked, as a beam does off its axis.
type unlitLamp struct{}

// Light is no light, from up and to one side, infinitely far.
func (unlitLamp) Light(_, _ maths.Vec) (maths.Vec, float64, maths.Light) {
	return maths.V(1, 1, 0).Unit(), math.Inf(1), maths.Light{}
}

// counted is a medium that counts how often its density is asked.
type counted struct {
	slab
	asked *int
}

// Density is the slab's, and one more asking.
func (medium counted) Density(point maths.Vec) float64 {
	*medium.asked++
	return medium.slab.Density(point)
}

// flat is a texture of one light everywhere.
type flat maths.Light

// At is the one light.
func (paint flat) At(maths.Vec) maths.Light { return maths.Light(paint) }

// fill is a flood of half white, the ambient lamp of these tests, and
// filling what it sends to a point that faces a way.
var fill = film.Flood{Strength: maths.Grey(0.5)}

// filling is the flood's light on something facing a way.
func filling(facing maths.Vec) maths.Light {
	_, _, light := fill.Light(maths.Vec{}, facing)
	return light
}

// What a lamp sends to a point is nothing past a wall for a lamp with
// a direction, and all of it for a lamp from everywhere, which casts
// no shadow. A lamp that sends no light is done with at once: nothing
// is marched toward it.
func TestWhatReachesAPointFromALamp(t *testing.T) {
	asked := 0
	over := counted{asked: &asked, slab: slab{density: 0.5,
		box: maths.Box{Min: maths.V(-9, 2, -9), Max: maths.V(9, 3, 9)}}}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Motion: resting(maths.V(0, 6, 0))})
	world.AddVolume(thing.Volume{Medium: over})
	scene := stage(world, 0)
	under := maths.Vec{}

	sun := thing.Sun(maths.Up, maths.Grey(1))
	if got := scene.reaching(sun, under, maths.Up); got.ambient() ||
		got.light != (maths.Light{}) {
		t.Errorf("past the ball the sun sends %+v", got)
	}
	// a sun to one side clears the ball and goes through the slab,
	// on a slant: root two of its depth
	slanting := thing.Sun(maths.V(1, 1, 0), maths.Grey(1))
	want := math.Exp(-0.5 * math.Sqrt2)
	slanted := scene.reaching(slanting, under, maths.Up).light.R
	if asked == 0 || !within(slanted, want, 1e-9) {
		t.Errorf("through the slab the slanting sun sends %v, want "+
			"%v; the slab was asked %d times", slanted, want, asked)
	}
	if got := scene.reaching(fill, under, maths.Up); !got.ambient() ||
		got.light != filling(maths.Up) {
		t.Errorf("the flood sends %+v, want all of %v from everywhere",
			got, filling(maths.Up))
	}
	asked = 0
	got := scene.reaching(unlitLamp{}, under, maths.Up)
	if got.light != (maths.Light{}) || asked != 0 {
		t.Errorf("a lamp with no light sends %v, and the slab was "+
			"asked %d times on the way to it", got.light, asked)
	}
}

// A lamp from everywhere lights a surface with the surface's colour
// times the lamp's light there, which the lamp has already weighed for
// the way the surface faces.
func TestAFloodLightsASurfaceByItsColour(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Colour: maths.RGB(0.8, 0.4, 0.2)}})
	world.Lamp(fill)
	got := lone(world, straightDown(3), 0).Light()
	want := maths.RGB(0.8, 0.4, 0.2).Times(filling(maths.Up))
	if !(want.R > 0.1) || !within(got.R, want.R, 1e-6) ||
		!within(got.G, want.G, 1e-6) || !within(got.B, want.B, 1e-6) {
		t.Errorf("under a flood the floor is %v, want %v", got, want)
	}
}

// In smoke a lamp from everywhere is weighed evenly, a half, like any
// lamp; and what the smoke scatters is its texture where it has one,
// not its plain colour.
func TestAFloodInSmokeIsWeighedEvenlyAndPaintedByTexture(t *testing.T) {
	const density, deep = 0.7, 2.0
	took := 1 - math.Exp(-density*deep)
	light := filling(alongZ.Dir.Neg()).R
	for name, each := range map[string]struct {
		worn *thing.Material
		want float64
	}{
		"plain smoke": {&thing.Material{Colour: maths.Grey(0.8)},
			0.8 * light * evenly * took},
		"textured smoke": {&thing.Material{Colour: maths.Grey(0.8),
			Texture: flat(maths.Grey(0.25))},
			0.25 * light * evenly * took},
	} {
		world := thing.NewWorld()
		world.AddVolume(thing.Volume{Medium: plate(density, -1, 1),
			Material: each.worn})
		world.Lamp(fill)
		got := through(stage(world, 0), alongZ, math.Inf(1)).added.R
		if !(each.want > 0.01) || !within(got, each.want, 1e-9) {
			t.Errorf("%s under a flood adds %v, want %v", name, got,
				each.want)
		}
	}
}

// In a gas a lamp from everywhere is weighed evenly too: one step
// gathers the lamp's light times a half, times what the gas scatters
// at each of the eye's bands, times the step's thickness.
func TestAFloodInAGasIsWeighedEvenly(t *testing.T) {
	const thick = 2e3
	air := gas.Air()
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: gas.Of{
		Shape: plate(1, -1, 1), In: air, Unit: 1}})
	world.Lamp(fill)
	scene := stage(world, 0)
	inside := crossings(scene.volumes, alongZ, math.Inf(1))[0]
	got := gasStep(scene, &inside, stride{thick: thick}, alongZ, 1,
		untouched()).added
	light := filling(alongZ.Dir.Neg())
	want := light.Scale(evenly).Times(byEye(air.Scatter)).Scale(thick)
	if !(want.B > want.R && want.R > 0) {
		t.Fatalf("the test's air scatters nothing to tell: %v", want)
	}
	if !within(got.R, want.R, 1e-12) || !within(got.G, want.G, 1e-12) ||
		!within(got.B, want.B, 1e-12) {
		t.Errorf("a gas step under a flood adds %v, want %v", got, want)
	}
}

// Under two lamps a point is shone on by the sum of the two.
func TestTwoLampsShineTheirSum(t *testing.T) {
	world := thing.NewWorld()
	world.Lamp(thing.Sun(maths.Up, maths.Grey(0.5)))
	world.Lamp(thing.Sun(maths.V(1, 0, 0), maths.Grey(0.25)))
	got := shone(stage(world, 0), maths.Vec{}, maths.Up)
	if !within(got, 0.75, 1e-12) {
		t.Errorf("two lamps of a half and a quarter shine %v", got)
	}
}

// A haze throws light forward. A step of it lit from behind the eye,
// so that the light must turn right round to come back, gathers by the
// phase at a cosine of minus one; lit from ahead, so that the light
// comes straight on to the eye, it gathers more.
func TestAHazeScattersForwardMoreThanBack(t *testing.T) {
	const thick = 2e3
	haze := gas.Haze()
	gathered := func(toward maths.Vec) float64 {
		world := thing.NewWorld()
		world.AddVolume(thing.Volume{Medium: gas.Of{
			Shape: plate(1, -1, 1), In: haze, Unit: 1}})
		world.Lamp(thing.Sun(toward, maths.Grey(1)))
		scene := stage(world, 0)
		inside := crossings(scene.volumes, alongZ, math.Inf(1))[0]
		return gasStep(scene, &inside, stride{thick: thick}, alongZ, 1,
			untouched()).added.G
	}
	// alongZ leaves the eye toward plus z from in front of the slab,
	// so nothing stands between the step and a lamp behind the eye
	ahead, behind := gathered(alongZ.Dir), gathered(alongZ.Dir.Neg())
	green := eyeBands[1]
	want := haze.PhaseAt(-1) * haze.Scatter(green) * thick
	if !(haze.PhaseAt(1) > 2*haze.PhaseAt(-1)) {
		t.Fatal("the test's haze is not lopsided")
	}
	if !within(behind, want, 1e-12) || !(ahead > 2*behind) {
		t.Errorf("lit from behind the eye the step gathers %v, want "+
			"%v; lit from ahead, %v", behind, want, ahead)
	}
}

// A surface turned from a lamp gets nothing from it, and not less
// than nothing: a ball that casts no shadow, lit from behind, is black
// in front.
func TestASurfaceTurnedFromALampIsUnlit(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Colour: maths.Grey(0.8)},
		Attrs:    thing.Attrs{Clear: true}})
	world.Lamp(thing.Sun(maths.V(0, 0, -1), maths.Grey(1)))
	front := maths.Ray{At: maths.V(0, 0, 5), Dir: maths.V(0, 0, -1)}
	if got := lone(world, front, 0); got.Light() != (maths.Light{}) ||
		got.Cover != 1 {
		t.Errorf("turned from its lamp the ball is %+v", got)
	}
}
