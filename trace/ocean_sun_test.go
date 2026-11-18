package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	seaocean "github.com/janearc/slayer/scene/ocean"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

func oceanSunFixture(t *testing.T, angle float64) (*thing.World,
	*thing.ClearMedium, *thing.Thing, look, maths.Vec) {
	t.Helper()
	const index, unit = 1.333, 1.0
	water, err := thing.NewClearMedium("test-ocean",
		maths.RGB(.2, .1, .05), unit)
	if err != nil {
		t.Fatal(err)
	}
	surface, err := shape.NewHeightField([]float64{0, 0, 0, 0}, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	boundary := &thing.Thing{Shape: surface,
		Material: &thing.Material{Through: 1, Gloss: 1, Index: index,
			ClearMedium: water}}
	toward := maths.V(math.Sin(angle), math.Cos(angle), 0)
	sun, err := emit.NewDistant(toward, maths.Grey(10))
	if err != nil {
		t.Fatal(err)
	}
	fill, err := emit.NewDistant(maths.V(0, 1, 0), maths.Grey(1))
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Things = append(world.Things, boundary)
	world.Lamps = []thing.Lamp{sun, fill}
	world.OceanSurface = &thing.OceanSurface{Boundary: boundary,
		Mean: maths.V(0, 0, 0), ExtentMetres: 20,
		MetresPerUnit: unit, DirectSunMeanPlane: true,
		SunRoute:     thing.OceanSunRouteGround,
		SunLampIndex: 0, FillLampIndex: 1}
	seen := look{holding: 1}
	seen.inside[0] = held{index: index, rank: 0, clearMedium: water}
	return world, water, boundary, seen, toward
}

func TestGroundlessAirOceanSunAcceptsOneTaggedLampAndSeparateSky(t *testing.T) {
	world, _, _, _, _ := oceanSunFixture(t, 0)
	world.Lamps = world.Lamps[:1]
	world.Sky = sky.Gradient{}
	world.OceanSurface.SunRoute = thing.OceanSunRouteGroundlessAir
	world.OceanSurface.FillLampIndex = -1
	if err := validateOceanDirectSunWorld(world); err != nil {
		t.Fatalf("single aired sun with independent sky was refused: %v", err)
	}
}

func TestGroundlessAirOceanSunKeepsExistingDirectionlessFloodSeparate(t *testing.T) {
	world, _, _, _, _ := oceanSunFixture(t, 0)
	world.Lamps = world.Lamps[:1]
	world.Lamps = append(world.Lamps, film.Flood{Strength: maths.Grey(0.25)})
	world.Sky = sky.Gradient{}
	world.OceanSurface.SunRoute = thing.OceanSunRouteGroundlessAir
	world.OceanSurface.FillLampIndex = -1
	if err := validateOceanDirectSunWorld(world); err != nil {
		t.Fatalf("existing separate directionless Flood was refused: %v", err)
	}
}

func TestGroundlessAirOceanSunRefusesMissingSkyFillOrExtraSun(t *testing.T) {
	for _, name := range []string{"missing sky", "fabricated fill", "extra distant", "extra film sun"} {
		t.Run(name, func(t *testing.T) {
			world, _, _, _, toward := oceanSunFixture(t, 0)
			world.Lamps = world.Lamps[:1]
			world.Sky = sky.Gradient{}
			world.OceanSurface.SunRoute = thing.OceanSunRouteGroundlessAir
			world.OceanSurface.FillLampIndex = -1
			switch name {
			case "missing sky":
				world.Sky = nil
			case "fabricated fill":
				world.OceanSurface.FillLampIndex = 0
			case "extra distant":
				other, err := emit.NewDistant(toward, maths.Grey(1))
				if err != nil {
					t.Fatal(err)
				}
				world.Lamps = append(world.Lamps, other)
			case "extra film sun":
				world.Lamps = append(world.Lamps, film.Sun{Toward: toward,
					Strength: maths.Grey(1), Size: 0.53 * maths.Degree})
			}
			if err := validateOceanDirectSunWorld(world); err == nil {
				t.Fatalf("unsupported groundless-air %s was accepted", name)
			}
		})
	}
}

func TestOceanMeanPlaneSunRequiresExplicitStagedRoute(t *testing.T) {
	world, _, _, _, _ := oceanSunFixture(t, 0)
	world.OceanSurface.SunRoute = thing.OceanSunRouteUnspecified
	if err := validateOceanDirectSunWorld(world); err == nil {
		t.Fatal("direct-sun mode without a staged source route was accepted")
	}
}

func TestOceanMeanPlaneSunConservesHorizontalPlaneIrradiance(t *testing.T) {
	for _, angle := range []float64{0, math.Pi / 6} {
		world, _, boundary, seen, towardAir := oceanSunFixture(t, angle)
		scene := stage(world, 0)
		towardWater, got, usable := scene.oceanSunLight(
			maths.V(0, -2, 0), maths.Up, seen)
		if !usable {
			t.Fatal("supported interior point was refused")
		}
		cosAir, cosWater := towardAir.Y, towardWater.Y
		from := scene.lifted(maths.V(0, -2, 0), maths.Up)
		waterDistance, hit := scene.oceanSun.placedBoundary.meet(
			maths.Ray{At: from, Dir: towardWater}, 0, math.Inf(1))
		if !hit {
			t.Fatal("direct path missed the sampled ocean boundary")
		}
		want := 10 * (1 - maths.Fresnel(cosAir, boundary.Material.Dense())) *
			cosAir * beerClear(.2, waterDistance)
		gotHorizontal := got.R * cosWater
		if math.Abs(gotHorizontal-want) > 1e-12 {
			t.Fatalf("horizontal irradiance got %.15g, want %.15g (direction %v)",
				gotHorizontal, want, towardWater)
		}
		if !(towardWater.Y > 0 && towardWater.Y >= towardAir.Y) ||
			math.Abs(towardWater.X*boundary.Material.Dense()-towardAir.X) > 1e-14 {
			t.Fatalf("refraction did not bend toward the normal: air %v water %v",
				towardAir, towardWater)
		}
	}
}

func TestOceanMeanPlaneSunKeepsOpaqueBlockersOnBothSegments(t *testing.T) {
	for _, segment := range []string{"water", "air"} {
		world, _, boundary, seen, towardAir := oceanSunFixture(t, math.Pi/9)
		towardWater, total := refracted(towardAir.Neg(), maths.Up,
			1.333)
		if total {
			t.Fatal("fixture unexpectedly has total internal reflection")
		}
		towardWater = towardWater.Neg().Unit()
		point := maths.V(0, -2, 0)
		planeAt := point.Add(towardWater.Scale((0 - point.Y) / towardWater.Y))
		blockAt := point.Add(towardWater.Scale(1))
		if segment == "air" {
			blockAt = planeAt.Add(towardAir.Scale(1))
		}
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: .1},
			Material: &thing.Material{}, Motion: thing.Still(
				maths.Subspace{Origin: blockAt})})
		scene := stage(world, 0)
		_, got, usable := scene.oceanSunLight(point, maths.Up, seen)
		if !usable || got != (maths.Light{}) {
			t.Fatalf("%s-segment opaque blocker was not honored: usable=%v light=%v boundary=%p", segment, usable, got, boundary)
		}
	}
}

func TestOceanMeanPlaneSunSkipsOnlyItsBoundary(t *testing.T) {
	world, _, boundary, _, _ := oceanSunFixture(t, 0)
	scene := stage(world, 0)
	ray := maths.Ray{At: maths.V(0, -1, 0), Dir: maths.Up}
	if !scene.crowd.blocked(ray, math.Inf(1)) {
		t.Fatal("legacy shadow query stopped treating ocean boundary as an opaque blocker")
	}
	if scene.crowd.blockedExcept(ray, math.Inf(1), boundary) {
		t.Fatal("direct-sun shadow query did not skip the exact ocean boundary")
	}
	blocker := world.Add(thing.Thing{Shape: shape.Sphere{Radius: .1},
		Material: &thing.Material{}, Motion: thing.Still(
			maths.Subspace{Origin: maths.V(0, .5, 0)})})
	scene = stage(world, 0)
	if !scene.crowd.blockedExcept(ray, math.Inf(1), boundary) {
		t.Fatalf("direct-sun shadow query skipped another opaque object %p", blocker)
	}
}

func TestOceanSunNoOptInKeepsLegacyLampEvaluation(t *testing.T) {
	world, _, _, seen, _ := oceanSunFixture(t, .2)
	world.OceanSurface.DirectSunMeanPlane = false
	scene := stage(world, 0)
	point, normal, colour := maths.V(0, -2, 0), maths.Up,
		maths.RGB(.3, .4, .5)
	got := scene.oceanLamplight(point, normal, colour, seen,
		world.Things[0])
	want := scene.lamplight(point, normal, colour)
	if got != want {
		t.Fatalf("no-opt-in lamp path changed: got %v, want %v", got, want)
	}
}

func TestOceanMeanPlaneSunRefusesUnsupportedSourcesAndMedia(t *testing.T) {
	t.Run("spot source", func(t *testing.T) {
		world, _, _, _, _ := oceanSunFixture(t, .2)
		world.Lamps[0] = film.Spot{Strength: maths.Grey(1)}
		if err := validateOceanDirectSunWorld(world); err == nil {
			t.Fatal("accepted a point/spot source in the mean-plane mode")
		}
	})
	t.Run("volume", func(t *testing.T) {
		world, _, _, _, _ := oceanSunFixture(t, .2)
		world.Volumes = append(world.Volumes, &thing.Volume{})
		if err := validateOceanDirectSunWorld(world); err == nil {
			t.Fatal("accepted volume transport together with mean-plane sun")
		}
	})
	t.Run("additional clear boundary", func(t *testing.T) {
		world, _, _, _, _ := oceanSunFixture(t, .2)
		world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
			Through: 1, Index: 1.5}})
		if err := validateOceanDirectSunWorld(world); err == nil {
			t.Fatal("accepted a nested transparent boundary")
		}
	})
	t.Run("haze", func(t *testing.T) {
		world, _, _, _, _ := oceanSunFixture(t, .2)
		world.Haze = thing.Haze{Density: .1}
		if err := validateOceanDirectSunWorld(world); err == nil {
			t.Fatal("accepted separate scene haze on the source path")
		}
	})
	t.Run("ocean scattering is opt-in on the same direct route", func(t *testing.T) {
		world, _, boundary, _, _ := oceanSunFixture(t, .2)
		medium, err := thing.NewClearMediumWithScattering("test-ocean",
			maths.Grey(.1), maths.Grey(.1), 0, 1, 1, 1)
		if err != nil {
			t.Fatal(err)
		}
		boundary.Material.ClearMedium = medium
		if err := validateOceanDirectSunWorld(world); err != nil {
			t.Fatalf("mean-plane interface refused a participating medium: %v", err)
		}
		if err := validateClearInscatterWorld(world, medium); err != nil {
			t.Fatalf("explicit ocean direct-sun scattering route was refused: %v", err)
		}
	})
	t.Run("ambient fill is not a scattering source", func(t *testing.T) {
		world, _, boundary, _, _ := oceanSunFixture(t, 0)
		medium, err := thing.NewClearMediumWithScattering("test-ocean",
			maths.Grey(.1), maths.Grey(.1), 0, 1, 1, 1)
		if err != nil {
			t.Fatal(err)
		}
		boundary.Material.ClearMedium = medium
		world.Lamps = append(world.Lamps, film.Flood{Strength: maths.Grey(.2)})
		if err := validateOceanDirectSunWorld(world); err == nil {
			t.Fatal("participating ocean silently omitted an ambient scattering source")
		}
	})
	t.Run("metre mismatch", func(t *testing.T) {
		world, _, boundary, _, _ := oceanSunFixture(t, .2)
		medium, err := thing.NewClearMedium("test-ocean", maths.Grey(.1), 2)
		if err != nil {
			t.Fatal(err)
		}
		boundary.Material.ClearMedium = medium
		if err := validateOceanDirectSunWorld(world); err == nil {
			t.Fatal("accepted incompatible metre conversion")
		}
	})
}

func TestOceanMeanPlaneSunRequiresMatchingInitialCameraState(t *testing.T) {
	world, medium, _, _, _ := oceanSunFixture(t, 0)
	eye := maths.Subspace{Origin: maths.V(0, -1, 0)}
	if err := validateOceanInitialMedia(world, eye, nil, 0); err == nil {
		t.Fatal("accepted a camera below the finite sea without an initial medium")
	}
	initial := []InitialMedium{{Index: 1.333,
		ClearMedium: medium}}
	if err := validateOceanInitialMedia(world, eye, initial, 0); err != nil {
		t.Fatalf("refused matching camera water state: %v", err)
	}
	if err := validateOceanInitialMedia(world,
		maths.Subspace{Origin: maths.V(0, 1, 0)}, initial, 0); err == nil {
		t.Fatal("accepted an ocean initial state for a camera above the interface")
	}
}

func TestOceanMeanPlaneSunRefusesOutsideFiniteFootprint(t *testing.T) {
	world, _, _, seen, _ := oceanSunFixture(t, 0)
	scene := stage(world, 0)
	_, _, usable := scene.oceanSunLight(maths.V(20, -1, 0), maths.Up,
		seen)
	if usable || scene.refusal() == nil {
		t.Fatal("outside-footprint source path was silently accepted")
	}
}

func TestOceanMeanPlaneSunUsesActualCrestAboveMeanPlane(t *testing.T) {
	world, _, boundary, seen, _ := oceanSunFixture(t, 0)
	crest, err := shape.NewHeightField([]float64{2, 2, 2, 2}, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	boundary.Shape = crest
	point := maths.V(0, 1, 0)
	if point.Y <= world.OceanSurface.Mean.Y || crest.Distance(point) >= 0 {
		t.Fatal("fixture is not inside a crest above the mean plane")
	}
	scene := stage(world, 0)
	toward, light, usable := scene.oceanSunLight(point, maths.Up, seen)
	if !usable || light == (maths.Light{}) || scene.refusal() != nil {
		t.Fatalf("sun path refused an interior point above mean plane: usable=%v light=%v refusal=%v", usable, light, scene.refusal())
	}
	from := scene.lifted(point, maths.Up)
	distance, hit := scene.oceanSun.placedBoundary.meet(
		maths.Ray{At: from, Dir: toward}, 0, math.Inf(1))
	if !hit || math.Abs(from.Add(toward.Scale(distance)).Y-2) > 1e-12 {
		t.Fatalf("water path did not end on the actual crest: distance %g hit %t", distance, hit)
	}
}

func TestOceanMeanPlaneSunCrossesAuthoredFlatMeanCollarOnce(t *testing.T) {
	world, _, boundary, seen, _ := oceanSunFixture(t, 0)
	field, ok := boundary.Shape.(*shape.HeightField)
	if !ok {
		t.Fatalf("fixture boundary shape is %T", boundary.Shape)
	}
	continuation, err := seaocean.NewFlatMeanContinuation(field, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	boundary.Shape = continuation
	if err := validateOceanDirectSunWorld(world); err != nil {
		t.Fatalf("refused supported continuation boundary: %v", err)
	}
	scene := stage(world, 0)
	for _, x := range []float64{0, 11} { // exact core and collar crossing
		point := maths.V(x, -2, 0)
		_, light, usable := scene.oceanSunLight(point, maths.Up, seen)
		if !usable {
			t.Fatalf("x=%g source path was not usable: %v", x, scene.refusal())
		}
		from := scene.lifted(point, maths.Up)
		distance, crossed := scene.oceanSun.placedBoundary.meet(
			maths.Ray{At: from, Dir: maths.Up}, 0, math.Inf(1))
		if !crossed || distance <= 0 {
			t.Fatalf("x=%g did not cross its one staged interface", x)
		}
		want := 10 * (1 - maths.Fresnel(1, boundary.Material.Dense())) *
			beerClear(.2, distance*scene.oceanSun.metresPerUnit)
		if math.Abs(light.R-want) > 1e-12 {
			t.Errorf("x=%g source crossed boundary with light %g, want one two-metre attenuation %g", x, light.R, want)
		}
	}
}

func TestOceanMeanPlaneSunReachesInfiniteFarFieldOutsideCore(t *testing.T) {
	world, _, boundary, seen, towardAir := oceanSunFixture(t, .2)
	field, ok := boundary.Shape.(*shape.HeightField)
	if !ok {
		t.Fatalf("fixture boundary shape is %T", boundary.Shape)
	}
	continuation, err := seaocean.NewInfiniteMeanContinuation(field, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	boundary.Shape = continuation
	world.OceanSurface.InfiniteMeanPlane = true
	if err := validateOceanDirectSunWorld(world); err != nil {
		t.Fatalf("refused explicit infinite mean continuation: %v", err)
	}
	scene := stage(world, 0)
	point := maths.V(100, -2, 0) // well outside the 20 m sampled core
	towardWater, light, usable := scene.oceanSunLight(point, maths.Up, seen)
	if !usable || light == (maths.Light{}) || scene.refusal() != nil {
		t.Fatalf("far-field sun path: usable=%t light=%v refusal=%v",
			usable, light, scene.refusal())
	}
	incoming, total := refracted(towardAir.Neg(), maths.Up,
		boundary.Material.Dense())
	if total {
		t.Fatal("fixture unexpectedly has total internal reflection")
	}
	wantTowardWater := incoming.Neg().Unit()
	if towardWater.Sub(wantTowardWater).Len() > 1e-12 {
		t.Fatalf("sun direction %v, want %v", towardWater, wantTowardWater)
	}
	from := scene.lifted(point, maths.Up)
	ray := maths.Ray{At: from, Dir: towardWater}
	distance, intersects := scene.oceanSun.placedBoundary.meet(ray, 0,
		math.Inf(1))
	if !intersects || !(distance > 0) {
		t.Fatalf("far-field ray did not reach the mean plane: distance=%g hit=%t",
			distance, intersects)
	}
	want := 10 * (1 - maths.Fresnel(towardAir.Y,
		boundary.Material.Dense())) * towardAir.Y / towardWater.Y *
		beerClear(.2, distance)
	if math.Abs(light.R-want) > 1e-5 {
		t.Fatalf("far-field light %g, want one mean-plane attenuation %g at %g m",
			light.R, want, distance)
	}
}

func TestOceanMeanPlaneSunRefusesMismatchedActiveMedium(t *testing.T) {
	world, _, _, seen, _ := oceanSunFixture(t, 0)
	scene := stage(world, 0)
	seen.inside[0] = held{index: 1.5, rank: 0}
	if _, _, usable := scene.oceanSunLight(maths.V(0, -1, 0),
		maths.Up, seen); usable || scene.refusal() == nil {
		t.Fatal("mismatched active medium was silently treated as ocean")
	}
}
