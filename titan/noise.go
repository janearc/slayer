package titan

import "math"

// smoothed is smooth noise at a point in space, from about minus one to
// one: nought at every whole-numbered point, rising and falling between
// them with no creases. The same seed and point give the same number for
// ever.
//
// it is Perlin's gradient noise (Perlin 2002), with each lattice corner's
// direction taken from a hash and not from his shuffled table. slayer's
// maths/noise does the same; this is written afresh, from memory, so that
// celestial does not lean on slayer.
func smoothed(seed uint64, at vector) float64 {
	floorX, floorY, floorZ := math.Floor(at.x), math.Floor(at.y),
		math.Floor(at.z)
	x, y, z := at.x-floorX, at.y-floorY, at.z-floorZ
	corner := func(dx, dy, dz float64) float64 {
		bits := mix(seed + golden)
		bits = mix(bits ^ mix(uint64(int64(floorX+dx))+golden))
		bits = mix(bits ^ mix(uint64(int64(floorY+dy))+golden))
		bits = mix(bits ^ mix(uint64(int64(floorZ+dz))+golden))
		return leaning(bits, x-dx, y-dy, z-dz)
	}
	u, v, w := eased(x), eased(y), eased(z)
	near := between(v, between(u, corner(0, 0, 0), corner(1, 0, 0)),
		between(u, corner(0, 1, 0), corner(1, 1, 0)))
	far := between(v, between(u, corner(0, 0, 1), corner(1, 0, 1)),
		between(u, corner(0, 1, 1), corner(1, 1, 1)))
	return between(w, near, far)
}

// edges is how many directions a lattice corner may lean in: the twelve
// edges of a cube. It's defined by Perlin's noise.
const edges = 12

// leaning is how far an offset goes along one of the twelve directions
// from a cube's middle to its edges, picked by a hash.
func leaning(bits uint64, x, y, z float64) float64 {
	switch bits % edges {
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

// eased is Perlin's quintic, 6t^5 - 15t^4 + 10t^3. Its slope and its
// bend are both nought at each end, so the lattice does not show.
func eased(t float64) float64 { return t * t * t * (t*(t*6-15) + 10) }

// between is a straight line from one number to another.
func between(t, from, to float64) float64 { return from + t*(to-from) }
