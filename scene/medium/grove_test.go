package medium

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

func TestGroveHasPrescribedOpticalDepthAndNoTreeCountMultiplier(t *testing.T) {
	weather := GroveWeather{Extinction: 0.006, Depth: 20, Coverage: 1}
	tree := Canopy{Radius: 40, Height: 30}
	one, err := NewGrove(weather, []Canopy{tree}, func(x, z float64) float64 { return 0 })
	if err != nil {
		t.Fatal(err)
	}
	many, err := NewGrove(weather, []Canopy{tree, tree, tree}, one.ground)
	if err != nil {
		t.Fatal(err)
	}
	// Smoothstep's integral from 0 to 1 is 1/2: a vertical ray
	// through 20 m has optical depth 0.006 * 20 / 2 = 0.06.
	tau := 0.0
	for i := range 2000 {
		at := maths.V(0, (float64(i)+0.5)*0.01, 0)
		got := one.Density(at)
		if got != many.Density(at) {
			t.Fatal("overlap multiplied the prescribed density")
		}
		tau += got * 0.01
	}
	if math.Abs(tau-0.06) > 1e-9 {
		t.Fatalf("optical depth %g, want 0.06", tau)
	}
	for _, at := range []maths.Vec{maths.V(0, -1, 0), maths.V(0, 20, 0), maths.V(41, 1, 0)} {
		if one.Density(at) != 0 {
			t.Errorf("mist outside its field at %v", at)
		}
	}
}

func TestGroveCoverageWindAndGroundAffectTheField(t *testing.T) {
	tree := Canopy{At: maths.V(0, 10, 0), Radius: 100, Height: 30}
	weather := GroveWeather{Extinction: 0.006, Depth: 20, Coverage: 0.6,
		Wind: maths.V(2, 0, 0), Fade: true, Seed: 23, GroundFloor: 10}
	grove, err := NewGrove(weather, []Canopy{tree}, func(x, z float64) float64 { return 10 })
	if err != nil {
		t.Fatal(err)
	}
	changed, nonzero := false, false
	for x := -20.0; x < 20; x++ {
		at := maths.V(x, 15, 0)
		moved := grove.Tick(2 * time.Second).Density(at.Add(maths.V(4, 0, 0)))
		if math.Abs(moved-grove.Density(at)) > 1e-14 {
			t.Fatal("wind did not advect the patches")
		}
		changed = changed || moved != grove.Density(at.Add(maths.V(4, 0, 0)))
		nonzero = nonzero || moved > 0
	}
	if !changed || !nonzero {
		t.Fatal("fixture failed to exercise moving mist")
	}
	weather.Coverage = 0
	clear, _ := NewGrove(weather, []Canopy{tree}, grove.ground)
	if clear.Density(maths.V(0, 15, 0)) != 0 {
		t.Error("zero coverage is not clear")
	}
	if grove.Density(maths.V(0, 9, 0)) != 0 {
		t.Error("mist below the ground")
	}
}

func TestGroveRejectsInvalidFields(t *testing.T) {
	weather := GroveWeather{Extinction: 0.01, Depth: 20, Coverage: 1}
	for _, tree := range []Canopy{{}, {Radius: math.Inf(1), Height: 10},
		{Radius: 1, Height: math.NaN()}} {
		if _, err := NewGrove(weather, []Canopy{tree}, func(x, z float64) float64 { return 0 }); err == nil {
			t.Error("accepted invalid canopy")
		}
	}
	weather.Depth = math.NaN()
	if _, err := NewGrove(weather, []Canopy{{Radius: 1, Height: 10}},
		func(x, z float64) float64 { return 0 }); err == nil {
		t.Error("accepted invalid weather")
	}
}

func TestThinMistDoesNotExplodeTheCanopyIndex(t *testing.T) {
	grove, err := NewGrove(GroveWeather{Extinction: 0.01, Depth: 0.1, Coverage: 1},
		[]Canopy{{Radius: 1000, Height: 20}}, func(x, z float64) float64 { return 0 })
	if err != nil {
		t.Fatal(err)
	}
	if len(grove.canopies) > 9 {
		t.Fatalf("one canopy occupies %d buckets", len(grove.canopies))
	}
	if grove.Density(maths.V(0, 0.05, 0)) != 0.005 {
		t.Error("index lost the canopy")
	}
}
