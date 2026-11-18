package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

func TestClearCloudPhaseChangesSourceButNotWaterOrTransmission(t *testing.T) {
	world, _, seen, _ := diffuseSkyOceanFixture(t, 1.333, 8)
	water, err := thing.NewClearMediumWithScattering("cloud phase water",
		maths.Grey(.1), maths.Grey(.01), 0, 1, 8, 20)
	if err != nil {
		t.Fatal(err)
	}
	ocean := world.OceanSurface
	ocean.Boundary.Material.ClearMedium = water
	ocean.DirectSunMeanPlane, ocean.DirectMoonMeanPlane = false, true
	ocean.SunRoute, ocean.FillLampIndex = thing.OceanSunRouteMoon, -1
	ocean.ClearCloudSkySamples = 0
	world.Lamps = world.Lamps[:1]
	field := medium.Fizz{Radius: 1, Height: 1, Thick: 1}
	legacy, err := medium.NewClearCloud(field)
	if err != nil {
		t.Fatal(err)
	}
	explicit, err := medium.NewClearCloudWithHenyeyGreenstein(field, 0)
	if err != nil {
		t.Fatal(err)
	}
	volume := world.AddVolume(thing.Volume{Medium: legacy,
		Material: &thing.Material{Colour: maths.Grey(1)},
		Motion:   thing.Still(maths.Subspace{Origin: maths.V(0, -2, 0)})})
	seen.inside[0].clearMedium = water
	seen.ray = maths.Ray{At: maths.V(0, -3, 0), Dir: maths.Up}
	first := stage(world, 0)
	old := throughClear(first, seen, 3, water)
	oldWater := clearWaterScatterSegment(first, seen, seen.ray, 0, .5, 3, water, maths.Grey(1))
	volume.Medium = explicit
	second := stage(world, 0)
	changed := throughClear(second, seen, 3, water)
	newWater := clearWaterScatterSegment(second, seen, seen.ray, 0, .5, 3, water, maths.Grey(1))
	if old.keep != changed.keep || oldWater != newWater {
		t.Fatal("cloud phase changed absorption or the water scattering law")
	}
	if old.added == changed.added {
		t.Fatal("explicit isotropic HG silently inherited water Rayleigh")
	}
}

func TestClearCloudPhaseSelectionAlsoWeightsRefractedSky(t *testing.T) {
	world, water, seen, _ := diffuseSkyOceanFixture(t, 1.333, 64)
	world.OceanSurface.DirectSunMeanPlane = false
	world.Lamps = nil
	scene := stage(world, 0)
	point := maths.V(0, -1, 0)
	inherited, ok := oceanClearSourceAtSample(scene, point, seen, maths.Up, water)
	if !ok {
		t.Fatal("legacy sky source failed")
	}
	explicit, ok := oceanClearSourceWithPhase(scene, point, seen,
		maths.Up, water, cloudPhaseSelection{g: 0, explicit: true})
	if !ok {
		t.Fatal("HG sky source failed")
	}
	if inherited.R <= 0 || explicit.R <= 0 || inherited == explicit {
		t.Fatal("cloud HG failed to select its own refracted sky phase")
	}
}

func TestClearCloudHGZeroIsIsotropicAndNotLegacyRayleigh(t *testing.T) {
	for _, cosine := range []float64{-1, -.5, 0, .5, 1} {
		if got := clearCloudHenyeyGreenstein(0, cosine); got != 1/(4*math.Pi) {
			t.Fatalf("explicit HG0(%g)=%g", cosine, got)
		}
	}
	if clearCloudHenyeyGreenstein(0, 1) == (gas.Gas{Phase: 0}).PhaseAt(1) {
		t.Fatal("explicit HG0 silently became legacy water Rayleigh")
	}
}

func TestClearCloudHGNormalizesAndHasAuthoredDirection(t *testing.T) {
	const n = 131072
	for _, g := range []float64{-.9, -.5, 0, .5, .9} {
		sum := 0.0
		for i := 0; i < n; i++ {
			cosine := -1 + 2*(float64(i)+.5)/n
			sum += clearCloudHenyeyGreenstein(g, cosine) * 4 * math.Pi / n
		}
		if math.Abs(sum-1) > 2e-6 {
			t.Fatalf("HG%g integrates to%g", g, sum)
		}
	}
	if clearCloudHenyeyGreenstein(.8, 1) <= clearCloudHenyeyGreenstein(.8, -1) {
		t.Fatal("positive g did not scatter forward")
	}
	if clearCloudHenyeyGreenstein(-.8, -1) <= clearCloudHenyeyGreenstein(-.8, 1) {
		t.Fatal("negative g did not scatter backward")
	}
}

func TestClearCloudHGNearEndpointsStaysFiniteAndPositive(t *testing.T) {
	for _, g := range []float64{math.Nextafter(1, 0), math.Nextafter(-1, 0)} {
		for _, cosine := range []float64{-1, 0, 1} {
			phase := clearCloudHenyeyGreenstein(g, cosine)
			if !(phase > 0) || math.IsNaN(phase) || math.IsInf(phase, 0) {
				t.Fatalf("HG(%g,%g)=%g", g, cosine, phase)
			}
		}
	}
}
