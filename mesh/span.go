package mesh

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// span is a box being grown: its low and high on each axis. an empty
// one is endlessly inside out, so anything joined to it is itself.
type span struct{ low, high [axes]float64 }

// emptySpan is a box around nothing.
func emptySpan() span {
	far := math.Inf(plusInfinity)
	return span{low: partsOf(far, far, far),
		high: partsOf(-far, -far, -far)}
}

// spanOf is the box around a face.
func spanOf(points []maths.Vec, at face) span {
	made := emptySpan()
	for _, corner := range at {
		made = made.with(parts(points[corner]))
	}
	return made
}

// with is the box grown to hold a point.
func (grow span) with(point [axes]float64) span {
	for axis := range point {
		grow.low[axis] = math.Min(grow.low[axis], point[axis])
		grow.high[axis] = math.Max(grow.high[axis], point[axis])
	}
	return grow
}

// joined is the box around both.
func (grow span) joined(other span) span {
	for axis := range grow.low {
		grow.low[axis] = math.Min(grow.low[axis], other.low[axis])
		grow.high[axis] = math.Max(grow.high[axis], other.high[axis])
	}
	return grow
}

// middle is the box's middle.
func (grow span) middle() [axes]float64 {
	var made [axes]float64
	for axis := range made {
		made[axis] = (grow.low[axis] + grow.high[axis]) / pair
	}
	return made
}

// area is half the box's surface, zero for an empty one: what the surface area
// heuristic weighs a side by.
func (grow span) area() float64 {
	wide := grow.high[alongX] - grow.low[alongX]
	tall := grow.high[alongY] - grow.low[alongY]
	deep := grow.high[alongZ] - grow.low[alongZ]
	if !(wide >= nought && tall >= nought && deep >= nought) {
		return nought
	}
	return wide*tall + tall*deep + deep*wide
}

// parts is a vector's three parts, to be taken by axis.
func parts(point maths.Vec) [axes]float64 {
	return partsOf(point.X, point.Y, point.Z)
}

// partsOf is three parts, to be taken by axis.
func partsOf(x, y, z float64) [axes]float64 { return [axes]float64{x, y, z} }

// vectorOf is three parts as a vector.
func vectorOf(made [axes]float64) maths.Vec {
	return maths.V(made[alongX], made[alongY], made[alongZ])
}
