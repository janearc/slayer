package skylight

import "math"

// rayleighCosine is the cosine of the turn Rayleigh scattering gives a
// photon, from a uniform draw from nought to one, by inverting the
// phase's distribution exactly.
//
// The phase, three eighths of one and mu squared, sums from minus one to
// mu to (mu cubed + 3 mu + 4) over 8. Setting that to the draw u gives
// mu cubed + 3 mu + 4 - 8u = 0, a cubic with one real root.
//
// Cardano's formula: with zeta = 4u - 2, mu is the cube root of zeta +
// root(zeta^2 + 1) plus the cube root of zeta - root(zeta^2 + 1); their
// product is minus one, so mu is A - 1/A for the first, A. The sign is
// taken apart so that A is never the small difference of near equals.
func rayleighCosine(draw float64) float64 {
	zeta := 4*draw - 2 // defined: the cubic's constant term, halved
	size := math.Abs(zeta)
	root := math.Cbrt(size + math.Sqrt(size*size+one))
	return math.Copysign(root-one/root, zeta)
}

// henyeyCosine is the cosine of the turn particles of asymmetry lean
// give a photon, from a uniform draw from nought to one, by the usual
// inversion of the Henyey-Greenstein distribution: one plus lean
// squared, less the square of (1 - lean^2)/(1 - lean + 2 lean u), over
// twice lean. Under the faintest lean, the turn is alike every way.
func henyeyCosine(lean, draw float64) float64 {
	if math.Abs(lean) < faintestLean {
		return two*draw - one
	}
	square := lean * lean
	part := (one - square) / (one - lean + two*lean*draw)
	return (one + square - part*part) / (two * lean)
}
