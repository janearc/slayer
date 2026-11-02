// Package shape describes geometry in local coordinates. Shapes are centred on
// their own origins and do not know their world placement or material. Each
// representation supplies the geometric queries required by tracing and
// collision detection.
package shape

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Shape is anything that can say where it could possibly be, and how far a
// point is from its surface.
type Shape interface {
	// Bounds is the box the shape stays inside, in its own space.
	Bounds() maths.Box
	// Distance is how far p is from the nearest bit of the surface, and
	// negative when p is inside.
	Distance(p maths.Vec) float64
}

// Hitter is a shape that can answer a ray exactly, and faster than marching.
type Hitter interface {
	// Hit is the distance along r at which it first meets the shape,
	// between near and far, if it does.
	Hit(r maths.Ray, near, far float64) (float64, bool)
}

// Spanner is a shape that can say where a ray enters it and where it
// leaves, so a ray can pass through it, as radiation does through a
// thing that is not opaque to it.
type Spanner interface {
	// Span is where the ray enters and leaves the shape between near
	// and far, if it crosses it at all.
	Span(r maths.Ray, near, far float64) (enter, leave float64, ok bool)
}

// Normaler is a shape that knows exactly which way its surface faces.
type Normaler interface {
	// Normal is the direction the surface faces at p, which is on it, one
	// unit long.
	Normal(p maths.Vec) maths.Vec
}

// Normal is which way the surface faces at p: exactly, when the shape knows,
// and otherwise the direction its distance grows fastest.
func Normal(s Shape, p maths.Vec) maths.Vec {
	if n, ok := s.(Normaler); ok {
		return n.Normal(p)
	}
	const e = 1e-5
	d := func(dx, dy, dz float64) float64 {
		return s.Distance(p.Add(maths.V(dx, dy, dz)))
	}
	return maths.V(
		d(e, 0, 0)-d(-e, 0, 0),
		d(0, e, 0)-d(0, -e, 0),
		d(0, 0, e)-d(0, 0, -e),
	).Unit()
}

// Sphere is a ball of the given radius, centred on its own zero.
type Sphere struct {
	Radius float64
}

// Bounds is the cube the sphere just fits in.
func (s Sphere) Bounds() maths.Box {
	r := s.Radius
	return maths.Box{Min: maths.V(-r, -r, -r), Max: maths.V(r, r, r)}
}

// Distance is how far p is from the centre, less the radius.
func (s Sphere) Distance(p maths.Vec) float64 { return p.Len() - s.Radius }

// Normal is straight out from the centre.
func (s Sphere) Normal(p maths.Vec) maths.Vec { return p.Unit() }

// Hit solves for where the ray crosses the sphere: the two points along it
// at distance Radius from the centre, and the nearer of them past near.
func (s Sphere) Hit(r maths.Ray, near, far float64) (float64, bool) {
	b := r.At.Dot(r.Dir)
	c := r.At.Dot(r.At) - s.Radius*s.Radius
	disc := b*b - c
	if disc < 0 {
		return 0, false
	}
	root := math.Sqrt(disc)
	for _, t := range [2]float64{-b - root, -b + root} {
		if t > near && t < far {
			return t, true
		}
	}
	return 0, false
}

// Span is where the ray enters the sphere and where it leaves.
func (s Sphere) Span(r maths.Ray, near, far float64) (float64, float64, bool) {
	b := r.At.Dot(r.Dir)
	c := r.At.Dot(r.At) - s.Radius*s.Radius
	disc := b*b - c
	if disc < 0 {
		return 0, 0, false
	}
	root := math.Sqrt(disc)
	enter, leave := math.Max(-b-root, near), math.Min(-b+root, far)
	return enter, leave, leave > enter
}

// Plane is the flat ground through its own zero, facing up, with no edge.
type Plane struct{}

// Bounds is everywhere: the plane has no edge.
func (Plane) Bounds() maths.Box { return maths.Everywhere() }

// Distance is how high p is above the ground.
func (Plane) Distance(p maths.Vec) float64 { return p.Y }

// Normal is up, everywhere on it.
func (Plane) Normal(maths.Vec) maths.Vec { return maths.Up }

// Hit is where the ray comes down to, or up to, height zero.
func (Plane) Hit(r maths.Ray, near, far float64) (float64, bool) {
	if r.Dir.Y == 0 {
		return 0, false
	}
	t := -r.At.Y / r.Dir.Y
	return t, t > near && t < far
}
