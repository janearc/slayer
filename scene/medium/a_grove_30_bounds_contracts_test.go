package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// aGrove30BoundsContractsFile holds the columns and the points tried in them.
const aGrove30BoundsContractsFile = "testdata/a-grove-30-bounds-contracts.csv"

// aGrove30BoundsContractsLevel is flat terrain at one height.
type aGrove30BoundsContractsLevel float64

// height is the terrain at x and z, which is the level everywhere.
func (level aGrove30BoundsContractsLevel) height(x, z float64) float64 {
	return float64(level)
}

// aGrove30BoundsContractsDensity is the density one metre above terrain with
// the mist's extinction: smoothstep at one eighth of the depth is 11/256, so
// the density is the extinction by 245 over 256.
func aGrove30BoundsContractsDensity(t *testing.T) float64 {
	t.Helper()
	extinction := picked(t, "a-grove-30-bounds-contracts-extinction")
	numerator := picked(t, "a-grove-30-bounds-contracts-numerator")
	denominator := picked(t, "a-grove-30-bounds-contracts-denominator")
	return extinction * numerator / denominator
}

// TestGroveIncludesItsPrescribedFloorAndCanopyTop holds that a point one
// metre above terrain is dense at the canopy's top and at the prescribed
// floor.
func TestGroveIncludesItsPrescribedFloorAndCanopyTop(t *testing.T) {
	cases := readSheet(t, aGrove30BoundsContractsFile)
	// The first column ends at the canopy top. The second one's terrain
	// continues below the prescribed floor, which clips it.
	for _, name := range []string{"canopy-top", "floor-clips-terrain"} {
		row := cases.row(t, aGrove30BoundsContractsFile,
			"a-grove-30-bounds-contracts-"+name)
		weather := GroveWeather{
			Extinction:  picked(t, "a-grove-30-bounds-contracts-extinction"),
			Depth:       picked(t, "a-grove-30-bounds-contracts-depth"),
			Coverage:    unity,
			GroundFloor: nought,
		}
		foot := maths.V(nought, cell(t, row, "base"), nought)
		trees := []Canopy{{At: foot,
			Radius: picked(t, "a-grove-30-bounds-contracts-radius"),
			Height: cell(t, row, "height")}}
		ground := aGrove30BoundsContractsLevel(cell(t, row, "terrain"))
		field, err := NewGrove(weather, trees, ground.height)
		if err != nil {
			t.Fatal(err)
		}
		want := aGrove30BoundsContractsDensity(t)
		tolerance := picked(t, "a-grove-30-bounds-contracts-tolerance")
		at := maths.V(nought, cell(t, row, "y"), nought)
		if got := field.Density(at); !(math.Abs(got-want) <= tolerance) {
			t.Fatalf("field boundary at y=%g: density %.17g, want %.17g",
				cell(t, row, "y"), got, want)
		}
	}
}
