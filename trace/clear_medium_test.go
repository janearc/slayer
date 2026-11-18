package trace

import (
	"math"
	"sync/atomic"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func opticalWater(t *testing.T, name string, extinction maths.Light,
	metresPerUnit float64) *thing.ClearMedium {
	t.Helper()
	medium, err := thing.NewClearMedium(name, extinction, metresPerUnit)
	if err != nil {
		t.Fatal(err)
	}
	return medium
}

func TestParticipatingClearMediumIntegratesOneOutgoingPathAgainstAnalyticSource(t *testing.T) {
	absorb := maths.RGB(0.1, 0.2, 0.3)
	scatter := maths.RGB(0.2, 0.3, 0.4)
	phase := 0.35
	medium, err := thing.NewClearMediumWithScattering("synthetic-source",
		absorb, scatter, phase, 1, 7, 5)
	if err != nil {
		t.Fatal(err)
	}
	const distance = 2.0
	incident := maths.RGB(2, 3, 4)
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Name: "black endpoint"},
		Motion:   resting(maths.V(0, distance, 0))})
	world.Lamp(thing.Sun(maths.V(1, 0, 0), incident))
	got, err := RenderWithInitialMedia(world,
		mediumEye(maths.Vec{}, maths.Up), film.Pinhole{}, 0, 1, 1,
		[]InitialMedium{{Index: 1, ClearMedium: medium}})
	if err != nil {
		t.Fatal(err)
	}
	cosine := 0.0 // lamp is along +x; camera path is along +y
	phaseValue := gas.Gas{Phase: phase}.PhaseAt(cosine)
	table, err := gas.NewVisibleTable([]gas.VisibleSample{
		{Nanometres: 465, Absorb: absorb.B, Scatter: scatter.B},
		{Nanometres: 550, Absorb: absorb.G, Scatter: scatter.G},
		{Nanometres: 610, Absorb: absorb.R, Scatter: scatter.R},
	}, phase)
	if err != nil {
		t.Fatal(err)
	}
	pixel := got.At(0, 0)
	for _, channel := range []struct {
		name           string
		value, a, s, l float64
		wavelength     float64
		pick           func(maths.Light) float64
	}{{"red", float64(pixel.R), absorb.R, scatter.R, incident.R,
		610, func(light maths.Light) float64 { return light.R }},
		{"green", float64(pixel.G), absorb.G, scatter.G, incident.G,
			550, func(light maths.Light) float64 { return light.G }},
		{"blue", float64(pixel.B), absorb.B, scatter.B, incident.B,
			465, func(light maths.Light) float64 { return light.B }}} {
		total := channel.a + channel.s
		want := channel.l * phaseValue * channel.s *
			(1 - math.Exp(-total*distance)) / total
		oneBandIncident := maths.Light{}
		switch channel.name {
		case "red":
			oneBandIncident.R = channel.l
		case "green":
			oneBandIncident.G = channel.l
		case "blue":
			oneBandIncident.B = channel.l
		}
		fromGas, err := table.SingleScatter(channel.wavelength,
			distance, oneBandIncident, cosine)
		if err != nil {
			t.Fatal(err)
		}
		if gasWant := channel.pick(fromGas); !within(gasWant, want, 1e-12) {
			t.Errorf("%s analytic gas primitive = %.9g, independent formula %.9g", channel.name, gasWant, want)
		}
		if !within(channel.value, want, 2e-6) {
			t.Errorf("%s source integral = %.9g, want %.9g", channel.name,
				channel.value, want)
		}
	}
}

func TestOceanSpotInscatterSelectsSpotsAndLeavesFloodDirectOnly(t *testing.T) {
	medium, err := thing.NewClearMediumWithScattering("spot-water",
		maths.RGB(.02, .01, .01), maths.RGB(.01, .005, .002),
		.7, 1, 12, 8)
	if err != nil {
		t.Fatal(err)
	}
	spot := film.Spot{At: maths.V(0, -2, 0), Aim: maths.V(0, -1, 0),
		Strength: maths.Grey(20), Cone: 60 * maths.Degree}
	flood := film.Flood{Strength: maths.Grey(100)}
	ray := maths.Ray{At: maths.V(0, -8, 0), Dir: maths.Up}
	onlySpot := &staged{lamps: []thing.Lamp{spot}}
	withFlood := &staged{lamps: []thing.Lamp{spot, flood}}
	got := clearInscatterSources(onlySpot, ray, 4, medium, true)
	withDirectFlood := clearInscatterSources(withFlood, ray, 4, medium, true)
	if got == (maths.Light{}) {
		t.Fatal("submerged spot added no bounded in-scattering")
	}
	if withDirectFlood != got {
		t.Fatalf("directionless flood entered the scatter source: spot %v, spot plus flood %v", got, withDirectFlood)
	}
}

func TestClearMediumPhaseZeroRetainsRayleighMeaning(t *testing.T) {
	rayleigh := gas.Gas{Phase: 0}.PhaseAt(0)
	wantRayleigh := 3 / (16 * math.Pi)
	isotropic := thing.IsotropicPhase()
	if !within(rayleigh, wantRayleigh, 1e-15) {
		t.Fatalf("phase zero = %.16g, want Rayleigh %.16g", rayleigh, wantRayleigh)
	}
	if rayleigh == isotropic {
		t.Fatalf("phase zero unexpectedly equals isotropic HG density %.16g", isotropic)
	}
}

func TestZeroScatteringKeepsLegacyClearMediumPictureBitForBit(t *testing.T) {
	extinction := maths.RGB(0.1, 0.2, 0.3)
	legacy, err := thing.NewClearMedium("legacy-picture", extinction, 1)
	if err != nil {
		t.Fatal(err)
	}
	participating, err := thing.NewClearMediumWithScattering("zero-scatter-picture",
		extinction, maths.Light{}, 0.7, 1, 12, 30)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Glow: maths.RGB(0.8, 0.6, 0.4)},
		Motion:   resting(maths.V(0, 4, 0))})
	render := func(medium *thing.ClearMedium) frame.Pixel {
		t.Helper()
		made, err := RenderWithInitialMedia(world,
			mediumEye(maths.Vec{}, maths.Up), film.Pinhole{}, 0, 1, 1,
			[]InitialMedium{{Index: 1, ClearMedium: medium}})
		if err != nil {
			t.Fatal(err)
		}
		return made.At(0, 0)
	}
	if got, want := render(participating), render(legacy); got != want {
		t.Fatalf("zero-scattering opt-in changed the legacy image: got %+v want %+v", got, want)
	}
}

func TestZeroExtinctionNamedBoundariesPreserveNoMediumPicture(t *testing.T) {
	zero, err := thing.NewClearMedium("zero-extinction", maths.Light{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	render := func(named bool) frame.Pixel {
		t.Helper()
		world := thing.NewWorld()
		entry := &thing.Material{Name: "entry", Through: 1, Gloss: 1, Index: 1.33}
		exit := &thing.Material{Name: "exit", Through: 1, Gloss: 1, Index: 1.33}
		if named {
			entry.ClearMedium, exit.ClearMedium = zero, zero
		}
		world.Add(thing.Thing{Shape: shape.Plane{}, Material: entry})
		world.Add(thing.Thing{Shape: shape.Plane{}, Material: exit,
			Motion: resting(maths.V(0, 2, 0))})
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.25},
			Material: &thing.Material{Glow: maths.Grey(1)},
			Motion:   resting(maths.V(0, 3, 0))})
		made, err := Render(world, mediumEye(maths.V(0, -1, 0), maths.Up),
			film.Pinhole{}, 0, 1, 1)
		if err != nil {
			t.Fatal(err)
		}
		return made.At(0, 0)
	}
	if got, want := render(true), render(false); got != want {
		t.Fatalf("zero-extinction named boundaries changed legacy pixels: got %+v want %+v", got, want)
	}
}

func TestParticipatingClearMediumHonoursIndependentStepAndDistanceBudgets(t *testing.T) {
	makeMedium := func(name string, steps int, maxDistance float64) *thing.ClearMedium {
		t.Helper()
		made, err := thing.NewClearMediumWithScattering(name,
			maths.RGB(0.1, 0.2, 0.3), maths.RGB(0.2, 0.3, 0.4),
			0, 1, steps, maxDistance)
		if err != nil {
			t.Fatal(err)
		}
		return made
	}
	worldFor := func(distance float64) *thing.World {
		world := thing.NewWorld()
		world.Add(thing.Thing{Shape: shape.Plane{},
			Material: &thing.Material{Name: "black endpoint"},
			Motion:   resting(maths.V(0, distance, 0))})
		world.Lamp(thing.Sun(maths.V(1, 0, 0), maths.Grey(1)))
		return world
	}
	render := func(medium *thing.ClearMedium, distance float64) frame.Pixel {
		t.Helper()
		got, err := RenderWithInitialMedia(worldFor(distance),
			mediumEye(maths.Vec{}, maths.Up), film.Pinhole{}, 0, 1, 1,
			[]InitialMedium{{Index: 1, ClearMedium: medium}})
		if err != nil {
			t.Fatal(err)
		}
		return got.At(0, 0)
	}
	coarse := render(makeMedium("coarse", 1, 20), 8)
	fine := render(makeMedium("fine", 16, 20), 8)
	if coarse != fine {
		t.Errorf("constant-illumination integral changed with step count: coarse=%+v fine=%+v", coarse, fine)
	}
	capped := render(makeMedium("capped", 4, 2), 8)
	want := render(makeMedium("two-metre", 8, 2), 2)
	if capped.R != want.R || capped.G != want.G || capped.B != want.B {
		t.Errorf("source beyond the explicit 2m path cap contributed: got %+v want %+v", capped, want)
	}
}

func TestParticipatingClearMediumRefusesUnmodeledLightPaths(t *testing.T) {
	water := opticalScatteringWater(t, "synthetic-refusal")
	checks := map[string]func(*thing.World){
		"clear boundary": func(world *thing.World) {
			world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
				Through: 1, Gloss: 1, Index: 1.33}})
		},
		"participating clear boundary": func(world *thing.World) {
			world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
				Through: 1, Gloss: 1, Index: 1.33, ClearMedium: water}})
		},
		"volume": func(world *thing.World) {
			world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1, Thick: 1}})
		},
		"sky":  func(world *thing.World) { world.Sky = even(maths.Grey(1)) },
		"haze": func(world *thing.World) { world.Haze = thing.Haze{Density: 0.1} },
		"built-in ambient flood": func(world *thing.World) {
			world.Lamp(film.Flood{Strength: maths.Grey(1)})
		},
		"too many lamps": func(world *thing.World) {
			for range maxClearInscatterLamps + 1 {
				world.Lamp(thing.Sun(maths.V(1, 0, 0), maths.Grey(1)))
			}
		},
		"field": func(world *thing.World) {
			world.AddField(thing.Field{Mass: 1, Reach: 40})
		},
	}
	for name, add := range checks {
		t.Run(name, func(t *testing.T) {
			world := thing.NewWorld()
			add(world)
			_, err := RenderWithInitialMedia(world,
				mediumEye(maths.Vec{}, maths.Up), film.Pinhole{}, 0, 1, 1,
				[]InitialMedium{{Index: 1.33, ClearMedium: water}})
			if err == nil {
				t.Fatal("unsupported incoming-light path was accepted")
			}
		})
	}
}

type directionlessInscatterLamp struct{}

func (directionlessInscatterLamp) Light(_, _ maths.Vec) (maths.Vec,
	float64, maths.Light) {
	return maths.Vec{}, math.Inf(1), maths.Grey(1)
}

func TestParticipatingClearMediumRefusesCustomAmbientLampAtRuntime(t *testing.T) {
	world := thing.NewWorld()
	world.Lamp(directionlessInscatterLamp{})
	_, err := RenderWithInitialMedia(world,
		mediumEye(maths.Vec{}, maths.Up), film.Pinhole{}, 0, 1, 1,
		[]InitialMedium{{Index: 1, ClearMedium: opticalScatteringWater(t, "custom-ambient")}})
	if err == nil {
		t.Fatal("custom directionless lamp was silently omitted from in-scattering")
	}
}

func TestParticipatingClearMediumRefusesFiniteNumericOverflow(t *testing.T) {
	t.Run("unrepresentable output light", func(t *testing.T) {
		medium, err := thing.NewClearMediumWithScattering("large-finite-light",
			maths.Light{}, maths.Grey(1e300), 0.5, 1, 4, 1)
		if err != nil {
			t.Fatal(err)
		}
		world := thing.NewWorld()
		world.Lamp(thing.Sun(maths.V(1, 0, 0), maths.Grey(1e300)))
		if _, err := RenderWithInitialMedia(world,
			mediumEye(maths.Vec{}, maths.Up), film.Pinhole{}, 0, 1, 1,
			[]InitialMedium{{Index: 1, ClearMedium: medium}}); err == nil {
			t.Fatal("finite inputs silently produced an unrepresentable float32 pixel")
		}
	})
	t.Run("overflowing world parameter", func(t *testing.T) {
		medium, err := thing.NewClearMediumWithScattering("extreme-world-unit",
			maths.Light{}, maths.Grey(1e-308), 0.5, 1e-308, 4, 1e308)
		if err != nil {
			t.Fatal(err)
		}
		world := thing.NewWorld()
		world.Lamp(thing.Sun(maths.V(1, 0, 0), maths.Grey(1)))
		if _, err := RenderWithInitialMedia(world,
			mediumEye(maths.Vec{}, maths.Up), film.Pinhole{}, 0, 1, 1,
			[]InitialMedium{{Index: 1, ClearMedium: medium}}); err == nil {
			t.Fatal("finite path budget silently overflowed its world-ray parameter")
		}
	})
}

func TestParticipatingClearMediumRefusesInvalidIntegrationDistanceAndScale(t *testing.T) {
	medium := opticalScatteringWater(t, "invalid-integration-input")
	validRay := maths.Ray{Dir: maths.Up}
	checks := []struct {
		name     string
		ray      maths.Ray
		distance float64
		want     uint32
	}{
		{name: "zero distance is a valid empty segment", ray: validRay,
			distance: 0},
		{name: "negative distance", ray: validRay, distance: -1,
			want: clearInscatterNumericalOverflow},
		{name: "NaN distance", ray: validRay, distance: math.NaN(),
			want: clearInscatterNumericalOverflow},
		{name: "zero ray scale", ray: maths.Ray{}, distance: 1,
			want: clearInscatterNumericalOverflow},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			fault := &atomic.Uint32{}
			scene := &staged{
				lamps:               []thing.Lamp{thing.Sun(maths.V(1, 0, 0), maths.Grey(1))},
				clearInscatterFault: fault,
			}
			_ = clearInscatter(scene, check.ray, check.distance, medium)
			if got := fault.Load(); got != check.want {
				t.Fatalf("fault = %d, want %d", got, check.want)
			}
		})
	}
}

func opticalScatteringWater(t *testing.T, name string) *thing.ClearMedium {
	t.Helper()
	medium, err := thing.NewClearMediumWithScattering(name,
		maths.RGB(0.1, 0.2, 0.3), maths.RGB(0.2, 0.3, 0.4),
		0, 1, 4, 10)
	if err != nil {
		t.Fatal(err)
	}
	return medium
}

func TestClearMediumDoesNotChangeLegacyRenderWithoutOptics(t *testing.T) {
	world := mediumWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.25}, Material: &thing.Material{Glow: maths.Grey(1)}, Motion: resting(maths.V(0, 3, 0))})
	eye := mediumEye(maths.V(0, -1, 0), maths.Up)
	legacy, err := Render(world, eye, film.Pinhole{}, 0, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	withEmptyOptics, err := RenderWithInitialMedia(world, eye, film.Pinhole{}, 0, 1, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := withEmptyOptics.At(0, 0), legacy.At(0, 0); got != want {
		t.Fatalf("empty optical state changed pixel: got %+v, want %+v", got, want)
	}
}

func TestClearMediumSkyCoverageAndZeroExtinction(t *testing.T) {
	world := thing.NewWorld()
	world.Sky = even(maths.RGB(0.7, 0.5, 0.2))
	eye := mediumEye(maths.V(0, 0, 0), maths.Up)
	baseline, err := Render(world, eye, film.Pinhole{}, 0, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	zero := opticalWater(t, "zero-water", maths.Grey(0), 1)
	unchanged, err := RenderWithInitialMedia(world, eye, film.Pinhole{}, 0,
		1, 1, []InitialMedium{{Index: 1.33, ClearMedium: zero}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := unchanged.At(0, 0), baseline.At(0, 0); got != want {
		t.Fatalf("zero extinction changed unbounded sky: got %+v, want %+v", got, want)
	}
	if got := unchanged.At(0, 0).Cover; got != 0 {
		t.Fatalf("unbounded sky with zero extinction has coverage %g, want 0", got)
	}

	redOnly := opticalWater(t, "red-only-water", maths.RGB(0.1, 0, 0), 1)
	filtered, err := RenderWithInitialMedia(world, eye, film.Pinhole{}, 0,
		1, 1, []InitialMedium{{Index: 1.33, ClearMedium: redOnly}})
	if err != nil {
		t.Fatal(err)
	}
	got := filtered.At(0, 0)
	if got.R != 0 || got.G != float32(0.5) || got.B != float32(0.2) || got.Cover != 0 {
		t.Fatalf("zero-extinction channels over unbounded sky = %+v, want R=0 G=0.5 B=0.2 Cover=0", got)
	}

	allBands := opticalWater(t, "all-bands-water", maths.Grey(0.1), 1)
	opaqueSky, err := RenderWithInitialMedia(world, eye, film.Pinhole{}, 0,
		1, 1, []InitialMedium{{Index: 1.33, ClearMedium: allBands}})
	if err != nil {
		t.Fatal(err)
	}
	if got := opaqueSky.At(0, 0); got.R != 0 || got.G != 0 || got.B != 0 || got.Cover != 1 {
		t.Fatalf("fully extinguished unbounded sky = %+v, want black with full optical coverage", got)
	}
}

func TestClearMediumDistanceUsesWorldRayParameterAndDirectionLength(t *testing.T) {
	water := opticalWater(t, "scaled-ray-water", maths.Grey(0.1), 1)
	input := frame.Pixel{R: 1, G: 1, B: 1}
	ray := maths.Ray{At: maths.V(0, -5, 0), Dir: maths.V(0, 2, 0)}
	got := attenuateClearSegment(input, water, ray, 3)
	want := math.Exp(-0.1 * 6) // ray parameter 3 times |Dir|=2
	if !within(float64(got.R), want, 1e-7) || !within(float64(got.Cover), 1-want, 1e-7) {
		t.Fatalf("scaled-ray attenuation = %+v, want RGB=%g Cover=%g", got, want, 1-want)
	}

	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Motion: thing.Still(maths.Subspace{Scale: 2})})
	placed := place(world, 0)
	unitRay := maths.Ray{At: maths.V(0, -5, 0), Dir: maths.Up}
	distance, hit := placed[0].meet(unitRay, 0, 10)
	if !hit {
		t.Fatal("scaled sphere did not intersect the unit world ray")
	}
	point := unitRay.Point(distance)
	if !within(distance, 3, 1e-12) || !within(point.Y, -2, 1e-12) {
		t.Fatalf("scaled-object intersection returned world parameter %g at %v; want t=3, point y=-2", distance, point)
	}
}

func TestClearMediumAttenuatesTheActualRefractedPath(t *testing.T) {
	world := thing.NewWorld()
	water := opticalWater(t, "test-water", maths.RGB(0.2, 0.1, 0.05), 2)
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Name: "water boundary", Through: 1, Gloss: 1, Index: 1.33, ClearMedium: water}})
	from := maths.V(0, -1, 0)
	direction := maths.V(math.Sin(maths.Degree*30), math.Cos(maths.Degree*30), 0)
	entryX := direction.X / direction.Y
	// refracted uses the inverse index ratio internally: entering water
	// reduces the ray's transverse component by 1.33.
	sineInWater := direction.X / 1.33
	cosineInWater := math.Sqrt(1 - sineInWater*sineInWater)
	// The ray reaches the sphere's near surface, not its center. The center
	// is three metres vertically beyond the interface and lies on the
	// refracted ray, so subtract the 0.25 m sphere radius from that path.
	pathToTarget := 3/cosineInWater - 0.25
	target := maths.V(entryX+3*sineInWater/cosineInWater, 3, 0)
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.25}, Material: &thing.Material{Glow: maths.Grey(1)}, Motion: resting(target)})
	got, err := Render(world, mediumEye(from, direction), film.Pinhole{}, 0, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	metres := pathToTarget * water.MetresPerUnit()
	fresnel := 1 - maths.Fresnel(math.Cos(maths.Degree*30), 1.33)
	want := maths.RGB(fresnel*math.Exp(-0.2*metres), fresnel*math.Exp(-0.1*metres), fresnel*math.Exp(-0.05*metres))
	for channel, pair := range map[string][2]float64{"red": {float64(got.At(0, 0).R), want.R}, "green": {float64(got.At(0, 0).G), want.G}, "blue": {float64(got.At(0, 0).B), want.B}} {
		if !within(pair[0], pair[1], 3e-6) {
			t.Errorf("%s bent-path transmission = %.9g, want %.9g", channel, pair[0], pair[1])
		}
	}
}

func TestClearMediumUsesFloat64HitDistanceNotPixelDepth(t *testing.T) {
	world := thing.NewWorld()
	water := opticalWater(t, "long-water", maths.Grey(1e-5), 1)
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
		Through: 1, Gloss: 1, Index: 1.33, ClearMedium: water,
	}})
	targetDistance := 1_000_000.07
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Glow: maths.Grey(1)},
		Motion:   resting(maths.V(0, targetDistance, 0))})
	if float64(float32(targetDistance)) == targetDistance {
		t.Fatal("fixture distance is exactly representable in pixel depth")
	}
	got, err := Render(world, mediumEye(maths.V(0, -1, 0), maths.Up),
		film.Pinhole{}, 0, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	distance := targetDistance
	fresnel := 1 - maths.Fresnel(1, 1.33)
	want := fresnel * math.Exp(-1e-5*distance)
	if !within(float64(got.At(0, 0).R), want, 2e-9) {
		t.Fatalf("long-path transmission = %.12g, want %.12g using float64 distance %.12g (pixel depth %.12g)",
			got.At(0, 0).R, want, distance, float32(distance))
	}
}

func TestClearMediumNestedRankAndExitRestoreTheLowerMedium(t *testing.T) {
	world := thing.NewWorld()
	water := opticalWater(t, "test-water", maths.RGB(0.1, 0.2, 0.3), 1)
	glass := opticalWater(t, "test-glass", maths.RGB(0.4, 0.3, 0.2), 1)
	glassBoundary := &thing.Material{Name: "glass boundary", Through: 1, Gloss: 1, Index: 1.5, Rank: 2, ClearMedium: glass}
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: glassBoundary})
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: glassBoundary, Motion: resting(maths.V(0, 2, 0))})
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Glow: maths.Grey(1)}, Motion: resting(maths.V(0, 3, 0))})
	from := maths.V(0, -1, 0)
	initial := []InitialMedium{{Index: 1.33, Rank: 0, ClearMedium: water}}
	got, err := RenderWithInitialMedia(world, mediumEye(from, maths.Up), film.Pinhole{}, 0, 1, 1, initial)
	if err != nil {
		t.Fatal(err)
	}
	waterGlass := 1 - maths.Fresnel(1, 1.5/1.33)
	glassWater := 1 - maths.Fresnel(1, 1.33/1.5)
	want := maths.RGB(waterGlass*glassWater*math.Exp(-0.1*2-0.4*2), waterGlass*glassWater*math.Exp(-0.2*2-0.3*2), waterGlass*glassWater*math.Exp(-0.3*2-0.2*2))
	for channel, pair := range map[string][2]float64{"red": {float64(got.At(0, 0).R), want.R}, "green": {float64(got.At(0, 0).G), want.G}, "blue": {float64(got.At(0, 0).B), want.B}} {
		if !within(pair[0], pair[1], 3e-6) {
			t.Errorf("%s nested transmission = %.9g, want %.9g", channel, pair[0], pair[1])
		}
	}
}

func TestInitialOpticalMediumAttenuatesBeforeItsExitBoundary(t *testing.T) {
	world := thing.NewWorld()
	boundaryWater := opticalWater(t, "test-water", maths.RGB(0.2, 0.1, 0.05), 3)
	initialWater := opticalWater(t, "test-water", maths.RGB(0.2, 0.1, 0.05), 3)
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Name: "water exit", Through: 1, Gloss: 1, Index: 1.33, ClearMedium: boundaryWater}, Motion: resting(maths.V(0, 2, 0))})
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Glow: maths.Grey(1)}, Motion: resting(maths.V(0, 3, 0))})
	initial := []InitialMedium{{Index: 1.33, ClearMedium: initialWater}}
	got, err := RenderWithInitialMedia(world, mediumEye(maths.V(0, 0, 0), maths.Up), film.Pinhole{}, 0, 1, 1, initial)
	if err != nil {
		t.Fatal(err)
	}
	fresnel := 1 - maths.Fresnel(1, 1/1.33)
	want := maths.RGB(fresnel*math.Exp(-0.2*6), fresnel*math.Exp(-0.1*6), fresnel*math.Exp(-0.05*6))
	for channel, pair := range map[string][2]float64{"red": {float64(got.At(0, 0).R), want.R}, "green": {float64(got.At(0, 0).G), want.G}, "blue": {float64(got.At(0, 0).B), want.B}} {
		if !within(pair[0], pair[1], 3e-6) {
			t.Errorf("%s initial-medium transmission = %.9g, want %.9g", channel, pair[0], pair[1])
		}
	}
}

func TestClearMediumRejectsDistinctIDsAtSameIndexAndRank(t *testing.T) {
	one := opticalWater(t, "first-water", maths.Grey(0.1), 1)
	two := opticalWater(t, "second-water", maths.Grey(0.1), 1)
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
		Through: 1, Index: 1.33, Rank: 0, ClearMedium: one,
	}})
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
		Through: 1, Index: 1.33, Rank: 0, ClearMedium: two,
	}, Motion: resting(maths.V(0, 2, 0))})
	if _, err := newView(world, mediumEye(maths.V(0, -1, 0), maths.Up),
		film.Pinhole{}, 0, 1, 1, 1); err == nil {
		t.Fatal("distinct medium IDs at one legacy index/rank stack key were accepted")
	}
}

func TestClearMediumRejectsAmbiguousIDsAndUnsupportedVolumes(t *testing.T) {
	one := opticalWater(t, "same-water", maths.RGB(0.1, 0.2, 0.3), 1)
	two := opticalWater(t, "same-water", maths.RGB(0.2, 0.2, 0.3), 1)
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Through: 1, Index: 1.33, ClearMedium: one}})
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Through: 1, Index: 1.33, ClearMedium: two}, Motion: resting(maths.V(0, 1, 0))})
	if _, err := newView(world, mediumEye(maths.V(0, -1, 0), maths.Up), film.Pinhole{}, 0, 1, 1, 1); err == nil {
		t.Fatal("one optical medium ID with different coefficients was accepted")
	}

	world = thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Through: 1, Index: 1.33, ClearMedium: one}})
	world.AddVolume(thing.NewVolume(medium.Puff{Radius: 1, Thick: 0.2}, nil))
	if _, err := newView(world, mediumEye(maths.V(0, -1, 0), maths.Up), film.Pinhole{}, 0, 1, 1, 1); err == nil {
		t.Fatal("mixed clear optical medium and volume transport was accepted")
	}
}

func TestClearCloudComposesWaterAndLocalSingleScatterBySegment(t *testing.T) {
	const density, waterSigma, slabNear, slabFar = 0.4, 0.2, 2.0, 4.0
	field, err := medium.NewClearCloud(plate(density, slabNear, slabFar))
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: field,
		Material: &thing.Material{Name: "cavitation cloud",
			Glow: maths.Grey(1)}})
	scene := stage(world, 0)
	water := opticalWater(t, "test-water", maths.Grey(waterSigma), 1)
	ray := maths.Ray{At: maths.V(0, 0, 0), Dir: maths.V(0, 0, 1)}
	const far = 6.0
	cloudOnly := through(scene, ray, far)
	combined := throughClear(scene, look{ray: ray}, far, water)
	if !within(combined.keep.R, cloudOnly.keep.R*math.Exp(-waterSigma*far), 1e-12) {
		t.Errorf("cloud and water transmission = %g, want independent product %g", combined.keep.R, cloudOnly.keep.R*math.Exp(-waterSigma*far))
	}
	if !(combined.added.R > 0 && combined.added.R < cloudOnly.added.R) {
		t.Fatalf("water did not attenuate locally scattered source: combined %v, cloud-only %v", combined.added, cloudOnly.added)
	}
	// The source is born between 2m and 4m. It must not receive the
	// full 6m background attenuation; bound it by those analytic paths.
	if !(combined.added.R > cloudOnly.added.R*math.Exp(-waterSigma*far) &&
		combined.added.R < cloudOnly.added.R*math.Exp(-waterSigma*slabNear)) {
		t.Errorf("local source was not weighted by its sampled water path: got %g, full-path bounds (%g,%g)", combined.added.R, cloudOnly.added.R*math.Exp(-waterSigma*far), cloudOnly.added.R*math.Exp(-waterSigma*slabNear))
	}

	// With a constant authored glow, each volume step has a known midpoint
	// and Beer factor. This checks the outgoing water leg at the actual
	// source depths, including cloud self-attenuation from nearer steps.
	glowing := thing.NewWorld()
	glowing.AddVolume(thing.Volume{Medium: field,
		Material: &thing.Material{Name: "constant cloud source",
			Glow: maths.Grey(1)}})
	glowingScene := stage(glowing, 0)
	glowingPass := throughClear(glowingScene, look{ray: ray}, far, water)
	stepLength := (slabFar - slabNear) / float64(strides)
	cloudKeep, wantAdded := 1.0, 0.0
	for step := 0; step < strides; step++ {
		along := slabNear + (float64(step)+0.5)*stepLength
		thick := density * stepLength
		absorb := 1 - math.Exp(-thick)
		wantAdded += cloudKeep * absorb * math.Exp(-waterSigma*along)
		cloudKeep *= 1 - absorb
	}
	if !within(glowingPass.added.R, wantAdded, 1e-12) {
		t.Errorf("outgoing source at cloud depths = %g, want analytic midpoint Beer sum %g", glowingPass.added.R, wantAdded)
	}
}

func TestClearCloudScattersRefractedSkyWithWaterOnBothLegs(t *testing.T) {
	const index = 1.333
	world, _, seen, _ := diffuseSkyOceanFixture(t, index, 8)
	water := opticalWater(t, "cloud-sky-water", maths.Grey(.08), 1)
	world.OceanSurface.ClearCloudSkySamples = 8
	world.OceanSurface.Boundary.Material.ClearMedium = water
	seen.inside[0].clearMedium = water
	cloud, err := medium.NewClearCloud(medium.Puff{Radius: .5, Thick: 2})
	if err != nil {
		t.Fatal(err)
	}
	world.AddVolume(thing.Volume{Medium: cloud,
		Material: &thing.Material{Name: "sub-resolution cavitation",
			Colour: maths.Grey(.8)},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, -2, 0)})})
	if err := ValidateClearState(world,
		[]InitialMedium{{Index: index, ClearMedium: water}}); err != nil {
		t.Fatalf("supported source/cloud composition refused: %v", err)
	}
	scene := stage(world, 0)
	seen.ray = maths.Ray{At: maths.V(0, -4, 0), Dir: maths.Up}
	passed := throughClear(scene, seen, 4, water)
	if !(passed.added.R > 0 && passed.added.G > 0 && passed.added.B > 0) {
		t.Fatalf("local cloud did not scatter transmitted sky: %v; %v",
			passed.added, scene.refusal())
	}
}

func TestNamedWaterStillRefusesUnmarkedVolume(t *testing.T) {
	water := opticalWater(t, "test-water", maths.Grey(.1), 1)
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Through: 1, Index: 1.33,
			ClearMedium: water}})
	world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1, Thick: .1}})
	if err := ValidateClearState(world, []InitialMedium{{Index: 1.33,
		ClearMedium: water}}); err == nil {
		t.Fatal("unmarked volume was admitted beside named water")
	}
}

func TestInitialClearMediumMustMatchNamedBoundaryProperties(t *testing.T) {
	boundary := opticalWater(t, "same-water", maths.RGB(0.1, 0.2, 0.3), 1)
	initial := opticalWater(t, "same-water", maths.RGB(0.1, 0.2, 0.4), 1)
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Through: 1, Index: 1.33, ClearMedium: boundary}})
	seen, err := newView(world, mediumEye(maths.V(0, -1, 0), maths.Up), film.Pinhole{}, 0, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := seen.withInitialMedia([]InitialMedium{{Index: 1.33, ClearMedium: initial}}); err == nil {
		t.Fatal("initial state with mismatched named-medium properties was accepted")
	}
}

func TestValidateClearStateSharesRenderPreflight(t *testing.T) {
	world := thing.NewWorld()
	water := opticalWater(t, "water", maths.RGB(0.1, 0.2, 0.3), 1)
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
		Through: 1, Index: 1.33, Rank: 2, ClearMedium: water,
	}})

	if err := ValidateClearState(world, nil); err != nil {
		t.Fatalf("valid boundary world: %v", err)
	}
	if err := ValidateClearState(world, []InitialMedium{{Index: 1.33,
		Rank: 2, ClearMedium: water}}); err != nil {
		t.Fatalf("matching initial medium: %v", err)
	}

	wrong := opticalWater(t, "other water", maths.RGB(0.1, 0.2, 0.3), 1)
	if err := ValidateClearState(world, []InitialMedium{{Index: 1.33,
		Rank: 2, ClearMedium: wrong}}); err == nil {
		t.Fatal("mismatched initial medium passed shared preflight")
	}
	if err := ValidateClearState(world, []InitialMedium{
		{Index: 1.33, Rank: 2, ClearMedium: water},
		{Index: 1.33, Rank: 2, ClearMedium: water},
	}); err == nil {
		t.Fatal("duplicate initial medium passed shared preflight")
	}
}

func TestValidateOceanSpotInscatterIsExplicitAndRejectsUnsupportedSources(t *testing.T) {
	medium, err := thing.NewClearMediumWithScattering("ocean spot water",
		maths.RGB(.02, .01, .01), maths.RGB(.01, .005, .002),
		.7, 1, 12, 8)
	if err != nil {
		t.Fatal(err)
	}
	boundary := thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
		Through: 1, Index: 1.333, ClearMedium: medium,
	}}
	world := thing.NewWorld()
	world.Add(boundary)
	world.OceanSurface = &thing.OceanSurface{Boundary: world.Things[0],
		Mean: maths.Vec{}, ExtentMetres: 20, MetresPerUnit: 1,
		SpotInscatter: true}
	spot := film.Spot{At: maths.V(0, -2, 0), Aim: maths.V(0, -1, 0),
		Cone: 60 * maths.Degree, Strength: maths.Grey(20)}
	flood := film.Flood{Strength: maths.Grey(100)}
	world.Lamp(spot)
	world.Lamp(flood)
	inside := []InitialMedium{{Index: 1.333, ClearMedium: medium}}
	if err := ValidateClearState(world, inside); err != nil {
		t.Fatalf("explicit submerged spot with direct-only flood: %v", err)
	}

	world.OceanSurface.SpotInscatter = false
	if err := ValidateClearState(world, inside); err == nil {
		t.Fatal("participating ocean medium without explicit spot opt-in was accepted")
	}
	world.OceanSurface.SpotInscatter = true
	world.Lamps = []thing.Lamp{flood}
	if err := ValidateClearState(world, inside); err == nil {
		t.Fatal("directionless flood alone was accepted as a scattering source")
	}
	world.Lamps = []thing.Lamp{film.Spot{At: maths.V(0, 2, 0),
		Aim: maths.V(0, -1, 0), Cone: 60 * maths.Degree}}
	if err := ValidateClearState(world, inside); err == nil {
		t.Fatal("spot above the ocean surface was accepted")
	}
	world.Lamps = []thing.Lamp{film.Spot{At: maths.V(0, -2, 0),
		Aim: maths.V(0, -1, 0), Cone: 120 * maths.Degree}}
	if err := ValidateClearState(world, inside); err == nil {
		t.Fatal("spot cone crossing the air-water interface was accepted")
	}
}
