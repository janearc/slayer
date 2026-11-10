package main

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/thing"
)

func TestStandFailureReportsItsAuthored4096OfferBudget(t *testing.T) {
	given := mustParse(t)
	given.set = onFixture(t, "above: 2000", "above: 9000")
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "below: 4000", "below: 9001", 1)
	if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	read := readBoundarySet(t, given.set)
	stage := staging{given: given, recipes: read, world: thing.NewWorld(),
		have: stock{}, radii: map[string]float64{}}
	if err := stage.terrain(read); err != nil {
		t.Fatal(err)
	}
	w, h, _ := stage.ground.Size()
	for row := 0; row < h; row++ {
		for column := 0; column < w; column++ {
			if !(stage.ground.At(column, row) < 9000) {
				t.Fatal("fixture unexpectedly reaches the forbidden height band")
			}
		}
	}
	stands := read.Stands()
	if len(stands) != 1 {
		t.Fatal("fixture lost its one actual stand request")
	}
	err = stage.scatter(stands[0])
	if err == nil || err.Error() != "no ground between 9000 and 9001 m in 4096 tries" {
		t.Fatalf("real unavailable band lost its authored retry diagnostic: %v", err)
	}
	if stage.made.things != 0 || len(stage.stood) != 0 || len(stage.world.Things) != 1 {
		t.Fatal("failed placement added or counted a specimen")
	}
}

func TestStandDiscardsActualOffTerrainOffersInsteadOfUsingTheZeroPoint(t *testing.T) {
	for _, outside := range []bool{false, true} {
		name, rangeText := "inside-control", "near: 2, far: 10"
		if outside {
			name, rangeText = "all-off-terrain", "near: 1000, far: 2000"
		}
		t.Run(name, func(t *testing.T) {
			given := mustParse(t)
			text := strings.Replace(fixtureScene, "alight: true", "alight: false", 1)
			text = strings.Replace(text, "count: 12", "count: 1\n        within: {of: the stone, from: 0, to: 360, "+rangeText+"}", 1)
			text = strings.Replace(text, "above: 2000", "above: -1", 1)
			text = strings.Replace(text, "below: 4000", "below: 10000", 1)
			given.set = onFixture(t, fixtureScene, text)
			_, _, ready, err := staged(given)
			if outside {
				if err == nil || !strings.Contains(err.Error(), "no ground between -1 and 10000 m in 4096 tries") {
					t.Fatalf("off-terrain offers were accepted or lost their actual refusal: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("real nearby placement control failed: %v", err)
			}
			defer ready.live.closeBeds()
		})
	}
}

// A tiny valid wedge around the actual first DEM cell samples height0.
// Its radius is below half the two-metre sample spacing, so this exercises
// both open band edges with real offers, not an injected ground function.
func TestStandHeightBandsKeepBothActualSampleEdgesOpen(t *testing.T) {
	for _, check := range []struct {
		name, above, below string
		refused            bool
	}{
		{"interior-control", "-1", "1", false},
		{"lower-edge", "0", "1", true},
		{"upper-edge", "-1", "0", true},
	} {
		t.Run(check.name, func(t *testing.T) {
			text := strings.Replace(fixtureScene, "alight: true", "alight: false", 1)
			text = strings.ReplaceAll(text, "east: 1050", "east: 1001")
			text = strings.Replace(text, "north: 8990", "north: 8999", 1)
			text = strings.Replace(text, "count: 12", "count: 1\n        within: {of: the stone, from: 0, to: 360, near: 0, far: 0.01}", 1)
			text = strings.Replace(text, "above: 2000", "above: "+check.above, 1)
			text = strings.Replace(text, "below: 4000", "below: "+check.below, 1)
			given := mustParse(t)
			given.set = onFixture(t, fixtureScene, text)
			_, made, ready, err := staged(given)
			if check.refused {
				want := "no ground between " + check.above + " and " + check.below + " m in 4096 tries"
				if err == nil || !strings.Contains(err.Error(), want) {
					t.Fatalf("actual sampled band edge was admitted or lost its refusal: %v", err)
				}
				return
			}
			if err != nil || made.things != 3 {
				t.Fatalf("zero-height interior control did not place its actual stand: %v, %d specimens", err, made.things)
			}
			defer ready.live.closeBeds()
		})
	}
}
