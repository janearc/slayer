package main

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/biome"
	"github.com/janearc/slayer/scene/frame"
)

// rangedHeights adds a height range answer to a level source.
type rangedHeights struct {
	flatHeights
	low, high float64
	has       bool
	fail      error
	asked     [][4]float64
}

// HeightRange records the query and returns the configured answer.
func (source *rangedHeights) HeightRange(west, south, east, north float64) (float64, float64, bool, error) {
	source.asked = append(source.asked, [4]float64{west, south, east, north})
	return source.low, source.high, source.has, source.fail
}

// rangedSite is a test site whose height source also reports ranges.
func rangedSite(t *testing.T, source *rangedHeights) *siteBuilder {
	t.Helper()
	site := roomySite(t)
	site.heights = source
	return site
}

// TestForestHeightsWithoutRanges checks that a source with no range method
// leaves the caller's envelope alone.
func TestForestHeightsWithoutRanges(t *testing.T) {
	site := roomySite(t)
	low, high, err := site.forestHeights(localBounds{MaxEast: 10, MaxSouth: 10}, -5, 50)
	if err != nil || low != -5 || high != 50 {
		t.Errorf("got %v %v %v", low, high, err)
	}
}

// TestForestHeightsQueriesCap checks the source rectangle asked for: the
// chart rectangle's half diagonal over the radius, in degrees about the
// centre of the region.
func TestForestHeightsQueriesCap(t *testing.T) {
	source := &rangedHeights{low: 10, high: 30, has: true}
	site := rangedSite(t, source)
	bounds := localBounds{MinEast: -1000, MinSouth: -1000, MaxEast: 1000, MaxSouth: 1000}
	low, high, err := site.forestHeights(bounds, 0, 100)
	if err != nil || low != 10 || high != 30 {
		t.Fatalf("got %v %v %v", low, high, err)
	}
	angle := math.Hypot(2000, 2000) / (2 * 1e6)
	west, south, east, north := source.asked[0][0], source.asked[0][1], source.asked[0][2], source.asked[0][3]
	if !near((north-south)*maths.Degree, 2*angle, 1e-12) {
		t.Errorf("latitude span = %v degrees, want %v radians", north-south, 2*angle)
	}
	latitude := 20 * maths.Degree
	want := 2 * math.Asin(math.Sin(angle)/math.Cos(latitude))
	if !near((east-west)*maths.Degree, want, 1e-9) {
		t.Errorf("longitude span = %v radians, want %v", (east-west)*maths.Degree, want)
	}
	if !near((west+east)/2, 10, 1e-6) || !near((south+north)/2, 20, 1e-6) {
		t.Errorf("rectangle is centred on %v, %v, want 10, 20", (west+east)/2, (south+north)/2)
	}
}

// TestForestHeightsFallbacks checks every case in which the declared envelope
// is kept: no coverage, a polar cap and a cap crossing the antimeridian.
func TestForestHeightsFallbacks(t *testing.T) {
	bounds := localBounds{MinEast: -1000, MinSouth: -1000, MaxEast: 1000, MaxSouth: 1000}
	site := rangedSite(t, &rangedHeights{low: 10, high: 30})
	if low, high, err := site.forestHeights(bounds, 0, 100); err != nil || low != 0 || high != 100 {
		t.Errorf("no coverage: %v %v %v", low, high, err)
	}
	for name, anchor := range map[string]frame.Location{
		"pole":         {Longitude: measure.Degrees(0), Latitude: measure.Degrees(89.99), Height: measure.Metres(0)},
		"antimeridian": {Longitude: measure.Degrees(179.99), Latitude: measure.Degrees(0), Height: measure.Metres(0)},
	} {
		sphere, err := frame.NewSphere(measure.Metres(1e6), anchor)
		if err != nil {
			t.Fatal(err)
		}
		source := &rangedHeights{low: 10, high: 30, has: true}
		site := rangedSite(t, source)
		site.frame = sphere
		basis, _ := sphere.Tangent(maths.Vec{})
		site.space = maths.NewSubspace(maths.Vec{}, basis, 1)
		if low, high, err := site.forestHeights(bounds, 0, 100); err != nil || low != 0 || high != 100 {
			t.Errorf("%s: %v %v %v", name, low, high, err)
		}
		if len(source.asked) != 0 {
			t.Errorf("%s: the source was asked about a cap it cannot cover", name)
		}
	}
}

// TestForestHeightsRefusals checks a failing source, nonsense ranges and a
// range outside the declared envelope.
func TestForestHeightsRefusals(t *testing.T) {
	bounds := localBounds{MinEast: -1000, MinSouth: -1000, MaxEast: 1000, MaxSouth: 1000}
	boom := errors.New("range failed")
	cases := []struct {
		name   string
		source *rangedHeights
		want   string
	}{
		{"source error", &rangedHeights{fail: boom}, "range failed"},
		{"not a number low", &rangedHeights{low: math.NaN(), high: 5, has: true}, "invalid forest source height bounds"},
		{"not a number high", &rangedHeights{low: 1, high: math.NaN(), has: true}, "invalid forest source height bounds"},
		{"infinite low", &rangedHeights{low: math.Inf(-1), high: 5, has: true}, "invalid forest source height bounds"},
		{"infinite high", &rangedHeights{low: 1, high: math.Inf(1), has: true}, "invalid forest source height bounds"},
		{"reversed", &rangedHeights{low: 9, high: 3, has: true}, "invalid forest source height bounds"},
		{"below the envelope", &rangedHeights{low: -1, high: 5, has: true}, "exceeds declared global height envelope"},
		{"above the envelope", &rangedHeights{low: 1, high: 101, has: true}, "exceeds declared global height envelope"},
	}
	for _, tc := range cases {
		_, _, err := rangedSite(t, tc.source).forestHeights(bounds, 0, 100)
		if err == nil || !containsText(err.Error(), tc.want) {
			t.Errorf("%s: error = %v, want one containing %q", tc.name, err, tc.want)
		}
	}
}

// TestRegionBounds checks the box about a region centred on the chart origin
// with heights 0 to 100 and a 10 metre extent. Its half width is the arc
// reach (radius + 100) times the half diagonal angle, plus half the height
// range, plus the extent, and its centre is 50 m up.
func TestRegionBounds(t *testing.T) {
	site := roomySite(t)
	bounds := localBounds{MinEast: -100, MinSouth: -100, MaxEast: 100, MaxSouth: 100}
	box := site.regionBounds(bounds, 0, 100, 10)
	angle := math.Hypot(200, 200) / (2 * 1e6)
	margin := (1e6+100)*angle + 50 + 10
	centre := box.Min.Add(box.Max).Scale(0.5)
	half := box.Max.Sub(box.Min).Scale(0.5)
	if !near(centre.Len(), 50, 1e-6) || !near(centre.Y, 50, 1e-6) {
		t.Errorf("centre = %v, want 50 m up", centre)
	}
	for _, got := range []float64{half.X, half.Y, half.Z} {
		if !near(got, margin, 1e-6) || got < margin {
			t.Errorf("half width = %v, want just over %v", got, margin)
		}
	}
}

// TestCandidateBounds checks the box about a root at the chart origin whose
// height runs from 0 to 10, with a 2 metre specimen extent: it is 2 wide on
// each side horizontally and 5 + 2 above and below the middle.
func TestCandidateBounds(t *testing.T) {
	site := roomySite(t)
	root := forestCandidate{Candidate: biome.Candidate{}, minimum: 0, maximum: 10}
	box := site.candidateBounds(root, 2)
	if !near(box.Min.X, -2, 1e-9) || !near(box.Max.X, 2, 1e-9) || !near(box.Min.Z, -2, 1e-9) || !near(box.Max.Z, 2, 1e-9) {
		t.Errorf("horizontal extent = %+v", box)
	}
	if !near(box.Min.Y, -2, 1e-9) || !near(box.Max.Y, 12, 1e-9) {
		t.Errorf("vertical extent = %v to %v, want -2 to 12", box.Min.Y, box.Max.Y)
	}
	if !(box.Min.X < -2 && box.Max.X > 2) {
		t.Error("the box was not widened outward past its exact size")
	}
}
