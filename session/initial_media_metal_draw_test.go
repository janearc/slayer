//go:build draw && darwin

package session

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// This crosses the command's drawMeasured dispatch, authored camera-stack
// transport, and the real Metal drawer. The paired no-stack draw checks that
// ordinary GPU jobs still take the unchanged path.
func TestMetalDrawMeasuredOptedInCameraMediumAndDefaultGPUPath(t *testing.T) {
	absorption := maths.RGB(.03, .015, .012)
	scattering := maths.RGB(.01, .005, .002)
	medium, err := thing.NewClearMediumWithScattering("Metal camera water",
		absorption, scattering, .7, 1, 24, 10)
	if err != nil {
		t.Fatal(err)
	}
	field, err := shape.NewHeightField([]float64{0, 0, 0, 0}, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	boundary := &thing.Thing{Shape: field, Material: &thing.Material{
		Through: 1, Gloss: 1, Index: 1.333, ClearMedium: medium}}
	world := thing.NewWorld()
	world.Things = append(world.Things, boundary)
	world.Lamps = []thing.Lamp{
		film.Spot{At: maths.V(0, -5, 0), Aim: maths.V(0, -1, 0),
			Cone: 45 * maths.Degree, Strength: maths.Grey(100)},
		film.Flood{Strength: maths.Grey(40)},
	}
	world.OceanSurface = &thing.OceanSurface{Boundary: boundary,
		Mean: maths.Vec{}, ExtentMetres: 20, MetresPerUnit: 1,
		SpotInscatter: true}
	eye := maths.Subspace{Origin: maths.V(0, -10, 0),
		Orientation: maths.LookAt(maths.V(0, -10, 0), maths.V(0, -9, 0))}
	initial := []trace.InitialMedium{{Index: 1.333, ClearMedium: medium}}
	given := asked{eye: &eye, onGPU: true, width: 1, height: 1,
		samples: 1, initialMedia: initial}

	metalFrame, _, err := drawMeasured(world, given, 0)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no Metal device")
	}
	if err != nil {
		t.Fatalf("authored high-level Metal draw: %v", err)
	}
	cpu, err := trace.SampledWithInitialMedia(world, eye, film.Pinhole{},
		0, 1, 1, 1, initial)
	if err != nil {
		t.Fatal(err)
	}
	if delta := math.Abs(float64(metalFrame.At(0, 0).R-cpu.At(0, 0).R)) +
		math.Abs(float64(metalFrame.At(0, 0).G-cpu.At(0, 0).G)) +
		math.Abs(float64(metalFrame.At(0, 0).B-cpu.At(0, 0).B)); delta > .01 {
		t.Fatalf("high-level Metal pixel %+v differs from CPU %+v (RGB L1 %g)",
			metalFrame.At(0, 0), cpu.At(0, 0), delta)
	}
	if got := metalFrame.At(0, 0); got.R <= 0 || got.G <= 0 || got.B <= 0 {
		t.Fatalf("opted-in Metal medium produced no lit pixel: %+v", got)
	}

	ordinary, _, err := drawMeasured(world, asked{eye: &eye, onGPU: true,
		width: 1, height: 1, samples: 1}, 0)
	if err != nil {
		t.Fatalf("default GPU draw without initial media: %v", err)
	}
	legacy, err := gpu.Sampled(world, eye, film.Pinhole{}, 0, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if ordinary.At(0, 0) != legacy.At(0, 0) {
		t.Fatalf("default high-level GPU pixel %+v differs from original GPU path %+v",
			ordinary.At(0, 0), legacy.At(0, 0))
	}
}
