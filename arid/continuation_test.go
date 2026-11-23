package arid

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
			node := maths.V(-extent/2+float64(column)*step,
				0, -extent/2+float64(row)*step)
			points = append(points, node)
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

func TestBoundedContinuationLimitsEdgeExtrapolation(t *testing.T) {
	// An edge rising one metre per cell should retain its slope as the
	// continuation approaches the flat outer surface.
	core, err := shape.NewHeightField([]float64{0, 0, 1, 0, 0, 1, 0, 0, 1}, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	surface, err := NewBoundedInfiniteMeanContinuation(core, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	inside, insideX, _ := surface.finite.sample(1, 0)
	outside, outsideX, _ := surface.finite.sample(1+1e-8, 0)
	if math.Abs(inside-outside) > 1e-6 || math.Abs(insideX-outsideX) > 1e-6 {
		t.Fatalf("edge is discontinuous: height %g/%g, slope %g/%g", inside, outside, insideX, outsideX)
	}
	for x := 1.0; x <= 3; x += .01 {
		height, _, _ := surface.finite.sample(x, 0)
		if height > 2 {
			t.Fatalf("edge extrapolation grew to %g at %g", height, x)
		}
	}
}

func TestBoundedContinuationPierWaves(t *testing.T) {
	core, err := NewHeightField(HeightFieldConfig{
		Sea:    oceantex.Sea{Wind: 4.1, Fetch: 7200, Depth: 14, Toward: .33},
		Extent: 680, Samples: 385, Seed: 183,
	})
	if err != nil {
		t.Fatal(err)
	}
	surface, err := NewBoundedInfiniteMeanContinuation(core, 0, 45)
	if err != nil {
		t.Fatal(err)
	}
	for z := -385.0; z <= 385; z += 2 {
		for x := -385.0; x <= 385; x += 2 {
			if math.Abs(x) < 340 && math.Abs(z) < 340 {
				continue
			}
			height, _, _ := surface.finite.sample(x, z)
			if math.Abs(height) > .5 {
				t.Fatalf("pier collar height %g at (%g,%g) exceeds half a metre", height, x, z)
			}
			if height < surface.finite.base || height > surface.finite.top {
				t.Fatalf("pier collar height %g exceeds its tracing bounds", height)
			}
		}
	}
	for _, point := range []maths.Vec{
		maths.V(350, 0, .37), maths.V(-360, 0, 120.37), maths.V(350, 0, 350),
	} {
		height, dx, dz := surface.finite.sample(point.X, point.Z)
		const epsilon = 1e-4
		x1, _, _ := surface.finite.sample(point.X+epsilon, point.Z)
		x0, _, _ := surface.finite.sample(point.X-epsilon, point.Z)
		z1, _, _ := surface.finite.sample(point.X, point.Z+epsilon)
		z0, _, _ := surface.finite.sample(point.X, point.Z-epsilon)
		if math.Abs(dx-(x1-x0)/(2*epsilon)) > 1e-5 ||
			math.Abs(dz-(z1-z0)/(2*epsilon)) > 1e-5 {
			t.Fatalf("bounded collar slope does not match height at %v", point)
		}
		ray := maths.Ray{At: point.Add(maths.V(0, 2, 0)), Dir: maths.Up.Neg()}
		along, hit := surface.Hit(ray, 0, 4)
		if !hit || math.Abs(along-(2-height)) > 1e-5 {
			t.Fatalf("bounded collar ray at %v hit %g/%v, want %g", point, along, hit, 2-height)
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

// Guards Hit's search of the collar for a ray that begins under the water
// there and never crosses the core (hitCoreOrCollar on the whole segment):
// without it the ray found no surface, or the mean plane's.
func TestInfiniteMeanContinuationFindsASkyRayFromUnderTheCollar(t *testing.T) {
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
	// A ray from under the water in the collar, climbing toward the sky,
	// nineteen centimetres under the surface it is made from.
	const depth = .19 // picked: well under a wave, well over a touch
	ray := underneath(surface, -161.45527311709725, 275.6393638580725, depth,
		maths.V(.7498211549005757, .6616405637984774, 0))
	if ray.At.Z <= surface.finite.half {
		t.Fatalf("ray starts at z=%g, inside the core", ray.At.Z)
	}
	if gap := surface.Distance(ray.At); math.Abs(gap+depth) > 1e-9 {
		t.Fatalf("ray start gap=%g, want -%g", gap, depth)
	}
	hit, ok := surface.Hit(ray, 0, math.Inf(1))
	if !ok {
		t.Fatal("an underwater sky ray missed the infinite mean collar")
	}
	if residual := surface.Distance(ray.Point(hit)); math.Abs(residual) > 1e-7 {
		t.Fatalf("sky ray hit residual=%g at %v", residual, ray.Point(hit))
	}
}

// Guards Hit's step past the core's edge: a ray that leaves the core under
// the water, its edge rounded inside by the ray's arithmetic, meets the
// surface in the collar beyond it (the afterCore search, begun a step past
// coreEnd); without it the crossing is lost at the seam.
func TestInfiniteMeanContinuationFindsAViewRayAcrossTheCoreEdgeBeforeAirSideSample(t *testing.T) {
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
	// A view segment that begins just under the water inside the core and
	// crosses the core's edge, whose scatter sample past the edge is just
	// air-side while the initial seawater stack remains held. It is made
	// from the surface: its start is found so that the edge rounds inside,
	// and its sample is the first point past the edge a march finds over
	// the surface.
	ray, coreEnd := seamRay(t, surface,
		maths.V(-0.6140080995203548, 0.0033849715853585234, 0.7892924652438844))
	if startGap := surface.Distance(ray.At); !(startGap < 0) {
		t.Fatalf("view segment starts outside water, gap=%g", startGap)
	}
	sampleAlong, found := firstAbove(surface, ray, coreEnd, marchStep,
		marchFar)
	if !found {
		t.Fatal("the view segment stays under the water past the core's edge")
	}
	sample := ray.Point(sampleAlong)
	if sampleGap := surface.Distance(sample); !(sampleGap > 0) ||
		math.Abs(sample.Z) <= surface.finite.half {
		t.Fatalf("scatter sample gap=%g at %v, want air-side past the core edge",
			sampleGap, sample)
	}
	hit, ok := surface.Hit(ray, 0, sampleAlong)
	if !ok {
		t.Fatal("missed the water crossing before the air-side sample")
	}
	if !(hit < sampleAlong) {
		t.Fatalf("water crossing t=%g is not before the source sample t=%g",
			hit, sampleAlong)
	}
	// the start is a millimetre from the edge and five under the surface,
	// which rises at most a millimetre in that run: no crossing before it
	if hit < coreEnd {
		t.Fatalf("water crossing t=%g is before the core's edge at t=%g",
			hit, coreEnd)
	}
	if residual := surface.Distance(ray.Point(hit)); math.Abs(residual) > 1e-7 {
		t.Fatalf("view-ray hit residual=%g at %v", residual, ray.Point(hit))
	}
}

// Guards the same collar search as the sky ray's, for a ray that begins a
// hair under the surface and dips as it goes, as a view ray does after a
// total internal reflection: the crossing is found within a step of the
// first point a march finds over the surface, not lost to the mean plane.
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
	// A view segment that follows a total internal reflection: it begins
	// in the initial water stack, a hair under the surface it is made from,
	// and a scatter sample along it has crossed to air. The sample is the
	// first point along the ray a march finds over the surface.
	const depth = 6e-5 // picked: a hair, as a total internal reflection leaves
	ray := underneath(surface, -166.0169509943855, 263.3858811823552, depth,
		maths.V(-0.6453308184336725, -0.07432086757624173, 0.7602792535788684))
	if startGap := surface.Distance(ray.At); math.Abs(startGap+depth) > 1e-9 {
		t.Fatalf("TIR view-ray start gap=%g, want water-side -%g", startGap,
			depth)
	}
	sampleAlong, found := firstAbove(surface, ray, 0, marchStep, marchFar)
	if !found {
		t.Fatal("the view segment stays under the water")
	}
	if sampleGap := surface.Distance(ray.Point(sampleAlong)); !(sampleGap > 0) {
		t.Fatalf("TIR source sample gap=%g, want air-side", sampleGap)
	}
	hit, ok := surface.Hit(ray, 0, sampleAlong)
	if !ok || !(hit < sampleAlong) {
		t.Fatalf("missed collar crossing before the air-side sample: hit=(%g,%t)",
			hit, ok)
	}
	if residual := surface.Distance(ray.Point(hit)); math.Abs(residual) > 1e-7 {
		t.Fatalf("TIR view-ray hit residual=%g at %v", residual,
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

// Guards the collar search for a grazing ray from a hair under the surface
// (hitCoreOrCollar on a segment that does not cross the core): it finds the
// crossing, and the bounded search and the unbounded one give the same
// first hit.
func TestInfiniteMeanContinuationFindsAGrazingViewRayCrossingInTheCollar(t *testing.T) {
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
	// A reflected view ray that remains in the initial water stack after a
	// total internal reflection, a hair under the continuation collar it is
	// made from, while a source sample along it is just above the collar.
	// The sample is the first point along the ray a march finds over it.
	const depth = 6.6e-5 // picked: a hair, as a grazing ray leaves
	ray := underneath(surface, -275.99602763588, 173.73196218084303, depth,
		maths.V(-0.8313319789250135, 0.1395933109731877, 0.5379598947395277))
	if gap := surface.Distance(ray.At); math.Abs(gap+depth) > 1e-9 {
		t.Fatalf("view start gap=%g, want water-side -%g", gap, depth)
	}
	sampleAlong, found := firstAbove(surface, ray, 0, marchStep, marchFar)
	if !found {
		t.Fatal("the view ray stays under the water")
	}
	sampleGap := surface.Distance(ray.Point(sampleAlong))
	if !(sampleGap > 0) {
		t.Fatalf("source sample gap=%g, want air-side", sampleGap)
	}
	hit, ok := surface.Hit(ray, 0, sampleAlong)
	t.Logf("water crossing t=%0.12g sample t=%0.12g start=%v sample=%v hitPoint=%v",
		hit, sampleAlong, ray.At, ray.Point(sampleAlong), ray.Point(hit))
	if !ok || !(hit < sampleAlong) {
		t.Fatalf("missed collar crossing before the source sample: hit=(%g,%t)", hit, ok)
	}
	fullHit, fullMeets := surface.Hit(ray, 0, math.Inf(1))
	if !fullMeets || math.Abs(fullHit-hit) > 1e-8 {
		t.Fatalf("extending far changed first collar hit: bounded=(%g,%t), infinite=(%g,%t)",
			hit, ok, fullHit, fullMeets)
	}
	if residual := surface.Distance(ray.Point(hit)); math.Abs(residual) > 1e-7 {
		t.Fatalf("collar hit residual=%g at %v", residual,
			ray.Point(hit))
	}
}

// Guards the same collar search for a source ray that climbs out of the
// collar toward the sky: the crossing is found, it is the first and not a
// later one, and it is the same whether the far end is infinite or just
// past it.
func TestInfiniteMeanContinuationFindsAWaterToSkyCollarCrossing(t *testing.T) {
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
	// A source ray that begins in water in the north-east collar, under the
	// collar it is made from, and points upward toward sky.
	const depth = 0.2348771131243448 // picked: a quarter of a metre under
	ray := underneath(surface, 29.616617015481552, 283.84295929193553, depth,
		maths.V(0.5970046455787908, 0.7720430635003749, 0.21802513905120813))
	if gap := surface.Distance(ray.At); math.Abs(gap+depth) > 1e-9 {
		t.Fatalf("water-side source gap=%g, want -%g", gap, depth)
	}
	hit, ok := surface.Hit(ray, 0, math.Inf(1))
	if !ok {
		t.Fatal("missed the collar crossing from underwater toward the far field")
	}
	t.Logf("water-to-sky collar hit t=%0.12g point=%v", hit, ray.Point(hit))
	// The surface stays between the box's base and top, so a ray that
	// climbs from under it crosses it before it climbs to the top.
	highest := (surface.finite.top - ray.At.Y) / ray.Dir.Y
	if hit <= 0 || hit > highest {
		t.Fatalf("water-to-sky first hit t=%g, want within (0, %g]", hit,
			highest)
	}
	// and it is the first crossing: a march from the start, in steps, is
	// under the surface until it, and over it at or after the hit
	first, over := firstAbove(surface, ray, 0, marchStep, marchFar)
	if !over || hit > first {
		t.Fatalf("water-to-sky hit t=%g is after the first point a march "+
			"finds over the surface, t=%g (%t)", hit, first, over)
	}
	boundedHit, bounded := surface.Hit(ray, 0, hit+.1)
	if !bounded || math.Abs(boundedHit-hit) > 1e-8 {
		t.Fatalf("near/far interval changed first collar hit: full=(%g,%t), bounded=(%g,%t)",
			hit, ok, boundedHit, bounded)
	}
	if residual := surface.Distance(ray.Point(hit)); math.Abs(residual) > 1e-7 {
		t.Fatalf("water-to-sky collar hit residual=%g at %v", residual,
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

// Guards the search past the core's edge (the afterCore search) for a ray
// that begins deep under the surface beside the edge and climbs out across
// it: found, the same with a near and far just round it, and on the surface.
func TestInfiniteMeanContinuationFindsADeepSourceRayCrossing(t *testing.T) {
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
	// A sky-source ray a metre and a fifth under the surface it is made
	// from, beside the core's edge.
	const depth = 1.223590525 // picked: a metre and a fifth, deeper than a wave's trough
	ray := underneath(surface, 55.2980698, 255.2732234, depth,
		maths.V(-0.5507286842, 0.6649548594, 0.5045125879))
	if gap := surface.Distance(ray.At); math.Abs(gap+depth) > 1e-9 {
		t.Fatalf("deep source ray gap=%g, want -%g", gap, depth)
	}
	hit, ok := surface.Hit(ray, 0, math.Inf(1))
	if !ok {
		t.Fatal("missed the deep source ray's crossing")
	}
	t.Logf("deep source ray intersects at t=%0.12g point=%v", hit,
		ray.Point(hit))
	bounded, boundedHit := surface.Hit(ray, 0, hit+.1)
	if !boundedHit || math.Abs(bounded-hit) > 1e-8 {
		t.Fatalf("deep source ray near/far first hit differs: full=%g bounded=%g (hit=%t)",
			hit, bounded, boundedHit)
	}
	if residual := surface.Distance(ray.Point(hit)); math.Abs(residual) > 1e-7 {
		t.Fatalf("deep source ray hit residual=%g at %v", residual,
			ray.Point(hit))
	}
}
