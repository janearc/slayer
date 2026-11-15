package rings

import "math"

// The numbers the package's arithmetic is made of, each defined.
const (
	none     = 0       // defined: no count, no length, no angle
	one      = 1       // defined: a whole, the hole's mass as a length
	two      = 2       // defined: halves in a whole
	halfTurn = math.Pi // defined: the angle round half a circle
	turn     = 2 * halfTurn
)

// stillMass is the mass of the hole every lens is made with: one, so
// every length here is in masses. stillSpin is its spin: none.
const (
	stillMass = one  // defined: lengths in masses
	stillSpin = none // defined: a hole that does not spin
)
