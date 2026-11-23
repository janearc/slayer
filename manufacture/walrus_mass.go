package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

const (
	// walrusThreshold is the summed density at which the hide begins.
	walrusThreshold = .30
	// walrusFalloff is the gaussian exponent of one mass's density.
	walrusFalloff = 2.1
	// walrusCutoff is how many radii out a mass is taken to be zero.
	walrusCutoff = 3.0
)

// walrusMass is one smooth lump of the body: a gaussian density over an
// ellipsoid whose axes need not line up with the world. The hide is where
// the summed density of every mass falls to walrusThreshold.
type walrusMass struct {
	centre   maths.Vec
	axes     [3]maths.Vec
	radius   [3]float64
	strength float64
	cutoff   float64
}

// walrusReach is how far, in radii, the hide lies from the centre of one
// mass of the given strength standing alone.
func walrusReach(strength float64) float64 {
	return math.Sqrt(math.Log(strength/walrusThreshold) / walrusFalloff)
}

// walrusBlob is a mass lined up with the world whose own hide would reach
// half out from its centre along each axis.
func walrusBlob(centre, half maths.Vec, strength float64) walrusMass {
	reach := walrusReach(strength)
	radius := [3]float64{half.X / reach, half.Y / reach, half.Z / reach}
	return walrusMass{
		centre: centre,
		axes: [3]maths.Vec{
			maths.V(1, 0, 0), maths.V(0, 1, 0), maths.V(0, 0, 1),
		},
		radius:   radius,
		strength: strength,
		cutoff:   walrusCutoff * math.Max(radius[0], math.Max(radius[1], radius[2])),
	}
}

// walrusPaddle is a flat mass lying along direction, with its thin axis
// as near to vertical as the direction allows. Half holds the half
// length along the direction, the half thickness and the half width.
func walrusPaddle(centre, direction, half maths.Vec, strength float64) walrusMass {
	along := direction.Unit()
	up := maths.V(0, 1, 0)
	thin := up.Sub(along.Scale(up.Dot(along))).Unit()
	wide := along.Cross(thin)
	reach := walrusReach(strength)
	radius := [3]float64{half.X / reach, half.Y / reach, half.Z / reach}
	return walrusMass{
		centre:   centre,
		axes:     [3]maths.Vec{along, thin, wide},
		radius:   radius,
		strength: strength,
		cutoff:   walrusCutoff * math.Max(radius[0], math.Max(radius[1], radius[2])),
	}
}

// density is this mass's share of the summed density at p.
func (mass walrusMass) density(p maths.Vec) float64 {
	offset := p.Sub(mass.centre)
	if math.Abs(offset.X) > mass.cutoff || math.Abs(offset.Y) > mass.cutoff ||
		math.Abs(offset.Z) > mass.cutoff {
		return 0
	}
	sum := 0.
	for i, axis := range mass.axes {
		along := offset.Dot(axis) / mass.radius[i]
		sum += along * along
	}
	return mass.strength * math.Exp(-walrusFalloff*sum)
}

// walrusSegmentDistance is how far p is from the nearest point of the
// segment from a to b.
func walrusSegmentDistance(p, a, b maths.Vec) float64 {
	axis := b.Sub(a)
	share := p.Sub(a).Dot(axis) / axis.Dot(axis)
	share = math.Max(0, math.Min(1, share))
	return p.Sub(a.Add(axis.Scale(share))).Len()
}

// walrusHash is a number from nought to one that is the same each time for
// the same lattice point and salt.
func walrusHash(x, y, z, salt int) float64 {
	const (
		first, second, third, fourth = 73856093, 19349663, 83492791, 2654435761
		mix                          = 0x45d9f3b
		shift                        = 15
		span                         = 1 << 24
	)
	bits := uint32(x)*first ^ uint32(y)*second ^ uint32(z)*third ^
		uint32(salt)*fourth
	bits ^= bits >> shift
	bits *= mix
	bits ^= bits >> shift
	return float64(bits%span) / span
}

// walrusNoise is smooth value noise at p, from nought to one, with one
// lattice point to the unit.
func walrusNoise(p maths.Vec) float64 {
	x, y, z := math.Floor(p.X), math.Floor(p.Y), math.Floor(p.Z)
	fx, fy, fz := maths.Smooth(0, 1, p.X-x), maths.Smooth(0, 1, p.Y-y),
		maths.Smooth(0, 1, p.Z-z)
	ix, iy, iz := int(x), int(y), int(z)
	row := func(dy, dz int) float64 {
		low := walrusHash(ix, iy+dy, iz+dz, 0)
		high := walrusHash(ix+1, iy+dy, iz+dz, 0)
		return low*(1-fx) + high*fx
	}
	near := row(0, 0)*(1-fy) + row(1, 0)*fy
	far := row(0, 1)*(1-fy) + row(1, 1)*fy
	return near*(1-fz) + far*fz
}
