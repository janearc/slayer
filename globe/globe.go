// Package globe implements a planet-sized sphere centred at the local origin.
// It supports views from ground level through orbital distances.
//
// Intersection uses the stable quadratic formula. At Earth’s radius, float64
// resolves the hit to roughly a nanometre; the stable form also handles larger
// scales. Tests compare with 200-bit arithmetic.
package globe

import (
	"errors"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// ErrGlobe is a globe with no size.
var ErrGlobe = errors.New("globe: a globe needs a radius above nought")

// Globe is a ball of a radius about its own zero.
type Globe struct{ radius float64 }

// New is a globe of a radius, or why there is none.
func New(radius float64) (Globe, error) {
	if !(radius > 0) || math.IsInf(radius, 0) {
		return Globe{}, ErrGlobe
	}
	return Globe{radius: radius}, nil
}

// Radius is how big the globe is.
func (ball Globe) Radius() float64 { return ball.radius }

// Bounds is the cube the globe fits in.
func (ball Globe) Bounds() maths.Box {
	edge := maths.V(ball.radius, ball.radius, ball.radius)
	return maths.Box{Min: edge.Neg(), Max: edge}
}

// Distance is how far a point is above the globe, less than zero inside.
func (ball Globe) Distance(at maths.Vec) float64 {
	return at.Len() - ball.radius
}

// Normal is the way out of the globe at a point on it.
func (ball Globe) Normal(at maths.Vec) maths.Vec { return at.Unit() }

// Hit is where a ray first meets the globe between near and far. the
// roots of t^2 + 2bt + c are taken as q and c/q, so neither is the
// small difference of two large numbers.
//
// c is the start's height times its distance from the far side, not
// its distance squared less the radius squared.
func (ball Globe) Hit(ray maths.Ray, near, far float64) (float64, bool) {
	way := ray.Dir.Unit()
	along := ray.At.Dot(way)
	from := ray.At.Len()
	gap := (from - ball.radius) * (from + ball.radius)
	inside := along*along - gap
	if inside < 0 {
		return 0, false
	}
	q := -(along + math.Copysign(math.Sqrt(inside), along))
	first, second := q, gap/q
	if first > second {
		first, second = second, first
	}
	for _, root := range []float64{first, second} {
		if root > near && root < far {
			return root, true
		}
	}
	return 0, false
}

// a globe is a shape a tracer meets exactly
var _ shape.Hitter = Globe{}
