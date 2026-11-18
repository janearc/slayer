//go:build draw && darwin

package gpu

import (
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

func TestMetalWakeFoamThroughOneMatchesCPUAndAddsDiffuseCoverage(t *testing.T) {
	field, err := shape.NewHeightField([]float64{0, 0, 0, 0}, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	makeWorld := func(enabled bool) *thing.World {
		material := &thing.Material{Name: "clear water", Colour: maths.Grey(.1),
			Through: 1, Gloss: 1, Index: 1.333}
		if enabled {
			material.WakeFoam = &thing.WakeFoamResponse{
				OriginEast: 5, Heading: 1.5707963267948966,
				HalfWidth: 5, Length: 48, Wavelength: 16,
				Coverage: 1, Band: 1,
				Reflectance: maths.RGB(.7, .8, .9), Satin: .8,
			}
		}
		world := thing.NewWorld()
		world.Add(thing.Thing{Shape: field, Material: material})
		world.Add(thing.Thing{Shape: shape.Plane{},
			Material: &thing.Material{Colour: maths.Grey(.3)},
			Motion:   thing.Still(maths.Subspace{Origin: maths.V(0, -10, 0)})})
		world.Lamp(thing.Sun(maths.Up, maths.Grey(5)))
		return world
	}
	eye := opticalEye(maths.V(0, 2, 0), maths.Up.Neg())
	lens := film.Pinhole{}
	controlWorld := makeWorld(false)
	candidateWorld := makeWorld(true)
	controlCPU, err := trace.Sampled(controlWorld, eye, lens, 0, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	candidateCPU, err := trace.Sampled(candidateWorld, eye, lens, 0, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if candidateCPU.At(0, 0).R <= controlCPU.At(0, 0).R+.05 {
		t.Fatalf("CPU foam did not add diffuse response: candidate %g, control %g",
			candidateCPU.At(0, 0).R, controlCPU.At(0, 0).R)
	}
	controlMetal, err := Sampled(controlWorld, eye, lens, 0, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	candidateMetal, err := Sampled(candidateWorld, eye, lens, 0, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	assertTransportPixel(t, "wake foam mixture", candidateMetal.At(0, 0),
		candidateCPU.At(0, 0), 3e-3)
	if candidateMetal.At(0, 0).R <= controlMetal.At(0, 0).R+.05 {
		t.Fatalf("Metal foam did not add diffuse response: candidate %g, control %g",
			candidateMetal.At(0, 0).R, controlMetal.At(0, 0).R)
	}
	zeroWorld := makeWorld(true)
	zeroWorld.Things[0].Material.WakeFoam.Coverage = 0
	zeroCPU, err := trace.Sampled(zeroWorld, eye, lens, 0, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	zeroMetal, err := Sampled(zeroWorld, eye, lens, 0, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if zeroCPU.At(0, 0) != controlCPU.At(0, 0) ||
		zeroMetal.At(0, 0) != controlMetal.At(0, 0) {
		t.Fatalf("zero coverage changed response: CPU%v/%v Metal%v/%v",
			zeroCPU.At(0, 0), controlCPU.At(0, 0),
			zeroMetal.At(0, 0), controlMetal.At(0, 0))
	}
}

func TestMetalWakeFoamDiffuseShareSurvivesTotalInternalReflection(t *testing.T) {
	field, err := shape.NewHeightField([]float64{0, 0, 0, 0}, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: field, Material: &thing.Material{
		Through: 1, Gloss: 1, Index: 1.333,
		WakeFoam: &thing.WakeFoamResponse{OriginEast: 5,
			Heading: 1.5707963267948966, HalfWidth: 5, Length: 48,
			Wavelength: 16, Coverage: 1, Band: 1,
			Reflectance: maths.Grey(.9), Satin: .8},
	}})
	world.Lamp(thing.Sun(maths.Up.Neg(), maths.Grey(5)))
	eye := opticalEye(maths.V(0, -2, 0), maths.V(.8, .6, 0))
	initial := []trace.InitialMedium{{Index: 1.333}}
	cpu, err := trace.SampledWithInitialMedia(world, eye, film.Pinhole{},
		0, 1, 1, 1, initial)
	if err != nil {
		t.Fatal(err)
	}
	device, profile, err := SampledWithInitialMedia(Software, world, eye,
		film.Pinhole{}, 0, 1, 1, 1, initial)
	if err != nil {
		t.Fatal(err)
	}
	if cpu.At(0, 0).R <= .03 || device.At(0, 0).R <= .03 {
		t.Fatalf("TIR erased foam diffuse share: CPU%v Metal%v",
			cpu.At(0, 0), device.At(0, 0))
	}
	assertTransportPixel(t, "foam under TIR", device.At(0, 0),
		cpu.At(0, 0), 3e-3)
	t.Logf("device=%q CPU=%v Metal=%v", profile.Device,
		cpu.At(0, 0), device.At(0, 0))
}
