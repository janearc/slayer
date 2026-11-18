package gpu

import (
	"math"
	"testing"
)

func TestCloudPhaseRejectsFloatEndpointRoundingAndUnderflow(t *testing.T) {
	for _, g := range []float64{-1, 1, math.Nextafter(1, 0), math.Nextafter(-1, 0), 1e-50, -1e-50, math.NaN(), math.Inf(1)} {
		if err := metalCloudAsymmetry(g); err == nil {
			t.Fatalf("unsafe Metal phase %g accepted", g)
		}
	}
	for _, g := range []float64{-.9, 0, .9, float64(math.Float32frombits(0x3f7fffff))} {
		if err := metalCloudAsymmetry(g); err != nil {
			t.Fatal(err)
		}
	}
}
