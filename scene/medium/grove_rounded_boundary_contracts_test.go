package medium

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The tables the rounded boundary contracts read: weathers and points by
// name, the signed coordinates that stay valid, and the cell widths against
// the coordinates they may index.
const (
	groveRoundedWeatherFile = "testdata/grove-rounded-weathers.csv"
	groveRoundedPointFile   = "testdata/grove-rounded-points.csv"
	groveRoundedAllowedFile = "testdata/grove-rounded-allowed.csv"
	groveRoundedBudgetFile  = "testdata/grove-rounded-budgets.csv"
)

// groveRoundedWeather is the weather of a row of the table of weathers.
func groveRoundedWeather(t *testing.T, name string) GroveWeather {
	t.Helper()
	row := readSheet(t, groveRoundedWeatherFile).row(t,
		groveRoundedWeatherFile, name)
	return GroveWeather{Depth: cell(t, row, "depth"),
		Extinction: cell(t, row, "extinction"),
		Coverage:   cell(t, row, "coverage")}
}

// groveRoundedPoint is the point of a row of the table of points.
func groveRoundedPoint(t *testing.T, name string) maths.Vec {
	t.Helper()
	row := readSheet(t, groveRoundedPointFile).row(t, groveRoundedPointFile,
		name)
	return maths.V(cell(t, row, "x"), cell(t, row, "y"), cell(t, row, "z"))
}

// groveRoundedTree is a canopy of a unit radius and a unit height at a point.
func groveRoundedTree(point maths.Vec) []Canopy {
	return []Canopy{{At: point, Radius: unity, Height: unity}}
}

// TestGroveRetainsItsCentreWhenHorizontalBoundsRoundTogether holds that a
// canopy whose bounds round to one extent still has mist at its centre.
func TestGroveRetainsItsCentreWhenHorizontalBoundsRoundTogether(t *testing.T) {
	// Binary64 spacing here is four metres. A one-metre canopy radius
	// rounds to the same extent on either side, but its centre remains
	// inside its own physical footprint and must retain its mist.
	weather := groveRoundedWeather(t, "rounded")
	canopy := Canopy{At: groveRoundedPoint(t, "rounded-tree"),
		Radius: unity,
		Height: picked(t, "grove-rounded-boundary-contracts-tall-height")}
	grove, err := NewGrove(weather, []Canopy{canopy}, flatGround)
	if err != nil {
		t.Fatal(err)
	}
	point := groveRoundedPoint(t, "rounded-probe")
	want := picked(t, "grove-rounded-boundary-contracts-centre-density")
	if got := grove.Density(point); got != want {
		t.Fatalf("mist at canopy centre: %g, want %g", got, want)
	}
}

// TestGroveTopRemainsEmptyWhenUnusedNoiseCoordinatesWouldOverflow holds that
// the top of a very thin layer is empty.
func TestGroveTopRemainsEmptyWhenUnusedNoiseCoordinatesWouldOverflow(
	t *testing.T) {
	weather := groveRoundedWeather(t, "thin")
	far := groveRoundedPoint(t, "far-tree")
	grove, err := NewGrove(weather, groveRoundedTree(far), flatGround)
	if err != nil {
		t.Fatal(err)
	}
	top := maths.V(far.X, weather.Depth, far.Z)
	if got := grove.Density(top); got != nought {
		t.Fatalf("mist exactly at the layer top: %g, want zero", got)
	}
}

// TestGroveRefusesBucketCoordinatesOutsideTheHostIntegerRange holds that a
// canopy whose bucket index is no integer is refused, and one whose index
// just fits is not.
func TestGroveRefusesBucketCoordinatesOutsideTheHostIntegerRange(
	t *testing.T) {
	weather := groveRoundedWeather(t, "unit")
	for _, name := range []string{"refused-west", "refused-south",
		"refused-east", "refused-north", "refused-overflow-x",
		"refused-overflow-z"} {
		point := groveRoundedPoint(t, name)
		// Check the pure conversion boundary first so a bad upper-limit
		// mutation fails before attempting an overflowing bucket walk.
		preflight := Grove{cell: unity}
		if preflight.indexable(point.X) && preflight.indexable(point.Z) {
			t.Fatalf("unrepresentable bucket passed preflight: %v", point)
		}
		_, err := NewGrove(weather, groveRoundedTree(point), flatGround)
		if err == nil || !strings.Contains(err.Error(), "bucket index range") {
			t.Fatalf("unrepresentable canopy bucket at %v: %v", point, err)
		}
	}
	// Both large signed coordinates below the conversion limit remain valid.
	for _, row := range readSheet(t, groveRoundedAllowedFile) {
		coordinate := cell(t, row, "coordinate")
		at := maths.V(coordinate, nought, coordinate)
		if _, err := NewGrove(weather, groveRoundedTree(at),
			flatGround); err != nil {
			t.Fatalf("representable canopy bucket at %g: %v", coordinate, err)
		}
	}
}

// TestGroveIndexBudgetUsesTheActualCellWidth holds that a coordinate may be
// indexed or not by the width of the cell it is divided by.
func TestGroveIndexBudgetUsesTheActualCellWidth(t *testing.T) {
	for _, row := range readSheet(t, groveRoundedBudgetFile) {
		preflight := Grove{cell: cell(t, row, "cell")}
		coordinate, valid := cell(t, row, "coordinate"), row["valid"] == "true"
		if got := preflight.indexable(coordinate); got != valid {
			t.Fatalf("coordinate %g cell %g: valid=%t, want %t", coordinate,
				preflight.cell, got, valid)
		}
	}
}
