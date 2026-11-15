package shape

import (
	"math"
	"math/rand"
	"testing"

	"github.com/janearc/slayer/maths"
)

// planeField is the plane height = x + z + offset over a square of the
// given extent, n by n samples. A plane is held exactly by its samples.
func planeField(t *testing.T, n int, extent, offset float64) *HeightField {
	t.Helper()
	step := extent / float64(n-1)
	heights := make([]float64, n*n)
	for row := 0; row < n; row++ {
		for column := 0; column < n; column++ {
			x := -extent/2 + float64(column)*step
			z := -extent/2 + float64(row)*step
			heights[row*n+column] = x + z + offset
		}
	}
	field, err := NewHeightField(heights, n, extent)
	if err != nil {
		t.Fatal(err)
	}
	return field
}

// The plane y = x + z + 8 has the normal (-1, 1, -1) / root three, so
// the point (0, 20, 0), 12 above it, is 12 / root three = 6.9282 from
// it. the foot of the perpendicular is at x = z = 4, inside the square
// of 40 across, so the square's edge does not change the distance. a
// bound that treated the gradient as root two (the steepest edge slope
// on one axis, 1, times the root of two) would say 12 / root two =
// 8.4853, more than the truth.
func TestADiagonalPlaneIsNotOverstated(t *testing.T) {
	field := planeField(t, 5, 40, 8)
	got := field.Distance(maths.V(0, 20, 0))
	want := 12 / math.Sqrt(3)
	if !(got <= want+1e-9) {
		t.Errorf("from (0, 20, 0) the distance is %v, more than the true %v",
			got, want)
	}
	// the slopes along x and z are one each: one over the root of three
	_, _, _, _, _, slant := field.Pack()
	if !(math.Abs(slant-1/math.Sqrt(3)) <= 1e-12) {
		t.Errorf("the slant is %v, want %v", slant, 1/math.Sqrt(3))
	}
}

// The plane y = x + z from (0, 1, 0) is 1 / root three = 0.5774 away,
// not 0.7071.
func TestADiagonalPlaneJustAboveIt(t *testing.T) {
	field := planeField(t, 5, 40, 0)
	got := field.Distance(maths.V(0, 1, 0))
	want := 1 / math.Sqrt(3)
	if !(got <= want+1e-9) {
		t.Errorf("the distance is %v, more than the true %v", got, want)
	}
}

// Over an axis ramp, a diagonal plane and a bumpy field, the distance
// never exceeds the true distance, which is found by sampling the
// surface finely and taking the nearest sample. the sampling can miss
// the nearest point by half a step across, so the true distance is
// allowed that much, times the root of one plus the steepest slope
// squared, as slack.
func TestDistanceNeverExceedsTheSampledTruth(t *testing.T) {
	const (
		n      = 5
		extent = 2.0
		dense  = 400
		slope  = 4.0 // the bumpy field's heights are within one, so edges rise at most 2 / 0.5
	)
	rng := rand.New(rand.NewSource(7))
	bumpy := make([]float64, n*n)
	for i := range bumpy {
		bumpy[i] = 2*rng.Float64() - 1
	}
	bumpField, err := NewHeightField(bumpy, n, extent)
	if err != nil {
		t.Fatal(err)
	}
	ramp := make([]float64, n*n)
	for row := 0; row < n; row++ {
		for column := 0; column < n; column++ {
			ramp[row*n+column] = -1 + float64(column)*0.5
		}
	}
	rampField, err := NewHeightField(ramp, n, extent)
	if err != nil {
		t.Fatal(err)
	}
	fields := map[string]*HeightField{
		"axis ramp":      rampField,
		"diagonal plane": planeField(t, n, extent, 0),
		"bumpy":          bumpField,
	}
	step := extent / dense
	slack := step * math.Sqrt2 / 2 * math.Sqrt(1+slope*slope)
	for name, field := range fields {
		for i := 0; i < 150; i++ {
			p := maths.V(extent*(rng.Float64()-0.5), 6*(rng.Float64()-0.5),
				extent*(rng.Float64()-0.5))
			nearest := math.Inf(1)
			for r := 0; r <= dense; r++ {
				for c := 0; c <= dense; c++ {
					x := -extent/2 + float64(c)*step
					z := -extent/2 + float64(r)*step
					q := maths.V(x, field.height(x, z), z)
					nearest = math.Min(nearest, p.Sub(q).Len())
				}
			}
			got := math.Abs(field.Distance(p))
			if !(got <= nearest+slack) {
				t.Fatalf("%s at %v: distance %v exceeds the sampled truth %v",
					name, p, got, nearest)
			}
		}
	}
}
