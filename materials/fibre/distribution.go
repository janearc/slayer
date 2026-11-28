package fibre

import "math"

// circularLogistic normalizes the logistic density on one azimuthal
// period, as in Chiang et al. (2016), Appendix A.
func circularLogistic(angle, scale float64) float64 {
	distance := math.Abs(math.Remainder(angle, 2*math.Pi))
	e := math.Exp(-distance / scale)
	return e / (scale * (1 + e) * (1 + e) * math.Tanh(math.Pi/(2*scale)))
}

// longitudinal is d'Eon's longitudinal distribution as in pbrt 4e,
// Eq. 9.49. Kappa is inverse variance. The dominant exponential of
// the Bessel term cancels the exponent in the denominator before
// evaluation, keeping the lobe finite even at small roughness.
func longitudinal(si, ci, so, co, kappa float64) float64 {
	cosines := ci * co
	argument := kappa * cosines
	exponent := kappa * min(0, cosines-si*so-1)
	normalizer := -math.Expm1(-2 * kappa)
	return kappa * scaledI0(argument) * math.Exp(exponent) / normalizer
}

// scaledI0 gives exp(-x) I0(x) for x >= 0. The power series and
// large-argument expansion are NIST DLMF 10.25.2 and 10.40.1.
func scaledI0(x float64) float64 {
	if x > 64 {
		inverse := 1 / x
		series := 1 + inverse*(1./8+inverse*(9./128+inverse*(225./3072+inverse*11025./98304)))
		return series / math.Sqrt(2*math.Pi*x)
	}
	term, sum := 1., 1.
	for order := 1.; ; order++ {
		ratio := x / (2 * order)
		term *= ratio * ratio
		if term < sum*1e-16 {
			return math.Exp(-x) * sum
		}
		sum += term
	}
}
