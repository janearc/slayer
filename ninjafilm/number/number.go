// Package number provides small numeric checks and helpers for combining the
// channels of linear RGB light.
package number

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// none is no amount at all, defined as zero.
const none = 0

// Positive is whether a number is greater than zero and finite: what a width, a
// rate or a slope has to be.
func Positive(amount float64) bool {
	return amount > none && amount <= math.MaxFloat64
}

// Sum is a light's three lamps added together.
func Sum(light maths.Light) float64 {
	return light.R + light.G + light.B
}

// Finite is whether every lamp of a light is a number: none endless,
// none not a number at all. It adds the sizes of the three lamps and
// asks that the total is a number. So three lamps that are each the
// largest number a float holds count as endless together.
func Finite(light maths.Light) bool {
	return math.Abs(light.R)+math.Abs(light.G)+math.Abs(light.B) <=
		math.MaxFloat64
}
