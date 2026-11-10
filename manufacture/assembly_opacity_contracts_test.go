package manufacture

import (
	"math"
	"testing"
)

func TestSmokeScatteringAcceptsBothClosedEndpoints(t *testing.T) {
	for _, albedo := range []float64{0, 1} {
		spec := burnerFixture()
		if spec.Smoke == nil {
			t.Fatal("endpoint fixture has no smoke")
		}
		spec.Albedo = albedo
		if _, err := Burn(spec); err != nil {
			t.Fatalf("scattering share %g refused: %v", albedo, err)
		}
	}
}

func TestFiniteModelValuesExcludeBothInfinitySigns(t *testing.T) {
	for _, value := range []float64{math.Inf(-1), math.Inf(1), math.NaN()} {
		if burnFinite(value) {
			t.Fatalf("nonfinite model value accepted: %g", value)
		}
	}
	for _, value := range []float64{-math.MaxFloat64, 0, math.MaxFloat64} {
		if !burnFinite(value) {
			t.Fatalf("finite model value refused: %g", value)
		}
	}
}
