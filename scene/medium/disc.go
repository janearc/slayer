package medium

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// firstEntry is the place of a profile's first entry.
const firstEntry = 0 // defined: the first place of a slice

// The numbers of a disc's ringlets and faces are chosen for the picture, in
// choices.yaml. discRinglets is how many ringlets there are when Ringlets
// says none; discRingY and discRingZ are where the ringlet noise is read on
// its other two axes; discOctaves is its octaves; and discFace is how much of
// the disc's half thickness each face is soft over.
var (
	discRinglets = choice("disc-ringlets")
	discRingY    = choice("disc-ring-y")
	discRingZ    = choice("disc-ring-z")
	discOctaves  = choiceCount("disc-octaves")
	discFace     = choice("disc-face")
)

// Disc is a flat ring of something round its own y axis: the rings of a
// giant, or, glowing, the disc round a hole. It runs from Inner to Outer
// and is Thick across, and how dense it is varies with the radius by a
// Profile, which is where the gaps are, and by ringlets of noise.
//
// Its radial edges are not soft: at Inner and at Outer the density is the
// profile's value there.
type Disc struct {
	Inner, Outer, Thick float64
	// Dense is the density where the profile is one. Zero means one. A
	// ring that keeps a third of the light straight through has Dense
	// of about 1.16 over its thickness: its soft faces take a twentieth
	// of the depth, so one over the thickness keeps e^-0.95, 0.39.
	Dense float64
	// Profile is the density from Inner to Outer, evenly spaced and
	// blended between: a gap is a low entry. Empty is one everywhere.
	Profile []float64
	// Grain is how much ringlets of noise thin the disc, from zero for a
	// smooth disc to one for ringlets as thin as the noise reaches: over a
	// million points, never below 0.13 of the density.
	Grain float64
	// Ringlets is how many ringlets there are across the disc; zero
	// means sixty.
	Ringlets float64
	Seed     uint64
}

// Bounds is the box the disc just fits in.
func (disc Disc) Bounds() maths.Box {
	outer, half := disc.Outer, disc.Thick/double
	return corners(maths.V(-outer, -half, -outer), maths.V(outer, half, outer))
}

// Span is where a ray crosses the slab the disc lies in, clipped to the
// cylinder of its outer radius.
func (disc Disc) Span(ray maths.Ray, near, far float64) (float64, float64,
	bool) {
	half := disc.Thick / double
	enter, leave := near, far
	if ray.Dir.Y == nought {
		if math.Abs(ray.At.Y) > half {
			return nought, nought, false
		}
	} else {
		lower := (-half - ray.At.Y) / ray.Dir.Y
		upper := (half - ray.At.Y) / ray.Dir.Y
		if lower > upper {
			lower, upper = upper, lower
		}
		enter, leave = math.Max(enter, lower), math.Min(leave, upper)
	}
	// the cylinder: the ray's x and z against the outer radius
	// Keep the recorded arm64 rounding order explicit, including when
	// coverage counters change the compiler's product selection.
	sideways := math.FMA(ray.Dir.X, ray.Dir.X, ray.Dir.Z*ray.Dir.Z)
	if sideways > nought {
		along := math.FMA(ray.At.X, ray.Dir.X, ray.At.Z*ray.Dir.Z) / sideways
		squared := math.FMA(ray.At.Z, ray.At.Z, ray.At.X*ray.At.X)
		offset := math.FMA(-disc.Outer, disc.Outer, squared) / sideways
		discriminant := math.FMA(along, along, -offset)
		if discriminant < nought {
			return nought, nought, false
		}
		root := math.Sqrt(discriminant)
		enter = math.Max(enter, -along-root)
		leave = math.Min(leave, -along+root)
	} else if ray.At.X*ray.At.X+ray.At.Z*ray.At.Z > disc.Outer*disc.Outer {
		return nought, nought, false
	}
	return enter, leave, leave > enter
}

// Density is the profile's density at the point's radius, thinned by the
// ringlets, soft at the disc's two faces, and zero outside it, and at a point
// that is not a number.
func (disc Disc) Density(point maths.Vec) float64 {
	half := disc.Thick / double
	if half <= nought || disc.Outer <= disc.Inner ||
		math.IsNaN(point.X+point.Y+point.Z) {
		return nought
	}
	radius := math.Hypot(point.X, point.Z)
	if radius < disc.Inner || radius > disc.Outer ||
		math.Abs(point.Y) > half {
		return nought
	}
	across := (radius - disc.Inner) / (disc.Outer - disc.Inner)
	density := disc.profile(across)
	if disc.Dense > nought {
		density *= disc.Dense
	}
	if disc.Grain > nought {
		ringlets := disc.Ringlets
		if ringlets <= nought {
			ringlets = discRinglets
		}
		ring := noise.Fractal(disc.Seed, across*ringlets, discRingY,
			discRingZ, discOctaves)
		thin := math.Max(nought, math.Min(unity, disc.Grain))
		density *= unity - thin*(halfway+halfway*ring)
	}
	// soft at each face, over the outer twentieth of the thickness
	return density * maths.Smooth(nought, discFace*half,
		half-math.Abs(point.Y))
}

// profile is the density at a share from zero at Inner to one at Outer. a
// share that is not a number, from a point that is none, is no density.
func (disc Disc) profile(share float64) float64 {
	if math.IsNaN(share) {
		return nought
	}
	entries := len(disc.Profile)
	switch entries {
	case nought:
		return unity
	case unity:
		return disc.Profile[firstEntry]
	}
	last := entries - unity
	place := math.Max(nought, math.Min(unity, share)) * float64(last)
	index := int(math.Floor(place))
	if index >= last {
		return disc.Profile[last]
	}
	fraction := place - float64(index)
	return disc.Profile[index]*(unity-fraction) +
		disc.Profile[index+unity]*fraction
}
