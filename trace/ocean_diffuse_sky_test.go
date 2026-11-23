package trace

import (
	"math"
	"strings"
	"sync"
	"testing"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

func diffuseSkyOceanFixture(test *testing.T, index float64,
	samples int,
) (*thing.World, *thing.ClearMedium, look, *staged) {
	return diffuseSkyOceanFixtureAt(test, index, samples, 0)
}

func diffuseSkyOceanFixtureAt(test *testing.T, index float64,
	samples int, sunAngle float64,
) (*thing.World, *thing.ClearMedium, look, *staged) {
	test.Helper()
	world, _, boundary, seen, _ := oceanSunFixture(test, sunAngle)
	core, err := shape.NewHeightField([]float64{0, 0, 0, 0}, 2, 20)
	if err != nil {
		test.Fatal(err)
	}
	continuation, err := arid.NewInfiniteMeanContinuation(core, 0, 5)
	if err != nil {
		test.Fatal(err)
	}
	boundary.Shape = continuation
	base, err := thing.NewClearMediumWithScattering("diffuse-test-ocean",
		maths.Grey(1e-12), maths.Grey(1e-12), 0, 1, 8, 12)
	if err != nil {
		test.Fatal(err)
	}
	medium, err := base.WithDiffuseSkyInscatter(samples)
	if err != nil {
		test.Fatal(err)
	}
	boundary.Material.Index = index
	boundary.Material.ClearMedium = medium
	seen.inside[0] = held{index: index, rank: boundary.Material.Rank,
		clearMedium: medium}
	world.Sky = sky.Gradient{Horizon: maths.Grey(4),
		Zenith: maths.Grey(4), Ground: maths.Grey(4)}
	world.OceanSurface.InfiniteMeanPlane = true
	scene := stage(world, 0)
	return world, medium, seen, scene
}

func TestDiffuseSkyInscatterSamplesNormalizedUpperHemisphere(t *testing.T) {
	const samples = 64
	world, _, seen, scene := diffuseSkyOceanFixture(t, 1, samples)
	if err := validateClearInscatterWorld(world,
		world.OceanSurface.Boundary.Material.ClearMedium); err != nil {
		t.Fatalf("valid infinite-mean sky-scattering ocean refused: %v", err)
	}
	got := oceanDiffuseSkyAtSample(scene, maths.V(0, -1e-5, 0), seen,
		maths.V(0, -1, 0))
	// At n=1, transmission is one and Snell's map is the identity. For a
	// constant sky and vertical downward view, Rayleigh p=3(1+mu^2)/(16π);
	// integrating it over the upper hemisphere is exactly one half.
	// Midpoint quadrature of mu^2 has mean 1/3 - 1/(12N^2),
	// so this finite sum has the exact value below; the continuous
	// hemisphere integral is 2 and the residual is its stated quadrature error.
	finiteQuadrature := 2 - 1/(8*float64(samples*samples))
	if math.Abs(got.R-finiteQuadrature) > 1e-12 ||
		math.Abs(got.G-finiteQuadrature) > 1e-12 ||
		math.Abs(got.B-finiteQuadrature) > 1e-12 ||
		math.Abs(got.G-2) > 1/(8*float64(samples*samples)) {
		t.Fatalf("constant-sky source integral=%v, finite analytic quadrature %.15g, continuous hemisphere integral 2",
			got, finiteQuadrature)
	}
}

func TestDiffuseSkyInscatterSupportsSkyOnlyOceanWithoutDirectSun(t *testing.T) {
	world, medium, seen, _ := diffuseSkyOceanFixture(t, 1.333, 16)
	world.OceanSurface.DirectSunMeanPlane = false
	world.Lamps = nil
	if err := validateClearInscatterWorld(world, medium); err != nil {
		t.Fatalf("sky-only scene rejected: %v", err)
	}
	scene := stage(world, 0)
	if scene.oceanSun == nil || scene.oceanSun.sunIndex != -1 {
		t.Fatalf("sky-only source context = %+v", scene.oceanSun)
	}
	if _, usable := oceanClearSourceAtSample(scene, maths.V(0, -1, 0),
		seen, maths.V(0, -1, 0), medium); !usable {
		t.Fatal("sky-only source evaluation failed")
	}
}

func TestSkyOnlyOceanValidatesInitialCameraMediumStack(t *testing.T) {
	world, medium, _, _ := diffuseSkyOceanFixture(t, 1.333, 8)
	world.OceanSurface.DirectSunMeanPlane = false
	below := maths.Subspace{Origin: maths.V(0, -1, 0)}
	above := maths.Subspace{Origin: maths.V(0, 1, 0)}
	initial := []InitialMedium{{Index: 1.333, ClearMedium: medium}}
	if err := validateOceanInitialMedia(world, below, nil, 0); err == nil {
		t.Fatal("accepted underwater camera without initial medium")
	}
	if err := validateOceanInitialMedia(world, below, initial, 0); err != nil {
		t.Fatalf("refused matching underwater medium: %v", err)
	}
	if err := validateOceanInitialMedia(world, above, initial, 0); err == nil {
		t.Fatal("accepted underwater stack for camera above surface")
	}
	changed, err := medium.WithDiffuseSkyInscatter(16)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateOceanInitialMedia(world, below,
		[]InitialMedium{{Index: 1.333, ClearMedium: changed}}, 0); err == nil {
		t.Fatal("accepted initial medium with different sky sampling")
	}
}

func TestSkyOnlyOceanRefusesInvalidBoundaries(t *testing.T) {
	for _, name := range []string{"absent", "hidden", "units", "material", "mean"} {
		t.Run(name, func(t *testing.T) {
			world, medium, _, _ := diffuseSkyOceanFixture(t, 1.333, 8)
			world.OceanSurface.DirectSunMeanPlane = false
			boundary := world.OceanSurface.Boundary
			switch name {
			case "absent":
				world.Things = nil
			case "hidden":
				boundary.Attrs.Hidden = true
			case "units":
				world.OceanSurface.MetresPerUnit = 2
			case "material":
				boundary.Material.Through = .9
			case "mean":
				world.OceanSurface.Mean.X = math.NaN()
			}
			if err := validateClearInscatterWorld(world, medium); err == nil {
				t.Fatalf("accepted invalid %s ocean boundary", name)
			}
		})
	}
}

func TestStageOceanWithoutBoundaryDoesNotPanic(t *testing.T) {
	world := &thing.World{OceanSurface: &thing.OceanSurface{}}
	if made := stage(world, 0); made == nil || made.oceanSun != nil {
		t.Fatalf("staged nil ocean boundary unexpectedly: %+v", made)
	}
}

func TestDiffuseSkyInscatterUsesSnellSolidAngleJacobian(t *testing.T) {
	const (
		index   = 1.333
		samples = 256
		depth   = 2.0
	)
	world, medium, seen, scene := diffuseSkyOceanFixture(t, index, samples)
	got := oceanDiffuseSkyAtSample(scene, maths.V(0, -depth, 0), seen,
		maths.V(0, -1, 0))
	// Independent dense air-solid-angle midpoint reference. The radiance
	// scale n_w^2 and dΩ Snell Jacobian n_a^2*cosAir/(n_w^2*cosWater)
	// cancel, leaving T*cosAir/cosWater in an air-direction quadrature.
	const referenceSamples = 100000
	extinction := medium.Extinction().G
	phaseModel := gas.Gas{Phase: medium.Phase()}
	var reference float64
	for sample := range referenceSamples {
		cosAir := (float64(sample) + .5) / referenceSamples
		sinAir := math.Sqrt(1 - cosAir*cosAir)
		sinWater := sinAir / index
		cosWater := math.Sqrt(1 - sinWater*sinWater)
		transmission := 1 - maths.Fresnel(cosAir, index)
		jacobianProduct := transmission * cosAir / cosWater
		waterDistance := depth / cosWater
		phase := phaseModel.PhaseAt(-cosWater)
		integrand := 4 * jacobianProduct *
			math.Exp(-extinction*waterDistance) * phase
		reference += integrand * 2 * math.Pi / referenceSamples
	}
	if math.Abs(got.G-reference) > 2e-6 {
		t.Fatalf("sky source=%g, dense Snell/Jacobian/Beer reference=%g", got.G, reference)
	}
	if !finite(got.R, got.G, got.B) {
		t.Fatalf("sky source is not finite: %v", got)
	}
	_ = world
}

func TestDiffuseSkyDirectionUsesTransmissionJacobianAndWaterBeer(t *testing.T) {
	const (
		index = 1.333
		depth = 2.0
		angle = math.Pi / 6
	)
	world, medium, seen, scene := diffuseSkyOceanFixture(t, index, 32)
	airToward := maths.V(math.Sin(angle), math.Cos(angle), 0)
	point := maths.V(0, -depth, 0)
	towardWater, got, usable := scene.oceanSkyLightFrom(point, seen,
		airToward)
	if !usable {
		t.Fatalf("single diffuse-sky sample was unusable: %v", scene.refusal())
	}
	sinWater := math.Sin(angle) / index
	wantToward := maths.V(sinWater, math.Sqrt(1-sinWater*sinWater), 0)
	if !towardWater.Near(wantToward, 1e-12) {
		t.Fatalf("air sky direction %v mapped to %v, want Snell direction %v",
			airToward, towardWater, wantToward)
	}
	distance, hit := scene.oceanSun.placedBoundary.meet(
		maths.Ray{At: point, Dir: wantToward}, 0, math.Inf(1))
	if !hit {
		t.Fatal("analytic ray did not meet the infinite mean continuation")
	}
	cosAir, cosWater := airToward.Y, wantToward.Y
	transmission := 1 - scene.oceanSun.material.SplitsBetween(cosAir, index)
	projectedJacobian := transmission * cosAir / cosWater
	extinction := medium.Extinction()
	want := maths.RGB(4*projectedJacobian*math.Exp(-extinction.R*distance),
		4*projectedJacobian*math.Exp(-extinction.G*distance),
		4*projectedJacobian*math.Exp(-extinction.B*distance))
	if math.Abs(got.R-want.R) > 1e-12 || math.Abs(got.G-want.G) > 1e-12 ||
		math.Abs(got.B-want.B) > 1e-12 {
		t.Fatalf("transmitted sky sample=%v, want T*cosAir/cosWater times water Beer %v",
			got, want)
	}
	_ = world
}

func TestClearCloudSkySourceAttenuatesIncomingWaterAndCloudLeg(t *testing.T) {
	const index, sigma = 1.333, 0.2
	point := maths.V(0, -1, 0)
	source := func(extinction float64, cloudOn bool) (maths.Light, float64) {
		world, _, seen, _ := diffuseSkyOceanFixture(t, index, 8)
		water := opticalWater(t, "cloud-source-water",
			maths.Grey(extinction), 1)
		boundary := world.OceanSurface.Boundary
		boundary.Material.ClearMedium = water
		world.OceanSurface.ClearCloudSkySamples = 8
		seen.inside[0].clearMedium = water
		if cloudOn {
			cloud, err := medium.NewClearCloud(medium.Puff{Radius: 0.5,
				Thick: 2})
			if err != nil {
				t.Fatal(err)
			}
			world.AddVolume(thing.Volume{Medium: cloud,
				Material: &thing.Material{Colour: maths.Grey(1)},
				Motion:   thing.Still(maths.Subspace{Origin: point})})
		}
		if err := ValidateClearState(world,
			[]InitialMedium{{Index: index, ClearMedium: water}}); err != nil {
			t.Fatalf("clear-cloud sky source world refused: %v", err)
		}
		made := stage(world, 0)
		_, incident, usable := made.oceanSkyLightFrom(point, seen,
			maths.Up)
		if !usable || incident == (maths.Light{}) {
			t.Fatalf("air-side source did not reach cloud sample: %v; %v",
				incident, made.refusal())
		}
		distance, hit := made.oceanSun.placedBoundary.meet(
			maths.Ray{At: point, Dir: maths.Up}, 0, math.Inf(1))
		if !hit {
			t.Fatal("source ray missed water boundary")
		}
		return incident, distance
	}
	noWater, distance := source(0, false)
	withWater, _ := source(sigma, false)
	wantRatio := math.Exp(-sigma * distance)
	if !within(withWater.G/noWater.G, wantRatio, 1e-12) {
		t.Fatalf("incoming source water Beer ratio=%g, want %g", withWater.G/noWater.G, wantRatio)
	}
	withCloud, _ := source(sigma, true)
	if !(withCloud.G > 0 && withCloud.G < withWater.G) {
		t.Fatalf("incoming local cloud did not attenuate sky source: clear %g, cloud %g", withWater.G, withCloud.G)
	}
}

func TestDiffuseSkyWaterSourceComposesWithOptedInClearCloud(t *testing.T) {
	world, water, seen, _ := diffuseSkyOceanFixture(t, 1.333, 8)
	world.OceanSurface.DirectSunMeanPlane = false
	world.Lamps = nil
	ray := maths.Ray{At: maths.V(0, -4, 0), Dir: maths.Up}
	seen.ray = ray
	withoutCloud := stage(world, 0)
	clearPath := throughClear(withoutCloud, seen, 4, water)
	if !(clearPath.added.R > 0 && clearPath.added.G > 0 &&
		clearPath.added.B > 0) {
		t.Fatalf("ocean diffuse-sky source contributed no light: %v",
			clearPath.added)
	}

	cloud, err := medium.NewClearCloud(medium.Fizz{Radius: 1,
		Height: 1, Thick: 10})
	if err != nil {
		t.Fatal(err)
	}
	motion := thing.Still(maths.Subspace{Origin: maths.V(0, -3.5, 0)})
	world.AddVolume(thing.Volume{Medium: cloud, Motion: motion,
		Material: &thing.Material{Name: "absorbing clear cloud"}})
	if err := ValidateClearState(world,
		[]InitialMedium{{Index: 1.333,
			ClearMedium: water}}); err != nil {
		t.Fatalf("explicit clear cloud alongside ocean scattering refused: %v", err)
	}
	withCloud := stage(world, 0)
	cloudPath := throughClear(withCloud, seen, 4, water)
	if !(cloudPath.added.R > 0 && cloudPath.added.G > 0 &&
		cloudPath.added.B > 0) {
		t.Fatalf("cloud composition discarded ocean source: %v; %v",
			cloudPath.added, withCloud.refusal())
	}
	if !(cloudPath.added.R < clearPath.added.R &&
		cloudPath.added.G < clearPath.added.G &&
		cloudPath.added.B < clearPath.added.B) {
		t.Fatalf("ocean source was not attenuated by cloud on the outgoing path: clear=%v cloud=%v",
			clearPath.added, cloudPath.added)
	}

	world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1,
		Thick: 0.1}, Material: &thing.Material{Name: "unmarked volume"}})
	if err := ValidateClearState(world,
		[]InitialMedium{{Index: 1.333,
			ClearMedium: water}}); err == nil ||
		!strings.Contains(err.Error(), "explicitly opted-in clear-cloud volumes") {
		t.Fatalf("unmarked volume was not refused beside ocean scattering: %v", err)
	}
}

func TestOceanDiffuseSkyAddsBoundedCameraPathSource(t *testing.T) {
	world, medium, seen, withSky := diffuseSkyOceanFixture(t, 1.333, 32)
	ray := maths.Ray{At: maths.V(0, -2, -2), Dir: maths.V(0, 0, 1)}
	with := oceanClearInscatter(withSky, seen, ray, 2, medium)
	withoutMedium, err := thing.NewClearMediumWithScattering(medium.Name(),
		medium.Absorption(), medium.Scattering(), medium.Phase(),
		medium.MetresPerUnit(), medium.InscatterSteps(),
		medium.MaxInscatterDistanceMetres())
	if err != nil {
		t.Fatal(err)
	}
	world.OceanSurface.Boundary.Material.ClearMedium = withoutMedium
	withoutSeen := seen
	withoutSeen.inside[0].clearMedium = withoutMedium
	without := oceanClearInscatter(stage(world, 0), withoutSeen, ray, 2,
		withoutMedium)
	source := oceanDiffuseSkyAtSample(withSky, maths.V(0, -2, -1.875), seen,
		ray.Dir)
	cell := 2.0 / float64(medium.InscatterSteps())
	want := 0.0
	for step := range medium.InscatterSteps() {
		begin := float64(step) * cell
		want += source.G * medium.Scattering().G *
			cellScatterIntegral(medium.Extinction().G, cell) *
			math.Exp(-medium.Extinction().G*begin)
	}
	got := with.G - without.G
	if math.Abs(got-want) > 1e-12*math.Max(1, want) {
		t.Fatalf("diffuse-sky camera contribution=%g, analytic bounded source integral=%g (with=%v without=%v)",
			got, want, with, without)
	}
}

func TestOceanDiffuseSkySurvivesBlockedDirectSun(t *testing.T) {
	const angle = math.Pi / 6
	world, medium, seen, _ := diffuseSkyOceanFixtureAt(t, 1.333, 64,
		angle)
	point := maths.V(0, -2, 0)
	sinWater := math.Sin(angle) / 1.333
	cosWater := math.Sqrt(1 - sinWater*sinWater)
	directTowardWater := maths.V(sinWater, cosWater, 0)
	blockAt := point.Add(directTowardWater.Scale(1 / cosWater))
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: .15},
		Material: &thing.Material{}, Motion: resting(blockAt)})
	scene := stage(world, 0)
	_, sun, usable := scene.oceanSunAtSample(point, seen)
	if !usable || sun != (maths.Light{}) {
		t.Fatalf("fixture did not block only the tagged direct sun: usable=%v sun=%v",
			usable, sun)
	}
	source, usable := oceanClearSourceAtSample(scene, point, seen,
		maths.V(0, -1, 0), medium)
	if !usable || source == (maths.Light{}) {
		t.Fatalf("blocked direct sun suppressed the visible sky source: usable=%v source=%v",
			usable, source)
	}
}

func TestDiffuseSkyInscatterRefusesMissingSkyOrFiniteOnlyOcean(t *testing.T) {
	for _, name := range []string{"missing sky", "finite footprint"} {
		t.Run(name, func(t *testing.T) {
			world, medium, _, _ := diffuseSkyOceanFixture(t, 1.333, 8)
			switch name {
			case "missing sky":
				world.Sky = nil
			case "finite footprint":
				world.OceanSurface.InfiniteMeanPlane = false
			}
			err := validateClearInscatterWorld(world, medium)
			if err == nil || !strings.Contains(err.Error(),
				"diffuse-sky in-scattering") {
				t.Fatalf("unsupported %s scene was silently accepted: %v", name, err)
			}
		})
	}
}

// moonlitCore is the moonlit sea's sampled waves, made once for the
// tests that stage it: making them is most of a second.
var moonlitCore = sync.OnceValues(func() (*shape.HeightField, error) {
	return arid.NewHeightField(arid.HeightFieldConfig{
		Sea:    oceantex.Sea{Wind: 7, Fetch: 50000, Toward: 0},
		Extent: 512, Samples: 257, Seed: 7, Seconds: 0,
	})
})

// moonlitSea is a wind sea of 512 metres with a collar of 64 out to a
// flat far field, placed 1500 metres down minus x and staged as the
// tagged ocean: the placed boundary, and its shape in its own space.
func moonlitSea(t *testing.T) (*placedThing, *arid.InfiniteMeanContinuation) {
	t.Helper()
	core, err := moonlitCore()
	if err != nil {
		t.Fatal(err)
	}
	boundaryShape, err := arid.NewInfiniteMeanContinuation(core, 0, 64)
	if err != nil {
		t.Fatal(err)
	}
	medium, err := thing.NewClearMediumWithScattering("test-water",
		maths.RGB(.01, .04, .06), maths.RGB(.03, .008, .004), 0,
		1, 8, 80)
	if err != nil {
		t.Fatal(err)
	}
	medium, err = medium.WithDiffuseSkyInscatter(16)
	if err != nil {
		t.Fatal(err)
	}
	boundary := &thing.Thing{Shape: boundaryShape,
		Material: &thing.Material{Through: 1, Gloss: 1, Index: 1.333,
			ClearMedium: medium},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(-1500, 0, 0)})}
	world := thing.NewWorld()
	world.Things = append(world.Things, boundary)
	world.OceanSurface = &thing.OceanSurface{Boundary: boundary,
		InfiniteMeanPlane: true, Mean: maths.V(-1500, 0, 0),
		ExtentMetres: 512, MetresPerUnit: 1}
	staged := stage(world, 0)
	if staged.oceanSun == nil || staged.oceanSun.placedBoundary == nil {
		t.Fatal("staged scene did not retain the placed scattering boundary")
	}
	return staged.oceanSun.placedBoundary, boundaryShape
}

// capturedWay is the way of a look eight degrees above level, out of
// the water, at a graze to the surface.
var capturedWay = maths.V(-0.8313319789250135, 0.1395933109731877,
	0.5379598947395277)

// climbing is a look along capturedWay from under the moonlit sea, and
// how far along it has climbed over the sea. It starts 276 metres from
// the sea's middle and so 20 past its core's edge, in the collar; a hundredth
// under the lowest the surface goes anywhere, and is followed to a
// hundredth over the highest. Its height is set from the surface's own
// bounds, so it crosses whatever the waves are.
func climbing(sea *arid.InfiniteMeanContinuation) (maths.Ray, float64) {
	low, high := sea.Bounds().Min.Y-0.01, sea.Bounds().Max.Y+0.01
	from := maths.V(-1775.99602763588, low, 173.73196218084303)
	return maths.Ray{At: from, Dir: capturedWay},
		(high - low) / capturedWay.Y
}

// A look that climbs out of the tagged sea at a graze, through the
// collar between its sampled core and its flat far field, meets the
// sea's surface on the way, asked through the placed thing as the
// in-scattering asks it. It starts under the whole surface and ends
// over the whole of it, so it crosses whatever the waves are; and a
// twentieth of a metre short of where it is said to cross, it is still
// under, and a twentieth past it is over.
func TestALookClimbingOutOfThePlacedSeaMeetsItsSurface(t *testing.T) {
	placed, sea := moonlitSea(t)
	ray, far := climbing(sea)
	local := placed.at.RayIn(ray)
	if under, over := sea.Distance(local.At),
		sea.Distance(local.Point(far)); !(under < 0 && over > 0) {
		t.Fatalf("the test's own look starts %g from the surface and "+
			"ends %g from it, want under and over", under, over)
	}
	along, hit := placed.meet(ray, 0, far)
	if !hit || !(along > 0 && along < far) {
		t.Fatalf("the placed sea was missed: hit=(%g,%t) before %g, "+
			"local ray %+v", along, hit, far, local)
	}
	if short, past := onEitherSide(sea, local, along); !(short < 0 &&
		past > 0) {
		t.Errorf("a twentieth either side of the crossing at %g the "+
			"look is %g and %g from the surface, want under and over",
			along, short, past)
	}
}

// crossingSide is how far, along a look, either side of a reported
// crossing the look is asked where it stands: a twentieth of a metre.
const crossingSide = 0.05

// onEitherSide is how far from the sea's surface a local look stands
// a twentieth of a metre before and after a place along it.
func onEitherSide(sea *arid.InfiniteMeanContinuation, local maths.Ray,
	along float64) (before, after float64) {
	return sea.Distance(local.Point(along - crossingSide)),
		sea.Distance(local.Point(along + crossingSide))
}

// Anti: a crossing reported a metre too early, or a metre too late, is
// not one the either-side check takes: a twentieth past the early one
// is still under, and a twentieth short of the late one is already
// over.
func TestACrossingReportedEarlyOrLateIsTurnedAwayBySidesCheck(
	t *testing.T) {
	placed, sea := moonlitSea(t)
	ray, far := climbing(sea)
	local := placed.at.RayIn(ray)
	along, hit := placed.meet(ray, 0, far)
	if !hit {
		t.Fatal("the placed sea was missed")
	}
	const wrong = 1.0
	if short, past := onEitherSide(sea, local, along-wrong); short < 0 &&
		past > 0 {
		t.Errorf("a crossing a metre early passes: %g and %g", short, past)
	}
	if short, past := onEitherSide(sea, local, along+wrong); short < 0 &&
		past > 0 {
		t.Errorf("a crossing a metre late passes: %g and %g", short, past)
	}
}

// Anti: a look that runs level under the whole surface, as far, meets
// none of it; and neither does the climbing look begun a metre deeper
// and stopped a metre along, 0.14 of a metre up and still under the
// lowest water.
func TestALookThatStaysUnderThePlacedSeaMeetsNoSurface(t *testing.T) {
	placed, sea := moonlitSea(t)
	ray, far := climbing(sea)
	level := ray
	level.Dir = maths.V(capturedWay.X, 0, capturedWay.Z).Unit()
	if along, hit := placed.meet(level, 0, far); hit {
		t.Errorf("a level look under the sea met it at %g", along)
	}
	sunk := ray
	sunk.At.Y -= 1
	if along, hit := placed.meet(sunk, 0, 1); hit {
		t.Errorf("a look stopped under the sea met it at %g", along)
	}
}
