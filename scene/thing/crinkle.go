package thing

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
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
type Crinkle struct {
	Size, Slope float64
	Seed        uint32
}

// crinkleOctaves is how many sizes of wrinkle lie one on another, each
// half the last, as a blanket's crumples do.
const crinkleOctaves = 3

// crinkleStep is how far apart, in widths of a wrinkle, the noise is
// read to find its slope.
const crinkleStep = 0.1

// Crinkled is the normal at a point of a surface with these wrinkles,
// seen along look: the normal tilted against the slope of the noise
// across it, scaled by Slope. A tilt that would turn the face away from
// the eye is not taken; the normal is left as it was.
func (c *Crinkle) Crinkled(point, normal, look maths.Vec) maths.Vec {
	if c == nil || !(c.Size > 0) || !(c.Slope > 0) {
		return normal
	}
	at := point.Scale(1 / c.Size)
	seed := uint64(c.Seed)
	here := noise.Fractal(seed, at.X, at.Y, at.Z, crinkleOctaves)
	slope := maths.V(
		noise.Fractal(seed, at.X+crinkleStep, at.Y, at.Z, crinkleOctaves)-here,
		noise.Fractal(seed, at.X, at.Y+crinkleStep, at.Z, crinkleOctaves)-here,
		noise.Fractal(seed, at.X, at.Y, at.Z+crinkleStep, crinkleOctaves)-here,
	).Scale(1 / crinkleStep)
	across := slope.Sub(normal.Scale(slope.Dot(normal)))
	tilted := normal.Sub(across.Scale(c.Slope)).Unit()
	if !(tilted.Dot(look) > 0.05) || math.IsNaN(tilted.X) {
		return normal
	}
	return tilted
}
