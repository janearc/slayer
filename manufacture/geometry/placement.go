package geometry

import (
	"github.com/janearc/slayer/maths"
	"math"
)

// Placed copies faces into the requested parent space.
func Placed[F Triangle](skin []F, space maths.Subspace) []F {
	out := make([]F, len(skin))
	for index, one := range skin {
		for corner, point := range one {
			out[index][corner] = space.Out(point)
		}
	}
	return out
}

// Enclosing returns the smallest box containing both inputs.
func Enclosing(a, b maths.Box) maths.Box {
	return maths.Box{Min: maths.V(math.Min(a.Min.X, b.Min.X), math.Min(a.Min.Y, b.Min.Y), math.Min(a.Min.Z, b.Min.Z)), Max: maths.V(math.Max(a.Max.X, b.Max.X), math.Max(a.Max.Y, b.Max.Y), math.Max(a.Max.Z, b.Max.Z))}
}

// Transformed encloses a box after placement in its parent space.
func Transformed(box maths.Box, space maths.Subspace) maths.Box {
	first := space.Out(box.Min)
	moved := maths.Box{Min: first, Max: first}
	for _, x := range []float64{box.Min.X, box.Max.X} {
		for _, y := range []float64{box.Min.Y, box.Max.Y} {
			for _, z := range []float64{box.Min.Z, box.Max.Z} {
				point := space.Out(maths.V(x, y, z))
				moved = Enclosing(moved, maths.Box{Min: point, Max: point})
			}
		}
	}
	return moved
}

// Crown makes the closed eight-face crown defined by a box.
func Crown[F Triangle](box maths.Box) []F {
	x, z := (box.Min.X+box.Max.X)/2, (box.Min.Z+box.Max.Z)/2
	y := (box.Min.Y + box.Max.Y) / 2
	bottom, top := maths.V(x, box.Min.Y, z), maths.V(x, box.Max.Y, z)
	ring := []maths.Vec{maths.V(box.Max.X, y, z), maths.V(x, y, box.Min.Z), maths.V(box.Min.X, y, z), maths.V(x, y, box.Max.Z)}
	var skin []F
	for i, point := range ring {
		next := ring[(i+1)%len(ring)]
		skin = append(skin, F{top, point, next}, F{bottom, next, point})
	}
	return skin
}
