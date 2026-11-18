package ocean

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/scene/shape"
)

func continuationCore(t *testing.T, flat bool) *shape.HeightField {
	t.Helper()
	const n, extent = 9, 8.0
	heights := make([]float64, n*n)
	for row := range n {
		for column := range n {
			x := -extent/2 + float64(column)*extent/float64(n-1)
			z := -extent/2 + float64(row)*extent/float64(n-1)
			if flat {
				heights[row*n+column] = 0
			} else {
				heights[row*n+column] = .125 + .03125*x - .015625*z + .0078125*x*z
			}
		}
	}
	core, err := shape.NewHeightField(heights, n, extent)
	if err != nil {
		t.Fatal(err)
	}
	return core
}

func continueCoreAt(t *testing.T, core *shape.HeightField, x, z float64) float64 {
	t.Helper()
	heights, n, extent, _, _, _ := core.Pack()
	step := extent / float64(n-1)
	gx := (x + extent/2) / step
	gz := (z + extent/2) / step
	column := min(int(gx), n-2)
	row := min(int(gz), n-2)
	tx, tz := gx-float64(column), gz-float64(row)
	at := row*n + column
	low := heights[at]*(1-tx) + heights[at+1]*tx
	upper := heights[at+n]*(1-tx) + heights[at+n+1]*tx
	return low*(1-tz) + upper*tz
}

func TestFlatMeanContinuationPreservesCoreHeightAndNormalExactly(t *testing.T) {
	core := continuationCore(t, false)
	continued, err := NewFlatMeanContinuation(core, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	surface := continued
	_, n, extent, _, _, _ := core.Pack()
	step := extent / float64(n-1)
	points := []maths.Vec{maths.V(0, 0, 0), maths.V(-1.25, 0, .75),
		maths.V(extent/2, 0, -extent/2), maths.V(-extent/2, 0, extent/2)}
	for row := range n {
		for column := range n {
			points = append(points, maths.V(-extent/2+float64(column)*step,
				0, -extent/2+float64(row)*step))
		}
	}
	for _, point := range points {
		wantHeight := continueCoreAt(t, core, point.X, point.Z)
		gotHeight, gotX, gotZ := surface.sample(point.X, point.Z)
		if gotHeight != wantHeight {
			t.Fatalf("core height at (%g,%g) = %.17g, want exact %.17g",
				point.X, point.Z, gotHeight, wantHeight)
		}
		wantNormal := core.Normal(maths.V(point.X, wantHeight, point.Z))
		gotNormal := surface.Normal(maths.V(point.X, wantHeight, point.Z))
		if gotNormal != wantNormal {
			t.Fatalf("core normal at (%g,%g) = %v, want exact %v; slopes %g,%g",
				point.X, point.Z, gotNormal, wantNormal, gotX, gotZ)
		}
	}
	if surface.Bounds() != (maths.Box{Min: maths.V(-6, surface.base, -6),
		Max: maths.V(6, surface.top, 6)}) {
		t.Fatalf("bounds = %+v", surface.Bounds())
	}
}

func TestFlatMeanContinuationNamesItsSpectralNormalRefusal(t *testing.T) {
	core := continuationCore(t, false)
	heights, _, _, _, _, _ := core.Pack()
	slopes := make([]float64, len(heights))
	for i := range slopes {
		slopes[i] = .125
	}
	spectral, err := core.WithApproximateSpectralNormals(slopes, slopes)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewFlatMeanContinuation(spectral, 0, 1); err == nil ||
		!strings.Contains(err.Error(), "shading-normal \"spectral-slopes\"") {
		t.Fatalf("flat-mean continuation error = %v, want named spectral-normal refusal", err)
	}
}

func TestFlatMeanContinuationHeightAndSlopeAreContinuousAtEdgesAndCorners(t *testing.T) {
	core := continuationCore(t, false)
	continued, err := NewFlatMeanContinuation(core, -.25, 2)
	if err != nil {
		t.Fatal(err)
	}
	surface := continued
	half := surface.half
	epsilon := 1e-8
	points := []struct{ x, z, nx, nz float64 }{
		{half, 0, 1, 0}, {-half, 0, -1, 0},
		{0, half, 0, 1}, {0, -half, 0, -1},
		{half, half, 1, 1}, {half, -half, 1, -1},
		{-half, half, -1, 1}, {-half, -half, -1, -1},
	}
	for _, point := range points {
		inX, inZ := point.x-point.nx*epsilon, point.z-point.nz*epsilon
		outX, outZ := point.x+point.nx*epsilon, point.z+point.nz*epsilon
		inside, ix, iz := surface.sample(inX, inZ)
		outside, ox, oz := surface.sample(outX, outZ)
		if math.Abs(outside-inside) > 1e-6 {
			t.Errorf("height seam at (%g,%g): inner %.12g outer %.12g",
				point.x, point.z, inside, outside)
		}
		if math.Abs(ox-ix) > 1e-6 || math.Abs(oz-iz) > 1e-6 {
			t.Errorf("slope seam at (%g,%g): inner (%g,%g) outer (%g,%g)",
				point.x, point.z, ix, iz, ox, oz)
		}
	}

	outer := surface.outerHalf
	for _, point := range []struct{ x, z float64 }{{outer, 0}, {-outer, 0},
		{0, outer}, {0, -outer}, {outer, outer}, {-outer, outer}} {
		height, dx, dz := surface.sample(point.x, point.z)
		if height != surface.mean || dx != 0 || dz != 0 {
			t.Errorf("outer continuation at (%g,%g) = (%g,%g,%g), want flat mean",
				point.x, point.z, height, dx, dz)
		}
	}
}

func TestFlatMeanContinuationRejectsInvalidOrOverBudgetInputs(t *testing.T) {
	core := continuationCore(t, false)
	for name, values := range map[string][2]float64{
		"zero width": {0, 0}, "negative width": {0, -1},
		"wide collar": {0, 8.0001}, "nan mean": {math.NaN(), 1},
		"infinite width": {0, math.Inf(1)},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewFlatMeanContinuation(core, values[0], values[1]); err == nil {
				t.Fatal("invalid continuation was accepted")
			}
		})
	}
	if _, err := NewFlatMeanContinuation(nil, 0, 1); err == nil {
		t.Fatal("nil core was accepted")
	}
}

func TestFlatMeanContinuationDistanceAndBoundsAreFinite(t *testing.T) {
	continued, err := NewFlatMeanContinuation(continuationCore(t, false), 0, 3)
	if err != nil {
		t.Fatal(err)
	}
	surface := continued
	box := surface.Bounds()
	for _, point := range []maths.Vec{
		maths.V(0, 2, 0), maths.V(box.Max.X, 0, box.Max.Z),
		maths.V(box.Max.X+1, -1, 0), maths.V(0, -2, box.Min.Z-1),
	} {
		if gap := surface.Distance(point); math.IsNaN(gap) || math.IsInf(gap, 0) {
			t.Fatalf("nonfinite distance at %v: %g", point, gap)
		}
	}
}

func TestInfiniteMeanContinuationHasNoLateralEdge(t *testing.T) {
	field := continuationCore(t, false)
	continued, err := NewInfiniteMeanContinuation(field, -.25, 2)
	if err != nil {
		t.Fatal(err)
	}
	box := continued.Bounds()
	if !math.IsInf(box.Min.X, -1) || !math.IsInf(box.Max.X, 1) ||
		!math.IsInf(box.Min.Z, -1) || !math.IsInf(box.Max.Z, 1) {
		t.Fatalf("infinite continuation bounds = %+v, want unbounded x/z", box)
	}
	for _, x := range []float64{-1e6, -20, 20, 1e6} {
		point := maths.V(x, -.25, 0)
		if got := continued.Distance(point); got != 0 {
			t.Errorf("far mean plane distance at x=%g is %g, want 0", x, got)
		}
		if got := continued.Normal(point); got != maths.Up {
			t.Errorf("far mean plane normal at x=%g is %v, want up", x, got)
		}
		if got := continued.Distance(maths.V(x, -1, 0)); !(got < 0) {
			t.Errorf("point below far mean plane at x=%g has signed distance %g", x, got)
		}
		if got := continued.Distance(maths.V(x, 1, 0)); !(got > 0) {
			t.Errorf("point above far mean plane at x=%g has signed distance %g", x, got)
		}
	}
	finite, err := NewFlatMeanContinuation(field, -.25, 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, point := range []maths.Vec{
		maths.V(-3, .4, -3), maths.V(0, .5, 0), maths.V(3, .2, 3),
	} {
		if got, want := continued.Normal(point), finite.Normal(point); got != want {
			t.Errorf("infinite option changed core/collar normal at %v: got %v, want %v", point, got, want)
		}
	}
}

func TestInfiniteMeanContinuationSolvesFarPlaneAndRejectsCoplanarRay(t *testing.T) {
	field, err := shape.NewHeightField(make([]float64, 9), 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	surface, err := NewInfiniteMeanContinuation(field, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	vertical := maths.Ray{At: maths.V(10, 2, -10), Dir: maths.V(0, -1, 0)}
	if got, hit := surface.Hit(vertical, 0, 3); !hit || got != 2 {
		t.Fatalf("far-plane hit = %g, %t; want exact t=2", got, hit)
	}
	coplanar := maths.Ray{At: maths.V(-10, 0, -10), Dir: maths.V(1, 0, 0)}
	if got, hit := surface.Hit(coplanar, 0, 30); hit {
		t.Fatalf("coplanar ray reported a non-isolated hit at %g", got)
	}
}

func TestInfiniteMeanContinuationUsesExactCoreHitForGrazingCrossing(t *testing.T) {
	core, err := NewHeightField(HeightFieldConfig{
		Sea:    oceantex.Sea{Wind: 7, Fetch: 50000, Toward: 0},
		Extent: 1024, Samples: 513, Seed: 7, Seconds: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	surface, err := NewInfiniteMeanContinuation(core, 0, 4)
	if err != nil {
		t.Fatal(err)
	}
	// Captured from the mean-sun proof's actual scattering sample. The
	// segment starts only 21 micrometres below this wave and reaches air at
	// the scattering sample, while the old conservative marcher exhausted
	// its step budget before finding the crossing.
	ray := maths.Ray{
		At:  maths.V(-16.38955244928809, -0.12612924633123349, -30.671611397581984),
		Dir: maths.V(-0.24855215536174888, 0.004632484328451868, -0.9686074365572338),
	}
	coreHit, coreMeets := surface.core.IntersectExact(ray, 0, math.Inf(1))
	if !coreMeets {
		t.Fatal("exact HeightField solver did not find the captured core crossing")
	}
	continuedHit, continuationMeets := surface.Hit(ray, 0, math.Inf(1))
	if !continuationMeets || math.Abs(continuedHit-coreHit) > 1e-9 {
		t.Fatalf("continued hit = (%g,%t), exact core hit = %g; want same first boundary", continuedHit, continuationMeets, coreHit)
	}
	point := ray.Point(continuedHit)
	if math.Abs(surface.Distance(point)) > 1e-8 {
		t.Fatalf("continued hit residual at %v is %g", point, surface.Distance(point))
	}
}

func TestInfiniteMeanContinuationFindsCapturedSkySampleInCollar(t *testing.T) {
	core, err := NewHeightField(HeightFieldConfig{
		Sea:    oceantex.Sea{Wind: 7, Fetch: 50000, Toward: 0},
		Extent: 512, Samples: 257, Seed: 7, Seconds: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	surface, err := NewInfiniteMeanContinuation(core, 0, 64)
	if err != nil {
		t.Fatal(err)
	}
	ray := maths.Ray{
		At:  maths.V(-161.45527311709725, -4.8277966920564825, 275.6393638580725),
		Dir: maths.V(.7498211549005757, .6616405637984774, 0),
	}
	if gap := surface.Distance(ray.At); gap >= 0 || math.Abs(gap+.19163615777287435) > 1e-6 {
		t.Fatalf("captured ray start gap=%g, want -0.1916361578", gap)
	}
	hit, ok := surface.Hit(ray, 0, math.Inf(1))
	if !ok {
		t.Fatal("captured underwater sky ray missed the infinite mean collar")
	}
	if residual := surface.Distance(ray.Point(hit)); math.Abs(residual) > 1e-7 {
		t.Fatalf("captured ray hit residual=%g at %v", residual, ray.Point(hit))
	}
}

func TestInfiniteMeanContinuationFindsMoonlightViewRayBeforeAirSideSample(t *testing.T) {
	core, err := NewHeightField(HeightFieldConfig{
		Sea:    oceantex.Sea{Wind: 7, Fetch: 50000, Toward: 0},
		Extent: 512, Samples: 257, Seed: 7, Seconds: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	surface, err := NewInfiniteMeanContinuation(core, 0, 64)
	if err != nil {
		t.Fatal(err)
	}
	// Captured from the actual moonlit Alfa draw after one mirror leg. The
	// source quadrature sampled this view segment at t=25 just air-side, while
	// the initial seawater stack remained held.
	ray := maths.Ray{
		At:  maths.V(-131.886821540482, -0.14435806042123842, 244.77062395449974),
		Dir: maths.V(-0.6140080995203548, 0.0033849715853585234, 0.7892924652438844),
	}
	const sampleAlong = 25.0
	if startGap := surface.Distance(ray.At); !(startGap < 0) {
		t.Fatalf("captured view segment starts outside water, gap=%g", startGap)
	}
	sample := ray.Point(sampleAlong)
	if sampleGap := surface.Distance(sample); !(sampleGap > 0) ||
		math.Abs(sampleGap-9.262488410291872e-05) > 1e-8 {
		t.Fatalf("captured scatter sample gap=%g at %v, want air-side +9.2624884e-5",
			sampleGap, sample)
	}
	coreBox := maths.Box{Min: maths.V(-surface.finite.half,
		math.Inf(-1), -surface.finite.half),
		Max: maths.V(surface.finite.half, math.Inf(1), surface.finite.half)}
	_, coreEnd, crossesCore := coreBox.Hit(ray, 0, sampleAlong)
	if !crossesCore {
		t.Fatal("captured view segment does not cross the sampled core edge")
	}
	seamProbe := ray.Point(math.Nextafter(coreEnd, math.Inf(1)))
	if math.Abs(seamProbe.Z) > surface.finite.half ||
		math.Abs(sample.Z) <= surface.finite.half {
		t.Fatalf("test no longer exercises a rounded core seam: seam probe z=%g, sample z=%g",
			seamProbe.Z, sample.Z)
	}
	hit, ok := surface.Hit(ray, 0, sampleAlong)
	if !ok {
		t.Fatal("missed the water crossing before the captured air-side sample")
	}
	if !(hit < sampleAlong) {
		t.Fatalf("water crossing t=%g is not before the source sample t=%g",
			hit, sampleAlong)
	}
	if residual := surface.Distance(ray.Point(hit)); math.Abs(residual) > 1e-7 {
		t.Fatalf("captured view-ray hit residual=%g at %v", residual,
			ray.Point(hit))
	}
}

func TestInfiniteMeanContinuationFindsTIRViewRayCrossingInCollar(t *testing.T) {
	core, err := NewHeightField(HeightFieldConfig{
		Sea:    oceantex.Sea{Wind: 7, Fetch: 50000, Toward: 0},
		Extent: 512, Samples: 257, Seed: 7, Seconds: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	surface, err := NewInfiniteMeanContinuation(core, 0, 64)
	if err != nil {
		t.Fatal(err)
	}
	// Captured from the matched post-fix Alfa draw: the preceding clear
	// boundary was total internal reflection, the source segment remains in
	// the initial water stack, and a t=5m scatter sample has crossed to air.
	ray := maths.Ray{
		At: maths.V(-166.0169509943855, 0.08982898583772062,
			263.3858811823552),
		Dir: maths.V(-0.6453308184336725, -0.07432086757624173,
			0.7602792535788684),
	}
	const sampleAlong = 5.0
	if startGap := surface.Distance(ray.At); !(startGap < 0) ||
		math.Abs(startGap+5.95840978470609e-05) > 1e-8 {
		t.Fatalf("captured TIR view-ray start gap=%g, want water-side -5.95841e-5",
			startGap)
	}
	if sampleGap := surface.Distance(ray.Point(sampleAlong)); !(sampleGap > 0) || math.Abs(sampleGap-0.005725400499896389) > 1e-8 {
		t.Fatalf("captured TIR source sample gap=%g, want air-side +0.0057254",
			sampleGap)
	}
	hit, ok := surface.Hit(ray, 0, sampleAlong)
	if !ok || !(hit < sampleAlong) {
		t.Fatalf("missed collar crossing before the captured air-side sample: hit=(%g,%t)",
			hit, ok)
	}
	if residual := surface.Distance(ray.Point(hit)); math.Abs(residual) > 1e-7 {
		t.Fatalf("captured TIR view-ray hit residual=%g at %v", residual,
			ray.Point(hit))
	}
}

func TestInfiniteMeanContinuationFindsCollarHitBeforeCore(t *testing.T) {
	surface, err := NewInfiniteMeanContinuation(continuationCore(t, false), -.25, 2)
	if err != nil {
		t.Fatal(err)
	}
	ray := maths.Ray{At: maths.V(5, .2, 0), Dir: maths.V(-1, 0, 0)}
	coreStart := 1.0 // x=4 is the core's east edge.
	hit, ok := surface.Hit(ray, 0, math.Inf(1))
	if !ok || !(hit < coreStart) {
		t.Fatalf("collar hit = (%g,%t), want a hit before core entry t=%g", hit, ok, coreStart)
	}
	if residual := surface.Distance(ray.Point(hit)); math.Abs(residual) > 1e-7 {
		t.Fatalf("collar hit residual = %g at %v", residual, ray.Point(hit))
	}
}

func TestFlatMeanContinuationOwnsSamplesAndBounds(t *testing.T) {
	const samples, extent = 9, 8.0
	heights := make([]float64, samples*samples)
	for i := range heights {
		heights[i] = .01 * float64(i-40)
	}
	core, err := shape.NewHeightField(heights, samples, extent)
	if err != nil {
		t.Fatal(err)
	}
	continued, err := NewFlatMeanContinuation(core, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	surface := continued
	point := maths.V(0, 0, 0)
	beforeHeight, _, _ := surface.sample(point.X, point.Z)
	beforeBounds := surface.Bounds()
	for i := range heights {
		heights[i] = 1000
	}
	afterHeight, _, _ := surface.sample(point.X, point.Z)
	if afterHeight != beforeHeight || surface.Bounds() != beforeBounds {
		t.Fatalf("caller mutation changed owned continuation: height %g=>%g bounds %+v=>%+v",
			beforeHeight, afterHeight, beforeBounds, surface.Bounds())
	}
}

func TestFlatMeanContinuationSlopeBoundEnclosesNumericalSamples(t *testing.T) {
	continued, err := NewFlatMeanContinuation(continuationCore(t, false), -.25, 2)
	if err != nil {
		t.Fatal(err)
	}
	surface := continued
	bound := 1 / surface.slant
	for row := range 257 {
		z := -surface.outerHalf + 2*surface.outerHalf*float64(row)/256
		for column := range 257 {
			x := -surface.outerHalf + 2*surface.outerHalf*float64(column)/256
			height, dx, dz := surface.sample(x, z)
			if !finite(height) || !finite(dx) || !finite(dz) {
				t.Fatalf("nonfinite sample at (%g,%g): %g (%g,%g)", x, z, height, dx, dz)
			}
			if slope := math.Hypot(dx, dz); slope > bound*(1+1e-12)+1e-12 {
				t.Fatalf("slope %g at (%g,%g) exceeds conservative bound %g", slope, x, z, bound)
			}
		}
	}
}

func TestInfiniteMeanContinuationFindsMatchedRays4MoonlightCrossing(t *testing.T) {
	core, err := NewHeightField(HeightFieldConfig{
		Sea:    oceantex.Sea{Wind: 7, Fetch: 50000, Toward: 0},
		Extent: 512, Samples: 257, Seed: 7, Seconds: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	surface, err := NewInfiniteMeanContinuation(core, 0, 64)
	if err != nil {
		t.Fatal(err)
	}
	// Captured from the 4-rays-per-side Alfa render. Its reflected view ray
	// remains in the initial water stack after TIR, while this source sample
	// is slightly above the continuation collar.
	ray := maths.Ray{
		At: maths.V(-275.99602763588, -0.1602377857834731,
			173.73196218084303),
		Dir: maths.V(-0.8313319789250135, 0.1395933109731877,
			0.5379598947395277),
	}
	const sampleAlong = 7.9281609747613855
	if gap := surface.Distance(ray.At); !(gap < 0) ||
		math.Abs(gap+6.604033739501678e-05) > 1e-8 {
		t.Fatalf("captured rays4 view start gap=%g, want water-side -6.60403e-5", gap)
	}
	sampleGap := surface.Distance(ray.Point(sampleAlong))
	if !(sampleGap > 0) || math.Abs(sampleGap-5.377404505635781e-05) > 1e-8 {
		t.Fatalf("captured rays4 source sample gap=%g, want air-side +5.37740e-5", sampleGap)
	}
	hit, ok := surface.Hit(ray, 0, sampleAlong)
	t.Logf("captured rays4 water crossing t=%0.12g sample t=%0.12g start=%v sample=%v hitPoint=%v",
		hit, sampleAlong, ray.At, ray.Point(sampleAlong), ray.Point(hit))
	if !ok || !(hit < sampleAlong) {
		t.Fatalf("missed collar crossing before captured rays4 source sample: hit=(%g,%t)", hit, ok)
	}
	fullHit, fullMeets := surface.Hit(ray, 0, math.Inf(1))
	if !fullMeets || math.Abs(fullHit-hit) > 1e-8 {
		t.Fatalf("extending far changed first collar hit: bounded=(%g,%t), infinite=(%g,%t)",
			hit, ok, fullHit, fullMeets)
	}
	if residual := surface.Distance(ray.Point(hit)); math.Abs(residual) > 1e-7 {
		t.Fatalf("captured rays4 collar hit residual=%g at %v", residual,
			ray.Point(hit))
	}
}

func TestInfiniteMeanContinuationFindsMatchedRays4WaterToSkyCollarCrossing(t *testing.T) {
	core, err := NewHeightField(HeightFieldConfig{
		Sea:    oceantex.Sea{Wind: 7, Fetch: 50000, Toward: 0},
		Extent: 512, Samples: 257, Seed: 7, Seconds: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	surface, err := NewInfiniteMeanContinuation(core, 0, 64)
	if err != nil {
		t.Fatal(err)
	}
	// Captured from the actual adaptive 4-rays moonlit render. This source ray
	// begins in water in the north-east collar and points upward toward sky.
	ray := maths.Ray{
		At: maths.V(29.616617015481552, -8.489090245342176,
			283.84295929193553),
		Dir: maths.V(0.5970046455787908, 0.7720430635003749,
			0.21802513905120813),
	}
	if gap := surface.Distance(ray.At); !(gap < 0) ||
		math.Abs(gap+0.2348771131243448) > 1e-9 {
		t.Fatalf("captured water-side source gap=%g", gap)
	}
	hit, ok := surface.Hit(ray, 0, math.Inf(1))
	if !ok {
		t.Fatal("missed the collar crossing from underwater toward the far field")
	}
	t.Logf("captured water-to-sky collar hit t=%0.12g point=%v", hit, ray.Point(hit))
	if hit < 10 || hit > 20 {
		t.Fatalf("captured water-to-sky first hit t=%g, want the measured crossing between 10 and 20", hit)
	}
	boundedHit, bounded := surface.Hit(ray, 0, hit+.1)
	if !bounded || math.Abs(boundedHit-hit) > 1e-8 {
		t.Fatalf("near/far interval changed first collar hit: full=(%g,%t), bounded=(%g,%t)",
			hit, ok, boundedHit, bounded)
	}
	if residual := surface.Distance(ray.Point(hit)); math.Abs(residual) > 1e-7 {
		t.Fatalf("captured water-to-sky collar hit residual=%g at %v", residual,
			ray.Point(hit))
	}
}

func TestInfiniteMeanContinuationGridSeamsAdvanceFromExactBoundaries(t *testing.T) {
	core := continuationCore(t, false)
	surface, err := NewInfiniteMeanContinuation(core, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	step := surface.finite.coreExtent / float64(surface.finite.samples-1)
	index := surface.finite.samples / 2
	line := -surface.finite.half + float64(index)*step
	for _, direction := range []float64{1, -1} {
		ray := maths.Ray{At: maths.V(line, 0, 0), Dir: maths.V(direction, 0, 0)}
		seam := surface.nextCoreGridSeam(ray, 0, 10*step)
		if !(seam > 0) || math.Abs(seam-step) > 1e-12 {
			t.Fatalf("direction %g exact-grid seam=%g, want next adjacent seam at %g",
				direction, seam, step)
		}

		along := 0.0
		for seen := 0; seen < surface.finite.samples; seen++ {
			next := surface.nextCoreGridSeam(ray, along, 10*step)
			if math.IsInf(next, 1) {
				break
			}
			if !(next-along > step*.5) {
				t.Fatalf("direction %g seam walk stalled at %g -> %g", direction,
					along, next)
			}
			along = next
			if seen == surface.finite.samples-1 {
				t.Fatalf("direction %g seam walk exceeded the finite core grid", direction)
			}
		}
	}
}

func TestInfiniteMeanContinuationFlatMeanPlaneHitsInsideCollar(t *testing.T) {
	const samples, extent, collar = 257, 512, 64
	core, err := shape.NewHeightField(make([]float64, samples*samples),
		samples, extent)
	if err != nil {
		t.Fatal(err)
	}
	surface, err := NewInfiniteMeanContinuation(core, 0, collar)
	if err != nil {
		t.Fatal(err)
	}
	// Captured by the authored flat-water gradient-sky draw. The plane
	// crossing lies just outside the finite core, inside its collar where
	// the ordinary meanPlaneHit shortcut intentionally declines the hit.
	ray := maths.Ray{
		At: maths.V(62.15806297849508, -28.320865357159185, 230.44421525320635),
		Dir: maths.V(-0.17795689499028677, 0.7280793581752806,
			0.6619907791838854),
	}
	want := -ray.At.Y / ray.Dir.Y
	crossing := ray.Point(want)
	if crossing.Z <= extent/2 || crossing.Z >= extent/2+collar {
		t.Fatalf("captured crossing %v is not inside the authored collar", crossing)
	}
	hit, ok := surface.Hit(ray, 0, math.Inf(1))
	if !ok || math.Abs(hit-want) > 1e-10 {
		t.Fatalf("flat mean-plane collar hit=(%g,%t), want analytic crossing t=%g",
			hit, ok, want)
	}
	point := ray.Point(hit)
	if math.Abs(surface.Distance(point)) > 1e-12 {
		t.Fatalf("flat mean-plane hit has signed gap %g at %v",
			surface.Distance(point), point)
	}
}

func TestInfiniteMeanContinuationFlatMeanPlaneHonorsRayInterval(t *testing.T) {
	surface, err := NewInfiniteMeanContinuation(continuationCore(t, true), 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	ray := maths.Ray{At: maths.V(5, -2, 0), Dir: maths.Up}
	for _, bounds := range [][2]float64{{0, 1}, {2, 3}, {0, 2}} {
		if along, ok := surface.Hit(ray, bounds[0], bounds[1]); ok {
			t.Fatalf("accepted flat hit %g outside open interval %v", along, bounds)
		}
	}
	if _, ok := surface.Hit(maths.Ray{At: maths.V(5, -2, 0),
		Dir: maths.V(1, 0, 0)}, 0, math.Inf(1)); ok {
		t.Fatal("accepted parallel ray as a flat-plane crossing")
	}
}

func TestInfiniteMeanContinuationFlatCoreAwayFromMeanKeepsCollar(t *testing.T) {
	surface, err := NewInfiniteMeanContinuation(continuationCore(t, true), 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	// A constant core is not an infinite plane when its authored mean differs.
	// Its height is zero in the core, one midway through the smooth collar,
	// and two outside it. The exact-plane shortcut must leave this unchanged.
	for _, row := range []struct{ x, along float64 }{{0, 3}, {5, 2}, {7, 1}} {
		ray := maths.Ray{At: maths.V(row.x, 3, 0), Dir: maths.V(0, -1, 0)}
		got, ok := surface.Hit(ray, 0, 5)
		if !ok || math.Abs(got-row.along) > 1e-10 {
			t.Fatalf("offset core x=%g hit=(%g,%t), want %g", row.x, got, ok, row.along)
		}
	}
}

func TestInfiniteMeanContinuationFindsPumpjet60mCapturedSourceRay(t *testing.T) {
	core, err := NewHeightField(HeightFieldConfig{
		Sea:    oceantex.Sea{Wind: 7, Fetch: 50000, Toward: 0},
		Extent: 512, Samples: 257, Seed: 7, Seconds: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	surface, err := NewInfiniteMeanContinuation(core, 0, 64)
	if err != nil {
		t.Fatal(err)
	}
	// Captured in pumpjet's 60m Alfa sky-source refusal; translate the world
	// origin by the authored ocean middle (-1500, 0, 0).
	ray := maths.Ray{
		At:  maths.V(55.2980698, -34.44412184, 255.2732234),
		Dir: maths.V(-0.5507286842, 0.6649548594, 0.5045125879),
	}
	if gap := surface.Distance(ray.At); !(gap < 0) ||
		math.Abs(gap+1.223590525) > 1e-7 {
		t.Fatalf("pumpjet captured local source ray gap=%g", gap)
	}
	hit, ok := surface.Hit(ray, 0, math.Inf(1))
	if !ok {
		t.Fatal("missed pumpjet's captured 60m lunar source ray")
	}
	t.Logf("pumpjet captured 60m source ray intersects at t=%0.12g point=%v",
		hit, ray.Point(hit))
	bounded, boundedHit := surface.Hit(ray, 0, hit+.1)
	if !boundedHit || math.Abs(bounded-hit) > 1e-8 {
		t.Fatalf("pumpjet ray near/far first hit differs: full=%g bounded=%g (hit=%t)",
			hit, bounded, boundedHit)
	}
}
