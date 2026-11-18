package main

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/biome"
)

// adaptiveSite is a forest site with a one metre cell, so the 60 metre region
// is split into quarters before its roots are listed.
func adaptiveSite(t *testing.T, eye, target maths.Vec) *siteBuilder {
	t.Helper()
	site := forestSite(t, eye, target)
	site.plan.Forest.Adaptive = true
	site.plan.Forest.Cell = 1
	site.plan.Forest.Density = 100
	return site
}

// TestCompleteEnvelope checks the envelope over every variant and detail,
// with and without the crown floor, which asks for a crown a saguaro cannot form.
func TestCompleteEnvelope(t *testing.T) {
	site := adaptiveSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	box, err := site.completeEnvelope()
	if err != nil {
		t.Fatal(err)
	}
	if box.Min.X != box.Min.Z || box.Max.X != box.Max.Z || box.Min.X != -box.Max.X || box.Min.Y > 0 || box.Max.Y < 5 {
		t.Errorf("envelope = %+v", box)
	}
	for _, key := range []prototypeKey{{kind: "saguaro", detail: "hull", seed: 5}, {kind: "saguaro", detail: "quick", seed: 6}, {kind: "saguaro", detail: "demo", seed: 5}, {kind: "saguaro", detail: "clumps", seed: 6}} {
		if _, found := site.store.values[key]; !found {
			t.Errorf("%v was not measured", key)
		}
	}
	if _, found := site.store.values[prototypeKey{kind: "saguaro", detail: "crown", seed: 5}]; found {
		t.Error("the crown was measured without the crown floor")
	}
	site.plan.Forest.GeometryMinimum = "crown"
	_, err = site.completeEnvelope()
	if err == nil || !containsText(err.Error(), "no spatial extent") {
		t.Errorf("the crown floor should measure the crown, which a saguaro cannot form: %v", err)
	}
}

// TestCompleteEnvelopeRefusals checks an unknown kind and a specimen outside
// the declared envelope.
func TestCompleteEnvelopeRefusals(t *testing.T) {
	site := adaptiveSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	site.plan.Forest.Radius = 0.1
	if _, err := site.completeEnvelope(); err == nil || !containsText(err.Error(), "visibility envelope") {
		t.Errorf("small envelope: %v", err)
	}
	site = adaptiveSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	site.plan.Forest.Mixture[0].Kind = "dragon"
	if _, err := site.completeEnvelope(); err == nil {
		t.Error("unknown kind accepted")
	}
}

// TestAdaptiveForest checks the adaptive run: regions are examined before
// cells, roots come back identified, and the work counters are filled.
func TestAdaptiveForest(t *testing.T) {
	site := adaptiveSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	if err := site.forest(0, 100); err != nil {
		t.Fatal(err)
	}
	record := site.record
	if record.ExaminedRegions < 4 || record.ExaminedCells < 1 {
		t.Errorf("regions %d cells %d: a 60 metre region of one metre cells must be split", record.ExaminedRegions, record.ExaminedCells)
	}
	if len(record.Roots) == 0 || record.Trees == 0 {
		t.Fatalf("roots %d trees %d", len(record.Roots), record.Trees)
	}
	if record.ExaminedRegions+record.ExaminedCells > site.plan.Budget.Cells {
		t.Errorf("work %d exceeds the budget", record.ExaminedRegions+record.ExaminedCells)
	}
}

// TestVisibleRootsRefusals checks the budget, a cancelled context and the
// regions that are skipped.
func TestVisibleRootsRefusals(t *testing.T) {
	site := adaptiveSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	forest := site.plan.Forest
	region, err := biome.New(biome.Definition{Name: "g", Seed: 1, Cell: metres(1), PerHectare: 100,
		Bounds:  biome.Rect{MinEast: -30, MinSouth: -30, MaxEast: 30, MaxSouth: 30},
		Species: []biome.Species{{Recipe: "saguaro", Weight: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	envelope := maths.Box{Min: maths.V(-3, -3, -3), Max: maths.V(3, 15, 3)}
	site.plan.Budget.Cells = 1
	if _, err := site.visibleRoots(region, forest.Bounds, 0, 100, envelope, 10); err == nil || !containsText(err.Error(), "budget exhausted") {
		t.Errorf("budget: %v", err)
	}
	site.plan.Budget.Cells = 100000
	ctx, cancel := cancelledContext(t)
	cancel()
	site.ctx = ctx
	if _, err := site.visibleRoots(region, forest.Bounds, 0, 100, envelope, 10); err == nil {
		t.Error("cancelled context accepted")
	}
	site = adaptiveSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	site.plan.Forest.MinimumHeight, site.plan.Forest.MaximumHeight = 200, 300
	roots, err := site.visibleRoots(region, forest.Bounds, 0, 100, envelope, 10)
	if err != nil || len(roots) != 0 || site.record.ExaminedRegions != 1 {
		t.Errorf("a height band above the source: %d roots, %d regions, %v", len(roots), site.record.ExaminedRegions, err)
	}
	site = adaptiveSite(t, maths.V(0, 200, 0), maths.V(0, 400, 0))
	roots, err = site.visibleRoots(region, forest.Bounds, 0, 100, envelope, 10)
	if err != nil || len(roots) != 0 || site.record.ExaminedRegions != 1 || site.record.AggregateRegions != 0 {
		t.Errorf("a region behind the camera: %d roots, %d regions, %v", len(roots), site.record.ExaminedRegions, err)
	}
	site = adaptiveSite(t, maths.V(0, 200000, 0), maths.V(0, 7, 0))
	roots, err = site.visibleRoots(region, forest.Bounds, 0, 100, envelope, 10)
	if err != nil || len(roots) != 0 || site.record.AggregateRegions != 1 {
		t.Errorf("a distant region: %d roots, %d aggregated, %v", len(roots), site.record.AggregateRegions, err)
	}
}
