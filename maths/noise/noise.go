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
	fx, fy, fz := math.Floor(x), math.Floor(y), math.Floor(z)
	ix, iy, iz := int64(fx), int64(fy), int64(fz)
	x, y, z = x-fx, y-fy, z-fz
	u, v, w := fade(x), fade(y), fade(z)

	g := func(dx, dy, dz int64) float64 {
		h := bits3(seed, ix+dx, iy+dy, iz+dz)
		return grad(h, x-float64(dx), y-float64(dy), z-float64(dz))
	}
	return lerp(w,
		lerp(v, lerp(u, g(0, 0, 0), g(1, 0, 0)),
			lerp(u, g(0, 1, 0), g(1, 1, 0))),
		lerp(v, lerp(u, g(0, 0, 1), g(1, 0, 1)),
			lerp(u, g(0, 1, 1), g(1, 1, 1))))
}

// finite is whether a number is one: not NaN, and not infinite either way.
func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// fade is Perlin's quintic, whose first two derivatives are zero at both ends,
// so the lattice does not show as creases.
func fade(t float64) float64 { return t * t * t * (t*(t*6-15) + 10) }

// lerp is a straight line from a to b.
func lerp(t, a, b float64) float64 { return a + t*(b-a) }

// grad is the dot product of an offset with one of the twelve edge
// directions of a cube, chosen by the low bits of a hash.
func grad(h uint64, x, y, z float64) float64 {
	switch h % 12 {
	case 0:
		return x + y
	case 1:
		return -x + y
	case 2:
		return x - y
	case 3:
		return -x - y
	case 4:
		return x + z
	case 5:
		return -x + z
	case 6:
		return x - z
	case 7:
		return -x - z
	case 8:
		return y + z
	case 9:
		return -y + z
	case 10:
		return y - z
	}
	return -y - z
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
		s := seed + uint64(o)*0x632be59bd9b4e019
		sum += amp * Gradient(s, x, y, z)
		norm += amp
		amp *= 0.5
		// 2.03 and not 2, so the octaves do not line up
		x, y, z = x*2.03, y*2.03, z*2.03
	}
	if norm == 0 {
		return 0
	}
	return sum / norm
}
