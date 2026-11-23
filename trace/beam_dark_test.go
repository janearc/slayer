package trace

import (
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The points a beam is asked about, and its width: the beam stands up
// from the origin's x and z, a quarter of a metre in radius.
var (
	// onTheAxis is a point on the beam's axis, a metre above its launch
	// plane, where it sends full light.
	onTheAxis = maths.V(0, 0, 0)
	// farOffTheAxis is a point a metre above the launch plane and 30
	// radii, 7.5 m, off the axis: the Gaussian there is exp(-2 * 30^2) =
	// exp(-1800), under the least float64, which is nought.
	farOffTheAxis = maths.V(7.5, 0, 0)
)

// darkScene is the beam of the work tests at full strength in a fog that
// counts its askings, and the sun too if asked: staged, with the count.
func darkScene(test *testing.T, sun bool) (*staged, *int) {
	test.Helper()
	asked := 0
	beam, err := emit.NewBeam(emit.Lit{At: maths.V(0, -crossingAlong, 0),
		Toward: maths.V(0, 1, 0), Strength: maths.Grey(1)},
		measure.Metres(crossingRadius), measure.Metres(crossingReach),
		measure.Degrees(0))
	if err != nil {
		test.Fatal(err)
	}
	world := thing.NewWorld()
	if sun {
		light := maths.Grey(1)
		world.Lamp(thing.Sun(maths.Up, light))
	}
	world.Lamp(beam)
	world.AddVolume(thing.Volume{Medium: counted{asked: &asked,
		slab: slab{density: crossingSigma, box: maths.Box{
			Min: maths.V(-workHalf, -crossingHeight, -crossingHeight),
			Max: maths.V(workHalf, crossingHeight, crossingHeight)}}},
		Material: &thing.Material{Colour: maths.Grey(1)}})
	scene := stage(world, 0)
	asked = 0
	return scene, &asked
}

// shadowsCast is how often the fog is asked when each lamp the scene
// lists for a step at the point is given its shadow.
func shadowsCast(scene *staged, asked *int, point maths.Vec) int {
	var room [lampsEvery]lit
	*asked = 0
	for _, one := range scene.marched(room[:], point, maths.V(-1, 0, 0),
		byLight) {
		scene.shadowed(one, point)
	}
	return *asked
}

// A beam that sends a point no light has no shadow ray cast for it: the
// fog along its axis is not asked, whatever the step's list holds.
func TestADarkBeamCastsNoShadowRay(t *testing.T) {
	scene, asked := darkScene(t, false)
	if got := shadowsCast(scene, asked, farOffTheAxis); got != 0 {
		t.Errorf("a beam far off its axis cost %d askings of the fog", got)
	}
}

// Anti: a point on the beam's axis does cast the shadow ray, which asks
// the fog, so the count that finds none for the dark beam can find some.
func TestALitBeamCastsAShadowRay(t *testing.T) {
	scene, asked := darkScene(t, false)
	if got := shadowsCast(scene, asked, onTheAxis); got == 0 {
		t.Error("a beam on its axis cast no shadow ray")
	}
}

// A point on the beam's axis lists both lamps, the sun and the beam, for
// the step; this is the control for the list at a point off the axis.
func TestALitBeamIsInTheStepsList(t *testing.T) {
	scene, _ := darkScene(t, true)
	var room [lampsEvery]lit
	got := scene.marched(room[:], onTheAxis, maths.V(-1, 0, 0), byLight)
	if len(got) != 2 {
		t.Errorf("a point on the axis lists %d lamps, want 2", len(got))
	}
}
