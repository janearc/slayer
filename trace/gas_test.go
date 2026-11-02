package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// earthlike is a world of a dark ball one unit across wearing the
// earth's air as a gas, a unit being the earth's radius, lit by a sun
// along plus x. The air's density is a share of sea level's, one at the
// surface, and the gas carries the amount, per metre.
func earthlike() *thing.World {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "ground",
			Colour: maths.Grey(0.1)}})
	world.AddVolume(thing.Volume{
		Medium: gas.Of{Shape: medium.Air{Radius: 1, Height: 8.5 / 6371,
			Thick: 1}, In: gas.Air(), Unit: 6371e3},
		Material: &thing.Material{Name: "air",
			Colour: maths.RGB(1, 0, 1)}})
	world.Lamp(thing.Sun(maths.V(1, 0, 0), maths.Grey(1)))
	return world
}

// looking is one pixel of the world seen from a place along a direction.
func looking(world *thing.World, from, along maths.Vec) maths.Light {
	eye := lookingAt(from, from.Add(along))
	lens := film.Pinhole{Wide: 1 * maths.Degree}
	return draw(world, eye, lens, 0, 3, 3).At(1, 1).Light()
}

// A gas air lights itself from its own numbers: straight up from the
// ground at noon the sky is blue, more blue than green than red, by
// rayleigh's fourth power across the eye's bands, and not the magenta
// the air's material is painted, which a gas volume never reads.
//
// Toward the sun along the horizon the light that gets through the
// long path is redder than the light from straight up.
func TestAGasAirMakesABlueSky(t *testing.T) {
	world := earthlike()
	up := maths.V(1.0001, 0, 0) // on the ground under the sun
	sky := looking(world, up, maths.V(1, 0, 0))
	if !(sky.B > sky.G && sky.G > sky.R && sky.R > 0) {
		t.Fatalf("the noon sky is %v", sky)
	}
	// how blue: the gas's own scattering at the blue band over the red,
	// which for air is rayleigh's fourth power, about three. What gets
	// through the air on the way moves it, so fifteen percent is let.
	air := gas.Air()
	for _, channel := range []struct {
		name string
		got  float64
		band maths.Band
	}{{"blue", sky.B, eyeBands[2]}, {"green", sky.G, eyeBands[1]}} {
		want := air.Scatter(channel.band) / air.Scatter(eyeBands[0])
		if !within(channel.got/sky.R, want, 0.15*want) {
			t.Errorf("the sky's %s over its red is %.3f, want %.3f",
				channel.name, channel.got/sky.R, want)
		}
	}
	// from a point on the terminator the sun lies along the horizon
	edge := maths.V(0, 1.0001, 0)
	low := looking(world, edge, maths.V(1, 0, 0))
	if !(low.R > 0 && low.B > 0) {
		t.Fatalf("the horizon toward the sun is %v", low)
	}
	if low.R/low.B <= sky.R/sky.B {
		t.Errorf("the horizon (%v) is not redder than the zenith (%v)",
			low, sky)
	}
	// at night, away from the sun, the air gives nothing
	night := looking(world, maths.V(-1.0001, 0, 0), maths.V(-1, 0, 0))
	if night != (maths.Light{}) {
		t.Errorf("the night sky is %v", night)
	}
}

// With no gas in a world, hasGas says so, gasClear is one, and a
// painted volume keeps the same share of all three channels. That the
// picture itself is unchanged is the golden puff's to say, not this
// test's.
func TestNoGasChangesNothing(t *testing.T) {
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1, Thick: 0.5},
		Material: &thing.Material{Name: "puff",
			Colour: maths.Grey(0.8)}})
	world.Lamp(thing.Sun(maths.V(0, 1, 0), maths.Grey(1)))
	scene := stage(world, 0)
	if hasGas(scene.volumes) {
		t.Fatal("a puff is a gas")
	}
	if passed := gasClear(scene.volumes, maths.Vec{}, maths.V(0, 1, 0),
		math.Inf(1)); passed != maths.Grey(1) {
		t.Errorf("gasClear with no gas is %v", passed)
	}
	keep := through(scene, maths.Ray{At: maths.V(0, 0, 5),
		Dir: maths.V(0, 0, -1)}, math.Inf(1)).keep
	if keep.R != keep.G || keep.G != keep.B || keep.R >= 1 || keep.R <= 0 {
		t.Errorf("a painted volume's keep is %v", keep)
	}
}
