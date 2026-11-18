package shape

import (
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/janearc/slayer/maths"
)

// flat is a level field at a height, three by three over two across.
func flat(t *testing.T, level float64) *HeightField {
	t.Helper()
	heights := make([]float64, 9)
	for i := range heights {
		heights[i] = level
	}
	field, err := NewHeightField(heights, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	return field
}

// A level field reads its level everywhere over the square, faces up,
// and its distance is the height above it.
func TestALevelFieldIsItsLevel(t *testing.T) {
	field := flat(t, 0.25)
	for _, at := range []maths.Vec{maths.V(0, 1, 0), maths.V(-0.9, 1, 0.7)} {
		if d := field.Distance(at); !closeTo(d, at.Y-0.25) {
			t.Errorf("over %v the distance is %v, want %v", at, d, at.Y-0.25)
		}
		if n := field.Normal(at); !closeTo(n.Y, 1) {
			t.Errorf("over %v the field faces %v", at, n)
		}
	}
	// beside the square the field is at least the sideways gap away
	if d := field.Distance(maths.V(2, 0.25, 0)); d < 1-1e-9 {
		t.Errorf("a metre beside the square the distance is %v", d)
	}
}

// A ramp faces square to its slope, and its distance never says more
// than the truth, so a march cannot step through it.
func TestARampFacesItsSlopeAndNeverOverstates(t *testing.T) {
	// rising one in one along x: columns at -1, 0, 1
	heights := []float64{-1, 0, 1, -1, 0, 1, -1, 0, 1}
	field, err := NewHeightField(heights, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := maths.V(-1, 1, 0).Unit()
	if n := field.Normal(maths.V(0.2, 5, -0.3)); !closeTo(n.Sub(want).Len(), 0) {
		t.Errorf("the ramp faces %v, want %v", n, want)
	}
	// the bound: |Distance| is at most the true distance to the plane
	truth := func(p maths.Vec) float64 {
		return math.Abs(p.Dot(want)) // the ramp is the plane x = y
	}
	for _, at := range []maths.Vec{maths.V(0, 0.5, 0), maths.V(0.3, 0.1, 0.2),
		maths.V(-0.5, -0.2, 0.4), maths.V(0.1, 2, -0.1)} {
		if d := math.Abs(field.Distance(at)); d > truth(at)+1e-9 {
			t.Errorf("at %v the distance %v overstates %v", at, d, truth(at))
		}
	}
}

func TestHeightFieldHitSolvesBilinearPatchesExactly(t *testing.T) {
	// h(x,z)=x*z on [-1,1]². Along x=z=-1+t at y=0, the ray
	// touches the saddle at t=1 with a double root.
	saddle, err := NewHeightField([]float64{1, -1, -1, 1}, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, implementsHitter := any(saddle).(Hitter); implementsHitter {
		t.Fatal("ordinary HeightField unexpectedly opts into exact Hitter dispatch")
	}
	tangent := maths.Ray{At: maths.V(-1, 0, -1), Dir: maths.V(1, 0, 1)}
	if got, ok := saddle.IntersectExact(tangent, 0, 3); !ok || !closeTo(got, 1) {
		t.Errorf("bilinear tangent hit = %v, %t; want 1", got, ok)
	}
	if _, ok := saddle.IntersectExact(tangent, 1, 3); ok {
		t.Error("accepted a hit at the exclusive near bound")
	}
	if _, ok := saddle.IntersectExact(tangent, 0, 1); ok {
		t.Error("accepted a hit at the exclusive far bound")
	}

	// A vertical ray at (.2,.3) meets the exact bilinear height .06.
	vertical := maths.Ray{At: maths.V(.2, 2, .3), Dir: maths.V(0, -1, 0)}
	if got, ok := saddle.IntersectExact(vertical, 0, 3); !ok || !closeTo(got, 1.94) {
		t.Errorf("vertical bilinear hit = %v, %t; want 1.94", got, ok)
	}
}

func TestHeightFieldHitHandlesOriginSidesGridEdgesAndFootprint(t *testing.T) {
	field := flat(t, 0)
	for _, test := range []struct {
		name string
		ray  maths.Ray
		near float64
		far  float64
		want float64
		hit  bool
	}{
		{"approach from above", maths.Ray{At: maths.V(0, 1e-12, 0), Dir: maths.V(0, -1, 0)}, 0, 1, 1e-12, true},
		{"approach from below", maths.Ray{At: maths.V(0, -2, 0), Dir: maths.V(0, 1, 0)}, 0, 3, 2, true},
		{"depart above", maths.Ray{At: maths.V(0, 1e-12, 0), Dir: maths.V(0, 1, 0)}, 0, 1, 0, false},
		{"depart below", maths.Ray{At: maths.V(0, -1e-12, 0), Dir: maths.V(0, -1, 0)}, 0, 1, 0, false},
		{"exact origin outward", maths.Ray{At: maths.V(0, 0, 0), Dir: maths.V(0, 1, 0)}, 0, 1, 0, false},
		{"edge of adjacent cells", maths.Ray{At: maths.V(0, 2, 0), Dir: maths.V(0, -1, 0)}, 0, 3, 2, true},
		{"outer footprint edge", maths.Ray{At: maths.V(1, 2, 0), Dir: maths.V(0, -1, 0)}, 0, 3, 2, true},
		{"outside footprint", maths.Ray{At: maths.V(1.0001, 2, 0), Dir: maths.V(0, -1, 0)}, 0, 3, 0, false},
		{"parallel above", maths.Ray{At: maths.V(0, 1, 0), Dir: maths.V(1, 0, 0)}, 0, 3, 0, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, ok := field.IntersectExact(test.ray, test.near, test.far)
			if ok != test.hit || ok && !closeTo(got, test.want) {
				t.Errorf("Hit = %v, %t; want %v, %t", got, ok, test.want, test.hit)
			}
		})
	}
}

func TestHeightFieldHitTraversesCellsInRayOrder(t *testing.T) {
	// A raised centre sample creates a continuous four-cell tent. This ray
	// crosses the x grid seam before reaching the rising patch.
	field, err := NewHeightField([]float64{0, 0, 0, 0, 1, 0, 0, 0, 0}, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	ray := maths.Ray{At: maths.V(-1, 0.5, 0), Dir: maths.V(1, 0, 0)}
	got, ok := field.IntersectExact(ray, 0, 2)
	if !ok || !closeTo(got, 0.5) {
		t.Errorf("first crossed-cell hit = %v, %t; want 0.5", got, ok)
	}
	// The next field touches its peak at the x seam between cells. The
	// first interval must retain a root that lies exactly at its exit.
	peak := []float64{0, 0, 0, 0, 1, 0, 0, 0, 0}
	seam, err := NewHeightField(peak, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	across := maths.Ray{At: maths.V(-1, 1, 0), Dir: maths.V(1, 0, 0)}
	if got, ok := seam.IntersectExact(across, 0, 2); !ok || !closeTo(got, 1) {
		t.Errorf("grid-seam tangent = %v, %t; want 1", got, ok)
	}
	// Enter the local patch at a large ray parameter, cross a cell seam,
	// then hit geometry in the adjacent cell. Cell advancement must not
	// depend on Nextafter changing the reconstructed world coordinate.
	rightPeak := []float64{0, 0, 0, 0, 0, 1, 0, 0, 0}
	late, err := NewHeightField(rightPeak, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	long := maths.Ray{At: maths.V(-2, 0.5, 0), Dir: maths.V(1e-16, 0, 0)}
	if got, ok := late.IntersectExact(long, 1e16, 3e16); !ok || !closeTo(got, 2.5e16) {
		t.Errorf("large-parameter adjacent-cell hit = %v, %t; want 2.5e16", got, ok)
	}
}

func TestHeightFieldHitClipsUnboundedRangesAndRejectsInvalidRays(t *testing.T) {
	field := flat(t, 0)
	upward := maths.Ray{At: maths.V(0, -2, 0), Dir: maths.V(0, 1, 0)}
	if got, ok := field.IntersectExact(upward, math.Inf(-1), math.Inf(1)); !ok || got != 2 {
		t.Errorf("unbounded interval hit = %v, %t; want 2", got, ok)
	}
	for _, ray := range []maths.Ray{
		{At: maths.V(math.Inf(1), 0, 0), Dir: maths.V(0, 1, 0)},
		{At: maths.V(0, 0, 0), Dir: maths.V(0, math.NaN(), 1)},
		{},
	} {
		if got, ok := field.IntersectExact(ray, 0, 10); ok {
			t.Errorf("invalid or stationary ray %v hit at %v", ray, got)
		}
	}
}

func TestApproximateSmoothHeightFieldChangesOnlyShadingNormals(t *testing.T) {
	heights := []float64{0, 0, 0, 0, 1, 0, 0, 0, 0}
	plain, err := NewHeightField(heights, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	smooth, err := plain.WithApproximateSmoothNormals()
	if err != nil {
		t.Fatal(err)
	}
	plainSamples, plainN, plainExtent, plainBottom, plainTop, plainSlant := plain.Pack()
	smoothSamples, smoothN, smoothExtent, smoothBottom, smoothTop, smoothSlant := smooth.Pack()
	if !slices.Equal(plainSamples, smoothSamples) || plainN != smoothN ||
		plainExtent != smoothExtent || plainBottom != smoothBottom ||
		plainTop != smoothTop || plainSlant != smoothSlant ||
		plain.Bounds() != smooth.Bounds() {
		t.Fatal("the shading approximation changed packed geometry or bounds")
	}
	for _, point := range []maths.Vec{
		maths.V(-1, 2, -1), maths.V(-0.2, 2, 0.3),
		maths.V(0, 2, 0), maths.V(0.7, -2, -0.1),
		maths.V(1.5, 3, -1.4),
	} {
		if got, want := smooth.Distance(point), plain.Distance(point); got != want {
			t.Errorf("distance at %v = %.17g, want exact core %.17g", point, got, want)
		}
	}
	if smooth.Normal(maths.V(0, 2, 0)) == plain.Normal(maths.V(0, 2, 0)) {
		t.Fatal("the opted-in shading normal kept the cell's geometric normal")
	}
}

func TestApproximateSpectralNormalsAreShadingOnlyAndValidateSlopes(t *testing.T) {
	heights := []float64{0, 0, 0, 0, 1, 0, 0, 0, 0}
	field, err := NewHeightField(heights, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	xSlopes := []float64{.1, .2, .3, .4, .5, .6, .7, .8, .9}
	zSlopes := []float64{-.9, -.8, -.7, -.6, -.5, -.4, -.3, -.2, -.1}
	spectral, err := field.WithApproximateSpectralNormals(xSlopes, zSlopes)
	if err != nil {
		t.Fatal(err)
	}
	if spectral.ApproximateShadingNormalMode() != "spectral-slopes" ||
		spectral.HasApproximateSmoothNormals() {
		t.Fatalf("spectral mode=%q smooth=%t",
			spectral.ApproximateShadingNormalMode(),
			spectral.HasApproximateSmoothNormals())
	}
	packed, n, extent, bottom, top, slant := field.Pack()
	got, gotN, gotExtent, gotBottom, gotTop, gotSlant := spectral.Pack()
	if !reflect.DeepEqual(packed, got) || n != gotN || extent != gotExtent ||
		bottom != gotBottom || top != gotTop || slant != gotSlant ||
		field.Bounds() != spectral.Bounds() {
		t.Fatal("spectral shading normals changed packed geometry or bounds")
	}
	for _, point := range []maths.Vec{maths.V(0, .5, 0), maths.V(2, -1, -3)} {
		if got, want := spectral.Distance(point), field.Distance(point); got != want {
			t.Errorf("spectral distance at %v = %.17g, want %.17g", point, got, want)
		}
	}
	want := maths.V(-xSlopes[4], 1, -zSlopes[4]).Unit()
	if got := spectral.Normal(maths.V(0, 1, 0)); got != want {
		t.Fatalf("spectral normal = %v, want %v", got, want)
	}
	if got, want := spectral.GeometricNormal(maths.V(0, 1, 0)),
		field.GeometricNormal(maths.V(0, 1, 0)); got != want {
		t.Fatalf("spectral field geometric normal = %v, want exact cell %v",
			got, want)
	}
	if spectral.GeometricNormal(maths.V(0, 1, 0)) == want {
		t.Fatal("spectral slope leaked into the geometric normal")
	}
	if _, err := field.WithApproximateSpectralNormals(xSlopes[:8], zSlopes); err == nil {
		t.Fatal("accepted mismatched spectral slope lengths")
	}
	bad := append([]float64(nil), xSlopes...)
	bad[4] = math.NaN()
	if _, err := field.WithApproximateSpectralNormals(bad, zSlopes); err == nil {
		t.Fatal("accepted a nonfinite spectral slope")
	}
}

func TestApproximateSmoothNormalsAreExactForAPlane(t *testing.T) {
	const samples, extent = 5, 4.0
	heights := make([]float64, samples*samples)
	step := extent / (samples - 1)
	for row := range samples {
		for column := range samples {
			x := -extent/2 + float64(column)*step
			z := -extent/2 + float64(row)*step
			heights[row*samples+column] = 5 + 2*x - 3*z
		}
	}
	field, err := NewHeightField(heights, samples, extent)
	if err != nil {
		t.Fatal(err)
	}
	smooth, err := field.WithApproximateSmoothNormals()
	if err != nil {
		t.Fatal(err)
	}
	want := maths.V(-2, 1, 3).Unit()
	for _, point := range []maths.Vec{
		maths.V(-2, 20, -2), maths.V(-0.3, -7, 1.1),
		maths.V(0, 0, 0), maths.V(2, 9, 2),
	} {
		if got := smooth.Normal(point); got.Sub(want).Len() > 1e-14 {
			t.Errorf("planar normal at %v = %v, want exact %v", point, got, want)
		}
	}
}

func TestApproximateSmoothNormalsAreContinuousAcrossCells(t *testing.T) {
	field, err := NewHeightField([]float64{0, 0, 0, 0, 1, 0, 0, 0, 0}, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	smooth, err := field.WithApproximateSmoothNormals()
	if err != nil {
		t.Fatal(err)
	}
	const delta = 1e-8
	for _, edge := range []struct{ x, z float64 }{{0, 0.3}, {0.3, 0}} {
		left := smooth.Normal(maths.V(edge.x-delta, 0, edge.z))
		right := smooth.Normal(maths.V(edge.x+delta, 0, edge.z))
		if got := left.Sub(right).Len(); got > 1e-7 {
			t.Errorf("normals across cell edge (%g,%g) differ by %.9g: %v / %v",
				edge.x, edge.z, got, left, right)
		}
	}
}

func TestDefaultHeightFieldKeepsItsCellNormal(t *testing.T) {
	heights := []float64{0, 0, 0, 0, 1, 0, 0, 0, 0}
	field, err := NewHeightField(heights, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	point := maths.V(0.2, 2, 0.3)
	want := maths.V(0.7, 1, 0.8).Unit()
	if got := field.Normal(point); got != want {
		t.Fatalf("default normal = %v, want exact cell normal %v", got, want)
	}
}

// The checks: too few samples, a wrong count, no extent, a hole.
func TestAHeightFieldRefusesWhatItCannotBe(t *testing.T) {
	if _, err := NewHeightField([]float64{1}, 1, 1); err == nil {
		t.Error("one sample was not refused")
	}
	if _, err := NewHeightField(make([]float64, 8), 3, 1); err == nil {
		t.Error("eight samples for nine were not refused")
	}
	if _, err := NewHeightField(make([]float64, 9), 3, 0); err == nil {
		t.Error("no extent was not refused")
	}
	holed := make([]float64, 9)
	holed[4] = math.NaN()
	if _, err := NewHeightField(holed, 3, 1); err == nil {
		t.Error("a hole was not refused")
	}
}

// closeTo is within a billionth, absolutely.
func closeTo(got, want float64) bool { return math.Abs(got-want) < 1e-9 }
