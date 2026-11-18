package main

import (
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/footprint"
)

// TestForestCanopyRefusal checks that a bad canopy sampling side stops the
// forest before any root is chosen.
func TestForestCanopyRefusal(t *testing.T) {
	site := forestSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	site.plan.Forest.AreaSide = 4
	if err := site.forest(0, 100); err == nil || !containsText(err.Error(), "canopy area sampling side") {
		t.Errorf("error = %v", err)
	}
	if site.record.Trees != 0 || site.record.GeneratedCandidates != 0 {
		t.Error("roots were chosen after the refusal")
	}
}

// TestForestAdaptiveEnvelopeRefusal checks that an adaptive forest asking for
// a crown floor reports the crown a saguaro cannot form.
func TestForestAdaptiveEnvelopeRefusal(t *testing.T) {
	site := adaptiveSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	site.plan.Forest.GeometryMinimum = "crown"
	if err := site.forest(0, 100); err == nil || !containsText(err.Error(), "no spatial extent") {
		t.Errorf("error = %v", err)
	}
}

// TestForestCullsRootsOutOfView checks that roots outside the frustum are
// counted as culled before any ground height is read for them.
func TestForestCullsRootsOutOfView(t *testing.T) {
	full := forestSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	if err := full.forest(0, 100); err != nil {
		t.Fatal(err)
	}
	side := forestSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	pose, err := maths.View(maths.V(-40, 12, -30), maths.V(-40, 12, 100), maths.V(0, 1, 0))
	if err != nil {
		t.Fatal(err)
	}
	side.view, err = footprint.New(*pose, measure.Degrees(60), 400, 300, 1, measure.Metres(0.1))
	if err != nil {
		t.Fatal(err)
	}
	if err := side.forest(0, 100); err != nil {
		t.Fatal(err)
	}
	if full.record.BoundedRoots != 0 {
		t.Errorf("a view over the whole region culled %d roots", full.record.BoundedRoots)
	}
	if side.record.BoundedRoots == 0 || side.record.GroundQueries >= full.record.GroundQueries {
		t.Errorf("culled %d, ground queries %d against %d for the full view", side.record.BoundedRoots, side.record.GroundQueries, full.record.GroundQueries)
	}
}

// TestForestElevationBand checks that a forest band above the ground places
// nothing although ground heights were queried, and that a fade at the band
// floor thins the trees without removing them.
func TestForestElevationBand(t *testing.T) {
	high := forestSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	high.plan.Forest.MinimumHeight = 50
	if err := high.forest(0, 100); err != nil {
		t.Fatal(err)
	}
	if high.record.Trees != 0 || high.record.GroundQueries == 0 {
		t.Errorf("trees %d ground queries %d", high.record.Trees, high.record.GroundQueries)
	}
	full := forestSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	if err := full.forest(0, 100); err != nil {
		t.Fatal(err)
	}
	faded := forestSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	// The ground is 7 m up, so a 14 m fade leaves half the density there.
	faded.plan.Forest.HeightFade = 14
	if err := faded.forest(0, 100); err != nil {
		t.Fatal(err)
	}
	if faded.record.Trees == 0 || faded.record.Trees >= full.record.Trees {
		t.Errorf("faded %d trees, full %d", faded.record.Trees, full.record.Trees)
	}
	if faded.record.Trees < full.record.Trees/4 || faded.record.Trees > 3*full.record.Trees/4 {
		t.Errorf("a half density should keep roughly half the %d trees, kept %d", full.record.Trees, faded.record.Trees)
	}
}

// TestForestBudgetsStopPlacement checks the instance and face allowances
// stop a forest that outgrows them.
func TestForestBudgetsStopPlacement(t *testing.T) {
	site := forestSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	site.plan.Budget.Instances = 5
	if err := site.forest(0, 100); err == nil || !containsText(err.Error(), "instance budget exhausted") {
		t.Errorf("instances: %v", err)
	}
	site = forestSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	// Two hull prototypes of 48 faces fit; the finer tiers do not.
	site.plan.Budget.Faces = 96
	if err := site.forest(0, 100); err == nil || !containsText(err.Error(), "stored faces") {
		t.Errorf("faces: %v", err)
	}
}

// TestVisibleRootsCellBudget checks that the allowance left for cell work
// when a region is finally selected is enforced by the selection itself.
func TestVisibleRootsCellBudget(t *testing.T) {
	site := adaptiveSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	site.plan.Budget.Cells = 20
	if err := site.forest(0, 100); err == nil {
		t.Error("twenty cells of work were enough for a 60 metre region of one metre cells")
	}
	if site.record.ExaminedRegions != 3 {
		t.Errorf("regions examined = %d, want the root, one half and one quarter of it", site.record.ExaminedRegions)
	}
}

// TestVisibleRootsHeightError checks that a failing height range ends the
// walk.
func TestVisibleRootsHeightError(t *testing.T) {
	site := adaptiveSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	site.heights = &rangedHeights{flatHeight(7), 1, 2, true, errBoom, nil}
	if err := site.forest(0, 100); err == nil || !containsText(err.Error(), "boom") {
		t.Errorf("error = %v", err)
	}
}

// flatHeight is a level source at one height.
func flatHeight(height float64) flatHeights { return flatHeights{height: height} }
