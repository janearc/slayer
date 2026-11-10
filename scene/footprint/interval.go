package footprint

import "math"

// interval encloses an expression evaluated on the supplied binary64 inputs.
// Each primitive widens one representable step. It is not a GPU float32 proof.
type interval struct{ low, high float64 }

// exact starts with the supplied representable scalar, without rounding it.
func exact(value float64) interval { return interval{value, value} }

// enclosed treats indeterminate arithmetic as unknown, never as empty space.
func enclosed(low, high float64) interval {
	if math.IsNaN(low) || math.IsNaN(high) {
		return interval{math.Inf(-1), math.Inf(1)}
	}
	return interval{math.Nextafter(low, math.Inf(-1)), math.Nextafter(high, math.Inf(1))}
}

// add and sub widen their extrema in the corresponding directions.
func (value interval) add(other interval) interval {
	return enclosed(value.low+other.low, value.high+other.high)
}

// sub widens the extrema of an interval subtraction.
func (value interval) sub(other interval) interval {
	return enclosed(value.low-other.high, value.high-other.low)
}

// mul encloses every endpoint product, retaining unknown for zero times
// infinity.
func (value interval) mul(other interval) interval {
	products := [4]float64{value.low * other.low, value.low * other.high,
		value.high * other.low, value.high * other.high}
	low, high := math.Inf(1), math.Inf(-1)
	for _, product := range products {
		if math.IsNaN(product) {
			return interval{math.Inf(-1), math.Inf(1)}
		}
		low, high = min(low, product), max(high, product)
	}
	return enclosed(low, high)
}

// div is used only with positive denominators; crossing zero stays unknown.
func (value interval) div(other interval) interval {
	if other.low <= 0 {
		return interval{math.Inf(-1), math.Inf(1)}
	}
	return value.mul(enclosed(1/other.high, 1/other.low))
}
