package medium

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// groveLevel is a ground at one height everywhere.
type groveLevel struct{ height float64 }

// ground is the height of the ground at any place: the same everywhere.
func (level groveLevel) ground(x, z float64) float64 { return level.height }

// groveSteadyWeather is the weather of the first and third tests, read from
// the chosen numbers: a clear-to-full mist of one extinction and depth.
func groveSteadyWeather(t *testing.T) GroveWeather {
	t.Helper()
	return GroveWeather{Extinction: picked(t, "grove-extinction"),
		Depth: picked(t, "grove-depth"), Coverage: unity}
}

// groveTree is the canopy of the first test: a round one on the ground.
func groveTree(t *testing.T) Canopy {
	t.Helper()
	return Canopy{Radius: picked(t, "grove-tree-radius"),
		Height: picked(t, "grove-tree-height")}
}

// groveOutside holds the mist to its field: it is nothing below the ground,
// above its depth and beyond the canopy's radius.
func groveOutside(t *testing.T, grove *Grove, weather GroveWeather) {
	t.Helper()
	below := maths.V(nought, -picked(t, "grove-below-depth"), nought)
	above := maths.V(nought, weather.Depth, nought)
	beyond := maths.V(picked(t, "grove-beyond-out"),
		picked(t, "grove-beyond-height"), nought)
	for _, at := range []maths.Vec{below, above, beyond} {
		if grove.Density(at) != nought {
			t.Errorf("mist outside its field at %v", at)
		}
	}
}

// A grove's optical depth is the one prescribed, however many trees overlap,
// and the mist is nothing outside its field.
func TestGroveHasPrescribedOpticalDepthAndNoTreeCountMultiplier(t *testing.T) {
	weather := groveSteadyWeather(t)
	tree := groveTree(t)
	one, err := NewGrove(weather, []Canopy{tree}, groveLevel{nought}.ground)
	if err != nil {
		t.Fatal(err)
	}
	many, err := NewGrove(weather, []Canopy{tree, tree, tree}, one.ground)
	if err != nil {
		t.Fatal(err)
	}
	// Smoothstep's integral from 0 to 1 is 1/2: a vertical ray
	// through 20 m has optical depth 0.006 * 20 / 2 = 0.06.
	var tau float64
	step := picked(t, "grove-optical-step")
	for i := range pickedCount(t, "grove-optical-samples") {
		at := maths.V(nought, (float64(i)+halfway)*step, nought)
		got := one.Density(at)
		if got != many.Density(at) {
			t.Fatal("overlap multiplied the prescribed density")
		}
		tau += got * step
	}
	want := picked(t, "grove-optical-depth")
	if !(math.Abs(tau-want) <= picked(t, "grove-optical-tolerance")) {
		t.Fatalf("optical depth %g, want %g", tau, want)
	}
	groveOutside(t, one, weather)
}

// groveMoving is the weather, canopy and ground of the second test, read
// from the chosen numbers: a patchy mist carried by a wind over raised
// ground.
func groveMoving(t *testing.T) (GroveWeather, Canopy, groveLevel) {
	t.Helper()
	floor := picked(t, "grove-ground-floor")
	tree := Canopy{At: maths.V(nought, floor, nought),
		Radius: picked(t, "grove-wide-radius"),
		Height: picked(t, "grove-tree-height")}
	weather := GroveWeather{Extinction: picked(t, "grove-extinction"),
		Depth:       picked(t, "grove-depth"),
		Coverage:    picked(t, "grove-patchy-coverage"),
		Wind:        maths.V(picked(t, "grove-wind-x"), nought, nought),
		Fade:        true,
		Seed:        uint64(pickedCount(t, "grove-seed")),
		GroundFloor: floor}
	return weather, tree, groveLevel{floor}
}

// groveAdvected counts the places along a walk where the wind has carried
// the patches by what it should, and whether any were moved and nonzero.
func groveAdvected(t *testing.T, grove *Grove) (changed, nonzero bool) {
	t.Helper()
	advect := maths.V(picked(t, "grove-advect-x"), nought, nought)
	seconds := time.Duration(pickedCount(t, "grove-tick-seconds"))
	end := picked(t, "grove-walk-end")
	for x := picked(t, "grove-walk-start"); x < end; x++ {
		at := maths.V(x, picked(t, "grove-moving-height"), nought)
		shifted := at.Add(advect)
		moved := grove.Tick(seconds * time.Second).Density(shifted)
		if !(math.Abs(moved-grove.Density(at)) <=
			picked(t, "grove-advect-tolerance")) {
			t.Fatal("wind did not advect the patches")
		}
		changed = changed || moved != grove.Density(shifted)
		nonzero = nonzero || moved > nought
	}
	return changed, nonzero
}

// Wind carries the patches, zero coverage is clear and there is no mist
// below the ground.
func TestGroveCoverageWindAndGroundAffectTheField(t *testing.T) {
	weather, tree, level := groveMoving(t)
	grove, err := NewGrove(weather, []Canopy{tree}, level.ground)
	if err != nil {
		t.Fatal(err)
	}
	changed, nonzero := groveAdvected(t, grove)
	if !changed || !nonzero {
		t.Fatal("fixture failed to exercise moving mist")
	}
	weather.Coverage = nought
	clear, _ := NewGrove(weather, []Canopy{tree}, grove.ground)
	moving := maths.V(nought, picked(t, "grove-moving-height"), nought)
	if clear.Density(moving) != nought {
		t.Error("zero coverage is not clear")
	}
	underground := maths.V(nought, picked(t, "grove-ground-below-height"),
		nought)
	if grove.Density(underground) != nought {
		t.Error("mist below the ground")
	}
}

// groveInvalidCanopies are canopies a grove must refuse: an empty one, an
// endless one and one with no height number.
func groveInvalidCanopies(t *testing.T) []Canopy {
	t.Helper()
	return []Canopy{{},
		{Radius: math.Inf(unity), Height: picked(t, "grove-invalid-height")},
		{Radius: picked(t, "grove-invalid-radius"), Height: math.NaN()}}
}

// A grove refuses a canopy or a weather that is not a number.
func TestGroveRejectsInvalidFields(t *testing.T) {
	weather := GroveWeather{Extinction: picked(t, "grove-refused-extinction"),
		Depth: picked(t, "grove-depth"), Coverage: unity}
	flat := groveLevel{nought}.ground
	for _, tree := range groveInvalidCanopies(t) {
		if _, err := NewGrove(weather, []Canopy{tree}, flat); err == nil {
			t.Error("accepted invalid canopy")
		}
	}
	weather.Depth = math.NaN()
	valid := Canopy{Radius: picked(t, "grove-invalid-radius"),
		Height: picked(t, "grove-invalid-height")}
	if _, err := NewGrove(weather, []Canopy{valid}, flat); err == nil {
		t.Error("accepted invalid weather")
	}
}

// A thin mist under a wide canopy keeps the canopy index to a few buckets.
func TestThinMistDoesNotExplodeTheCanopyIndex(t *testing.T) {
	weather := GroveWeather{Extinction: picked(t, "grove-refused-extinction"),
		Depth: picked(t, "grove-thin-depth"), Coverage: unity}
	wide := Canopy{Radius: picked(t, "grove-thin-radius"),
		Height: picked(t, "grove-thin-height")}
	grove, err := NewGrove(weather, []Canopy{wide}, groveLevel{nought}.ground)
	if err != nil {
		t.Fatal(err)
	}
	if len(grove.canopies) > pickedCount(t, "grove-buckets-most") {
		t.Fatalf("one canopy occupies %d buckets", len(grove.canopies))
	}
	thin := maths.V(nought, picked(t, "grove-thin-point-height"), nought)
	if grove.Density(thin) != picked(t, "grove-thin-density") {
		t.Error("index lost the canopy")
	}
}
