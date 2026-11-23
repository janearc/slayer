package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// underSky is a world of a sky and so many volume sky samples, with
// nothing else in it.
func underSky(t *testing.T, sky thing.Sky, samples int) *thing.World {
	t.Helper()
	asked, err := thing.NewVolumeIllumination(samples)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Sky, world.VolumeIllumination = sky, asked
	return world
}

// A point of a volume takes the mean of the sky's radiance over the
// whole sphere, with no cosine: under a sky of one radiance everywhere,
// that radiance. Were the looks weighed by a cosine to straight up, as a
// floor weighs them, the mean of the cosine over the sphere's upper half
// and nought over its lower would give a quarter of it.
func TestAVolumeUnderAnEvenSkyTakesTheSkysRadiance(t *testing.T) {
	blueSky := even(skyBlue)
	scene := stage(underSky(t, blueSky, 8), 0)
	lightIs(t, "the volume sky", scene.volumeSkyAt(maths.Vec{}), skyBlue,
		1e-12)
}

// The looks come in pairs, one the other turned right round, and what
// stands in a look's way hides its sky. A floor under the point hides
// the one of each pair that looks down, so the point takes half the
// sky's radiance: half the sphere of a sky that is the same everywhere.
func TestAFloorHidesHalfTheVolumeSky(t *testing.T) {
	world := underSky(t, even(skyBlue), 8)
	world.Add(thing.Thing{Shape: shape.Plane{}})
	got := stage(world, 0).volumeSkyAt(maths.V(0, 1, 0))
	lightIs(t, "the volume sky over a floor", got, skyBlue.Scale(0.5), 1e-12)
}

// Under a sky that brightens with height the mean over the sphere is
// the sky at the horizon: each look up is as far above it as its pair,
// looking down, is below. A sky of one at the horizon rising half a unit
// a radian is one. The eight looks are no nearer straight up or down
// than a sine of 0.875, 61 degrees, where the table is the sky's own.
func TestTheVolumeSkyIsTheMeanOverTheWholeSphere(t *testing.T) {
	scene := stage(underSky(t, rising{floor: 1, slope: 0.5}, 8), 0)
	lightIs(t, "the volume sky", scene.volumeSkyAt(maths.Vec{}),
		maths.Grey(1), 1e-9)
}

// Anti: a volume takes no sky that was not asked for, and none where
// there is no sky.
func TestAVolumeTakesNoSkyUnasked(t *testing.T) {
	unasked := thing.NewWorld()
	unasked.Sky = even(skyBlue)
	skyless := underSky(t, nil, 8)
	for name, world := range map[string]*thing.World{
		"no samples asked": unasked, "no sky": skyless} {
		if got := stage(world, 0).volumeSkyAt(maths.Vec{}); got !=
			(maths.Light{}) {
			t.Errorf("with %s the volume sky is %v", name, got)
		}
	}
}

// With the volume sky asked for, a bit of smoke sends its colour times
// the sky's mean radiance, and its colour times each lamp's light by
// the even phase, one in four pi. Smoke of 0.8, 0.4 and 0.2 under a sky
// of 0.2, 0.4 and 0.8 is 0.16 in every channel from the sky, and under
// a sun of two 0.8, 0.4 and 0.2 times 2 / (4 pi) more.
func TestSmokeUnderAnAskedSkyAddsTheSkyAndWeighsLampsByTheEvenPhase(
	t *testing.T) {
	world := underSky(t, even(skyBlue), 8)
	light := maths.Grey(2)
	world.Lamp(thing.Sun(maths.Up, light))
	bit := smokeBit(sheet(0.7, -1, 1))
	got := scattered(stage(world, 0), &bit, midway, alongZ, false)
	want := maths.Grey(0.16).Add(smokeColour.Scale(2 / (4 * math.Pi)))
	lightIs(t, "smoke under a sky and a sun", got, want, 1e-12)
}

// Anti: without the volume sky asked for, a sky adds nothing to smoke.
// The same bit under the same sun sends the same with a sky and
// without one.
func TestSmokeUnderAnUnaskedSkyTakesNoneOfIt(t *testing.T) {
	bit := smokeBit(sheet(0.7, -1, 1))
	bare := thing.NewWorld()
	bareLight := maths.Grey(2)
	bare.Lamp(thing.Sun(maths.Up, bareLight))
	skied := thing.NewWorld()
	skiedLight := maths.Grey(2)
	skied.Lamp(thing.Sun(maths.Up, skiedLight))
	skied.Sky = even(skyBlue)
	without := scattered(stage(bare, 0), &bit, midway, alongZ, false)
	with := scattered(stage(skied, 0), &bit, midway, alongZ, false)
	if with != without || !(without.R > 0.1) {
		t.Errorf("smoke sends %v under an unasked sky and %v under none",
			with, without)
	}
}
