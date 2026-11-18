//go:build draw && darwin

package gpu

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// TestMetalInitialClearMediumExtinctionMatchesCPU is a small headless device
// check. The controls cover the unchanged no-optics path and zero extinction;
// optical cases cover both interface directions, TIR, and a nested stack.
func TestMetalInitialClearMediumExtinctionMatchesCPU(t *testing.T) {
	lens := film.Pinhole{}
	origin := maths.Vec{}
	straight := opticalEye(origin, maths.Up)

	legacyWorld := opticalTransportWorld(nil)
	legacyCPU, err := trace.Sampled(legacyWorld, straight, lens, 0, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	legacyMetal, err := Sampled(legacyWorld, straight, lens, 0, 1, 1, 1)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no Metal device")
	}
	if err != nil {
		t.Fatalf("legacy control draw: %v", err)
	}
	assertTransportPixel(t, "legacy no-medium control", legacyMetal.At(0, 0),
		legacyCPU.At(0, 0), 2e-4)
	t.Logf("legacy no-medium control CPU=%+v Metal=%+v",
		legacyCPU.At(0, 0), legacyMetal.At(0, 0))

	zero, err := thing.NewClearMedium("synthetic-zero-extinction",
		maths.Grey(0), 3)
	if err != nil {
		t.Fatal(err)
	}
	zeroWorld := opticalTransportWorld(zero)
	zeroCPU, err := trace.Sampled(zeroWorld, straight, lens, 0, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	zeroMetal, zeroProfile, err := SampledWithInitialMedia(Software,
		zeroWorld, straight, lens, 0, 1, 1, 1, nil)
	if err != nil {
		t.Fatalf("zero-extinction optical draw: %v", err)
	}
	assertTransportPixel(t, "zero-extinction control", zeroMetal.At(0, 0),
		zeroCPU.At(0, 0), 2e-4)
	assertTransportPixel(t, "zero extinction versus legacy",
		zeroMetal.At(0, 0), legacyMetal.At(0, 0), 2e-4)
	t.Logf("zero-extinction control device=%q CPU=%+v Metal=%+v",
		zeroProfile.Device, zeroCPU.At(0, 0), zeroMetal.At(0, 0))

	waterExtinction := maths.RGB(0.08, 0.15, 0.30)
	t.Run("underwater exit", func(t *testing.T) {
		boundary, err := thing.NewClearMedium("synthetic-water-exit",
			waterExtinction, 3)
		if err != nil {
			t.Fatal(err)
		}
		camera, err := thing.NewClearMedium("synthetic-water-exit",
			waterExtinction, 3)
		if err != nil {
			t.Fatal(err)
		}
		world := opticalTransportWorld(boundary)
		initial := []trace.InitialMedium{{Index: 1.33,
			ClearMedium: camera}}
		transmission := 1 - maths.Fresnel(1, 1/1.33)
		expected := beerExpected(waterExtinction, 2*boundary.MetresPerUnit(),
			transmission)
		renderOpticalCase(t, "underwater exit", world, straight, lens,
			initial, expected, 3e-4)
	})

	t.Run("air entry", func(t *testing.T) {
		water, err := thing.NewClearMedium("synthetic-water-entry",
			waterExtinction, 3)
		if err != nil {
			t.Fatal(err)
		}
		world := opticalTransportWorld(water)
		transmission := 1 - maths.Fresnel(1, 1.33)
		expected := beerExpected(waterExtinction,
			water.MetresPerUnit(), transmission)
		renderOpticalCase(t, "air entry", world, straight, lens, nil,
			expected, 3e-4)
	})

	t.Run("total internal reflection", func(t *testing.T) {
		extinction := maths.RGB(0.05, 0.10, 0.15)
		water, err := thing.NewClearMedium("synthetic-water-tir",
			extinction, 2)
		if err != nil {
			t.Fatal(err)
		}
		world := thing.NewWorld()
		world.Add(thing.Thing{Shape: shape.Plane{},
			Material: &thing.Material{Name: "water exit", Through: 1,
				Gloss: 1, Index: 1.33, ClearMedium: water},
			Motion: thing.Still(maths.NewSubspace(maths.V(0, 2, 0),
				maths.Identity, 1))})
		world.Add(thing.Thing{Shape: shape.Plane{},
			Material: &thing.Material{Name: "reflected target",
				Glow: maths.Grey(1)},
			Motion: thing.Still(maths.NewSubspace(maths.V(0, 0, 0),
				maths.Identity, 1))})
		angle := 60 * maths.Degree
		from := maths.V(0, 1, 0)
		eye := opticalEye(from, maths.V(math.Sin(angle), math.Cos(angle), 0))
		camera, err := thing.NewClearMedium("synthetic-water-tir",
			extinction, 2)
		if err != nil {
			t.Fatal(err)
		}
		initial := []trace.InitialMedium{{Index: 1.33,
			ClearMedium: camera}}
		if math.Sin(angle) <= 1/1.33 {
			t.Fatal("fixture angle does not exceed the water-to-air critical angle")
		}
		expected := beerExpected(extinction, 12, 1)
		renderOpticalCase(t, "total internal reflection", world, eye, lens,
			initial, expected, 8e-4)
	})

	t.Run("nested water and glass", func(t *testing.T) {
		waterExtinction := maths.RGB(0.03, 0.06, 0.12)
		glassExtinction := maths.RGB(0.20, 0.10, 0.05)
		waterCamera, err := thing.NewClearMedium("synthetic-nested-water",
			waterExtinction, 2)
		if err != nil {
			t.Fatal(err)
		}
		glass, err := thing.NewClearMedium("synthetic-nested-glass",
			glassExtinction, 1)
		if err != nil {
			t.Fatal(err)
		}
		world := thing.NewWorld()
		glassBoundary := &thing.Material{Name: "glass sheet", Through: 1,
			Gloss: 1, Index: 1.5, Rank: 2, ClearMedium: glass}
		world.Add(thing.Thing{Shape: shape.Plane{}, Material: glassBoundary,
			Motion: thing.Still(maths.NewSubspace(maths.V(0, 2, 0),
				maths.Identity, 1))})
		world.Add(thing.Thing{Shape: shape.Plane{}, Material: glassBoundary,
			Motion: thing.Still(maths.NewSubspace(maths.V(0, 4, 0),
				maths.Identity, 1))})
		world.Add(thing.Thing{Shape: shape.Plane{},
			Material: &thing.Material{Name: "nested target", Glow: maths.Grey(1)},
			Motion: thing.Still(maths.NewSubspace(maths.V(0, 5, 0),
				maths.Identity, 1))})
		initial := []trace.InitialMedium{{Index: 1.33, Rank: 0,
			ClearMedium: waterCamera}}
		waterGlass := 1 - maths.Fresnel(1, 1.5/1.33)
		glassWater := 1 - maths.Fresnel(1, 1.33/1.5)
		extinction := maths.RGB(
			waterExtinction.R*6+glassExtinction.R*2,
			waterExtinction.G*6+glassExtinction.G*2,
			waterExtinction.B*6+glassExtinction.B*2)
		expected := beerExpected(extinction, 1, waterGlass*glassWater)
		renderOpticalCase(t, "nested water and glass", world, straight, lens,
			initial, expected, 1e-3)
	})
}

func TestMetalOptedInOceanSpotScatteringMatchesCPU(t *testing.T) {
	absorption := maths.RGB(.02, .01, .01)
	scattering := maths.RGB(.01, .005, .002)
	makeWorld := func(medium *thing.ClearMedium, enabled bool) *thing.World {
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
			SpotInscatter: enabled}
		return world
	}
	medium, err := thing.NewClearMediumWithScattering("Metal spot water",
		absorption, scattering, .7, 1, 24, 10)
	if err != nil {
		t.Fatal(err)
	}
	world := makeWorld(medium, true)
	initial := []trace.InitialMedium{{Index: 1.333, ClearMedium: medium}}
	eye := opticalEye(maths.V(0, -10, 0), maths.Up)
	lens := film.Pinhole{}
	cpu, err := trace.SampledWithInitialMedia(world, eye, lens, 0, 1, 1, 1, initial)
	if err != nil {
		t.Fatalf("CPU spot scattering draw: %v", err)
	}
	metalFrame, profile, err := SampledWithInitialMedia(Software, world, eye,
		lens, 0, 1, 1, 1, initial)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no Metal device")
	}
	if err != nil {
		t.Fatalf("Metal spot scattering draw: %v", err)
	}
	assertTransportPixel(t, "ocean spot single scattering",
		metalFrame.At(0, 0), cpu.At(0, 0), 3e-3)

	controlMedium, err := thing.NewClearMedium("Metal spot water",
		maths.RGB(.03, .015, .012), 1)
	if err != nil {
		t.Fatal(err)
	}
	controlWorld := makeWorld(controlMedium, false)
	control, err := trace.SampledWithInitialMedia(controlWorld, eye, lens,
		0, 1, 1, 1, []trace.InitialMedium{{Index: 1.333,
			ClearMedium: controlMedium}})
	if err != nil {
		t.Fatal(err)
	}
	for _, channel := range []struct {
		name string
		got  float32
		base float32
	}{{"red", cpu.At(0, 0).R, control.At(0, 0).R},
		{"green", cpu.At(0, 0).G, control.At(0, 0).G},
		{"blue", cpu.At(0, 0).B, control.At(0, 0).B}} {
		if !(channel.got > channel.base) {
			t.Errorf("CPU %s output %g did not exceed no-scatter control %g",
				channel.name, channel.got, channel.base)
		}
	}
	t.Logf("Metal device=%q scattered CPU=%+v Metal=%+v control=%+v",
		profile.Device, cpu.At(0, 0), metalFrame.At(0, 0), control.At(0, 0))
}

func renderOpticalCase(t *testing.T, label string, world *thing.World,
	eye maths.Subspace, lens film.Sensor, initial []trace.InitialMedium,
	expected [3]float64, tolerance float64,
) {
	t.Helper()
	cpu, err := trace.SampledWithInitialMedia(world, eye, lens, 0,
		1, 1, 1, initial)
	if err != nil {
		t.Fatalf("CPU %s draw: %v", label, err)
	}
	made, profile, err := SampledWithInitialMedia(Software, world, eye,
		lens, 0, 1, 1, 1, initial)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no Metal device")
	}
	if err != nil {
		t.Fatalf("Metal %s draw: %v", label, err)
	}
	cpuPixel, metalPixel := cpu.At(0, 0), made.At(0, 0)
	assertTransportPixel(t, label+" CPU parity", metalPixel, cpuPixel,
		tolerance)
	for index, channel := range []struct {
		name string
		got  float32
	}{{"red", metalPixel.R}, {"green", metalPixel.G},
		{"blue", metalPixel.B}} {
		if math.Abs(float64(channel.got)-expected[index]) > tolerance {
			t.Errorf("%s Metal %s = %.8f, analytic expected %.8f (tol %.1g)",
				label, channel.name, channel.got, expected[index], tolerance)
		}
	}
	if cpuPixel.Cover != 1 || metalPixel.Cover != 1 {
		t.Errorf("%s coverage CPU/Metal = %g/%g, want 1",
			label, cpuPixel.Cover, metalPixel.Cover)
	}
	t.Logf("%s device=%q CPU=%+v Metal=%+v expected RGB=(%.6f, %.6f, %.6f) tolerance=%.1g",
		label, profile.Device, cpuPixel, metalPixel, expected[0],
		expected[1], expected[2], tolerance)
}

func opticalTransportWorld(medium *thing.ClearMedium) *thing.World {
	world := thing.NewWorld()
	boundary := &thing.Material{Name: "flat interface", Through: 1,
		Gloss: 1, Index: 1.33, ClearMedium: medium}
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: boundary,
		Motion: thing.Still(maths.NewSubspace(maths.V(0, 2, 0),
			maths.Identity, 1))})
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Name: "glowing target", Glow: maths.Grey(1)},
		Motion: thing.Still(maths.NewSubspace(maths.V(0, 3, 0),
			maths.Identity, 1))})
	return world
}

func opticalEye(from, direction maths.Vec) maths.Subspace {
	return maths.NewSubspace(from, maths.LookAt(from, from.Add(direction)), 1)
}

func beerExpected(extinction maths.Light, pathMetres, interfaceTransmission float64) [3]float64 {
	return [3]float64{
		interfaceTransmission * math.Exp(-extinction.R*pathMetres),
		interfaceTransmission * math.Exp(-extinction.G*pathMetres),
		interfaceTransmission * math.Exp(-extinction.B*pathMetres),
	}
}

func assertTransportPixel(t *testing.T, label string, got, want frame.Pixel,
	tolerance float64) {
	t.Helper()
	for _, channel := range []struct {
		name string
		got  float32
		want float32
	}{{"red", got.R, want.R}, {"green", got.G, want.G},
		{"blue", got.B, want.B}, {"cover", got.Cover, want.Cover}} {
		if math.Abs(float64(channel.got-channel.want)) > tolerance {
			t.Errorf("%s %s = %.8f, want %.8f (tol %.1g)", label,
				channel.name, channel.got, channel.want, tolerance)
		}
	}
}

// A deliberately inconsistent spectral normal tilts a flat interface far
// enough to predict total internal reflection. The actual heightfield is
// horizontal, so the underwater ray must leave the water and reach the target.
// This exercises packed spectral shading normals and the separate exact
// interface normal through a real Metal draw.
func TestMetalSpectralHeightFieldClearBoundaryUsesGeometricNormal(t *testing.T) {
	field, err := shape.NewHeightField(make([]float64, 9), 3, 40)
	if err != nil {
		t.Fatal(err)
	}
	xSlope := make([]float64, 9)
	for index := range xSlope {
		xSlope[index] = 10
	}
	field, err = field.WithApproximateSpectralNormals(xSlope,
		make([]float64, 9))
	if err != nil {
		t.Fatal(err)
	}
	water := &thing.Material{Name: "water", Through: 1, Gloss: 1,
		Index: 1.33}
	target := &thing.Material{Name: "exit target", Glow: maths.Grey(1)}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: field, Material: water})
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: target,
		Motion: thing.Still(maths.NewSubspace(maths.V(0, 3, 0),
			maths.Identity, 1))})
	eye := opticalEye(maths.Vec{Y: -1}, maths.V(0.6, 0.8, 0))
	lens := film.Pinhole{}
	initial := []trace.InitialMedium{{Index: 1.33}}
	cpu, err := trace.SampledWithInitialMedia(world, eye, lens, 0,
		1, 1, 1, initial)
	if err != nil {
		t.Fatalf("CPU geometric-normal control: %v", err)
	}
	device, profile, err := SampledWithInitialMedia(Software, world, eye,
		lens, 0, 1, 1, 1, initial)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no Metal device")
	}
	if err != nil {
		t.Fatalf("Metal geometric-normal draw: %v", err)
	}
	got, want := device.At(0, 0), cpu.At(0, 0)
	assertTransportPixel(t, "spectral clear HeightField geometry", got, want,
		3e-4)
	if want.R < 0.5 || got.R < 0.5 {
		t.Fatalf("underwater ray failed to exit to glowing target: CPU=%+v Metal=%+v",
			want, got)
	}
	t.Logf("device=%q CPU=%+v Metal=%+v", profile.Device, want, got)
}

// The complementary HeightField case confirms geometry still produces true
// total internal reflection when the physical incidence exceeds critical.
func TestMetalHeightFieldGeometricNormalPreservesTotalInternalReflection(t *testing.T) {
	field, err := shape.NewHeightField(make([]float64, 9), 3, 40)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: field,
		Material: &thing.Material{Name: "water", Through: 1, Gloss: 1,
			Index: 1.33},
		Motion: thing.Still(maths.NewSubspace(maths.V(0, 2, 0),
			maths.Identity, 1))})
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Name: "reflected target", Glow: maths.Grey(1)},
		Motion: thing.Still(maths.NewSubspace(maths.V(0, -3, 0),
			maths.Identity, 1))})
	angle := 60 * maths.Degree
	eye := opticalEye(maths.Vec{}, maths.V(math.Sin(angle), math.Cos(angle), 0))
	initial := []trace.InitialMedium{{Index: 1.33}}
	cpu, err := trace.SampledWithInitialMedia(world, eye, film.Pinhole{}, 0,
		1, 1, 1, initial)
	if err != nil {
		t.Fatalf("CPU total-internal-reflection control: %v", err)
	}
	device, profile, err := SampledWithInitialMedia(Software, world, eye,
		film.Pinhole{}, 0, 1, 1, 1, initial)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no Metal device")
	}
	if err != nil {
		t.Fatalf("Metal total-internal-reflection draw: %v", err)
	}
	got, want := device.At(0, 0), cpu.At(0, 0)
	assertTransportPixel(t, "HeightField total internal reflection", got,
		want, 3e-4)
	if want.R < 0.5 || got.R < 0.5 {
		t.Fatalf("TIR ray did not reflect to the target: CPU=%+v Metal=%+v",
			want, got)
	}
	t.Logf("device=%q CPU=%+v Metal=%+v", profile.Device, want, got)
}
