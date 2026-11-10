//go:build darwin

package gpu_test

import (
	_ "embed"
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/burning-trousers/mocks/mediummock"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
	"go.uber.org/mock/gomock"
)

// firstBeamPixel is defined as the first pixel of a Go slice.
const firstBeamPixel = 0

// beamUnit is defined as the exact-to-reference ratio.
const beamUnit = 1

//go:embed testdata/beam-integral.yaml
var integralRaw []byte

type uniformBeamFog struct {
	Extent     maths.Box `json:"extent"`
	Extinction float64   `json:"extinction"`
}

func (fog uniformBeamFog) density(at maths.Vec) float64 {
	low, high := fog.Extent.Min, fog.Extent.Max
	if at.X < low.X || at.X > high.X || at.Y < low.Y || at.Y > high.Y || at.Z < low.Z || at.Z > high.Z {
		return beamZero
	}
	return fog.Extinction
}

type integralPlan struct {
	beamPlan `json:",inline" yaml:",inline"`
	Fog      uniformBeamFog `json:"fog"`
	Views    []struct {
		Centre   float64 `json:"centre"`
		Expected float64 `json:"expected"`
	} `json:"views"`
}

func TestGuideBeamIntegrationMatchesAnIndependentGaussianIntegral(test *testing.T) {
	var plan integralPlan
	if err := yamlfile.Read("testdata/beam-integral.yaml", integralRaw, &plan); err != nil {
		test.Fatal(err)
	}
	lens, err := film.PinholeOf(plan.Wide * maths.Degree)
	if err != nil {
		test.Fatal(err)
	}
	eye, err := maths.View(plan.Eye, plan.Target, maths.YAxis)
	if err != nil {
		test.Fatal(err)
	}
	for _, view := range plan.Views {
		world := plan.world(test, view.Centre)
		cpu, err := trace.Sampled(world, *eye, lens, beamZero, plan.Width, plan.Height, plan.Samples)
		if err != nil {
			test.Fatal(err)
		}
		assertBeamIntegral(test, cpu, view.Expected, plan.Tolerance)
		device, err := gpu.Sampled(world, *eye, lens, beamZero, plan.Width, plan.Height, plan.Samples)
		if errors.Is(err, metal.ErrNoDevice) {
			test.Skip("no Metal device")
		}
		if err != nil {
			test.Fatal(err)
		}
		assertBeamIntegral(test, device, view.Expected, plan.Tolerance)
		test.Logf("beam centre %g, integral %.9g; CPU %.9g Metal %.9g", view.Centre, view.Expected, cpu.Pix[firstBeamPixel].R, device.Pix[firstBeamPixel].R)
	}
}

func (plan integralPlan) world(test *testing.T, centre float64) *thing.World {
	test.Helper()
	point := plan.Source
	point.At.X = centre
	beam, err := emit.NewBeam(point, measure.Metres(plan.Radius),
		measure.Metres(plan.Reach), measure.Degrees(plan.Divergence))
	if err != nil {
		test.Fatal(err)
	}
	world := thing.NewWorld()
	world.Lamp(beam)
	fog := mediummock.NewMockMedium(gomock.NewController(test))
	fog.EXPECT().Bounds().Return(plan.Fog.Extent).AnyTimes()
	fog.EXPECT().Density(gomock.Any()).DoAndReturn(plan.Fog.density).AnyTimes()
	world.AddVolume(thing.NewVolume(fog, plan.Volume.Material))
	return world
}

func assertBeamIntegral(test *testing.T, picture *frame.Frame, expected, tolerance float64) {
	test.Helper()
	pixel := picture.Pix[firstBeamPixel]
	for _, channel := range []float32{pixel.R, pixel.G, pixel.B} {
		if !(math.Abs(float64(channel)/expected-beamUnit) <= tolerance) {
			test.Fatal("narrow beam disagrees with analytic transport", channel, expected)
		}
	}
}
