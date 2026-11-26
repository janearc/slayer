package maths

import "math"

// CountBetween picks one whole number uniformly from an inclusive range.
func CountBetween(least, most, share float64) int {
	wide := most - least + 1
	return int(least + math.Min(math.Floor(share*wide), wide-1))
}
