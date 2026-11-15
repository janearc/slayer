package noise

import (
	"math"
	"testing"
)

func TestGradientDirectionsPreserveExactValues(t *testing.T) {
	values := []float64{0, math.Copysign(0, -1), 0.125, -0.75,
		math.SmallestNonzeroFloat64, math.MaxFloat64}
	for h := uint64(0); h < 120; h++ {
		for _, x := range values {
			for _, y := range values {
				for _, z := range values {
					got, want := grad(h, x, y, z), referenceGrad(h, x, y, z)
					if math.Float64bits(got) != math.Float64bits(want) {
						t.Fatalf("gradient %d (%g,%g,%g): %g want %g", h, x, y, z, got, want)
					}
				}
			}
		}
	}
}

var gradientResult float64

func BenchmarkGradientDirections(b *testing.B) {
	sum := 0.0
	for i := range b.N {
		sum += grad(mix(uint64(i)), .125, -.75, .375)
	}
	gradientResult = sum
}

func TestGradientMatchesOriginalLatticeHashes(t *testing.T) {
	for i := range int64(10000) {
		x, y, z := Hash(1, i)*100-50, Hash(2, i)*100-50, Hash(3, i)*100-50
		got, want := Gradient(uint64(i), x, y, z),
			referenceGradient(uint64(i), x, y, z)
		if math.Float64bits(got) != math.Float64bits(want) {
			t.Fatalf("lattice %d: %g want %g", i, got, want)
		}
	}
}
