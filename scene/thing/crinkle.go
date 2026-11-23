package thing

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/texture"
)

// Crinkle is a surface's wrinkles, as a spacecraft's blankets have:
// how wide they run, in metres; how steeply they tilt its face, the
// share of the slope of its noise that bends the normal; and a seed,
// below 2^24 so the GPU's floats hold it whole. The tilt bends what the
// surface mirrors and how its lamps light it, so a smooth sheet breaks
// into the scattered glints a crumpled one gives.
//
// The wrinkles are laid in the world's own space, not the thing's: a
// thing that moves slides through them. The GPU's kernel draws the same
// noise.
//
// Facet, if over nought, cuts the surface in flat facets a Size across
// as well, each tipping the normal by up to Facet, as a chisel leaves
// stone. The CPU tracer draws them; the GPU's kernel does not.
type Crinkle struct {
	Size, Slope float64
	Seed        uint32
	Facet       float64
}

// The counts a crinkle's arithmetic is written with.
const (
	// crinkleNone is defined: no amount, and a pixel of no width.
	crinkleNone = 0
	// crinkleWhole is defined: one amount, the whole of a share.
	crinkleWhole = 1
)

// crinkleOctaves is how many sizes of wrinkle lie one on another, each
// half the last, as a blanket's crumples do. It is read from picks.yaml.
var crinkleOctaves = int(pick("crinkle-octaves"))

// crinkleStep is how far apart, in widths of a wrinkle, the noise is
// read to find its slope. It is read from picks.yaml.
var crinkleStep = pick("crinkle-step")

// crinkleFacingLeast is the least cosine to the eye a tilted normal may
// have and be kept. It is read from picks.yaml.
var crinkleFacingLeast = pick("crinkle-facing-least")

// crinklePositive reports whether a number is above nothing. A number that
// is no number is not.
func crinklePositive(value float64) bool {
	return value > crinkleNone
}

// CrinkledWithin is the normal at a point of a surface with these wrinkles,
// seen along look: the normal tilted against the slope of the noise
// across it, scaled by Slope. A tilt that would turn the face away from
// the eye is not taken, nor is one that is no number; the normal is left
// as it was.
//
// It is for a pixel that is a given width across at the point, in metres.
// Wrinkles finer than the pixel cannot be seen as wrinkles and would only
// glitter, so their tilt fades with the pixel's width, by texture.Detail,
// to none: a far wall is flat, a near one wrinkled. A width of nought is a
// lone ray, and the whole tilt.
func (crinkle *Crinkle) CrinkledWithin(point, normal, look maths.Vec,
	pixel float64) maths.Vec {
	if crinkle == nil || !crinklePositive(crinkle.Size) ||
		!(crinklePositive(crinkle.Slope) || crinklePositive(crinkle.Facet)) {
		return normal
	}
	shown := texture.Detail(pixel, crinkle.Size)
	if !crinklePositive(shown) {
		return normal
	}
	at := point.Scale(crinkleWhole / crinkle.Size)
	slope := crinkleSlope(uint64(crinkle.Seed), at)
	along := slope.Dot(normal)
	across := slope.Sub(normal.Scale(along))
	tilted := normal.Sub(across.Scale(crinkle.Slope * shown)).
		Add(crinkle.chisel(point, normal, shown)).Unit()
	if !(tilted.Dot(look) > crinkleFacingLeast) {
		return normal
	}
	return tilted
}

// crinkleSlope is the slope of the noise at a point, in widths of a
// wrinkle: the noise a step along each axis less the noise there, over
// the step.
func crinkleSlope(seed uint64, at maths.Vec) maths.Vec {
	here := noise.Fractal(seed, at.X, at.Y, at.Z, crinkleOctaves)
	return maths.V(
		noise.Fractal(seed, at.X+crinkleStep, at.Y, at.Z, crinkleOctaves)-here,
		noise.Fractal(seed, at.X, at.Y+crinkleStep, at.Z, crinkleOctaves)-here,
		noise.Fractal(seed, at.X, at.Y, at.Z+crinkleStep, crinkleOctaves)-here,
	).Scale(crinkleWhole / crinkleStep)
}
