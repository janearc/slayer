package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// aGrove20CoverageContractsFile holds the seeds, coverages and densities the
// grove is held to.
const aGrove20CoverageContractsFile = "testdata/" +
	"a-grove-20-coverage-contracts.csv"

// aGrove20CoverageContractsLevel is flat ground at one height.
type aGrove20CoverageContractsLevel float64

// height is the ground at x and z, which is the level everywhere.
func (level aGrove20CoverageContractsLevel) height(x, z float64) float64 {
	return float64(level)
}

// aGrove20CoverageContractsField is the grove of one row: its seed and
// coverage, over the one tree on ground that is level.
func aGrove20CoverageContractsField(t *testing.T, row map[string]string,
	ground float64, trees []Canopy) *Grove {
	t.Helper()
	weather := GroveWeather{
		Extinction:  picked(t, "a-grove-20-coverage-contracts-extinction"),
		Depth:       picked(t, "a-grove-20-coverage-contracts-depth"),
		Coverage:    cell(t, row, "coverage"),
		Seed:        uint64(cellCount(t, row, "seed")),
		GroundFloor: ground,
	}
	field, err := NewGrove(weather, trees,
		aGrove20CoverageContractsLevel(ground).height)
	if err != nil {
		t.Fatal(err)
	}
	return field
}

// These actual fields have unusually strong three-octave noise. Coverage
// zero and one are exact appearance choices, independent of that noise.
func TestGroveCoverageEndpointsIgnoreEvenStrongNoisePeaks(t *testing.T) {
	coordinate := picked(t, "a-grove-20-coverage-contracts-coordinate")
	ground := picked(t, "a-grove-20-coverage-contracts-ground")
	point := maths.V(coordinate, coordinate, coordinate)
	trees := []Canopy{{At: maths.V(coordinate, ground, coordinate),
		Radius: picked(t, "a-grove-20-coverage-contracts-radius"),
		Height: picked(t, "a-grove-20-coverage-contracts-tall")}}
	tolerance := picked(t, "a-grove-20-coverage-contracts-tolerance")
	for _, name := range []string{"clear", "filled"} {
		row := readSheet(t, aGrove20CoverageContractsFile).row(t,
			aGrove20CoverageContractsFile,
			"a-grove-20-coverage-contracts-"+name)
		field := aGrove20CoverageContractsField(t, row, ground, trees)
		want := cell(t, row, "want")
		if got := field.Density(point); !(math.Abs(got-want) <= tolerance) {
			t.Fatalf("coverage %g seed %d density %.17g, want %.17g",
				cell(t, row, "coverage"), cellCount(t, row, "seed"), got, want)
		}
	}
}
