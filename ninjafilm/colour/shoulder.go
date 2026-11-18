package colour

import "math"

// Scale is the scale a of the tone's shoulder, knee + a ln(1 + (y -
// knee)/a), solved so that the top lands on white. A tone with no
// shoulder has none, and its scale is nought.
//
// This is the only definition of the shoulder's scale in slayer. Squeeze
// rolls light off by it, and render/colour's Tone develops through this
// package. One restatement reads it: the Metal kernel finish in
// improv/moonlake/omnipositor/gpudevelop/develop.go.
//
// The GPU cannot call Go, so that kernel restates the arithmetic, with
// this scale passed in its Params. TestTheKernelsShoulderIsTheTones
// there holds the two together.
//
//ninja:vow practice slayer's gpu pictures take their shoulder's scale here
func (tone Tone) Scale() float64 { return tone.scale }

// scaleOf is the scale a of the shoulder knee + a ln(1 + (y - knee)/a),
// whose slope at the knee is one, at which it reaches white at top.
//
// Put a = v (top - knee). White at top is v ln(1 + 1/v) = c, c being the
// knee's room to white over top - knee, under one since top is past one.
// The left side rises with v, from nought toward one, so v is found by
// bisection over the floats themselves, between bounds that hold it.
//
// The low bound is the least normal float, whose reciprocal is finite;
// a c under its v ln(1 + 1/v), about 1.6e-305, a top past about 1e289,
// finds v there and reaches white before top. The high bound is c / (1 -
// c): there v ln(1 + 1/v) is at least v / (1 + v), which is c.
//
// Each step halves the count of floats between the bounds, fewer than
// two to the sixty-third, so after bisections steps they are next to each
// other and v is found to the last bit.
func scaleOf(knee, top float64) float64 {
	span := top - knee
	share := (whole - knee) / span
	low := math.Float64bits(leastNormal)
	high := max(low, math.Float64bits(share/(whole-share)))
	for range bisections {
		middle := low + (high-low)/pair
		guess := math.Float64frombits(middle)
		if guess*math.Log1p(whole/guess) < share {
			low = middle
		} else {
			high = middle
		}
	}
	return math.Float64frombits(high) * span
}

// what the shoulder's scale is solved with
const (
	// bisections is how many steps scaleOf halves its bounds in: one for
	// each bit of a float64, past the sixty-three the gap needs.
	bisections = 64 // defined: a float64's bits
	// leastNormal is the least normal float64, scaleOf's low bound: its
	// reciprocal is finite, where a lesser one's may not be.
	leastNormal = 0x1p-1022 // defined: two to the minus 1022
)
