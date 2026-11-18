//go:build draw && darwin

package gpu

import (
	"errors"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// A ray that never leaves a homogeneous absorbing medium travels infinitely
// far through it before reaching a mathematical sky at infinity. No sky
// radiance reaches the camera when all extinction channels are positive.
func TestMetalInitialAbsorbingMediumExtinguishesUnboundedSky(t *testing.T) {
	water, err := thing.NewClearMedium("synthetic unbounded water",
		maths.RGB(.03, .008, .004), 1)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Sky = sky.Gradient{Horizon: maths.RGB(.12, .22, .34),
		Zenith: maths.RGB(.035, .09, .18), Ground: maths.RGB(.008, .025, .055)}
	initial := []trace.InitialMedium{{Index: 1.333, ClearMedium: water}}
	for _, each := range []struct {
		name string
		way  maths.Vec
	}{
		{"horizontal", maths.V(0, 0, -1)},
		{"downward", maths.V(0, -.1, -1)},
		{"upward", maths.V(0, .1, -1)},
	} {
		t.Run(each.name, func(t *testing.T) {
			eye := opticalEye(maths.Vec{}, each.way)
			renderOpticalCase(t, each.name, world, eye, film.Pinhole{},
				initial, [3]float64{0, 0, 0}, 1e-5)
		})
	}
}

// A zero-extinction channel remains transparent even over an unbounded path;
// an off-axis object also makes this a miss in a nonempty scene.
func TestMetalUnboundedSkyPreservesClearChannels(t *testing.T) {
	water, err := thing.NewClearMedium("partly clear water", maths.RGB(.03, 0, .004), 1)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Sky = sky.Gradient{Horizon: maths.RGB(.12, .22, .34),
		Zenith: maths.RGB(.035, .09, .18), Ground: maths.RGB(.008, .025, .055)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Glow: maths.Grey(1)}, Motion: at(40, 0, 0)})
	initial := []trace.InitialMedium{{Index: 1.333, ClearMedium: water}}
	eye := opticalEye(maths.Vec{}, maths.V(0, 0, -1))
	cpu, err := trace.SampledWithInitialMedia(world, eye, film.Pinhole{}, 0, 1, 1, 1, initial)
	if err != nil {
		t.Fatal(err)
	}
	made, _, err := SampledWithInitialMedia(Software, world, eye, film.Pinhole{}, 0, 1, 1, 1, initial)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no Metal device")
	}
	if err != nil {
		t.Fatal(err)
	}
	pixel := made.At(0, 0)
	assertTransportPixel(t, "partly clear sky", pixel, cpu.At(0, 0), 1e-5)
	if pixel.R != 0 || pixel.B != 0 || pixel.G < .21 || pixel.G > .23 || pixel.Cover != 0 {
		t.Errorf("got %+v, want only the green horizon and zero coverage", pixel)
	}
}

// With no surface hit, the bounded single-scatter integration still runs
// along the visible water path; extinguishing the sky must not extinguish it.
func TestMetalUnboundedWaterRetainsSpotScattering(t *testing.T) {
	water, err := thing.NewClearMediumWithScattering("scattering water",
		maths.RGB(.02, .01, .01), maths.RGB(.01, .005, .002), .7, 1, 24, 10)
	if err != nil {
		t.Fatal(err)
	}
	field, err := shape.NewHeightField([]float64{0, 0, 0, 0}, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	boundary := &thing.Thing{Shape: field, Material: &thing.Material{
		Through: 1, Gloss: 1, Index: 1.333, ClearMedium: water}}
	world := thing.NewWorld()
	world.Things = append(world.Things, boundary)
	world.Lamps = []thing.Lamp{film.Spot{At: maths.V(0, -5, 0),
		Aim: maths.V(0, -1, 0), Cone: 45 * maths.Degree, Strength: maths.Grey(100)}}
	world.OceanSurface = &thing.OceanSurface{Boundary: boundary, Mean: maths.Vec{},
		ExtentMetres: 20, MetresPerUnit: 1, SpotInscatter: true}
	initial := []trace.InitialMedium{{Index: 1.333, ClearMedium: water}}
	eye := opticalEye(maths.V(0, -10, 0), maths.V(0, -1, 0))
	cpu, err := trace.SampledWithInitialMedia(world, eye, film.Pinhole{}, 0, 1, 1, 1, initial)
	if err != nil {
		t.Fatal(err)
	}
	made, _, err := SampledWithInitialMedia(Software, world, eye, film.Pinhole{}, 0, 1, 1, 1, initial)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no Metal device")
	}
	if err != nil {
		t.Fatal(err)
	}
	assertTransportPixel(t, "unbounded spot scattering", made.At(0, 0), cpu.At(0, 0), 3e-3)
	if made.At(0, 0).R <= 0 {
		t.Errorf("spot scattering was lost: %+v", made.At(0, 0))
	}
}
