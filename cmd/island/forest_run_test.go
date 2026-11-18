package main

import (
	"context"
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/biome"
	"github.com/janearc/slayer/scene/footprint"
	"github.com/janearc/slayer/scene/thing"
)

// lookDown gives a perspective at the given height above the site looking
// at the point target on the ground.
func lookDown(t *testing.T, eye, target maths.Vec) footprint.Perspective {
	t.Helper()
	pose, err := maths.View(eye, target, maths.V(0, 0, -1))
	if err != nil {
		t.Fatal(err)
	}
	view, err := footprint.New(*pose, measure.Degrees(60), 400, 300, 1, measure.Metres(0.1))
	if err != nil {
		t.Fatal(err)
	}
	return view
}

// forestSite is a site with a small saguaro forest about the chart origin,
// seen from the given eye. Saguaro has every detail tier and builds quickly.
func forestSite(t *testing.T, eye, target maths.Vec) *siteBuilder {
	t.Helper()
	site := roomySite(t)
	site.view = lookDown(t, eye, target)
	forest := &forestPlan{Name: "grove", Seed: 5, Cell: 10, Density: 400, Variety: 2,
		Bounds:        localBounds{MinEast: -30, MinSouth: -30, MaxEast: 30, MaxSouth: 30},
		MinimumHeight: 0, MaximumHeight: 100, Clearance: 1, Radius: 3, Height: 15,
		Thresholds: [4]float64{1, 4, 12, 40}, GeometryMinimum: "", AreaSide: 0}
	forest.Mixture = append(forest.Mixture, struct {
		Kind   string  `yaml:"kind" json:"kind"`
		Weight float64 `yaml:"weight" json:"weight"`
	}{Kind: "saguaro", Weight: 1})
	site.plan.Forest = forest
	site.plan.Budget.Cells, site.plan.Budget.Candidates = 100000, 100000
	return site
}

// TestForestNothingToDo checks that a missing forest and a zero density both
// leave the site untouched.
func TestForestNothingToDo(t *testing.T) {
	site := roomySite(t)
	if err := site.forest(0, 100); err != nil {
		t.Fatal(err)
	}
	site.plan.Forest = &forestPlan{}
	if err := site.forest(0, 100); err != nil || site.record.ForestPreparation != 0 || len(site.world.Things) != 0 {
		t.Errorf("zero density: %v", err)
	}
}

// TestForestPlacesTrees checks the non-adaptive run from close above: every
// placed tree is a record in bounds, trees and aggregates add up to the
// records, and each tree is one placed part.
func TestForestPlacesTrees(t *testing.T) {
	site := forestSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	if err := site.forest(0, 100); err != nil {
		t.Fatal(err)
	}
	record := site.record
	if record.Trees < 20 {
		t.Fatalf("only %d trees placed from a density of 400 per hectare over 0.36 hectare", record.Trees)
	}
	if record.Trees+record.Aggregated != len(record.Roots) {
		t.Errorf("trees %d + aggregated %d != %d records", record.Trees, record.Aggregated, len(record.Roots))
	}
	if len(site.world.Things) != record.Instances || record.Instances == 0 || record.Instances%record.Trees != 0 {
		t.Errorf("things %d instances %d trees %d: each tree is a whole number of parts", len(site.world.Things), record.Instances, record.Trees)
	}
	if record.ExaminedCells < 36 || record.GeneratedCandidates < record.Trees {
		t.Errorf("cells %d candidates %d", record.ExaminedCells, record.GeneratedCandidates)
	}
	tiers := map[string]bool{"aggregate": true, "hull": true, "clumps": true, "quick": true, "demo": true}
	for _, root := range record.Roots {
		at := (&forestPaint{space: site.space, sphere: site.frame}).chart(root.Position)
		if at.X < -30 || at.X >= 30 || at.Z < -30 || at.Z >= 30 || !tiers[root.Detail] || root.Kind != "saguaro" || root.Variant < 0 || root.Variant > 1 {
			t.Errorf("bad root record %+v at %v", root, at)
		}
	}
	if record.ForestPreparation <= 0 || record.ForestSelection < 0 || record.ForestPlacement < 0 {
		t.Errorf("timings %v %v %v", record.ForestPreparation, record.ForestSelection, record.ForestPlacement)
	}
}

// TestForestIdentityIsCameraIndependent checks that a root keeps its kind and
// variant when the camera moves: same identifier, same recipe.
func TestForestIdentityIsCameraIndependent(t *testing.T) {
	near := forestSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	far := forestSite(t, maths.V(60, 80, 0), maths.V(0, 7, 0))
	if err := near.forest(0, 100); err != nil {
		t.Fatal(err)
	}
	if err := far.forest(0, 100); err != nil {
		t.Fatal(err)
	}
	seen := map[biome.ID]rootRecord{}
	for _, root := range near.record.Roots {
		seen[root.ID] = root
	}
	shared := 0
	for _, root := range far.record.Roots {
		if other, found := seen[root.ID]; found {
			shared++
			if other.Kind != root.Kind || other.Variant != root.Variant || other.Position != root.Position {
				t.Errorf("root %v changed with the camera: %+v then %+v", root.ID, other, root)
			}
		}
	}
	if shared < 10 {
		t.Errorf("only %d roots in both views", shared)
	}
}

// TestForestClearingsExcludeRoots checks that a reserved clearing keeps
// trees away by its radius plus the specimen radius and clearance.
func TestForestClearingsExcludeRoots(t *testing.T) {
	open := forestSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	if err := open.forest(0, 100); err != nil {
		t.Fatal(err)
	}
	closed := forestSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	closed.clearings = []clearing{{east: 0, south: 0, radius: 8}}
	if err := closed.forest(0, 100); err != nil {
		t.Fatal(err)
	}
	if closed.record.Trees >= open.record.Trees {
		t.Errorf("a clearing removed nothing: %d then %d", open.record.Trees, closed.record.Trees)
	}
	for _, root := range closed.record.Roots {
		at := (&forestPaint{space: closed.space, sphere: closed.frame}).chart(root.Position)
		if math.Hypot(at.X, at.Z) < 8+1+1-1e-6 {
			t.Errorf("tree at %v inside the clearing", at)
		}
	}
}

// TestForestOutsideView checks that a region behind the camera is skipped
// and says so.
func TestForestOutsideView(t *testing.T) {
	site := forestSite(t, maths.V(0, 200, 0), maths.V(0, 400, 0))
	if err := site.forest(0, 100); err != nil {
		t.Fatal(err)
	}
	if site.record.RegionSkipped != "outside view" || len(site.world.Things) != 0 || site.record.RegionAggregated {
		t.Errorf("record = %+v", site.record)
	}
}

// TestForestWholeRegionAggregated checks that a region too far to resolve a
// single tree is left to the authored canopy appearance.
func TestForestWholeRegionAggregated(t *testing.T) {
	site := forestSite(t, maths.V(0, 200000, 0), maths.V(0, 7, 0))
	if err := site.forest(0, 100); err != nil {
		t.Fatal(err)
	}
	if !site.record.RegionAggregated || len(site.world.Things) != 0 || len(site.record.Roots) != 0 {
		t.Errorf("record = %+v", site.record)
	}
}

// TestForestRefusals checks the refusals before any placement.
func TestForestRefusals(t *testing.T) {
	cases := []struct {
		name   string
		change func(*forestPlan)
		want   string
	}{
		{"unknown kind", func(f *forestPlan) { f.Mixture[0].Kind = "dragon" }, "dragon"},
		{"envelope too small", func(f *forestPlan) { f.Radius = 0.1 }, "exceeds its declared visibility envelope"},
		{"no name", func(f *forestPlan) { f.Name = "" }, "invalid region"},
	}
	for _, tc := range cases {
		site := forestSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
		tc.change(site.plan.Forest)
		if err := site.forest(0, 100); err == nil || !containsText(err.Error(), tc.want) {
			t.Errorf("%s: error = %v, want one containing %q", tc.name, err, tc.want)
		}
	}
	site := forestSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	site.plan.Budget.Cells = 3
	if err := site.forest(0, 100); err == nil {
		t.Error("a cell budget of 3 was enough for 36 cells")
	}
	site = forestSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	site.heights.(*flatHeights).absent = true
	if err := site.forest(0, 100); err == nil || !containsText(err.Error(), "no source height") {
		t.Errorf("no ground: %v", err)
	}
	site = forestSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	site.ctx = ctx
	if err := site.forest(0, 100); err == nil {
		t.Error("a cancelled context was ignored")
	}
}

// paintedWorld adds an altitude-painted ground to the site's world so that
// canopy appearance has a texture to adjust.
func paintedWorld(site *siteBuilder) *altitudePaint {
	paint := &altitudePaint{forest: &forestPaint{}}
	site.world.Add(thing.Thing{Material: &thing.Material{Texture: paint}})
	return paint
}
