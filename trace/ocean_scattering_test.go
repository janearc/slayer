package trace

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

func oceanParticipatingFixture(test *testing.T, angle float64) (*thing.World,
	*thing.ClearMedium, *thing.Thing, look, maths.Vec) {
	test.Helper()
	world, _, boundary, seen, toward := oceanSunFixture(test, angle)
	medium, err := thing.NewClearMediumWithScattering("test-ocean",
		maths.RGB(.12, .08, .04), maths.RGB(.03, .02, .01), .3,
		1, 8, 12)
	if err != nil {
		test.Fatal(err)
	}
	boundary.Material.ClearMedium = medium
	seen.inside[0].clearMedium = medium
	return world, medium, boundary, seen, toward
}

func TestOceanScatteringSunCrossesInterfaceAndAttenuatesInWater(t *testing.T) {
	world, medium, _, seen, towardAir := oceanParticipatingFixture(t, math.Pi/6)
	scene := stage(world, 0)
	towardWater, got, usable := scene.oceanSunAtSample(maths.V(0, -2, 0), seen)
	if !usable {
		t.Fatal("refracted direct sun was not usable at an interior sample")
	}
	airAngle := math.Pi / 6
	waterSine := math.Sin(airAngle) / 1.333
	wantToward := maths.V(waterSine, math.Sqrt(1-waterSine*waterSine), 0)
	if !towardWater.Near(wantToward, 1e-12) {
		t.Fatalf("sample-to-sun direction = %v; Snell prediction %v", towardWater, wantToward)
	}
	if towardWater == towardAir {
		t.Fatal("water-side sun direction was not refracted")
	}
	reflectance := scene.oceanSun.material.SplitsBetween(towardAir.Y, 1.333)
	distanceMetres := 2 / wantToward.Y
	extinction := medium.Extinction()
	for _, channel := range []struct {
		name  string
		got   float64
		lamp  float64
		sigma float64
	}{{"red", got.R, 10, extinction.R},
		{"green", got.G, 10, extinction.G},
		{"blue", got.B, 10, extinction.B}} {
		want := channel.lamp * (1 - reflectance) *
			channelBeer(channel.sigma, distanceMetres) *
			towardAir.Y / wantToward.Y
		if math.Abs(channel.got-want) > 1e-11*math.Max(1, want) {
			t.Errorf("%s transmitted source=%g; Fresnel/projection/Beer oracle %g", channel.name, channel.got, want)
		}
	}
}

func TestOceanSingleScatterIntegratesRefractedSunAlongWaterRay(t *testing.T) {
	world, medium, boundary, _, _ := oceanParticipatingFixture(t, 0)
	sphereAt := resting(maths.V(0, -2, 1))
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Colour: maths.Grey(.8)},
		Motion:   sphereAt})
	const index = 1.333
	eye := mediumEye(maths.V(0, -2, -4), maths.V(0, 0, 1))
	media := []InitialMedium{{Index: index, Rank: 0, ClearMedium: medium}}
	with, err := RenderWithInitialMedia(world, eye,
		film.Pinhole{Wide: .01}, time.Duration(0), 1, 1, media)
	if err != nil {
		t.Fatalf("single-scatter ocean render: %v", err)
	}
	extinction := medium.Extinction()
	scattering := medium.Scattering()
	withoutMedium, err := thing.NewClearMedium("test-ocean",
		extinction, 1)
	if err != nil {
		t.Fatal(err)
	}
	boundary.Material.ClearMedium = withoutMedium
	world.OceanSurface.DirectSunMeanPlane = false
	without, err := RenderWithInitialMedia(world, eye,
		film.Pinhole{Wide: .01}, time.Duration(0), 1, 1,
		[]InitialMedium{{Index: index, Rank: 0, ClearMedium: withoutMedium}})
	if err != nil {
		t.Fatalf("extinction-only control render: %v", err)
	}
	distance, cell, steps := 4.0, 4.0/8, 8
	reflectance := boundary.Material.SplitsBetween(1, index)
	phase := (gas.Gas{Phase: .3}).PhaseAt(0)
	incident := 10 * (1 - reflectance) * channelBeer(extinction.G, 2)
	want := 0.0
	for step := range steps {
		begin := float64(step) * cell
		want += incident * phase * scattering.G *
			cellScatterIntegral(extinction.G, cell) *
			math.Exp(-extinction.G*begin)
	}
	got := float64(with.At(0, 0).G - without.At(0, 0).G)
	if math.Abs(got-want) > 2e-6*math.Max(1, want) {
		t.Fatalf("in-scattered green increment=%g; independent 8-cell Beer/source integral=%g (path %g m)", got, want, distance)
	}
}

func TestOceanSingleScatterRespectsOpaqueSunBlocker(t *testing.T) {
	world, _, _, seen, _ := oceanParticipatingFixture(t, 0)
	sphereAt := resting(maths.V(0, -1, 0))
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: .2},
		Material: &thing.Material{Colour: maths.Grey(.5)},
		Motion:   sphereAt})
	scene := stage(world, 0)
	_, got, usable := scene.oceanSunAtSample(maths.V(0, -2, 0), seen)
	if !usable {
		t.Fatal("opaque blocker should produce a usable, shadowed source query")
	}
	if got != (maths.Light{}) {
		t.Fatalf("opaque object between sample and interface transmitted source: %v", got)
	}
}

func TestOceanSingleScatterDoesNotSampleDiffuseSky(t *testing.T) {
	world, medium, _, seen, _ := oceanParticipatingFixture(t, 0)
	ray := maths.Ray{At: maths.V(0, -2, -2), Dir: maths.V(0, 0, 1)}
	withoutSky := stage(world, 0)
	withoutSkySource := oceanClearInscatter(withoutSky, seen, ray, 2,
		medium)
	withSkyWorld := *world
	withSkyWorld.Sky = sky.Gradient{}
	if err := validateClearInscatterWorld(&withSkyWorld, medium); err != nil {
		t.Fatalf("ocean direct-sun scattering should preserve, not reject, its camera sky: %v", err)
	}
	withSky := stage(&withSkyWorld, 0)
	withSkySource := oceanClearInscatter(withSky, seen, ray, 2, medium)
	if withSkySource != withoutSkySource {
		t.Fatalf("camera sky altered the direct-sun-only scatter source: with sky %v, without sky %v", withSkySource, withoutSkySource)
	}
}

func TestOceanSingleScatterAirStartRecordsNumericFault(t *testing.T) {
	world, _, boundary, _, _ := oceanSunFixture(t, 0)
	medium, err := thing.NewClearMediumWithScattering("test-ocean",
		maths.Grey(.1), maths.Grey(.01), -.999999, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	boundary.Material.ClearMedium = medium
	sun, err := emit.NewDistant(maths.Up, maths.Grey(1e307))
	if err != nil {
		t.Fatal(err)
	}
	world.Lamps[world.OceanSurface.SunLampIndex] = sun
	sphereAt := resting(maths.V(0, -2, -2))
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: .25},
		Material: &thing.Material{Colour: maths.Grey(.8)},
		Motion:   sphereAt})
	eye := mediumEye(maths.V(0, 2, -2), maths.V(0, -1, 0))
	view, err := newView(world, eye, film.Pinhole{}, 0, 1, 1, 1)
	if err != nil {
		t.Fatalf("air-start view: %v", err)
	}
	view, err = view.withInitialMedia(nil)
	if err != nil {
		t.Fatalf("air-start state: %v", err)
	}
	if view.holding != 0 {
		t.Fatalf("air-start camera was seeded with %d media", view.holding)
	}
	if _, err := RenderWithInitialMedia(world, eye, film.Pinhole{}, 0,
		1, 1, nil); err == nil ||
		!strings.Contains(err.Error(), "clear-medium in-scattering produced") {
		t.Fatalf("air-start ray crossing into scattering water did not return its numerical fault: %v", err)
	}
	scene := stage(world, 0)
	scene.enableClearInscatterFault(view)
	if scene.clearInscatterFault == nil {
		t.Fatal("air-start scene did not allocate a refusal flag for its reachable participating ocean")
	}
	waterLook := look{ray: maths.Ray{At: maths.V(0, -1, -2),
		Dir: maths.V(0, -1, 0)}, holding: 1}
	waterLook.inside[0] = held{index: 1.333, clearMedium: medium}
	_ = oceanClearInscatter(scene, waterLook, waterLook.ray, .1, medium)
	if err := scene.refusal(); err == nil ||
		!strings.Contains(err.Error(), "clear-medium in-scattering produced") {
		t.Fatalf("air-start source overflow did not become a render error: %v", err)
	}
}

func TestAirStartWithoutScatteringKeepsNoFaultState(t *testing.T) {
	world, _, _, _, _ := oceanSunFixture(t, 0)
	eye := mediumEye(maths.V(0, 2, -2), maths.V(0, -1, 0))
	view, err := newView(world, eye, film.Pinhole{}, 0, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	scene := stage(world, 0)
	scene.enableClearInscatterFault(view)
	if scene.clearInscatterFault != nil {
		t.Fatal("non-scattering air-start scene allocated the in-scatter fault state")
	}
}

func TestOceanScatteringRequiresItsExplicitDirectSunRoute(t *testing.T) {
	world, _, _, _, _ := oceanParticipatingFixture(t, 0)
	world.OceanSurface.DirectSunMeanPlane = false
	if _, err := validateClearWorld(world); err == nil {
		t.Fatal("participating water without explicit mean-plane direct sun was accepted")
	}
}

func TestOceanScatteringZeroAndNoOptInKeepLegacyPixels(t *testing.T) {
	world, medium, boundary, _, _ := oceanParticipatingFixture(t, 0)
	mediumWithoutSource, err := thing.NewClearMediumWithScattering(
		"test-ocean", medium.Extinction(), maths.Light{}, .3, 1, 8, 12)
	if err != nil {
		t.Fatal(err)
	}
	legacyMedium, err := thing.NewClearMedium("test-ocean",
		medium.Extinction(), 1)
	if err != nil {
		t.Fatal(err)
	}
	withoutOceanTag := *world.OceanSurface
	withoutOceanTag.DirectSunMeanPlane = false
	world.OceanSurface = &withoutOceanTag
	boundary.Material.ClearMedium = mediumWithoutSource
	eye := mediumEye(maths.V(0, -2, -4), maths.V(0, 0, 1))
	zero, err := RenderWithInitialMedia(world, eye,
		film.Pinhole{Wide: .01}, 0, 1, 1,
		[]InitialMedium{{Index: 1.333, Rank: 0, ClearMedium: mediumWithoutSource}})
	if err != nil {
		t.Fatalf("zero-scatter legacy render: %v", err)
	}
	boundary.Material.ClearMedium = legacyMedium
	world.OceanSurface = nil
	legacy, err := RenderWithInitialMedia(world, eye,
		film.Pinhole{Wide: .01}, 0, 1, 1,
		[]InitialMedium{{Index: 1.333, Rank: 0, ClearMedium: legacyMedium}})
	if err != nil {
		t.Fatalf("legacy render without ocean descriptor: %v", err)
	}
	if zero.At(0, 0) != legacy.At(0, 0) {
		t.Fatalf("zero-scattering/no-ocean-opt-in changed the legacy pixel: %v vs %v",
			zero.At(0, 0), legacy.At(0, 0))
	}
}

func channelBeer(extinction, metres float64) float64 {
	return math.Exp(-extinction * metres)
}

func cellScatterIntegral(extinction, metres float64) float64 {
	if extinction == 0 {
		return metres
	}
	return -math.Expm1(-extinction*metres) / extinction
}
