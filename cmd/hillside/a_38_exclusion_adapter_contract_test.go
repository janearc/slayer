package main

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The legacy reader accepts a clearance without a target. That must not
// invent an exclusion circle at the world origin. A named target does
// exclude its actual nearby offers; the otherwise identical control fits.
func TestActualStandExclusionsRequireATargetAndRejectNearbyOffers(t *testing.T) {
	for _, check := range []struct {
		name, block string
		refused     bool
		translated  bool
	}{
		{"no-exclusion", "", false, false},
		{"clearance-only", "        exclude: {clearance: {value: 7, unit: m}}\n", false, false},
		{"named-exclusion", "        exclude: {around: the stone, clearance: {value: 7, unit: m}}\n", true, false},
		{"translated-named-exclusion", "        exclude: {around: the stone, clearance: {value: 7, unit: m}}\n", true, true},
	} {
		t.Run(check.name, func(t *testing.T) {
			text := strings.Replace(fixtureScene, "alight: true", "alight: false", 1)
			if !check.translated {
				text = strings.ReplaceAll(text, "east: 1050", "east: 1001")
			}
			text = strings.Replace(text, "north: 8990", "north: 8999", 1)
			text = strings.Replace(text, "count: 12", "count: 1\n"+check.block+
				"        within: {of: the stone, from: 0, to: 360, near: 0, far: 0.01}", 1)
			text = strings.Replace(text, "above: 2000", "above: -1", 1)
			text = strings.Replace(text, "below: 4000", "below: 10000", 1)
			given := mustParse(t)
			given.set = onFixture(t, fixtureScene, text)
			_, made, ready, err := staged(given)
			if check.refused {
				if err == nil || !strings.Contains(err.Error(), "no ground between -1 and 10000 m in 4096 tries") {
					t.Fatalf("nearby named exclusion lost its actual refusal: %v", err)
				}
				return
			}
			if err != nil || made.things != 3 {
				t.Fatalf("a missing exclusion target blocked a valid stand: %v, %d specimens", err, made.things)
			}
			defer ready.live.closeBeds()
		})
	}
}

// The separate 3-4-5 canopy check holds enclosing-radius geometry. Here
// use the actual manufactured target radius to test the comparison itself,
// at adjacent binary64 distances and exact touching clearance.
func TestExclusionAllowsExactClearanceBeyondTheActualTargetRadius(t *testing.T) {
	text := strings.Replace(fixtureScene, "alight: true", "alight: false", 1)
	text = strings.ReplaceAll(text, "east: 1050", "east: 1001")
	text = strings.Replace(text, "count: 12", "count: 12\n        exclude: {around: the stone, clearance: {value: 7, unit: m}}", 1)
	given := mustParse(t)
	given.set = onFixture(t, fixtureScene, text)
	read := readBoundarySet(t, given.set)
	stage := staging{given: given, recipes: read, world: thing.NewWorld(),
		have: stock{}, places: map[string]maths.Vec{}, radii: map[string]float64{}}
	planet, hasPlanet := read.Planet()
	if err := stage.place(read, planet, hasPlanet); err != nil {
		t.Fatal(err)
	}
	stands := read.Stands()
	if len(stands) != 1 {
		t.Fatal("fixture lost its checked stand")
	}
	name, clearance := stands[0].Exclude()
	centre, exists := stage.places[name]
	radius := stage.radii[name]
	if !exists || name != "the stone" || clearance != 7 || centre.X != 0 || !(radius > 0 && radius < 100) {
		t.Fatal("fixture lost its actual target, finite radius or exact-distance origin")
	}
	edge := 7 + radius
	for _, check := range []struct {
		distance float64
		allowed  bool
	}{
		{0, false}, {math.Nextafter(edge, 0), false},
		{edge, true}, {math.Nextafter(edge, math.Inf(1)), true},
	} {
		point := maths.V(check.distance, centre.Y, centre.Z)
		if got := stage.allowed(stands[0], point); got != check.allowed {
			t.Fatalf("distance %.17g from radius%g with7m clearance: allowed=%v, want%v", check.distance, radius, got, check.allowed)
		}
	}
}
