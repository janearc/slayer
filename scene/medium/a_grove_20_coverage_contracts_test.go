package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// These actual fields have unusually strong three-octave noise. Coverage
// zero and one are exact appearance choices, independent of that noise.
func TestGroveCoverageEndpointsIgnoreEvenStrongNoisePeaks(t *testing.T) {
	const coordinate = 40132.0
	const ground = 40128.0
	point := maths.V(coordinate, coordinate, coordinate)
	trees := []Canopy{{At: maths.V(coordinate, ground, coordinate), Radius: 16, Height: 16}}
	for _, row := range []struct {
		seed           uint64
		coverage, want float64
	}{
		{6856140, 0, 0}, {36335586, 1, .005},
	} {
		field, err := NewGrove(GroveWeather{Extinction: .01, Depth: 8, Coverage: row.coverage,
			Seed: row.seed, GroundFloor: ground}, trees, func(x, z float64) float64 { return ground })
		if err != nil {
			t.Fatal(err)
		}
		if got := field.Density(point); !(math.Abs(got-row.want) <= 1e-15) {
			t.Fatalf("coverage %g seed %d density %.17g, want %.17g", row.coverage, row.seed, got, row.want)
		}
	}
}
