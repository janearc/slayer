package fuelbed

import "github.com/janearc/slayer/research"

// Grass applies the calibrated example from Mandel et al., section 7.1.
// These coefficients describe that proxy experiment, not a tree species.
// sources verify holds the printed mantissas to the pinned PDF page.
//
// Powers, units and roles are checked against the final section 7.1 list
// and held by TestGrassProxyCoefficientsHoldTheirPublishedPowers.
// The paper also prints B with 10^4 earlier; the final list uses 10^2.
func Grass(s Spec) Spec {
	s.Diffusion = research.Value("fuel-diffusion-mantissa") * 1e-1
	s.Heating = research.Value("fuel-heating-mantissa") * 1e2
	s.Activation = research.Value("fuel-activation-mantissa") * 1e2
	s.Cooling = research.Value("fuel-cooling-mantissa") * 1e-5
	s.Consumption = research.Value("fuel-consumption-mantissa") * 1e-1
	return s
}
