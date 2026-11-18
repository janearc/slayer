package main

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// TestCanopyAppearanceSkipsWithoutSampling checks that a forest with no area
// side leaves the record and paint alone.
func TestCanopyAppearanceSkipsWithoutSampling(t *testing.T) {
	site := forestSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	paint := paintedWorld(site)
	if err := site.canopyAppearance(); err != nil {
		t.Fatal(err)
	}
	if site.record.CanopyAreaSamples != 0 || paint.forest.areaSampled || len(site.store.values) != 0 {
		t.Error("canopy appearance did work without an area side")
	}
}

// TestCanopyAppearanceMeasuresCover checks the sampled area, the cover it
// implies, the ray count and the reuse of measured areas.
func TestCanopyAppearanceMeasuresCover(t *testing.T) {
	site := forestSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	site.plan.Forest.AreaSide = 16
	paint := paintedWorld(site)
	if err := site.canopyAppearance(); err != nil {
		t.Fatal(err)
	}
	record := site.record
	if record.CanopyAreaSamples != 2*16*16 {
		t.Errorf("rays = %d, want one 16 by 16 grid per variant", record.CanopyAreaSamples)
	}
	extent := 0.0
	for _, seed := range []uint64{5, 6} {
		bounds := site.store.values[prototypeKey{kind: "saguaro", detail: "clumps", seed: seed}].bounds
		extent = math.Max(extent, (bounds.Max.X-bounds.Min.X)*(bounds.Max.Z-bounds.Min.Z))
	}
	if record.CanopyArea <= 0 || record.CanopyArea > extent {
		t.Errorf("mean area = %v for a footprint box of about %v", record.CanopyArea, extent)
	}
	// Density 400 per hectare, so cover is one minus exp of minus area times 0.04.
	if want := 1 - math.Exp(-400*record.CanopyArea/10000); !near(record.CanopyCover, want, 1e-12) {
		t.Errorf("cover = %v, want %v", record.CanopyCover, want)
	}
	if !paint.forest.areaSampled || !near(paint.forest.areaDensity, 400*record.CanopyArea/10000, 1e-12) {
		t.Errorf("paint = %+v", paint.forest)
	}
	if err := site.canopyAppearance(); err != nil {
		t.Fatal(err)
	}
	if site.record.CanopyAreaSamples != 2*16*16 {
		t.Errorf("a second pass sampled again: %d rays", site.record.CanopyAreaSamples)
	}
}

// TestCanopyAppearanceWeightsMixture checks that two members of one kind with
// different weights give the same mean as one member.
func TestCanopyAppearanceWeightsMixture(t *testing.T) {
	one := forestSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	one.plan.Forest.AreaSide = 16
	if err := one.canopyAppearance(); err != nil {
		t.Fatal(err)
	}
	two := forestSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
	two.plan.Forest.AreaSide = 16
	two.plan.Forest.Mixture = append(two.plan.Forest.Mixture, two.plan.Forest.Mixture[0])
	two.plan.Forest.Mixture[0].Weight, two.plan.Forest.Mixture[1].Weight = 1, 3
	if err := two.canopyAppearance(); err != nil {
		t.Fatal(err)
	}
	if !near(one.record.CanopyArea, two.record.CanopyArea, 1e-12) {
		t.Errorf("areas %v and %v differ", one.record.CanopyArea, two.record.CanopyArea)
	}
}

// TestCanopyAppearanceRefusals checks each refusal.
func TestCanopyAppearanceRefusals(t *testing.T) {
	cases := []struct {
		name   string
		change func(*forestPlan)
		want   string
	}{
		{"zero weights", func(f *forestPlan) { f.Mixture[0].Weight = 0 }, "invalid canopy mixture area"},
		{"side too small", func(f *forestPlan) { f.AreaSide = 4 }, "canopy area sampling side"},
		{"side too large", func(f *forestPlan) { f.AreaSide = 200 }, "canopy area sampling side"},
		{"envelope too small", func(f *forestPlan) { f.Radius = 0.1 }, "visibility envelope"},
		{"unknown kind", func(f *forestPlan) { f.Mixture[0].Kind = "dragon" }, "dragon"},
	}
	for _, tc := range cases {
		site := forestSite(t, maths.V(0, 40, 0), maths.V(0, 7, 0))
		site.plan.Forest.AreaSide = 16
		tc.change(site.plan.Forest)
		if err := site.canopyAppearance(); err == nil || !containsText(err.Error(), tc.want) {
			t.Errorf("%s: error = %v, want one containing %q", tc.name, err, tc.want)
		}
	}
}
