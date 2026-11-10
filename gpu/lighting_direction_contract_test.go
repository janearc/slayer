//go:build darwin

package gpu

import (
	"math"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// referenceBlue is uniform incident radiance, independent of the lamps.
type referenceBlue struct{}

// Light returns the same known radiance in every direction.
func (referenceBlue) Light(maths.Vec) maths.Light {
	return maths.RGB(0, 0, 0.125)
}

// These are analytic references, rather than one renderer's answer used to
// judge another. A unit white Lambertian surface sends 1/pi toward the eye
// under unit normal irradiance, and zero from a lamp behind its normal.
func TestIndependentLightingSurfaceDirections(test *testing.T) {
	for _, backend := range []string{"CPU", "Metal", "Native"} {
		test.Run(backend, func(test *testing.T) {
			checkIndependentSurface(test, backend, maths.Up, false,
				maths.RGB(1/math.Pi, 0, 0))
			checkIndependentSurface(test, backend, maths.Up.Neg(), false,
				maths.Light{})
			checkIndependentSurface(test, backend, maths.Up.Neg(), true,
				maths.RGB(0, 0, 0.125))
		})
	}
}

// A separate blue sky can fill the visible face even when the red lamp is
// behind it. This distinguishes an extra source from a reversed light vector.
func checkIndependentSurface(test *testing.T, backend string,
	direction maths.Vec, sky bool, expected maths.Light,
) {
	test.Helper()
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: floor(test, 8),
		Material: thing.Diffuse("white reference", maths.Grey(1))})
	world.Lamp(independentLamp(test, direction, maths.RGB(1, 0, 0)))
	if sky {
		world.Sky = referenceBlue{}
	}
	eye := maths.Subspace{Origin: maths.V(0, 3, 0),
		Orientation: maths.LookAtUp(maths.V(0, 3, 0),
			maths.Vec{}, maths.ZAxis)}
	got := independentLightingPixel(test, backend, world, eye)
	assertIndependentLight(test, got, expected)
}

// A ceiling above the mist hides an overhead lamp, but cannot hide a lamp
// below it. The camera looks horizontally and never intersects the ceiling.
func TestIndependentLightingMistShadowDirections(test *testing.T) {
	for _, backend := range []string{"CPU", "Metal", "Native"} {
		test.Run(backend, func(test *testing.T) {
			above := independentMistPixel(test, backend, maths.Up, true)
			assertIndependentLight(test, above, maths.Light{})
			for _, below := range []bool{false, true} {
				direction := maths.Up
				if below {
					direction = direction.Neg()
				}
				lit := independentMistPixel(test, backend, direction, below)
				if !(lit.R > 0) || math.IsInf(lit.R, 0) {
					test.Fatalf("unblocked mist lost its light: %+v", lit)
				}
			}
		})
	}
}

// independentMistPixel keeps the blocker outside the camera's horizontal ray.
func independentMistPixel(test *testing.T, backend string,
	direction maths.Vec, ceiling bool,
) maths.Light {
	test.Helper()
	world := thing.NewWorld()
	world.Lamp(independentLamp(test, direction, maths.Grey(1)))
	world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1, Thick: 1},
		Material: thing.Diffuse("reference mist", maths.Grey(1))})
	if ceiling {
		world.Add(thing.Thing{Shape: floor(test, 8),
			Material: thing.Diffuse("opaque ceiling", maths.Light{}),
			Motion:   thing.Still(maths.Subspace{Origin: maths.V(0, 2, 0)})})
	}
	from := maths.V(0, 0, 4)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
	return independentLightingPixel(test, backend, world, eye)
}

// independentLamp uses the same irradiance-lamp type as the scene's Moon.
func independentLamp(test *testing.T, direction maths.Vec,
	light maths.Light,
) emit.Distant {
	test.Helper()
	lamp, err := emit.NewDistant(direction, light)
	if err != nil {
		test.Fatal(err)
	}
	return lamp
}

// Every backend is checked against the same external expectations.
func independentLightingPixel(test *testing.T, backend string,
	world *thing.World, eye maths.Subspace,
) maths.Light {
	test.Helper()
	var picture *frame.Frame
	var err error
	switch backend {
	case "CPU":
		picture, err = trace.Sampled(world, eye, film.Pinhole{}, 0, 3, 3, 1)
	case "Metal":
		picture, err = Sampled(world, eye, film.Pinhole{}, 0, 3, 3, 1)
	case "Native":
		picture, _, err = SampledAccelerated(world, eye, film.Pinhole{},
			0, 3, 3, 1)
	default:
		test.Fatal("unknown lighting reference backend", backend)
	}
	if err != nil {
		test.Fatal(err)
	}
	return picture.At(1, 1).Light()
}

// assertIndependentLight allows float32 rounding around the analytic values.
func assertIndependentLight(test *testing.T, got, expected maths.Light) {
	test.Helper()
	for channel, pair := range [][2]float64{{got.R, expected.R},
		{got.G, expected.G}, {got.B, expected.B}} {
		if !(math.Abs(pair[0]-pair[1]) <= 0.00001) {
			test.Fatalf("channel %d: got %.9g, independent reference %.9g",
				channel, pair[0], pair[1])
		}
	}
}
