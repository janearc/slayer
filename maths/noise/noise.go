// Package noise supplies deterministic hashes and smooth seeded noise. The
// values look irregular but are reproducible without mutable random state.
package noise

import "math"

// mix is splitmix64's finaliser: every bit of the input reaches every bit
// of the output.
func mix(z uint64) uint64 {
	z ^= z >> 30
	z *= 0xbf58476d1ce4e5b9
	z ^= z >> 27
	z *= 0x94d049bb133111eb
	return z ^ z>>31
}

// unit is the top 53 bits of a hash as a number from zero up to one.
func unit(z uint64) float64 { return float64(z>>11) / (1 << 53) }

// golden is the odd constant splitmix64 steps by. added to an index before it
// is mixed, it keeps index zero from mixing to zero, which mix does with zero.
const golden = 0x9e3779b97f4a7c15

// Hash is a number from zero up to one for a seed and an index. The same pair
// gives the same number forever.
func Hash(seed uint64, i int64) float64 {
	return unit(mix(seed ^ mix(uint64(i)+golden)))
}

// Hash2 is Hash for a pair of indices.
func Hash2(seed uint64, i, j int64) float64 {
	j2 := mix(uint64(j) + golden)
	return unit(mix(seed ^ mix(uint64(i)+j2)))
}

// Hash3 is Hash for three indices.
func Hash3(seed uint64, i, j, k int64) float64 {
	return unit(bits3(seed, i, j, k))
}

// bits3 is the whole hash of three indices, for a caller that wants
// several numbers out of one: a lattice corner's gradient takes its low
// bits.
func bits3(seed uint64, i, j, k int64) uint64 {
	return mix(seed ^ mix(uint64(i)+mix(uint64(j)+
		mix(uint64(k)+golden))))
}

// Gradient is three-dimensional gradient noise at a point: zero on every
// integer lattice point, smooth between them, and seldom past one either way:
// it can be, a little, as 1.033 at seed 41885775 and (0.35, 0.5, 0.475). over a
// million points it was never past 0.98.
//
// a coordinate is a number within 2^53 of zero. past that a float has no
// fraction, so the noise is the same everywhere; past 2^63 the lattice index is
// whatever the machine makes of it; NaN gives NaN.
//
// Perlin's improved noise, with the gradient at each corner taken from the hash
// rather than a shuffled table. A table is 256 entries of one fixed shuffle;
// the hash gives every seed its own field, which is how two planets differ.
func Gradient(seed uint64, x, y, z float64) float64 {
	at := cellAt(x, y, z)
	return at.gradient(seed)
}

// cell is a point in its lattice cell, worked out once for every seed
// asked there: its place in the cell, the fades of it, and each
// corner's hash before the seed is mixed in, by its step along x, y, z.
type cell struct {
	x, y, z, u, v, w float64
	placed           [cellEnds][cellEnds][cellEnds]uint64
}

// cellAt is the cell a point is in, and where in it.
func cellAt(x, y, z float64) cell {
	fx, fy, fz := math.Floor(x), math.Floor(y), math.Floor(z)
	ix, iy, iz := int64(fx), int64(fy), int64(fz)
	at := cell{x: x - fx, y: y - fy, z: z - fz}
	at.u, at.v, at.w = fade(at.x), fade(at.y), fade(at.z)
	inner := corners(iy, iz)
	for dx := range int64(cellEnds) {
		for dy := range cellEnds {
			for dz := range cellEnds {
				at.placed[dx][dy][dz] = mix(uint64(ix+dx) + inner[dy][dz])
			}
		}
	}
	return at
}

// gradient is Gradient's noise for a seed at the cell's point: each
// corner's hash finished with the seed, as bits3 finishes it, graded,
// and blended across the cell.
func (at cell) gradient(seed uint64) float64 {
	g := func(dx, dy, dz int64) float64 {
		h := mix(seed ^ at.placed[dx][dy][dz])
		return grad(h, at.x-float64(dx), at.y-float64(dy),
			at.z-float64(dz))
	}
	u, v, w := at.u, at.v, at.w
	return lerp(w,
		lerp(v, lerp(u, g(0, 0, 0), g(1, 0, 0)),
			lerp(u, g(0, 1, 0), g(1, 1, 0))),
		lerp(v, lerp(u, g(0, 0, 1), g(1, 0, 1)),
			lerp(u, g(0, 1, 1), g(1, 1, 1))))
}

// corners is the part of bits3 that the four columns of a lattice cell
// share, by the corner's step along y and along z: the hash of j and k,
// which every corner's i is then added to. Worked once a cell, it is the
// same number bits3 works for each of the eight corners.
func corners(j, k int64) [cellEnds][cellEnds]uint64 {
	var inner [cellEnds][cellEnds]uint64
	for dz := range int64(cellEnds) {
		under := mix(uint64(k+dz) + golden)
		for dy := range int64(cellEnds) {
			inner[dy][dz] = mix(uint64(j+dy) + under)
		}
	}
	return inner
}

// cellEnds is the two ends of a lattice cell along an axis, its low
// corner and its high, true by definition.
const cellEnds = 2

// finite is whether a number is one: not NaN, and not infinite either way.
func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// fade is Perlin's quintic, whose first two derivatives are zero at both ends,
// so the lattice does not show as creases.
func fade(t float64) float64 { return t * t * t * (t*(t*6-15) + 10) }

// lerp is a straight line from a to b.
func lerp(t, a, b float64) float64 { return a + t*(b-a) }

// grad is the dot product of an offset with one of the twelve edge
// directions of a cube, chosen by the low bits of a hash.
//
// The edge is looked up, not switched on: the hash makes the choice
// unpredictable, and a mispredicted branch was most of the noise's time.
// Each term is the offset's number times one or minus one, which is
// exact, so the sum is the one the switch made, to the bit.
func grad(h uint64, x, y, z float64) float64 {
	along := edges[h%cubeEdges]
	offset := [3]float64{x, y, z}
	return along.firstSign*offset[along.first] +
		along.secondSign*offset[along.second]
}

// cubeEdges is how many edges a cube has, true by definition.
const cubeEdges = 12

// edge is one edge direction of a cube as grad reads it: which two of an
// offset's numbers it adds, x, y or z by place, and the sign of each.
type edge struct {
	first, second         int
	firstSign, secondSign float64
}

// the places of an offset's numbers, and the signs an edge gives them
const (
	alongX, alongY, alongZ = 0, 1, 2
	plus, minus            = 1.0, -1.0
)

// edges are the twelve, in the order of Perlin's improved noise: x+y,
// -x+y, x-y, -x-y, then x and z, then y and z, the same way round.
var edges = [cubeEdges]edge{
	{alongX, alongY, plus, plus}, {alongX, alongY, minus, plus},
	{alongX, alongY, plus, minus}, {alongX, alongY, minus, minus},
	{alongX, alongZ, plus, plus}, {alongX, alongZ, minus, plus},
	{alongX, alongZ, plus, minus}, {alongX, alongZ, minus, minus},
	{alongY, alongZ, plus, plus}, {alongY, alongZ, minus, plus},
	{alongY, alongZ, plus, minus}, {alongY, alongZ, minus, minus},
}

// Fractal is gradient noise summed over octaves, each about twice the
// frequency and half the amplitude of the one before. It is what makes a
// coastline ragged at every scale rather than round at one.
//
// the sum is divided by its octaves' amplitudes added up, so it is never
// wider than one octave, which is Gradient and can pass one a little.
//
// the octaves seldom peak together, so it is narrower the more there are:
// over a million points, 0.98 at most for one octave, 0.83 for two, and
// 0.65 for five or eight.
//
// each octave's field is the seed plus a step times the octave, so two
// seeds a step apart share fields an octave apart. an octave whose
// position is past what a float holds, which a thousand or so do, is not
// added, nor any after it.
func Fractal(seed uint64, x, y, z float64, octaves int) float64 {
	sum, amp, norm := 0.0, 1.0, 0.0
	for o := range octaves {
		if !finite(x) || !finite(y) || !finite(z) {
			break
		}
		s := seed + uint64(o)*octaveStep
		sum += amp * Gradient(s, x, y, z)
		norm += amp
		amp *= octaveFall
		x, y, z = x*octaveRise, y*octaveRise, z*octaveRise
	}
	if norm == 0 {
		return 0
	}
	return sum / norm
}

// the octaves of a fractal: each octave's field is the seed plus this
// step times the octave; each is this share of the one before it; and
// each is this many times finer, 2.03 and not 2, so the octaves do not
// line up. All three are picked, and are what slayer has always drawn.
const (
	octaveStep = 0x632be59bd9b4e019
	octaveFall = 0.5
	octaveRise = 2.03
)

// Fractals is Fractal at one point for several seeds: into[i] is
// Fractal(seeds[i], x, y, z, octaves), to the bit. Each octave's cell,
// its fades and the seed-free part of its corners' hashes are worked
// once for all the seeds, which is most of a fractal's work beside the
// grading. into must be as long as seeds.
func Fractals(seeds []uint64, x, y, z float64, octaves int,
	into []float64) {
	into = into[:len(seeds)]
	clear(into)
	amp, norm := 1.0, 0.0
	for o := range octaves {
		if !finite(x) || !finite(y) || !finite(z) {
			break
		}
		at := cellAt(x, y, z)
		for index, seed := range seeds {
			into[index] += amp * at.gradient(seed+uint64(o)*octaveStep)
		}
		norm += amp
		amp *= octaveFall
		x, y, z = x*octaveRise, y*octaveRise, z*octaveRise
	}
	for index := range into {
		if norm == 0 {
			into[index] = 0
			continue
		}
		into[index] /= norm
	}
}
