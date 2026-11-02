// Package grain implements seeded Perlin gradient noise for print grain. Values
// are zero at integer lattice points and interpolate smoothly between them,
// usually within roughly minus one to one.
//
// Gradients come from the emulsion hash rather than a fixed permutation table.
// Each seed defines its own field without mutable state, clock reads, or a
// shared random generator.
package grain

import (
	"math"

	"github.com/janearc/slayer/ninjafilm/hash"
)

// Perlin's quintic is 6t^5 - 15t^4 + 10t^3. It is written here as t
// cubed times (t times (t times 6 minus 15) plus 10).
const (
	// quinticLead is the 6 of the quintic. It is defined by the quintic.
	quinticLead = 6
	// quinticSecond is the 15 of the quintic. It is defined by it.
	quinticSecond = 15
	// quinticThird is the 10 of the quintic. It is defined by it.
	quinticThird = 10
)

// The twelve gradients are the twelve edge directions of a cube. Each
// is two of the three axes with a sign on each.
//
// The pairs of axes run xy then xz then yz. Four numbers go to a pair.
// The first axis is positive on the even numbers. The second is positive
// when the number is under two.
const (
	// edges is how many edge directions a cube has. It is defined by
	// the cube.
	edges = 12
	// choices is how many numbers go to a pair of axes. It is defined
	// as the four ways to sign two axes.
	choices = 4
	// coupled is how many pairs share their first axis, x. It is
	// defined by the numbering above.
	coupled = 2
	// lastAxis is the place of z among the axes. It is defined as two.
	lastAxis = 2
	// nextAxis is how far on the second axis of a pair is from its first,
	// before z stops it. It is defined as one.
	nextAxis = 1
	// signsBy is how many signs an axis has. It is defined as two.
	signsBy = 2
	// positive is the plus sign. It is defined as one.
	positive = 1
	// dimensions is how many axes space has. It is defined as three.
	dimensions = 3
	// none is no amount. It is defined as zero.
	none = 0
	// whole is one. It is defined as one.
	whole = 1
)

// cell is the lattice cell a point lies in, and where in it the point
// is.
type cell struct {
	seed       uint64
	ix, iy, iz int64
	x, y, z    float64
}

// Gradient is three-dimensional gradient noise at a point. It is zero on every
// integer lattice point and smooth between them. It is seldom past one either
// way. It reaches 1.033 at seed 41885775 near the middle of a cell. Over a
// million points it was never past 0.98.
//
// Coordinates are numbers within 2^53 of zero. Past that a float has no
// fraction, so the noise is the same everywhere. Past 2^63 the lattice index is
// whatever the machine makes of it. NaN gives NaN.
//
// It is Perlin's improved noise. The gradient at each corner is taken from the
// hash. Tables are 256 entries of one fixed shuffle. The hash gives every seed
// its own field. That is how two planets differ.
func Gradient(seed uint64, x, y, z float64) float64 {
	fx, fy, fz := math.Floor(x), math.Floor(y), math.Floor(z)
	at := cell{seed: seed, ix: int64(fx), iy: int64(fy), iz: int64(fz),
		x: x - fx, y: y - fy, z: z - fz}
	smoothX, smoothY, smoothZ := fade(at.x), fade(at.y), fade(at.z)
	low := at.plane(none, smoothX, smoothY)
	high := at.plane(whole, smoothX, smoothY)
	return lerp(smoothZ, low, high)
}

// plane is the noise on one face of the cell. It is a step up in z or
// none. It is blended across x and then across y.
func (at cell) plane(dz int64, smoothX, smoothY float64) float64 {
	near := lerp(smoothX, at.corner(none, none, dz),
		at.corner(whole, none, dz))
	far := lerp(smoothX, at.corner(none, whole, dz),
		at.corner(whole, whole, dz))
	return lerp(smoothY, near, far)
}

// corner is the gradient of one corner of the cell dotted with the way
// from that corner to the point.
func (at cell) corner(dx, dy, dz int64) float64 {
	bits := hash.Bits(at.seed, at.ix+dx, at.iy+dy, at.iz+dz)
	return slope(bits, at.x-float64(dx), at.y-float64(dy), at.z-float64(dz))
}

// fade is Perlin's quintic. Its first two derivatives are zero at both ends, so
// the lattice does not show as creases.
func fade(along float64) float64 {
	cubed := along * along * along
	return cubed * (along*(along*quinticLead-quinticSecond) + quinticThird)
}

// lerp is a straight line from one value to another, along its length.
func lerp(along, from, to float64) float64 { return from + along*(to-from) }

// slope is the dot product of an offset with one of the twelve edge
// directions of a cube. The low bits of a hash choose it.
func slope(bits uint64, x, y, z float64) float64 {
	which := int(bits % edges)
	pair, signs := which/choices, which%choices
	parts := [dimensions]float64{x, y, z}
	first := parts[pair/coupled]
	second := parts[min(pair+nextAxis, lastAxis)]
	leading := float64(positive - signsBy*(signs%signsBy))
	trailing := float64(positive - signsBy*(signs/signsBy))
	return leading*first + trailing*second
}
