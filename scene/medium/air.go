package medium

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Air is an atmosphere held against a body: thickest at the surface, at
// Radius, and thinning upward by a factor of e every Height, the way air
// does. It reaches out to Reach heights, past which it is taken as
// nothing.
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

// Reach is how many scale heights up the air is still counted.
const Reach = 6

// top is the radius past which the air is nothing.
func (a Air) top() float64 { return a.Radius + Reach*a.Height }

// Bounds is the cube the air's top just fits in.
func (a Air) Bounds() maths.Box {
	r := a.top()
	return maths.Box{Min: maths.V(-r, -r, -r), Max: maths.V(r, r, r)}
}

// Span is where a ray crosses the air's top.
func (a Air) Span(r maths.Ray, near, far float64) (float64, float64, bool) {
	return sphere(a.top(), r, near, far)
}

// Density is Thick at the surface, falling by e every Height above it, and zero
// below it or past the top.
func (a Air) Density(p maths.Vec) float64 {
	if a.Height <= 0 {
		return 0
	}
	up := p.Len() - a.Radius
	if up < 0 || up > Reach*a.Height {
		return 0
	}
	return a.Thick * math.Exp(-up/a.Height)
}
