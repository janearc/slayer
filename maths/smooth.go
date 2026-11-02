package maths

// What Smooth is made of, each defined. The step is the cubic 3t^2 - 2t^3,
// which is zero at zero and one at one with no slope at either.
const (
	below = 0 // defined: the step's value before it begins
	above = 1 // defined: the step's value once it has ended
	three = 3 // defined: the cubic's constant, 3t^2 - 2t^3
	two   = 2 // defined: the cubic's multiplier, 3t^2 - 2t^3
)

// Smooth is zero up to start and one from end, eased between. It is the
// step with no corner, for a line's edge or a tile's seam. start is below
// end. Given otherwise it is a plain step at start.
//
// If x is no number the result is none. Start or end as no number is
// promised nothing: the result is zero or one or none.
//
//ninja:vow practice a library: its callers are in other modules
func Smooth(start, end, x float64) float64 {
	if x <= start {
		return below
	}
	if x >= end {
		return above
	}
	along := (x - start) / (end - start)
	return along * along * (three - two*along)
}
