//go:build draw && darwin

package gpu_test

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies/scrooge"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// The thin beam of these draws: a column of 0.05 m radius and no
// divergence, standing up the y axis from a metre below the origin, in a
// puff of haze round the origin, seen by one pixel straight at its axis
// from the east.
const (
	// thinRadius is the beam's launch radius, in metres.
	thinRadius = 0.05
	// thinPeak is the beam's peak strength, in every channel.
	thinPeak = 8.0
	// thinReach is how far the beam runs, in metres.
	thinReach = 8.0
	// thinLaunch is the height of the launch plane, in metres.
	thinLaunch = -1.0
	// hazeRadius is the puff's radius, in metres.
	hazeRadius = 10.0
	// hazeThick is the puff's density at its middle, per metre.
	hazeThick = 0.05
	// thinWide is the lens's field of view, in degrees.
	thinWide = 50.0
	// thinNear is how far the eye stands from the axis in the near
	// draw, in metres. Its distance squared, 1600, has a float32 step of
	// 1.2e-4, half a percent of the square of the beam's margin, 0.0225.
	thinNear = 40.0
	// thinFloor is the least a pixel that holds the beam may read. The
	// beam's light is worked: across the diameter a Gaussian of 1/e^2
	// radius 0.05 m holds 0.05 * sqrt(pi / 2) = 0.0627 m of peak; the
	// haze scatters half of a lamp's light toward the eye (the engine's
	// even phase), at its density of 0.05 at the middle; so 0.5 * 8 *
	// 0.05 * 0.0627 = 0.0125. Of that the eye keeps exp(-0.25): the
	// density falls from 0.05 to nothing over ten metres, a smoothstep
	// whose mean is a half, so 0.05 * 10 * 0.5 = 0.25. The beam keeps
	// about exp(-0.05) over its metre from the launch plane. So
	// 0.0125 * 0.78 * 0.95 = 0.0093, and the floor is a little under
	// half of that.
	thinFloor = 0.004
	// apartTolerance is how far the beam's pixels may differ between
	// the two ways of refining and be the same, as a share. The two ways
	// refine the same cells inside the beam's cylinder, and the cells
	// they differ in lie past three radii, where the Gaussian holds
	// exp(-18) = 1.5e-8 of its peak; float32 sums over a hundred steps
	// add a millionth. A thousandth is the share the draw of this beam
	// against the host's keeps in beam_test.go.
	apartTolerance = 1e-3
)

// thinWorld is the thin beam in its haze, or the haze alone.
func thinWorld(test *testing.T, withBeam bool) *thing.World {
	test.Helper()
	world := thing.NewWorld()
	if withBeam {
		beam, err := emit.NewBeam(emit.Lit{At: maths.V(0, thinLaunch, 0),
			Toward: maths.YAxis, Strength: maths.Grey(thinPeak)},
			measure.Metres(thinRadius), measure.Metres(thinReach),
			measure.Degrees(0))
		if err != nil {
			test.Fatal(err)
		}
		world.Lamp(beam)
	}
	world.AddVolume(thing.Volume{
		Medium: medium.Puff{Radius: hazeRadius, Thick: hazeThick},
		Material: &thing.Material{Colour: maths.Grey(1),
			Name: "thin beam haze"}})
	return world
}

// thinPixel is the red of the one pixel a device draws of a world from
// the east at that distance, with the scrooge switch at that value.
func thinPixel(test *testing.T, world *thing.World, distance float64,
	value string) float64 {
	test.Helper()
	test.Setenv(scrooge.Variable, value)
	eye, err := maths.View(maths.V(distance, 0, 0), maths.V(0, 0, 0),
		maths.YAxis)
	if err != nil {
		test.Fatal(err)
	}
	lens, err := film.PinholeOf(thinWide * maths.Degree)
	if err != nil {
		test.Fatal(err)
	}
	drawn, err := gpu.Sampled(world, *eye, lens, 0, 1, 1, 1)
	if errors.Is(err, metal.ErrNoDevice) {
		test.Skip("no Metal device")
	}
	if err != nil {
		test.Fatal(err)
	}
	return float64(drawn.Pix[0].R)
}

// beamIsPresent is whether a pixel of the thin beam holds the beam.
func beamIsPresent(red float64) bool { return red >= thinFloor }

// alikeApart is whether the beam's pixel is the same drawn apart as in
// boxes, as a share of the boxes' pixel.
func alikeApart(apart, boxes float64) bool {
	return math.Abs(apart-boxes) <= apartTolerance*boxes
}

// The thin beam, near, is drawn the same way with the cylinders as with
// the boxes. Through its axis the ray's stretch is 0.3 m, three radii
// each side, in both, so the two refine the same cells.
func TestAThinBeamIsDrawnTheSameApartAsInBoxesNearby(test *testing.T) {
	world := thinWorld(test, true)
	boxes := thinPixel(test, world, thinNear, "")
	apart := thinPixel(test, world, thinNear, "1")
	if !beamIsPresent(boxes) {
		test.Fatalf("boxes mode draws no beam: %g, want %g or more",
			boxes, thinFloor)
	}
	if !alikeApart(apart, boxes) {
		test.Errorf("apart draws %g, boxes %g", apart, boxes)
	}
}

// Anti: the haze with no beam is drawn as no beam, in both ways, so the
// check that the beam is present reports a draw without one.
func TestAHazeWithNoBeamIsReportedAbsent(test *testing.T) {
	world := thinWorld(test, false)
	for _, value := range []string{"", "1"} {
		if red := thinPixel(test, world, thinNear, value); beamIsPresent(
			red) {
			test.Errorf("with %q a haze with no beam reads present: %g",
				value, red)
		}
	}
}

// guideFrame is the guide scene drawn with the scrooge switch at a value.
func guideFrame(test *testing.T, scene *beamScene, value string) *frame.Frame {
	test.Helper()
	test.Setenv(scrooge.Variable, value)
	plan := scene.plan
	drawn, err := gpu.Sampled(scene.world, *scene.eye, scene.lens, 0,
		plan.Width, plan.Height, plan.Samples)
	if errors.Is(err, metal.ErrNoDevice) {
		test.Skip("no Metal device")
	}
	if err != nil {
		test.Fatal(err)
	}
	return drawn
}

// The guide scene's picture, of a beam in mist from a slant, is the
// same drawn apart as in boxes, across its whole frame.
func TestTheGuideSceneIsDrawnTheSameApartAsInBoxes(test *testing.T) {
	scene := guideScene(test)
	boxes := guideFrame(test, scene, "")
	apart := guideFrame(test, scene, "1")
	energy, differ := beamDifference(boxes, apart)
	if !(energy >= scene.plan.Minimum) {
		test.Fatalf("boxes mode draws no beam: %g", energy)
	}
	if !(differ/energy <= apartTolerance) {
		test.Errorf("apart differs from boxes by %g of %g", differ, energy)
	}
}

// Anti: the guide scene with its beam taken away holds less than the
// least energy a beam leaves, so the check that the beam is present
// reports it.
func TestTheGuideSceneWithNoBeamIsReportedAbsent(test *testing.T) {
	scene := guideScene(test)
	scene.world.Lamps = nil
	nothing := guideFrame(test, scene, "1")
	energy, _ := beamDifference(nothing, nothing)
	if energy >= scene.plan.Minimum {
		test.Errorf("a scene with no beam holds %g", energy)
	}
}
