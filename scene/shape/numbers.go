package shape

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The numbers every shape here is built from. Each says what it is.
const (
	nought = 0   // defined: no amount
	unity  = 1   // defined: one
	pair   = 2   // defined: two
	half   = 0.5 // defined: one half
)

// finiteNumber is whether value is a real number: neither not a number nor
// infinite.
func finiteNumber(value float64) bool {
	return math.Abs(value) <= math.MaxFloat64
}

// nowhere is the box of a shape that has no surface: the one point at the
// origin.
func nowhere() maths.Box { return maths.Box{} }
