package trace

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// worldBox is a placed thing's box in the world: its shape's box with
// each corner taken out to the world, widened a hair so that rounding
// in the taking never leaves a meeting outside it. A shape with no edge
// has none.
func worldBox(placed *placedThing) maths.Box {
	local := placed.thing.Shape.Bounds()
	if !edged(local) {
		return maths.Everywhere()
	}
	box := emptyBox()
	for corner := range 8 {
		at := placed.at.Out(cornerOf(local, corner))
		box = joined(box, maths.Box{Min: at, Max: at})
	}
	return widened(box)
}

// cornerOf is one of a box's eight corners, by the bits of its number:
// the first bit picks x's side, the second y's, the third z's.
func cornerOf(box maths.Box, corner int) maths.Vec {
	pick := func(bit int, low, high float64) float64 {
		if corner&bit == 0 {
			return low
		}
		return high
	}
	return maths.V(pick(1, box.Min.X, box.Max.X),
		pick(2, box.Min.Y, box.Max.Y), pick(4, box.Min.Z, box.Max.Z))
}

// widened is a box made larger on every side by a billionth of its
// size and its distance from the origin, whichever is more.
func widened(box maths.Box) maths.Box {
	hair := 1e-9*reach(box) + math.SmallestNonzeroFloat64
	margin := maths.V(hair, hair, hair)
	return maths.Box{Min: box.Min.Sub(margin), Max: box.Max.Add(margin)}
}

// reach is how big a box is where floats are concerned: its size or its
// distance from the origin, whichever is more.
func reach(box maths.Box) float64 {
	return max(box.Max.Sub(box.Min).Len(), box.Min.Len(), box.Max.Len())
}

// emptyBox is a box around nothing, which joins to anything as that
// thing.
func emptyBox() maths.Box {
	inf := math.Inf(1)
	return maths.Box{Min: maths.V(inf, inf, inf),
		Max: maths.V(-inf, -inf, -inf)}
}

// joined is the box around two boxes.
func joined(one, other maths.Box) maths.Box {
	return maths.Box{
		Min: maths.V(min(one.Min.X, other.Min.X), min(one.Min.Y,
			other.Min.Y), min(one.Min.Z, other.Min.Z)),
		Max: maths.V(max(one.Max.X, other.Max.X), max(one.Max.Y,
			other.Max.Y), max(one.Max.Z, other.Max.Z)),
	}
}

// edged is whether a box has an edge on every side.
func edged(box maths.Box) bool {
	for _, side := range [6]float64{box.Min.X, box.Min.Y, box.Min.Z,
		box.Max.X, box.Max.Y, box.Max.Z} {
		if math.IsInf(side, 0) || math.IsNaN(side) {
			return false
		}
	}
	return true
}

// widest is the axis along which a box is longest: zero for x, one for y, two
// for z.
func widest(box maths.Box) int {
	size := box.Max.Sub(box.Min)
	switch {
	case size.X >= size.Y && size.X >= size.Z:
		return 0
	case size.Y >= size.Z:
		return 1
	}
	return 2
}

// middle is the middle of a box along one axis.
func middle(box maths.Box, axis int) float64 {
	switch axis {
	case 0:
		return (box.Min.X + box.Max.X) / 2
	case 1:
		return (box.Min.Y + box.Max.Y) / 2
	}
	return (box.Min.Z + box.Max.Z) / 2
}
