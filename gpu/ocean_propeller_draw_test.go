//go:build draw && darwin

package gpu

import (
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
	"math"
	"testing"
)

func TestMetalMoonPropellerInterleavingMatchesCPU(t *testing.T) {
	for _, mode := range []string{"per-gap", "ray-spacing"} {
		t.Run(mode, func(t *testing.T) {
			water, err := thing.NewClearMediumWithScattering("mixed lunar water", maths.RGB(.025, .015, .008), maths.RGB(.012, .018, .02), .8, 1, 16, 20)
			if err != nil {
				t.Fatal(err)
			}
			field, err := shape.NewHeightField([]float64{0, 0, 0, 0}, 2, 200)
			if err != nil {
				t.Fatal(err)
			}
			moon, err := emit.NewDistant(maths.V(.6, .8, 0), maths.RGB(.03, .04, .05))
			if err != nil {
				t.Fatal(err)
			}
			world := thing.NewWorld()
			world.Sky = sky.Gradient{}
			boundary := world.Add(thing.Thing{Shape: field, Material: &thing.Material{Through: 1, Gloss: 1, Index: 1.333, ClearMedium: water}})
			world.Lamps = []thing.Lamp{moon}
			world.OceanSurface = &thing.OceanSurface{Boundary: boundary, Mean: maths.Vec{}, ExtentMetres: 200, MetresPerUnit: 1, DirectMoonMeanPlane: true, SunRoute: thing.OceanSunRouteMoon, SunLampIndex: 0, FillLampIndex: -1, ClearCloudWaterStepMode: mode}
			cloud, err := medium.NewClearCloud(propellerFixture(t, 5))
			if err != nil {
				t.Fatal(err)
			}
			world.AddVolume(thing.Volume{Medium: cloud, Material: &thing.Material{Colour: maths.Grey(.9)}, Motion: thing.Still(maths.NewSubspace(maths.V(0, -6, -3), maths.Identity, 1))})
			initial := []trace.InitialMedium{{Index: 1.333, ClearMedium: water}}
			eye := opticalEye(maths.V(.9, -6, -8), maths.V(0, 0, 1))
			cpu, err := trace.SampledWithInitialMedia(world, eye, film.Pinhole{}, 0, 1, 1, 1, initial)
			if err != nil {
				t.Fatal(err)
			}
			got, profile, err := SampledWithInitialMedia(Software, world, eye, film.Pinhole{}, 0, 1, 1, 1, initial)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%s device=%q CPU=%v Metal=%v", mode, profile.Device, cpu.At(0, 0), got.At(0, 0))
			a, b := cpu.At(0, 0), got.At(0, 0)
			for channel, pair := range [][2]float32{{a.R, b.R}, {a.G, b.G}, {a.B, b.B}} {
				if math.Abs(float64(pair[0]-pair[1])) > math.Abs(float64(pair[0]))*.003+1e-9 {
					t.Fatalf("mixed channel%d CPU%g Metal%g", channel, pair[0], pair[1])
				}
			}
			if b.B <= 0 {
				t.Fatal("mixed lunar source was not invoked")
			}
			encoded, err := encodeKeptWithInitialMedia(world, nil, eye, film.Pinhole{}, 0, 1, 1, nil, initial)
			if err != nil {
				t.Fatal(err)
			}
			if encoded.boxedCPU.Cells != 0 || len(encoded.boxes.bytes) != 4*len(propellerFields) {
				t.Fatal("wrapped propeller was baked instead of analytic")
			}
			config := propellerFixture(t, 5).Parameters()
			config.VisualOnset = false
			disabled, err := medium.NewPropellerCavitation(config)
			if err != nil {
				t.Fatal(err)
			}
			zeroCloud, err := medium.NewClearCloud(disabled)
			if err != nil {
				t.Fatal(err)
			}
			world.Volumes[0].Medium = zeroCloud
			zero, _, err := SampledWithInitialMedia(Software, world, eye, film.Pinhole{}, 0, 1, 1, 1, initial)
			if err != nil {
				t.Fatal(err)
			}
			world.Volumes = nil
			bare, _, err := SampledWithInitialMedia(Software, world, eye, film.Pinhole{}, 0, 1, 1, 1, initial)
			if err != nil {
				t.Fatal(err)
			}
			if zero.At(0, 0) != bare.At(0, 0) {
				t.Fatalf("zero cloud duplicated or lost water source: wrapped%v bare%v", zero.At(0, 0), bare.At(0, 0))
			}
			if math.Abs(float64(b.B-bare.At(0, 0).B)) < 1e-7 {
				t.Fatal("occupied propeller made no visible transport contribution")
			}
		})
	}
}
