package medium

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestGroveRetainsItsCentreWhenHorizontalBoundsRoundTogether(t *testing.T) {
	// Binary64 spacing here is four metres. A one-metre canopy radius
	// rounds to the same extent on either side, but its centre remains
	// inside its own physical footprint and must retain its mist.
	const coordinate = 0x1p54
	weather := GroveWeather{Depth: 8, Extinction: 0.01, Coverage: 1}
	grove, err := NewGrove(weather, []Canopy{
		{At: maths.V(coordinate, 0, coordinate), Radius: 1, Height: 10},
	}, func(_, _ float64) float64 { return 0 })
	if err != nil {
		t.Fatal(err)
	}
	point := maths.V(coordinate, 4, coordinate)
	if got := grove.Density(point); got != 0.005 {
		t.Fatalf("mist at canopy centre: %g, want 0.005", got)
	}
}

func TestGroveTopRemainsEmptyWhenUnusedNoiseCoordinatesWouldOverflow(t *testing.T) {
	weather := GroveWeather{Depth: 1e-300, Extinction: 0.01, Coverage: 0.5}
	grove, err := NewGrove(weather, []Canopy{
		{At: maths.V(1e10, 0, 0), Radius: 1, Height: 1},
	}, func(_, _ float64) float64 { return 0 })
	if err != nil {
		t.Fatal(err)
	}
	if got := grove.Density(maths.V(1e10, weather.Depth, 0)); got != 0 {
		t.Fatalf("mist exactly at the layer top: %g, want zero", got)
	}
}

func TestGroveRefusesBucketCoordinatesOutsideTheHostIntegerRange(t *testing.T) {
	weather := GroveWeather{Depth: 1, Extinction: 0.01, Coverage: 1}
	for _, point := range []maths.Vec{
		maths.V(-1e100, 0, 0), maths.V(0, 0, -1e100),
		maths.V(1e100, 0, 0), maths.V(0, 0, 1e100),
		maths.V(0x1p63, 0, 0), maths.V(0, 0, 0x1p63),
	} {
		// Check the pure conversion boundary first so a bad upper-limit
		// mutation fails before attempting an overflowing bucket walk.
		preflight := Grove{cell: 1}
		if preflight.indexable(point.X) && preflight.indexable(point.Z) {
			t.Fatalf("unrepresentable bucket passed preflight: %v", point)
		}
		_, err := NewGrove(weather, []Canopy{
			{At: point, Radius: 1, Height: 1},
		}, func(_, _ float64) float64 { return 0 })
		if err == nil || !strings.Contains(err.Error(), "bucket index range") {
			t.Fatalf("unrepresentable canopy bucket at %v: %v", point, err)
		}
	}
	// Both large signed coordinates below the conversion limit remain valid.
	for _, coordinate := range []float64{-0x1p63, -0x1p62, 0x1p62} {
		_, err := NewGrove(weather, []Canopy{
			{At: maths.V(coordinate, 0, coordinate), Radius: 1, Height: 1},
		}, func(_, _ float64) float64 { return 0 })
		if err != nil {
			t.Fatalf("representable canopy bucket at %g: %v", coordinate, err)
		}
	}
}

func TestGroveIndexBudgetUsesTheActualCellWidth(t *testing.T) {
	for _, test := range []struct {
		cell, coordinate float64
		valid            bool
	}{
		{0.5, 0x1p62, false}, {2, 0x1p63, true},
		{1, -0x1p63, true}, {1, -0x1p64, false},
	} {
		preflight := Grove{cell: test.cell}
		if got := preflight.indexable(test.coordinate); got != test.valid {
			t.Fatalf("coordinate %g cell %g: valid=%t, want %t",
				test.coordinate, test.cell, got, test.valid)
		}
	}
}
