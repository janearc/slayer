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
	Density(p maths.Vec) float64
	// Bounds is the box outside which the density is zero, so a ray marches
	// only where there is something to add up.
	//
	// Plume, Jet and Tail reach past theirs, as they were first drawn:
	// Plume by a quarter of Thick at its side, the other two by a
	// hundredth. they are not widened here.
	Bounds() maths.Box
}

// Timed is a medium that moves: Tick is the medium as it is at a time,
// which the tracer asks for once a picture. A medium that is not Timed
// is the same at every time.
type Timed interface {
	Medium
	Tick(t time.Duration) Medium
}

// Spanner is a medium that can say exactly where a ray enters and leaves
// it, closer than its box: a ball by its sphere. A march then starts on
// the medium itself, so every ray's first step lies at the same depth and
// a surface that is really a steep rise in density does not terrace.
type Spanner interface {
	// Span is where the ray enters and leaves the medium between near
	// and far, if it does.
	Span(r maths.Ray, near, far float64) (enter, leave float64, ok bool)
}

// sphere is where a ray enters and leaves a ball of radius rad about the
// origin, between near and far: the two roots, each kept inside the range.
func sphere(rad float64, r maths.Ray, near, far float64) (float64, float64,
	bool) {
	b := r.At.Dot(r.Dir)
	c := r.At.Dot(r.At) - rad*rad
	disc := b*b - c
	if disc < 0 {
		return 0, 0, false
	}
	root := math.Sqrt(disc)
	enter, leave := math.Max(-b-root, near), math.Min(-b+root, far)
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
func (p Puff) Bounds() maths.Box {
	r := p.Radius
	return maths.Box{Min: maths.V(-r, -r, -r), Max: maths.V(r, r, r)}
}

// Span is where a ray crosses the puff's ball.
func (p Puff) Span(r maths.Ray, near, far float64) (float64, float64, bool) {
	return sphere(p.Radius, r, near, far)
}

// Density is Thick at the middle, falling smoothly to zero at the edge.
func (p Puff) Density(at maths.Vec) float64 {
	if p.Radius <= 0 {
		return 0
	}
	return p.Thick * (1 - maths.Smooth(0, 1, at.Len()/p.Radius))
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
func (w Wispy) Bounds() maths.Box {
	if w.Of == nil {
		return maths.Box{}
	}
	return w.Of.Bounds()
}

// Span is the medium's own, when it has one; otherwise its box's.
func (w Wispy) Span(r maths.Ray, near, far float64) (float64, float64, bool) {
	if sp, ok := w.Of.(Spanner); ok {
		return sp.Span(r, near, far)
	}
	return w.Bounds().Hit(r, near, far)
}

// Density is the medium's, thinned by the noise at that point.
func (w Wispy) Density(p maths.Vec) float64 {
	if w.Of == nil {
		return 0
	}
	d := w.Of.Density(p)
	if d == 0 {
		return 0
	}
	scale := w.Scale
	if scale <= 0 {
		scale = 1
	}
	n := noise.Fractal(w.Seed, p.X/scale, p.Y/scale, p.Z/scale, 4)
	thin := math.Max(0, math.Min(1, w.Amount))
	return d * (1 - thin*(0.5+0.5*n))
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
func (g Giant) Bounds() maths.Box {
	r := g.Radius
	return maths.Box{Min: maths.V(-r, -r, -r), Max: maths.V(r, r, r)}
}

// Span is where a ray crosses the giant's ball: its cloud tops.
func (g Giant) Span(r maths.Ray, near, far float64) (float64, float64, bool) {
	return sphere(g.Radius, r, near, far)
}

// Density is zero outside Radius and rises smoothly to Thick by Depth below it,
// staying there all the way down.
func (g Giant) Density(p maths.Vec) float64 {
	if g.Radius <= 0 || g.Depth <= 0 {
		return 0
	}
	below := g.Radius - p.Len()
	if below <= 0 {
		return 0
	}
	return g.Thick * maths.Smooth(0, g.Depth, below)
}
