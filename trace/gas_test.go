package trace

import (
	"math"
	"testing"

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

// EyeOptics is a gas's extinction and scatter at 610, 550 and 465 nm, read
// here from the gas at each by hand, and the numbers a placed volume keeps.
func TestEyeOpticsAreTheGassNumbersAtTheEyesBands(t *testing.T) {
	for name, air := range map[string]gas.Gas{"air": gas.Air(),
		"haze": gas.Haze(), "air per planet": gas.Air().Per(6371e3)} {
		extinction, scatter := EyeOptics(air)
		bands := [3]maths.Band{{Nanometres: 610}, {Nanometres: 550},
			{Nanometres: 465}}
		gotE := [3]float64{extinction.R, extinction.G, extinction.B}
		gotS := [3]float64{scatter.R, scatter.G, scatter.B}
		for index, band := range bands {
			if !(gotE[index] == air.Extinction(band)) ||
				!(gotS[index] == air.Scatter(band)) || !(gotS[index] > 0) {
				t.Errorf("%s at %v nm: extinction %v scatter %v, want %v and %v",
					name, band.Nanometres, gotE[index], gotS[index],
					air.Extinction(band), air.Scatter(band))
			}
		}
		world := thing.NewWorld()
		world.AddVolume(thing.Volume{Medium: gas.Of{
			Shape: medium.Puff{Radius: 1, Thick: 1}, In: air, Unit: 1}})
		placed := stage(world, 0).volumes[0]
		if placed.extinction != extinction || placed.scatter != scatter {
			t.Errorf("%s: a placed volume keeps %v and %v, not %v and %v",
				name, placed.extinction, placed.scatter, extinction, scatter)
		}
	}
}
