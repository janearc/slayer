package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// slitCylinder closes an annular arc, including its two cut ends.
// An inner radius of zero makes a solid roof sector with the same slit.
type slitCylinder struct {
	outer, inner, height, opening float64
}

// skin winds each boundary outward. Adjacent faces share exact vertices.
func (shape slitCylinder) skin(fine cut) surface {
	var skin surface
	for side := range fine.around {
		here := shape.point(side, fine)
		next := shape.point(side+spaceOne, fine)
		skin = quad(skin, here[spaceZero], here[spaceOne],
			next[spaceOne], next[spaceZero])
		skin = shape.caps(skin, here, next)
	}
	first := shape.point(spaceZero, fine)
	last := shape.point(fine.around, fine)
	skin = quad(skin, first[spaceZero], first[arcInnerLow],
		first[arcInnerHigh], first[spaceOne])
	return quad(skin, last[spaceZero], last[spaceOne],
		last[arcInnerHigh], last[arcInnerLow])
}

// caps uses a triangle fan at the axis, rather than a collapsed inner quad.
// Distinct corners avoid relying on roundoff to discard a zero-area face.
func (shape slitCylinder) caps(skin surface,
	here, next [arcCorners]maths.Vec,
) surface {
	if shape.inner == spaceZero {
		bottom := face{
			here[spaceZero], next[spaceZero], here[arcInnerLow]}
		top := face{here[spaceOne], here[arcInnerHigh], next[spaceOne]}
		return append(skin, bottom, top)
	}
	skin = quad(skin, here[arcInnerLow], next[arcInnerLow],
		next[arcInnerHigh], here[arcInnerHigh])
	skin = quad(skin, here[spaceZero], next[spaceZero],
		next[arcInnerLow], here[arcInnerLow])
	return quad(skin, here[spaceOne], here[arcInnerHigh],
		next[arcInnerHigh], next[spaceOne])
}

// arcCorners is defined by the four outer/inner low/high corner roles.
// arcInnerLow and arcInnerHigh are defined as the two inner corner indices.
const arcCorners, arcInnerLow, arcInnerHigh = 4, 2, 3

// point retains the same two radii at each end of a vertical wall.
func (shape slitCylinder) point(side int, fine cut) [arcCorners]maths.Vec {
	angle := shape.opening*spaceHalf +
		(fullTurn-shape.opening)*float64(side)/float64(fine.around)
	x, z := math.Sin(angle), -math.Cos(angle)
	direction := maths.V(x, spaceZero, z)
	outer := direction.Scale(shape.outer)
	inner := direction.Scale(shape.inner)
	lift := maths.YAxis.Scale(shape.height)
	return [arcCorners]maths.Vec{
		outer, outer.Add(lift), inner, inner.Add(lift)}
}
