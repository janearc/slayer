package main

import (
	"context"
	"errors"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/frame"
	"github.com/janearc/slayer/scene/thing"
)

// newTestSite builds a site on the test sphere with a level height source,
// an empty world and a mist floor store of the given byte limit.
func newTestSite(t *testing.T, height float64, limit int64) (*siteBuilder, *flatHeights) {
	t.Helper()
	sphere := testSphere(t)
	basis, err := sphere.Tangent(maths.Vec{})
	if err != nil {
		t.Fatal(err)
	}
	relative, err := frame.NewRelative(maths.Vec{})
	if err != nil {
		t.Fatal(err)
	}
	source := &flatHeights{height: height}
	return &siteBuilder{ctx: t.Context(), plan: &sitePlan{}, frame: sphere, heights: source, relative: relative,
		space: maths.NewSubspace(maths.Vec{}, basis, 1), world: thing.NewWorld(),
		store: &prototypeStore{values: map[prototypeKey]prototype{}}, floors: &floorStore{limit: limit}}, source
}

// testMist is a mist layer over a 200 by 200 metre square with a 3 by 3 floor.
func testMist() mistPlan {
	return mistPlan{PatchScale: 50, Extinction: 0.01, Depth: 20, Coverage: 0.5, Base: 10, Seed: 1,
		Bounds:    localBounds{MinEast: -100, MinSouth: -100, MaxEast: 100, MaxSouth: 100},
		FloorSide: 3, FloorTolerance: 1e-6, FloorIterations: 20}
}

// TestSiteGround checks the placed point and height at the chart centre and
// that every query is counted.
func TestSiteGround(t *testing.T) {
	site, source := newTestSite(t, 7, 0)
	position, height, err := site.ground(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if height != 7 {
		t.Errorf("height = %v, want 7", height)
	}
	if want := site.frame.Centre().Add(site.chartUp(0, 0).Scale(1e6 + 7)); !near(position.Sub(want).Len(), 0, 1e-6) {
		t.Errorf("position = %v, want %v", position, want)
	}
	if site.record.GroundQueries != 1 || source.calls != 1 {
		t.Errorf("queries recorded %d, made %d", site.record.GroundQueries, source.calls)
	}
}

// TestSiteGroundRefusals checks cancelled, absent and failing sources.
func TestSiteGroundRefusals(t *testing.T) {
	boom := errors.New("source failed")
	site, source := newTestSite(t, 0, 0)
	source.absent = true
	if _, _, err := site.ground(1, 1); err == nil || !containsText(err.Error(), "no source height") {
		t.Errorf("absent source: %v", err)
	}
	source.absent, source.fail = false, boom
	if _, _, err := site.ground(1, 1); !errors.Is(err, boom) {
		t.Errorf("failing source: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	site.ctx = ctx
	before := source.calls
	if _, _, err := site.ground(1, 1); !errors.Is(err, context.Canceled) || source.calls != before {
		t.Errorf("cancelled: %v after %d new queries", err, source.calls-before)
	}
}

// TestMistFloorSamplesSource checks the grid shape, the heights and the
// build counter. The centre sample sits at the source height, and the corners
// sag by the sphere's curvature, about d squared over twice the radius.
func TestMistFloorSamplesSource(t *testing.T) {
	site, _ := newTestSite(t, 7, 0)
	floor, err := site.mistFloor(testMist())
	if err != nil {
		t.Fatal(err)
	}
	if floor.side != 3 || len(floor.heights) != 9 {
		t.Fatalf("side %d with %d heights", floor.side, len(floor.heights))
	}
	if !near(floor.heights[4], 7, 1e-6) {
		t.Errorf("centre = %v, want 7", floor.heights[4])
	}
	sag := 7 - 2e4/(2*1e6)
	if !near(floor.heights[0], sag, 1e-3) || !near(floor.heights[8], sag, 1e-3) {
		t.Errorf("corners = %v and %v, want about %v", floor.heights[0], floor.heights[8], sag)
	}
	if site.record.MistFloorBuilds != 1 || site.record.MistFloorHits != 0 {
		t.Errorf("builds %d hits %d", site.record.MistFloorBuilds, site.record.MistFloorHits)
	}
}

// TestMistFloorCache checks reuse, byte accounting and oldest-first eviction.
// A three by three grid is 72 bytes, so a 100 byte limit holds one grid.
func TestMistFloorCache(t *testing.T) {
	site, source := newTestSite(t, 7, 100)
	first, err := site.mistFloor(testMist())
	if err != nil {
		t.Fatal(err)
	}
	queries := source.calls
	again, err := site.mistFloor(testMist())
	if err != nil {
		t.Fatal(err)
	}
	if source.calls != queries || site.record.MistFloorHits != 1 || site.record.MistFloorBuilds != 1 {
		t.Errorf("a repeat should hit: queries %d to %d, hits %d, builds %d", queries, source.calls, site.record.MistFloorHits, site.record.MistFloorBuilds)
	}
	if again.heights[4] != first.heights[4] || site.floors.bytes != 72 {
		t.Errorf("cached grid differs or bytes = %d", site.floors.bytes)
	}
	other := testMist()
	other.Bounds.MaxEast = 50
	if _, err := site.mistFloor(other); err != nil {
		t.Fatal(err)
	}
	if site.record.MistFloorEvictions != 1 || len(site.floors.values) != 1 || site.floors.bytes != 72 {
		t.Errorf("evictions %d, entries %d, bytes %d", site.record.MistFloorEvictions, len(site.floors.values), site.floors.bytes)
	}
	if _, err := site.mistFloor(testMist()); err != nil {
		t.Fatal(err)
	}
	if site.record.MistFloorBuilds != 3 {
		t.Errorf("the evicted grid should be rebuilt: builds = %d", site.record.MistFloorBuilds)
	}
}

// TestMistFloorRefusals checks each refusal before and during a build.
func TestMistFloorRefusals(t *testing.T) {
	small := testMist()
	small.FloorSide = 1
	large := testMist()
	large.FloorSide = 130
	cases := []struct {
		name  string
		mist  mistPlan
		limit int64
		setup func(*siteBuilder, *flatHeights)
		want  string
	}{
		{"side too small", small, 0, nil, "invalid mist floor sampling side"},
		{"side too large", large, 0, nil, "invalid mist floor sampling side"},
		{"store missing", testMist(), 0, func(s *siteBuilder, _ *flatHeights) { s.floors = nil }, "store is unavailable"},
		{"grid over the budget", testMist(), 50, nil, "exceeds its cache payload budget"},
		{"source missing", testMist(), 0, func(_ *siteBuilder, h *flatHeights) { h.absent = true }, "missing source below mist floor"},
		{"no iterations", func() mistPlan { m := testMist(); m.FloorIterations = 0; return m }(), 0, nil, "did not converge"},
		{"cancelled", testMist(), 0, func(s *siteBuilder, _ *flatHeights) {
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			s.ctx = ctx
		}, "canceled"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			site, source := newTestSite(t, 7, tc.limit)
			if tc.setup != nil {
				tc.setup(site, source)
			}
			if _, err := site.mistFloor(tc.mist); err == nil || !containsText(err.Error(), tc.want) {
				t.Errorf("error = %v, want one containing %q", err, tc.want)
			}
		})
	}
}

// TestMistFloorInconsistentCache checks the refusal when the byte count says
// the store is full but it holds no grid to evict.
func TestMistFloorInconsistentCache(t *testing.T) {
	site, _ := newTestSite(t, 7, 100)
	site.floors.bytes = 90
	if _, err := site.mistFloor(testMist()); err == nil || !containsText(err.Error(), "accounting is inconsistent") {
		t.Errorf("error = %v", err)
	}
}

// TestMistLayerAddsVolume checks the volume placed for a mist layer, with
// and without an edge fade, and that a bad layer places nothing.
func TestMistLayerAddsVolume(t *testing.T) {
	site, _ := newTestSite(t, 7, 0)
	plain := testMist()
	faded := testMist()
	faded.EdgeFade = 20
	site.plan.Atmosphere = []mistPlan{plain, faded}
	if err := site.mist(); err != nil {
		t.Fatal(err)
	}
	volumes := site.world.Volumes
	if len(volumes) != 2 {
		t.Fatalf("%d volumes, want 2", len(volumes))
	}
	if _, tapered := volumes[0].Medium.(taperedMist); tapered {
		t.Error("an unfaded layer was tapered")
	}
	if tapered, ok := volumes[1].Medium.(taperedMist); !ok || tapered.fade != 20 {
		t.Errorf("a faded layer is %T", volumes[1].Medium)
	}
	if volumes[0].Material.Name != "authored regional mist" {
		t.Errorf("material = %q", volumes[0].Material.Name)
	}
	box := volumes[0].Medium.Bounds()
	if box.Min != maths.V(-100, 10, -100) || box.Max != maths.V(100, 30, 100) {
		t.Errorf("deck bounds = %+v", box)
	}
}

// TestMistRefusals checks that a floor failure and a bad weather field both
// stop the layer, and that the first failing layer ends the list.
func TestMistRefusals(t *testing.T) {
	site, _ := newTestSite(t, 7, 0)
	bad := testMist()
	bad.FloorSide = 1
	site.plan.Atmosphere = []mistPlan{bad, testMist()}
	if err := site.mist(); err == nil {
		t.Error("a bad floor was accepted")
	}
	if len(site.world.Volumes) != 0 {
		t.Errorf("%d volumes placed after a failure", len(site.world.Volumes))
	}
	weather := testMist()
	weather.Coverage = 2
	if err := site.mistLayer(weather); err == nil {
		t.Error("a mist layer with coverage above one was accepted")
	}
}
