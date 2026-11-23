package main

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/frame"
	"github.com/janearc/slayer/scene/medium"
)

// near reports whether two numbers agree within a small absolute tolerance.
func near(a, b, tolerance float64) bool { return math.Abs(a-b) <= tolerance }

// TestEdgeWeight checks the horizontal taper at hand-worked distances. Smooth
// is three times the square minus twice the cube of the fraction travelled.
func TestEdgeWeight(t *testing.T) {
	bounds := localBounds{MinEast: 0, MinSouth: 0, MaxEast: 100, MaxSouth: 40}
	cases := []struct {
		name       string
		fade, e, s float64
		want       float64
	}{
		{"west of region", 10, -1, 20, 0},
		{"east edge is outside", 10, 100, 20, 0},
		{"north of region", 10, 50, -1, 0},
		{"south edge is outside", 10, 50, 40, 0},
		{"no fade is full inside", 0, 0, 0, 1},
		{"on the edge with fade", 10, 0, 20, 0},
		{"halfway through fade", 10, 5, 20, 0.5},
		{"quarter through fade", 8, 2, 20, 3.0/16 - 2.0/64},
		{"nearest edge is the south one", 10, 50, 35, 0.5},
		{"beyond the fade", 10, 50, 20, 1},
	}
	for _, tc := range cases {
		if got := edgeWeight(bounds, tc.fade, tc.e, tc.s); !near(got, tc.want, 1e-12) {
			t.Errorf("%s: weight = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestHeightWeight checks both ends of an elevation band thin as authored.
func TestHeightWeight(t *testing.T) {
	cases := []struct {
		name   string
		fade   float64
		height float64
		want   float64
	}{
		{"below the band", 10, -1, 0},
		{"above the band", 10, 101, 0},
		{"no fade", 0, 50, 1},
		{"band floor", 10, 0, 0},
		{"halfway up the lower fade", 10, 5, 0.5},
		{"middle of band", 10, 50, 1},
		{"halfway down the upper fade", 10, 95, 0.5},
		{"band ceiling", 10, 100, 0},
	}
	for _, tc := range cases {
		if got := heightWeight(0, 100, tc.fade, tc.height); !near(got, tc.want, 1e-12) {
			t.Errorf("%s: weight = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestHeightColour checks palette steps, soft boundaries and the base colour.
func TestHeightColour(t *testing.T) {
	red, green, blue := [3]float64{1, 0, 0}, [3]float64{0, 1, 0}, [3]float64{0, 0, 1}
	bands := []heightBand{{Below: 100, Colour: red}, {Below: 200, Colour: green}, {Below: 300, Colour: blue}}
	base := maths.RGB(0.5, 0.5, 0.5)
	step := altitudePaint{bands: bands, base: base}
	soft := altitudePaint{bands: bands, base: base, transition: 10}
	cases := []struct {
		name   string
		paint  altitudePaint
		height float64
		want   maths.Light
	}{
		{"first band", step, 50, rgb(red)},
		{"band boundary belongs above", step, 100, rgb(green)},
		{"last band", step, 250, rgb(blue)},
		{"above all bands", step, 300, base},
		{"no bands", altitudePaint{base: base}, 5, base},
		{"soft, well inside", soft, 50, rgb(red)},
		{"soft, at the boundary", soft, 100, rgb(red).Scale(0.5).Add(rgb(green).Scale(0.5))},
		{"soft, start of blend", soft, 90, rgb(red)},
		{"soft, end of blend", soft, 110, rgb(green)},
		{"soft, last boundary is a step", soft, 300, base},
	}
	for _, tc := range cases {
		if got := tc.paint.heightColour(tc.height); got != tc.want {
			t.Errorf("%s: colour = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestForestPlanDetail checks tier selection against the four thresholds and
// the authored geometry floor.
func TestForestPlanDetail(t *testing.T) {
	forest := forestPlan{Thresholds: [4]float64{1, 2, 3, 4}}
	cases := []struct {
		pixels float64
		floor  string
		want   string
	}{
		{0.5, "", "aggregate"}, {1, "", "aggregate"},
		{1.5, "", "hull"}, {2, "", "hull"},
		{2.5, "", "clumps"}, {3, "", "clumps"},
		{3.5, "", "quick"}, {4, "", "quick"},
		{4.5, "", "demo"},
		{1.5, "crown", "crown"},
		{0.5, "crown", "aggregate"},
		{2.5, "crown", "clumps"},
	}
	for _, tc := range cases {
		forest.GeometryMinimum = tc.floor
		if got := forest.detail(tc.pixels); got != tc.want {
			t.Errorf("pixels %v floor %q: detail = %q, want %q", tc.pixels, tc.floor, got, tc.want)
		}
	}
}

// TestForestPlanContains checks each face of the visibility envelope.
func TestForestPlanContains(t *testing.T) {
	forest := forestPlan{Radius: 5, Height: 20}
	box := func(x0, y0, z0, x1, y1, z1 float64) maths.Box {
		return maths.Box{Min: maths.V(x0, y0, z0), Max: maths.V(x1, y1, z1)}
	}
	if err := forest.contains(box(-5, -5, -5, 5, 20, 5)); err != nil {
		t.Errorf("a box on the envelope was refused: %v", err)
	}
	for name, outside := range map[string]maths.Box{
		"west": box(-5.1, 0, 0, 1, 1, 1), "east": box(0, 0, 0, 5.1, 1, 1),
		"north": box(0, 0, -5.1, 1, 1, 1), "south": box(0, 0, 0, 1, 1, 5.1),
		"below": box(0, -5.1, 0, 1, 1, 1), "above": box(0, 0, 0, 1, 20.1, 1),
	} {
		if err := forest.contains(outside); err == nil {
			t.Errorf("%s: a box beyond the envelope was accepted", name)
		}
	}
}

// TestFiniteArea checks which numbers count as an area.
func TestFiniteArea(t *testing.T) {
	cases := map[float64]bool{0: true, 3.5: true, -1: false, math.Inf(1): false, math.Inf(-1): false, math.NaN(): false}
	for value, want := range cases {
		if got := finiteArea(value); got != want {
			t.Errorf("finiteArea(%v) = %v, want %v", value, got, want)
		}
	}
}

// TestTransformedBounds checks that a quarter turn about up swaps the box's
// horizontal extents and a shift moves it whole.
func TestTransformedBounds(t *testing.T) {
	box := maths.Box{Min: maths.V(-1, 0, -3), Max: maths.V(1, 2, 3)}
	shift := maths.NewSubspace(maths.V(10, 20, 30), maths.Identity, 1)
	moved := transformedBounds(box, shift)
	if moved.Min != maths.V(9, 20, 27) || moved.Max != maths.V(11, 22, 33) {
		t.Errorf("shifted box = %+v", moved)
	}
	quarterTurn := maths.NewSubspace(maths.Vec{}, maths.Rotation(maths.YAxis, math.Pi/2), 1)
	turned := transformedBounds(box, quarterTurn)
	if !near(turned.Max.X, 3, 1e-12) || !near(turned.Max.Z, 1, 1e-12) || !near(turned.Min.X, -3, 1e-12) || !near(turned.Min.Z, -1, 1e-12) ||
		!near(turned.Min.Y, 0, 1e-12) || !near(turned.Max.Y, 2, 1e-12) {
		t.Errorf("turned box = %+v", turned)
	}
	scaled := transformedBounds(box, maths.NewSubspace(maths.Vec{}, maths.Identity, 2))
	if scaled.Min != maths.V(-2, 0, -6) || scaled.Max != maths.V(2, 4, 6) {
		t.Errorf("scaled box = %+v", scaled)
	}
}

// TestFloorGridHeight checks bilinear interpolation on a two by two grid and
// the not-a-number result outside it.
func TestFloorGridHeight(t *testing.T) {
	floor := floorGrid{bounds: localBounds{MinEast: 0, MinSouth: 0, MaxEast: 10, MaxSouth: 20}, side: 2,
		heights: []float64{0, 10, 20, 30}}
	cases := []struct{ x, z, want float64 }{
		{0, 0, 0}, {10, 0, 10}, {0, 20, 20}, {10, 20, 30},
		{5, 0, 5}, {0, 10, 10}, {5, 10, 15}, {2.5, 5, 2.5 + 5},
	}
	for _, tc := range cases {
		if got := floor.height(tc.x, tc.z); !near(got, tc.want, 1e-12) {
			t.Errorf("height(%v, %v) = %v, want %v", tc.x, tc.z, got, tc.want)
		}
	}
	for _, out := range [][2]float64{{-0.1, 5}, {10.1, 5}, {5, -0.1}, {5, 20.1}} {
		if got := floor.height(out[0], out[1]); !math.IsNaN(got) {
			t.Errorf("height%v = %v, want NaN", out, got)
		}
	}
	wide := floorGrid{bounds: floor.bounds, side: 3, heights: []float64{0, 1, 2, 10, 11, 12, 20, 21, 22}}
	if got := wide.height(7.5, 15); !near(got, 1.5+15, 1e-12) {
		t.Errorf("three-sided grid height = %v, want 16.5", got)
	}
}

// constantDensity is a medium that reports one density everywhere inside a box.
type constantDensity struct {
	box     maths.Box
	density float64
	ticks   int
}

// Bounds returns the box of the field.
func (field constantDensity) Bounds() maths.Box { return field.box }

// Finest returns an arbitrary sampling step.
func (field constantDensity) Finest() float64 { return 1 }

// Density returns the configured value.
func (field constantDensity) Density(maths.Vec) float64 { return field.density }

// Tick returns a copy that has counted one advance.
func (field constantDensity) Tick(time.Duration) medium.Medium {
	field.ticks++
	return field
}

// TestTaperedMist checks the elliptical support, the fade at hand-worked
// distances, and the pass-through methods.
func TestTaperedMist(t *testing.T) {
	box := maths.Box{Min: maths.V(-10, 0, -5), Max: maths.V(10, 4, 5)}
	field := constantDensity{box: box, density: 2}
	mist := taperedMist{field: field, fade: 2.5, finest: 0.75}
	if mist.Bounds() != box || mist.Finest() != 0.75 {
		t.Errorf("bounds %+v finest %v", mist.Bounds(), mist.Finest())
	}
	// The smaller radius is 5, so the fade starts at a normalised distance of 0.5.
	cases := []struct {
		name string
		at   maths.Vec
		want float64
	}{
		{"centre", maths.V(0, 1, 0), 2},
		{"inside the fade start", maths.V(5, 1, 0), 2},
		{"halfway through the fade", maths.V(7.5, 1, 0), 1},
		{"on the ellipse", maths.V(10, 1, 0), 0},
		{"outside along the short axis", maths.V(0, 1, 5), 0},
		{"rectangle corner is outside the ellipse", maths.V(10, 1, 5), 0},
	}
	for _, tc := range cases {
		if got := mist.Density(tc.at); !near(got, tc.want, 1e-12) {
			t.Errorf("%s: density = %v, want %v", tc.name, got, tc.want)
		}
	}
	ticked, ok := mist.Tick(time.Second).(taperedMist)
	if !ok || ticked.fade != 2.5 || ticked.finest != 0.75 {
		t.Fatalf("tick lost the taper: %#v", mist.Tick(time.Second))
	}
	if inner, _ := ticked.field.(constantDensity); inner.ticks != 1 {
		t.Errorf("the underlying field was advanced %d times, want 1", inner.ticks)
	}
	still := taperedMist{field: stillField{box: box}, fade: 1}
	if got, ok := still.Tick(time.Second).(taperedMist); !ok || got.field != (stillField{box: box}) {
		t.Error("a field with no clock was changed by Tick")
	}
}

// stillField is a medium without a clock.
type stillField struct{ box maths.Box }

// Bounds returns the box of the field.
func (field stillField) Bounds() maths.Box { return field.box }

// Finest returns an arbitrary sampling step.
func (field stillField) Finest() float64 { return 1 }

// Density returns a fixed value.
func (field stillField) Density(maths.Vec) float64 { return 1 }

// flatHeights is a height source with one value everywhere, a missing region
// or a fixed error.
type flatHeights struct {
	height float64
	absent bool
	fail   error
	calls  int
}

// Height returns the configured answer and counts the call.
func (source *flatHeights) Height(float64, float64) (float64, bool, error) {
	source.calls++
	return source.height, !source.absent, source.fail
}

// testSphere is a small reference sphere anchored at a fixed place.
func testSphere(t *testing.T) frame.Sphere {
	t.Helper()
	sphere, err := frame.NewSphere(measure.Metres(1e6), frame.Location{
		Longitude: measure.Degrees(10), Latitude: measure.Degrees(20), Height: measure.Metres(0)})
	if err != nil {
		t.Fatal(err)
	}
	return sphere
}

// TestChartUp checks the chart centre is straight up and that distance east
// tilts the radial direction by distance over radius.
func TestChartUp(t *testing.T) {
	sphere := testSphere(t)
	basis, err := sphere.Tangent(maths.Vec{})
	if err != nil {
		t.Fatal(err)
	}
	site := &siteBuilder{frame: sphere, space: maths.NewSubspace(maths.Vec{}, basis, 1)}
	if got := site.chartUp(0, 0); got != basis.Y {
		t.Errorf("chart centre up = %v, want %v", got, basis.Y)
	}
	angle := 1000.0 / 1e6
	up := site.chartUp(1000, 0)
	if !near(up.Dot(basis.Y), math.Cos(angle), 1e-12) || !near(up.Dot(basis.X), math.Sin(angle), 1e-12) || !near(up.Len(), 1, 1e-12) {
		t.Errorf("east tilt = %v", up)
	}
	south := site.chartUp(0, 1000)
	if !near(south.Dot(basis.Z), math.Sin(angle), 1e-12) {
		t.Errorf("south tilt = %v", south)
	}
}

// TestForestPaintChart checks the chart inverts the position chart: a point
// reached by chartUp maps back to its east and south offsets.
func TestForestPaintChart(t *testing.T) {
	sphere := testSphere(t)
	basis, err := sphere.Tangent(maths.Vec{})
	if err != nil {
		t.Fatal(err)
	}
	space := maths.NewSubspace(maths.Vec{}, basis, 1)
	site := &siteBuilder{frame: sphere, space: space}
	paint := forestPaint{space: space, sphere: sphere}
	for _, at := range [][2]float64{{0, 0}, {1200, 0}, {0, -800}, {-300, 450}} {
		radius := sphere.Radius().InMetres()
		world := sphere.Centre().Add(site.chartUp(at[0], at[1]).Scale(radius))
		got := paint.chart(world)
		if !near(got.X, at[0], 1e-6) || !near(got.Z, at[1], 1e-6) || got.Y != 0 {
			t.Errorf("chart of %v = %v", at, got)
		}
	}
}

// TestFloorHeightConverges checks the solved floor of a level source: its
// height above the site origin.
func TestFloorHeightConverges(t *testing.T) {
	sphere := testSphere(t)
	basis, _ := sphere.Tangent(maths.Vec{})
	source := &flatHeights{height: 7}
	site := &siteBuilder{ctx: t.Context(), frame: sphere, heights: source,
		space: maths.NewSubspace(maths.Vec{}, basis, 1)}
	got, err := site.floorHeight(0, 0, 1e-6, 20)
	if err != nil {
		t.Fatal(err)
	}
	if !near(got, 7, 1e-6) {
		t.Errorf("floor height = %v, want 7", got)
	}
	if site.record.MistHeightQueries < 1 || source.calls != site.record.MistHeightQueries {
		t.Errorf("queries recorded %d, made %d", site.record.MistHeightQueries, source.calls)
	}
}

// TestFloorHeightRefusals checks the cancelled, absent, failing and
// non-converging cases.
func TestFloorHeightRefusals(t *testing.T) {
	sphere := testSphere(t)
	basis, _ := sphere.Tangent(maths.Vec{})
	space := maths.NewSubspace(maths.Vec{}, basis, 1)
	boom := errors.New("source failed")
	ctx := t.Context()
	cases := []struct {
		name       string
		source     *flatHeights
		iterations int
		tolerance  float64
		want       string
	}{
		{"source error", &flatHeights{fail: boom}, 5, 1e-6, "source failed"},
		{"no source", &flatHeights{absent: true}, 5, 1e-6, "missing source below mist floor"},
		{"no iterations", &flatHeights{height: 7}, 0, 1e-6, "did not converge"},
		{"tolerance too tight for one pass", &flatHeights{height: 7}, 1, 1e-9, "did not converge"},
	}
	for _, tc := range cases {
		site := &siteBuilder{ctx: ctx, frame: sphere, heights: tc.source, space: space}
		_, err := site.floorHeight(0, 0, tc.tolerance, tc.iterations)
		if err == nil || !containsText(err.Error(), tc.want) {
			t.Errorf("%s: error = %v, want one containing %q", tc.name, err, tc.want)
		}
	}
}

// TestFloorHeightStopsWhenCancelled checks that a cancelled context ends the
// search before any source query.
func TestFloorHeightStopsWhenCancelled(t *testing.T) {
	sphere := testSphere(t)
	basis, _ := sphere.Tangent(maths.Vec{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	source := &flatHeights{height: 7}
	site := &siteBuilder{ctx: ctx, frame: sphere, heights: source, space: maths.NewSubspace(maths.Vec{}, basis, 1)}
	if _, err := site.floorHeight(0, 0, 1e-6, 5); !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want cancellation", err)
	}
	if source.calls != 0 {
		t.Errorf("%d source queries after cancellation", source.calls)
	}
}
