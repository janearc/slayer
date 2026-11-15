package named

import "fmt"

// Bounds are the least and most a number may be. The least may hold the
// number at it or only those above; the most holds the number at it. The
// zero Bounds holds every number.
type Bounds struct {
	low, high edge
}

// edge is one end of a parameter's bounds: where it is, whether there is
// one, and whether the number at it is outside.
type edge struct {
	at        float64
	set, open bool
}

// unbounded holds every number.
var unbounded = Bounds{}

// atLeast holds least and every number above it.
func atLeast(least float64) Bounds {
	return Bounds{low: edge{at: least, set: true}}
}

// above holds every number above least, and not least itself.
func above(least float64) Bounds {
	return Bounds{low: edge{at: least, set: true, open: true}}
}

// between holds least, most and every number between them.
func between(least, most float64) Bounds {
	return Bounds{low: edge{at: least, set: true},
		high: edge{at: most, set: true}}
}

// Least is the lowest number held, if there is one; open says the number
// itself is outside and only those above it are held.
func (bounds Bounds) Least() (least float64, open, set bool) {
	return bounds.low.at, bounds.low.open, bounds.low.set
}

// Most is the highest number held, if there is one. The number itself is
// always held.
func (bounds Bounds) Most() (most float64, set bool) {
	return bounds.high.at, bounds.high.set
}

// Holds is whether the number is within the bounds.
func (bounds Bounds) Holds(value float64) bool {
	return bounds.low.under(value) && bounds.high.over(value)
}

// under is whether the value is on the held side of a low edge.
func (low edge) under(value float64) bool {
	if !low.set {
		return true
	}
	if low.open {
		return value > low.at
	}
	return value >= low.at
}

// over is whether the value is on the held side of a high edge, which
// always holds the number at it.
func (high edge) over(value float64) bool {
	return !high.set || value <= high.at
}

// String is the bounds in words: "any number", "at least 8", "above 0",
// "from -5 to 90".
func (bounds Bounds) String() string {
	low, high := bounds.low, bounds.high
	if high.set {
		return fmt.Sprintf("from %v to %v", low.at, high.at)
	}
	if low.open {
		return fmt.Sprintf("above %v", low.at)
	}
	if low.set {
		return fmt.Sprintf("at least %v", low.at)
	}
	return "any number"
}
