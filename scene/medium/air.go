package medium

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Air is an atmosphere held against a body: thickest at the surface, at
// Radius, and thinning upward by a factor of e every Height, the way air
// does. It reaches out to six heights, past which it is taken as nothing.
//
// Below the surface there is none; the surface is a thing of its own and
// stops the ray.
type Air struct {
	Radius float64
	// Height is the scale height: how far up the air thins by e.
	Height float64
	// Thick is the density at the surface.
	Thick float64
}

// airReach is how many scale heights up the air is still counted. It is
// chosen: at six the air is a quarter of a hundredth of the surface's.
const airReach = 6

// top is the radius past which the air is nothing.
func (air Air) top() float64 { return air.Radius + airReach*air.Height }

// Bounds is the cube the air's top just fits in.
func (air Air) Bounds() maths.Box {
	return cube(air.top())
}

// Span is where a ray crosses the air's top.
func (air Air) Span(ray maths.Ray, near, far float64) (float64, float64,
	bool) {
	return sphere(air.top(), ray, near, far)
}

// Density is Thick at the surface, falling by e every Height above it, and
// zero below it or past the top.
func (air Air) Density(point maths.Vec) float64 {
	if air.Height <= nought {
		return nought
	}
	above := point.Len() - air.Radius
	if above < nought || above > airReach*air.Height {
		return nought
	}
	return air.Thick * math.Exp(-above/air.Height)
}
