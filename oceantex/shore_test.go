package oceantex

import (
	"math"
	"testing"
)

// The wave number solves the dispersion relation at any depth.
func TestWaveNumber(t *testing.T) {
	for _, depth := range []float64{0.5, 3, 20, 500} {
		for _, omega := range []float64{0.3, 1, 2.5} {
			k := (Sea{Depth: depth}).number(omega)
			near(t, "dispersion", Gravity*k*math.Tanh(k*depth), omega*omega,
				1e-9)
		}
	}
	near(t, "deep water", (Sea{}).number(1), 1/Gravity, 1e-12)
}
