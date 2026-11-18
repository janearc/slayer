package skylight

import "math"

// The numbers the walk is built on, each with where it comes from.
const (
	none     = 0           // defined: nothing; the first index
	one      = 1           // defined: a whole; the next index
	two      = 2           // defined: a pair; twice
	half     = 0.5         // defined: one of two
	unkept   = -1          // defined: a boundary that keeps no radiance
	fullTurn = 2 * math.Pi // defined: a circle's angle, in radians
	sphere   = 4 * math.Pi // defined: the solid angle of every direction
	corners  = 8           // defined: a cube's, the reads trilinear takes

	// mostEvents is the cap on scatterings and reflections a photon may
	// have. Under Venus's cloud a photon has a few hundred; one with a
	// hundred thousand is caught in air no planet has.
	mostEvents = 100000 // picked: see above

	// lightest is the weight under which Russian roulette is played: a
	// photon below it lives at twice its weight half the time, or dies.
	lightest = 0.01 // picked: a hundredth, where a photon's light is
	// too little to follow at full cost

	// grazing is the cosine from the vertical under which a crossing's
	// one over the cosine is replaced by its mean over that band, two
	// over grazing, which is right when the radiance is even across it.
	grazing = 0.01 // picked: within a degree of the horizon

	// faintestLean is the asymmetry under which particles scatter alike
	// every way: below it the Henyey-Greenstein inversion loses more to
	// rounding than the lean is worth.
	faintestLean = 1e-6 // picked: see above

	// exponentBits and fraction turn a 64-bit draw into a number from
	// nought to under one: a float64 keeps 53 bits of fraction.
	exponentBits = 11      // defined: 64 less 53
	fraction     = 1 << 53 // defined: two to the 53
)
