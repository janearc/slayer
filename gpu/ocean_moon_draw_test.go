//go:build draw && darwin

package gpu

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// The synthetic emitter isolates transport numerics. Production plans copy
// the actual staged Moon emitter through exactly the same validated path.
func TestMetalMoonMeanInterfaceSingleScatterMatchesCPU(t *testing.T) {
	for _, phase := range []float64{0, .8} {
		t.Run(fmt.Sprintf("phase-%g", phase), func(t *testing.T) { testMetalMoonMeanInterface(t, phase) })
	}
}

func testMetalMoonMeanInterface(t *testing.T, phase float64) {
	water, err := thing.NewClearMediumWithScattering("lunar test water",
		maths.RGB(.025, .015, .008), maths.RGB(.012, .018, .02), phase, 1, 32, 10)
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
	boundary := world.Add(thing.Thing{Shape: field, Material: &thing.Material{
		Through: 1, Gloss: 1, Index: 1.333, ClearMedium: water}})
	world.Lamps = []thing.Lamp{moon}
	world.OceanSurface = &thing.OceanSurface{Boundary: boundary, Mean: maths.Vec{},
		ExtentMetres: 200, MetresPerUnit: 1, DirectMoonMeanPlane: true,
		SunRoute: thing.OceanSunRouteMoon, SunLampIndex: 0, FillLampIndex: -1}
	initial := []trace.InitialMedium{{Index: 1.333, ClearMedium: water}}
	encoded, err := encodeKeptWithInitialMedia(world, nil, opticalEye(maths.V(0, -5, 0), maths.V(0, -1, 0)), film.Pinhole{}, 0, 1, 1, nil, initial)
	if err != nil {
		t.Fatal(err)
	}
	kernel, err := metal.Open(sourceWithOpticalState(0, probeVolumes)+`
kernel void moon_probe(SLAYER_TRACE_ARGUMENTS) {
 World world=SLAYER_WORLD_INITIALIZER;
 Reached moon=ocean_moon_incident(world,world.clear_media[0],float3(0,-5,0));
 out[0]=moon.light.x; out[1]=moon.light.y; out[2]=moon.light.z;
 out[3]=world.clear_media[0].moon_light.w;
}`, "moon_probe")
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	probe, err := kernel.Grid(1, 1, 1, 4, encoded.inputs())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("incident probe=%v", probe)
	cosWater := math.Sqrt(1 - (1-.8*.8)/(1.333*1.333))
	projected := (1 - maths.Fresnel(.8, 1.333)) * .8 / cosWater
	for channel, expected := range []float64{.03 * projected * math.Exp(-.037*5/cosWater), .04 * projected * math.Exp(-.033*5/cosWater), .05 * projected * math.Exp(-.028*5/cosWater)} {
		if math.Abs(float64(probe[channel])-expected) > expected*3e-6 {
			t.Fatalf("incident channel%d=%g want%g", channel, probe[channel], expected)
		}
	}
	for _, centre := range []maths.Vec{
		maths.V((.6/1.333)*(2.5/cosWater), -2.5, 0),
		maths.V((.6/1.333)*(5/cosWater)+1.2, 1.6, 0),
	} {
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: .4}, Material: &thing.Material{Colour: maths.Grey(.5)}, Motion: thing.Still(maths.NewSubspace(centre, maths.Identity, 1))})
		shadow, err := encodeKeptWithInitialMedia(world, nil, opticalEye(maths.V(0, -5, 0), maths.V(0, -1, 0)), film.Pinhole{}, 0, 1, 1, nil, initial)
		if err != nil {
			t.Fatal(err)
		}
		got, err := kernel.Grid(1, 1, 1, 4, shadow.inputs())
		if err != nil {
			t.Fatal(err)
		}
		if got[0] != 0 || got[1] != 0 || got[2] != 0 {
			t.Fatalf("opaque source-path blocker%v leaked%v", centre, got)
		}
		world.Things = world.Things[:len(world.Things)-1]
	}
	for _, direction := range []maths.Vec{maths.V(0, -1, 0), maths.V(0, 0, -1), maths.V(.4, .9, 0)} {
		eye := opticalEye(maths.V(0, -5, 0), direction)
		cpu, err := trace.SampledWithInitialMedia(world, eye, film.Pinhole{}, 0, 1, 1, 1, initial)
		if err != nil {
			t.Fatal(err)
		}
		got, profile, err := SampledWithInitialMedia(Software, world, eye, film.Pinhole{}, 0, 1, 1, 1, initial)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("device=%q dir=%v CPU=%v Metal=%v", profile.Device, direction, cpu.At(0, 0), got.At(0, 0))
		want, pixel := cpu.At(0, 0), got.At(0, 0)
		for channel, pair := range [][2]float32{{pixel.R, want.R}, {pixel.G, want.G}, {pixel.B, want.B}} {
			if math.Abs(float64(pair[0]-pair[1])) > 3e-5*math.Abs(float64(pair[1]))+1e-10 {
				t.Fatalf("channel%d Metal%g CPU%g", channel, pair[0], pair[1])
			}
		}
		if got.At(0, 0).B <= 0 {
			t.Fatal("Moon single-scatter source was not invoked")
		}
	}
	eye := opticalEye(maths.V(99, -5, 0), maths.V(0, -1, 0))
	if _, _, err := SampledWithInitialMedia(Software, world, eye, film.Pinhole{}, 0, 1, 1, 1, initial); err == nil || !strings.Contains(err.Error(), "finite water footprint") {
		t.Fatalf("source miss silently became dark water: %v", err)
	}
}
