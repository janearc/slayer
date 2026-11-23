// Package grain is seeded Perlin gradient noise for print grain. Values are
// zero at integer lattice points and interpolate smoothly between them,
// usually within roughly minus one to one.
//
// The noise itself is slayer's one noise, in maths/noise: gradients come
// from its hash rather than a fixed permutation table, so each seed defines
// its own field without mutable state, clock reads, or a shared random
// generator. This package says it in the print's words, and its tests hold
// it to the print's recorded numbers.
package grain

import "github.com/janearc/slayer/maths/noise"

// Gradient is three-dimensional gradient noise at a point. It is zero on every
// integer lattice point and smooth between them. It is seldom past one either
// way. It reaches 1.033 at seed 41885775 near the middle of a cell. Over a
// million points it was never past 0.98.
//
// What a coordinate may be, and what one too large to have a fraction or no
// number at all gives, is as maths/noise says of its Gradient.
func Gradient(seed uint64, x, y, z float64) float64 {
	return noise.Gradient(seed, x, y, z)
}
