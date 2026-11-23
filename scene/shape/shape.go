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
	// Distance is a lower bound on how far point is from the nearest bit
	// of the surface: never more than the true distance, because a ray is
	// marched forward by it and a longer step goes through the surface.
	// It is negative when point is inside.
	Distance(point maths.Vec) float64
}

// Hitter is a shape that can answer a ray exactly, and faster than marching.
type Hitter interface {
	// Hit is the distance along ray, counted in lengths of ray.Dir, at
	// which it first meets the shape, between near and far, if it does.
	// The distance means something only when the second answer is true.
	Hit(ray maths.Ray, near, far float64) (float64, bool)
}

// Spanner is a shape that can say where a ray enters it and where it
// leaves, so a ray can pass through it, as radiation does through a
// thing that is not opaque to it.
type Spanner interface {
	// Span is where the ray enters and leaves the shape between near
	// and far, if it crosses it at all.
	Span(ray maths.Ray, near, far float64) (enter, leave float64, ok bool)
}

// normaler is a shape that knows exactly which way its surface faces.
type normaler interface {
	// Normal is the direction the surface faces at point, which is on it,
	// one unit long.
	Normal(point maths.Vec) maths.Vec
}

// GeometricNormaler is a shape whose appearance normal may differ from its
// exact surface normal. Optical boundaries use this normal for the side a ray
// enters, reflects toward, or refracts into; Normal remains the shading normal.
type GeometricNormaler interface {
	GeometricNormal(point maths.Vec) maths.Vec
}

// normalStep is how far either side of a point the marched normal measures
// the distance, in the shape's own units. It is chosen, in picks.yaml.
var normalStep = pick("normal-step")

// Normal is which way the surface faces at point: exactly, when the shape
// knows, and otherwise the direction its distance grows fastest, measured
// normalStep either side. Far from the origin, where adding normalStep to a
// coordinate changes nothing, the marched answer is the zero vector.
func Normal(solid Shape, point maths.Vec) maths.Vec {
	if exact, ok := solid.(normaler); ok {
		return exact.Normal(point)
	}
	return maths.V(
		distanceGrowth(solid, point, maths.XAxis),
		distanceGrowth(solid, point, maths.YAxis),
		distanceGrowth(solid, point, maths.ZAxis),
	).Unit()
}

// distanceGrowth is how much the distance changes across normalStep either
// side of point along one axis.
func distanceGrowth(solid Shape, point, axis maths.Vec) float64 {
	ahead := point.Add(axis.Scale(normalStep))
	behind := point.Add(axis.Scale(-normalStep))
	return solid.Distance(ahead) - solid.Distance(behind)
}

// GeometricNormal is the exact surface normal when a shape distinguishes it
// from its shading normal, and otherwise the shape's ordinary normal.
func GeometricNormal(solid Shape, point maths.Vec) maths.Vec {
	if exact, ok := solid.(GeometricNormaler); ok {
		return exact.GeometricNormal(point)
	}
	return Normal(solid, point)
}

// Sphere is a ball of the given radius, centred on its own zero. A radius
// that is not above zero, or is not a finite number, makes no ball: its
// Bounds is the one point at the origin, its Distance is infinite, and Hit
// and Span meet nothing.
type Sphere struct {
	Radius float64
}

// valid is whether the radius makes a ball.
func (sphere Sphere) valid() bool {
	return sphere.Radius > nought && finiteNumber(sphere.Radius)
}

// Bounds is the cube the sphere just fits in.
func (sphere Sphere) Bounds() maths.Box {
	if !sphere.valid() {
		return nowhere()
	}
	reach := sphere.Radius
	return maths.Box{Min: maths.V(-reach, -reach, -reach),
		Max: maths.V(reach, reach, reach)}
}

// Distance is how far point is from the centre, less the radius.
func (sphere Sphere) Distance(point maths.Vec) float64 {
	if !sphere.valid() {
		return math.Inf(unity)
	}
	return point.Len() - sphere.Radius
}

// Normal is straight out from the centre. At the centre itself no way is
// out, and it is the zero vector.
func (sphere Sphere) Normal(point maths.Vec) maths.Vec { return point.Unit() }

// unitSlack and cancellationLimit are chosen, in picks.yaml. They tell the
// plain form of the crossings from the careful one.
var (
	unitSlack         = pick("sphere-unit-slack")
	cancellationLimit = pick("sphere-cancellation-limit")
)

// crossings are the two distances along the ray, in lengths of its Dir, at
// which it is a radius from the centre, nearer first. A ray whose Dir is one
// long, to within unitSlack, from a start no more than cancellationLimit
// squared radii from the centre, is solved in the plain form, as it always
// has been.
//
// Any other ray, whose Dir has another length or whose start is so
// far that the squares swallow the radius, is solved from its closest
// approach to the centre, which keeps the width of a small ball far away.
func (sphere Sphere) crossings(ray maths.Ray) (first, last float64, ok bool) {
	if !sphere.valid() {
		return nought, nought, false
	}
	lengthSquared := ray.Dir.Dot(ray.Dir)
	startSquared := ray.At.Dot(ray.At)
	if math.Abs(lengthSquared-unity) <= unitSlack &&
		startSquared <= cancellationLimit*sphere.Radius*sphere.Radius {
		return sphere.plainCrossings(ray, startSquared)
	}
	return sphere.carefulCrossings(ray, lengthSquared)
}

// plainCrossings solves the quadratic for a ray one long and not far: its
// roots are minus b, and the root of b squared less c.
func (sphere Sphere) plainCrossings(ray maths.Ray,
	startSquared float64) (first, last float64, ok bool) {
	b := ray.At.Dot(ray.Dir)
	c := startSquared - sphere.Radius*sphere.Radius
	disc := b*b - c
	if disc < nought {
		return nought, nought, false
	}
	root := math.Sqrt(disc)
	return -b - root, -b + root, true
}

// carefulCrossings solves from the ray's closest approach to the centre, in
// lengths of the ray's Dir, whatever they are.
func (sphere Sphere) carefulCrossings(ray maths.Ray,
	lengthSquared float64) (first, last float64, ok bool) {
	if !(lengthSquared > nought) || math.IsInf(lengthSquared, unity) {
		return nought, nought, false
	}
	closest := -ray.At.Dot(ray.Dir) / lengthSquared
	foot := ray.At.Add(ray.Dir.Scale(closest))
	room := sphere.Radius*sphere.Radius - foot.Dot(foot)
	if !(room >= nought) {
		return nought, nought, false
	}
	reach := math.Sqrt(room / lengthSquared)
	return closest - reach, closest + reach, true
}

// Hit solves for where the ray crosses the sphere: the two points along it
// at distance Radius from the centre, and the nearer of them past near. A
// miss is zero and false. Distances are in lengths of ray.Dir.
func (sphere Sphere) Hit(ray maths.Ray, near, far float64) (float64, bool) {
	first, last, ok := sphere.crossings(ray)
	if !ok {
		return nought, false
	}
	for _, along := range [...]float64{first, last} {
		if along > near && along < far {
			return along, true
		}
	}
	return nought, false
}

// Span is where the ray enters the sphere and where it leaves, in lengths
// of ray.Dir.
func (sphere Sphere) Span(ray maths.Ray, near, far float64) (float64, float64, bool) {
	first, last, ok := sphere.crossings(ray)
	if !ok {
		return nought, nought, false
	}
	enter, leave := math.Max(first, near), math.Min(last, far)
	return enter, leave, leave > enter
}

// Plane is the flat ground through its own zero, facing up, with no edge.
type Plane struct{}

// Bounds is everywhere: the plane has no edge.
func (Plane) Bounds() maths.Box { return maths.Everywhere() }

// Distance is how high point is above the ground.
func (Plane) Distance(point maths.Vec) float64 { return point.Y }

// Normal is up, everywhere on it.
func (Plane) Normal(maths.Vec) maths.Vec { return maths.Up }

// Hit is where the ray comes down to, or up to, height zero. A ray along
// the ground misses; one that meets it outside near and far gives where it
// would have, and false.
func (Plane) Hit(ray maths.Ray, near, far float64) (float64, bool) {
	if ray.Dir.Y == nought {
		return nought, false
	}
	along := -ray.At.Y / ray.Dir.Y
	return along, along > near && along < far
}
