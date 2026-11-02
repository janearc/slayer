// Package fake is the harness's own scene parts, each answering in
// closed form: a plane, a sphere, a point lamp, a flat sky, a slab of
// gas and a sensor of aimed rays. Fixtures and benchmarks are built of
// them.
//
// They are not gomock's, because a mock's reflection would be most of
// what a benchmark measured. The plane, the sphere and the lamp count
// how often they are asked, when given counts.
//
// The counts use the standard library's typed atomics rather than
// go.uber.org/atomic, which they have superseded since go 1.19.
package fake

import (
	"math"
	"sync/atomic"

	"github.com/janearc/slayer/maths"
)

// Counts are how often the parts of a scene were asked something. A
// part made with nil counts counts nothing, so a timed run is not
// slowed by its own counting: four cores adding to one counter made it
// most of what a benchmark measured.
type Counts struct {
	hits   atomic.Int64
	lights atomic.Int64
}

// Hits is how many times a shape was asked where a ray meets it.
func (counts *Counts) Hits() int64 { return counts.hits.Load() }

// Lights is how many times a lamp was asked what light reaches a point.
func (counts *Counts) Lights() int64 { return counts.lights.Load() }

// hit counts one ask of a shape, when there are counts to keep.
func (counts *Counts) hit() {
	if counts != nil {
		counts.hits.Add(1)
	}
}

// light counts one ask of a lamp, when there are counts to keep.
func (counts *Counts) light() {
	if counts != nil {
		counts.lights.Add(1)
	}
}

// Plane is a square of ground, level, centred on the origin: half is
// how far it reaches from the centre along x and along z.
type Plane struct {
	half   float64
	counts *Counts
}

// NewPlane is a square of ground reaching half from the origin, counted
// in counts.
func NewPlane(half float64, counts *Counts) Plane {
	return Plane{half: half, counts: counts}
}

// thin is how far above and below the plane its box reaches.
const thin = 1e-6

// Bounds is the box the plane stays inside.
func (ground Plane) Bounds() maths.Box {
	return maths.Box{Min: maths.V(-ground.half, -thin, -ground.half),
		Max: maths.V(ground.half, thin, ground.half)}
}

// Distance is how far a point is from the plane's level.
func (ground Plane) Distance(at maths.Vec) float64 { return math.Abs(at.Y) }

// Hit is where a ray crosses the plane inside its square.
func (ground Plane) Hit(ray maths.Ray, near, far float64) (float64, bool) {
	ground.counts.hit()
	if ray.Dir.Y == 0 {
		return 0, false
	}
	along := -ray.At.Y / ray.Dir.Y
	if along < near || along > far {
		return 0, false
	}
	met := ray.Point(along)
	inside := math.Abs(met.X) <= ground.half &&
		math.Abs(met.Z) <= ground.half
	return along, inside
}

// Normal is up, everywhere on the plane.
func (ground Plane) Normal(maths.Vec) maths.Vec { return maths.V(0, 1, 0) }

// Sphere is a ball of some radius about a centre.
type Sphere struct {
	centre maths.Vec
	radius float64
	counts *Counts
}

// NewSphere is a ball about centre, counted in counts.
func NewSphere(centre maths.Vec, radius float64, counts *Counts) Sphere {
	return Sphere{centre: centre, radius: radius, counts: counts}
}

// Bounds is the cube the ball fits in.
func (ball Sphere) Bounds() maths.Box {
	reach := maths.V(ball.radius, ball.radius, ball.radius)
	low, high := ball.centre.Sub(reach), ball.centre.Add(reach)
	return maths.Box{Min: low, Max: high}
}

// Distance is how far a point is from the ball's surface, negative
// inside.
func (ball Sphere) Distance(at maths.Vec) float64 {
	return at.Sub(ball.centre).Len() - ball.radius
}

// Hit is where a ray first meets the ball between near and far.
func (ball Sphere) Hit(ray maths.Ray, near, far float64) (float64, bool) {
	ball.counts.hit()
	enter, leave, crossed := ball.crossing(ray)
	switch {
	case !crossed:
		return 0, false
	case enter >= near && enter <= far:
		return enter, true
	case leave >= near && leave <= far:
		return leave, true
	}
	return 0, false
}

// crossing is where a ray, extended both ways, enters and leaves the
// ball, if it meets it.
func (ball Sphere) crossing(ray maths.Ray) (float64, float64, bool) {
	from := ray.At.Sub(ball.centre)
	along := from.Dot(ray.Dir)
	square := ray.Dir.Dot(ray.Dir)
	rest := from.Dot(from) - ball.radius*ball.radius
	under := along*along - square*rest
	if under < 0 {
		return 0, 0, false
	}
	root := math.Sqrt(under)
	return (-along - root) / square, (-along + root) / square, true
}

// Normal is the way out of the ball at a point on it.
func (ball Sphere) Normal(at maths.Vec) maths.Vec {
	return at.Sub(ball.centre).Unit()
}

// Lamp is a point of light: its light at one unit away, falling off
// as the square of the distance.
type Lamp struct {
	at       maths.Vec
	strength maths.Light
	counts   *Counts
}

// NewLamp is a point of light at a place, counted in counts.
func NewLamp(at maths.Vec, strength maths.Light, counts *Counts) Lamp {
	return Lamp{at: at, strength: strength, counts: counts}
}

// Light is which way the lamp lies from a point, how far, and the light
// that reaches the point from it.
func (lamp Lamp) Light(at, _ maths.Vec) (maths.Vec, float64, maths.Light) {
	lamp.counts.light()
	toward := lamp.at.Sub(at)
	far := toward.Len()
	return toward.Scale(1 / far), far, lamp.strength.Scale(1 / (far * far))
}

// Aimed is a sensor whose pixels are rays the harness chose: pixel x of
// the one row is ray x, so a fixture knows exactly where each looks.
type Aimed struct{ rays []maths.Ray }

// NewAimed is a sensor of one row, one pixel for each ray.
func NewAimed(rays ...maths.Ray) Aimed { return Aimed{rays: rays} }

// Ray is the ray pixel x looks along; the row and the size are the
// harness's to keep to one row of len(rays). A pixel past the last is
// answered with the last ray: the tracer's pixelAngle asks for the pixel
// beside the middle one, which is outside a picture one or two wide.
func (sensor Aimed) Ray(x, _, _, _ int, _ maths.Band) maths.Ray {
	return sensor.rays[min(x, len(sensor.rays)-1)]
}

// Sky is the same light from every direction.
type Sky struct{ light maths.Light }

// NewSky is a sky of one light everywhere.
func NewSky(light maths.Light) Sky { return Sky{light: light} }

// Light is the sky's light, whichever way one looks.
func (sky Sky) Light(maths.Vec) maths.Light { return sky.light }

// Slab is a box of medium of one density throughout: light going a
// distance d through it keeps exp(-density d) of itself.
type Slab struct {
	box     maths.Box
	density float64
}

// NewSlab is a box of medium between two corners, of one density.
func NewSlab(low, high maths.Vec, density float64) Slab {
	return Slab{box: maths.Box{Min: low, Max: high}, density: density}
}

// Bounds is the slab's box.
func (slab Slab) Bounds() maths.Box { return slab.box }

// Density is the slab's density inside its box and nothing outside.
func (slab Slab) Density(at maths.Vec) float64 {
	inside := at.X >= slab.box.Min.X && at.X <= slab.box.Max.X &&
		at.Y >= slab.box.Min.Y && at.Y <= slab.box.Max.Y &&
		at.Z >= slab.box.Min.Z && at.Z <= slab.box.Max.Z
	if inside {
		return slab.density
	}
	return 0
}

// Span is where a ray enters and leaves the slab's box between near and
// far, worked out exactly, one pair of faces at a time.
func (slab Slab) Span(ray maths.Ray, near, far float64) (float64, float64,
	bool) {
	enter, leave := near, far
	for _, faces := range slab.faces(ray) {
		if faces.way == 0 {
			if faces.from < faces.low || faces.from > faces.high {
				return 0, 0, false
			}
			continue
		}
		in := (faces.low - faces.from) / faces.way
		out := (faces.high - faces.from) / faces.way
		enter = max(enter, min(in, out))
		leave = min(leave, max(in, out))
	}
	return enter, leave, enter <= leave
}

// pair is two of the slab's faces, square to one axis, and where a ray
// starts along that axis and which way it goes.
type pair struct{ low, high, from, way float64 }

// faces is the slab's three pairs of faces as a ray meets them.
func (slab Slab) faces(ray maths.Ray) [3]pair {
	low, high := slab.box.Min, slab.box.Max
	return [3]pair{
		{low: low.X, high: high.X, from: ray.At.X, way: ray.Dir.X},
		{low: low.Y, high: high.Y, from: ray.At.Y, way: ray.Dir.Y},
		{low: low.Z, high: high.Z, from: ray.At.Z, way: ray.Dir.Z},
	}
}
