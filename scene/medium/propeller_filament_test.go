package medium

import (
	"math"
	"testing"
)

// A blade's core is continuous across its cut at propellerCoreSupport
// standard deviations: just inside it the density is within a float32's
// rounding of nought, as it is just outside, not a step of about 0.011.
func TestAFilamentReachesZeroAtItsCutWithoutAStep(t *testing.T) {
	const width = 0.37
	cut := propellerCoreSupport * propellerCoreSupport * width * width
	for _, nudge := range []float64{1e-3, 1e-6, 1e-9, 0} {
		inside := filament(cut*(1-nudge), width)
		outside := filament(cut*(1+nudge), width)
		if inside < 0 || inside > 2*nudge+1e-15 {
			t.Errorf("just inside the cut by %g, density %g, want within 2x",
				nudge, inside)
		}
		if outside != 0 {
			t.Errorf("outside the cut, density %g, want none", outside)
		}
	}
}

// Anti: at the centre the core is still whole less its cut value, and it
// falls with distance.
func TestAFilamentIsStillAGaussianInsideItsCut(t *testing.T) {
	const width = 0.5
	floor := math.Exp(-0.5 * propellerCoreSupport * propellerCoreSupport)
	if got := filament(0, width); math.Abs(got-(1-floor)) > 1e-12 {
		t.Errorf("at the centre, density %g, want %g", got, 1-floor)
	}
	if filament(0.04, width) >= filament(0.01, width) {
		t.Error("the core does not fall with distance")
	}
}
