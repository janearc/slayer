package emit

import (
	"math"
	"testing"
)

// rayIntegral integrates path length through an emitting unit ball, seen
// from the observer. The radial s^2 in the volume element cancels 1/s^2.
// This uses ray chords, without the logarithmic solution or its series.
func rayIntegral(distance float64) float64 {
	const steps = 8192
	limit := 1.0
	chord := func(mu float64) float64 {
		return -distance*mu + math.Sqrt(1-distance*distance*(1-mu*mu))
	}
	if distance >= 1 {
		limit = math.Pi / 2
		chord = func(angle float64) float64 {
			s, c := math.Sincos(angle)
			return 2 * s * c * c / (distance * distance * math.Sqrt(1-s*s/(distance*distance)))
		}
	}
	low := 0.0
	if distance < 1 {
		low = -1
	}
	step := (limit - low) / steps
	sum := chord(low) + chord(limit)
	for i := 1; i < steps; i++ {
		weight := 2.0
		if i%2 != 0 {
			weight = 4
		}
		sum += weight * chord(low+float64(i)*step)
	}
	return 1.5 * sum * step / 3
}

func TestFiniteLightInteriorAndCutoffsAgreeWithRayQuadrature(t *testing.T) {
	for _, ratio := range []float64{0, .1, .25, .5, .9, .99, 1.01, 1.5, 4, 4.5, 5, 20} {
		want := rayIntegral(ratio)
		if !(want > 0) || math.IsInf(want, 0) {
			t.Fatalf("invalid quadrature at %g: %g", ratio, want)
		}
		for _, radius := range []float64{.1, 1, 7} {
			got := volumeFalloff(ratio*radius, radius) * radius * radius
			// The far-series remainder is below 5e-7 at q=4. The
			// independent integral leaves a factor of two for rounding.
			if !(math.Abs(got-want) <= 1e-6*want) {
				t.Fatalf("q=%g R=%g: falloff %g, ray integral %g", ratio, radius, got, want)
			}
		}
	}
}
