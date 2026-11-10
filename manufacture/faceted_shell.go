package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// facetedShell is a closed, slitted frustum with a fixed architectural outline.
// Its polygon count describes a building, not surface-detail tessellation.
type facetedShell struct {
	base, top, wall, height, opening float64
	sides                            int
}

// skin closes both wall faces, their top and bottom, and the slit ends.
func (shape facetedShell) skin(cut) surface {
	var skin surface
	caps := slitCylinder{inner: shape.base - shape.wall}
	for side := range shape.sides {
		here := shape.point(side)
		next := shape.point(side + spaceOne)
		skin = quad(skin, here[spaceZero], here[spaceOne],
			next[spaceOne], next[spaceZero])
		skin = caps.caps(skin, here, next)
	}
	first, last := shape.point(spaceZero), shape.point(shape.sides)
	skin = quad(skin, first[spaceZero], first[arcInnerLow],
		first[arcInnerHigh], first[spaceOne])
	return quad(skin, last[spaceZero], last[spaceOne],
		last[arcInnerHigh], last[arcInnerLow])
}

// point keeps adjoining faces on exactly the same polygon corner.
func (shape facetedShell) point(side int) [arcCorners]maths.Vec {
	angle := shape.opening*spaceHalf +
		(fullTurn-shape.opening)*float64(side)/float64(shape.sides)
	sine, cosine := math.Sincos(angle)
	direction := maths.V(sine, spaceZero, -cosine)
	up := maths.YAxis.Scale(shape.height)
	return [arcCorners]maths.Vec{
		direction.Scale(shape.base), direction.Scale(shape.top).Add(up),
		direction.Scale(shape.base - shape.wall),
		direction.Scale(shape.top - shape.wall).Add(up)}
}
