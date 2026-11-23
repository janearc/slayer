// Package medium describes density fields that rays pass through: smoke,
// clouds, atmospheres, and gas giants. Density is evaluated in local space;
// materials supply optical properties, and the renderer integrates them.
//
// A medium has density throughout its extent rather than an opaque surface. A
// gas giant can therefore thicken continuously below its visible cloud tops.
package medium

import (
	"math"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// The numbers every medium's arithmetic is written with, each true by
// definition.
const (
	nought  = 0   // defined: the additive identity
	unity   = 1   // defined: the multiplicative identity
	halfway = 0.5 // defined: maps noise from minus one to one onto nought to one
	double  = 2   // defined: the factor between a radius and a width
)

// origin is the middle of a medium's own space.
var origin = maths.V(nought, nought, nought)

// cube is the box of a half width about the origin.
func cube(half float64) maths.Box { return maths.SegmentBox(origin, origin, half) }

// corners is the box between two corners, taken in either order.
func corners(first, second maths.Vec) maths.Box {
	return maths.SegmentBox(first, second, nought)
}

// Medium is anything with a thickness at every point.
type Medium interface {
	// Density is how thick the medium is at p, per unit of distance in
	// the medium's own space: light going a distance d through density
	// k keeps exp(-k d) of itself. Nought is clear.
	//
	// Because the unit is the medium's own, a medium scaled up keeps its
	// look: twice as big and half as thick per unit of the world, the
	// same through the middle. A giant at any size is as opaque as its
	// numbers say.
	Density(point maths.Vec) float64
	// Bounds is the box outside which the density is exactly zero, at
	// every point, so a ray marches only where there is something to add
	// up. Inside it the density may be zero too.
	Bounds() maths.Box
}

// Timed is a medium that moves: Tick is the medium as it is at a time,
// which the tracer asks for once a picture. A medium that is not Timed
// is the same at every time.
type Timed interface {
	Medium
	Tick(at time.Duration) Medium
}

// Spanner is a medium that can say exactly where a ray enters and leaves
// it, closer than its box: a ball by its sphere. A march then starts on
// the medium itself, so every ray's first step lies at the same depth and
// a surface that is really a steep rise in density does not terrace.
type Spanner interface {
	// Span is where the ray enters and leaves the medium between near
	// and far, if it does.
	Span(ray maths.Ray, near, far float64) (enter, leave float64, ok bool)
}

// Interval is one contiguous occupied ray segment in a medium's local units.
// An empty interval has Leave <= Enter.
type Interval struct{ Enter, Leave float64 }

// SplitSpanner preserves both sides of a hollow shell without marching its
// empty interior. At most two disjoint intervals are returned, nearest first.
type SplitSpanner interface {
	Spans(ray maths.Ray, near, far float64) [shellLegs]Interval
}

// sphere is where a ray enters and leaves a ball of the radius about the
// origin, between near and far: the two roots, each kept inside the range.
func sphere(radius float64, ray maths.Ray, near, far float64) (float64,
	float64, bool) {
	// Name the fused operations used by the recorded arm64 renderer.
	// Otherwise coverage instrumentation can change which product the
	// compiler rounds first, moving the same ray's roots by a last bit.
	along := math.FMA(ray.At.Z, ray.Dir.Z,
		math.FMA(ray.At.X, ray.Dir.X, ray.At.Y*ray.Dir.Y))
	squared := math.FMA(ray.At.Z, ray.At.Z,
		math.FMA(ray.At.Y, ray.At.Y, ray.At.X*ray.At.X))
	offset := math.FMA(-radius, radius, squared)
	discriminant := math.FMA(along, along, -offset)
	if discriminant < nought {
		return nought, nought, false
	}
	root := math.Sqrt(discriminant)
	enter, leave := math.Max(-along-root, near), math.Min(-along+root, far)
	return enter, leave, leave > enter
}

// Puff is a ball of something, thickest in the middle and thinning to
// nothing at its edge: a cloud, a puff of smoke.
type Puff struct {
	Radius float64
	// Thick is the density at the middle.
	Thick float64
}

// Bounds is the cube the puff just fits in.
func (puff Puff) Bounds() maths.Box {
	return cube(puff.Radius)
}

// Span is where a ray crosses the puff's ball.
func (puff Puff) Span(ray maths.Ray, near, far float64) (float64, float64,
	bool) {
	return sphere(puff.Radius, ray, near, far)
}

// Density is Thick at the middle, falling smoothly to zero at the edge.
func (puff Puff) Density(point maths.Vec) float64 {
	if puff.Radius <= nought {
		return nought
	}
	return puff.Thick * (unity - maths.Smooth(nought, unity,
		point.Len()/puff.Radius))
}

// Wispy is a medium made ragged by noise: its density thinned by a fractal
// noise at every point, never by more than Amount of it, so a smooth puff
// becomes smoke.
//
// Seed makes every wisp its own; Scale is how big the wisps are, in the
// medium's own units; Amount is how much of the density the noise may take
// away, from zero for none to one for as much as the noise reaches: over a
// million points it kept 0.16 to 0.84.
type Wispy struct {
	Of     Medium
	Seed   uint64
	Scale  float64
	Amount float64
}

// Bounds is the medium's own.
func (wispy Wispy) Bounds() maths.Box {
	if wispy.Of == nil {
		return corners(origin, origin)
	}
	return wispy.Of.Bounds()
}

// Span is the medium's own, when it has one; otherwise its box's.
func (wispy Wispy) Span(ray maths.Ray, near, far float64) (float64, float64,
	bool) {
	if spanner, ok := wispy.Of.(Spanner); ok {
		return spanner.Span(ray, near, far)
	}
	return wispy.Bounds().Hit(ray, near, far)
}

// Tick is the wisps of the medium as it is at a time: the medium moves as
// it does, when it does, and the noise stands still.
func (wispy Wispy) Tick(at time.Duration) Medium {
	if timed, ok := wispy.Of.(Timed); ok {
		wispy.Of = timed.Tick(at)
	}
	return wispy
}

// Density is the medium's, thinned by the noise at that point.
func (wispy Wispy) Density(point maths.Vec) float64 {
	if wispy.Of == nil {
		return nought
	}
	density := wispy.Of.Density(point)
	if density == nought {
		return nought
	}
	scale := wispy.Scale
	if scale <= nought {
		scale = unity
	}
	grain := noise.Fractal(wispy.Seed, point.X/scale, point.Y/scale,
		point.Z/scale, chosen.wispyOctaves)
	thin := math.Max(nought, math.Min(unity, wispy.Amount))
	return density * (unity - thin*(halfway+halfway*grain))
}

// Giant is the body of a gas giant: clear above Radius, and below it
// thickening over Depth until nothing gets through. There is no surface;
// what looks like one is where the light stops.
type Giant struct {
	Radius float64
	// Depth is how far below Radius the density reaches Thick.
	Depth float64
	// Thick is the density at full depth.
	Thick float64
}

// Bounds is the cube the giant just fits in.
func (giant Giant) Bounds() maths.Box {
	return cube(giant.Radius)
}

// Span is where a ray crosses the giant's ball: its cloud tops.
func (giant Giant) Span(ray maths.Ray, near, far float64) (float64, float64,
	bool) {
	return sphere(giant.Radius, ray, near, far)
}

// Density is zero outside Radius and rises smoothly to Thick by Depth below it,
// staying there all the way down.
func (giant Giant) Density(point maths.Vec) float64 {
	if giant.Radius <= nought || giant.Depth <= nought {
		return nought
	}
	below := giant.Radius - point.Len()
	if below <= nought {
		return nought
	}
	return giant.Thick * maths.Smooth(nought, giant.Depth, below)
}
