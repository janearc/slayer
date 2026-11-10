package medium

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// Disc is a flat ring of something round its own y axis: the rings of a
// giant, or, glowing, the disc round a hole. It runs from Inner to Outer
// and is Thick across, and how dense it is varies with the radius by a
// Profile, which is where the gaps are, and by ringlets of noise.
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
func (d Disc) Bounds() maths.Box {
	r, h := d.Outer, d.Thick/2
	return maths.Box{Min: maths.V(-r, -h, -r), Max: maths.V(r, h, r)}
}

// Span is where a ray crosses the slab the disc lies in, clipped to the
// cylinder of its outer radius.
func (d Disc) Span(r maths.Ray, near, far float64) (float64, float64, bool) {
	h := d.Thick / 2
	enter, leave := near, far
	if r.Dir.Y == 0 {
		if math.Abs(r.At.Y) > h {
			return 0, 0, false
		}
	} else {
		t0, t1 := (-h-r.At.Y)/r.Dir.Y, (h-r.At.Y)/r.Dir.Y
		if t0 > t1 {
			t0, t1 = t1, t0
		}
		enter, leave = math.Max(enter, t0), math.Min(leave, t1)
	}
	// the cylinder: the ray's x and z against the outer radius
	// Keep the recorded arm64 rounding order explicit, including when
	// coverage counters change the compiler's product selection.
	a := math.FMA(r.Dir.X, r.Dir.X, r.Dir.Z*r.Dir.Z)
	if a > 0 {
		b := math.FMA(r.At.X, r.Dir.X, r.At.Z*r.Dir.Z) / a
		squared := math.FMA(r.At.Z, r.At.Z, r.At.X*r.At.X)
		c := math.FMA(-d.Outer, d.Outer, squared) / a
		disc := math.FMA(b, b, -c)
		if disc < 0 {
			return 0, 0, false
		}
		root := math.Sqrt(disc)
		enter = math.Max(enter, -b-root)
		leave = math.Min(leave, -b+root)
	} else if r.At.X*r.At.X+r.At.Z*r.At.Z > d.Outer*d.Outer {
		return 0, 0, false
	}
	return enter, leave, leave > enter
}

// Density is the profile's density at the point's radius, thinned by the
// ringlets, soft at the disc's two faces, and zero outside it, and at a point
// that is not a number.
func (d Disc) Density(p maths.Vec) float64 {
	h := d.Thick / 2
	if h <= 0 || d.Outer <= d.Inner || math.IsNaN(p.X+p.Y+p.Z) {
		return 0
	}
	r := math.Hypot(p.X, p.Z)
	if r < d.Inner || r > d.Outer || math.Abs(p.Y) > h {
		return 0
	}
	t := (r - d.Inner) / (d.Outer - d.Inner)
	dens := d.profile(t)
	if d.Dense > 0 {
		dens *= d.Dense
	}
	if d.Grain > 0 {
		n := d.Ringlets
		if n <= 0 {
			n = 60
		}
		ring := noise.Fractal(d.Seed, t*n, 0.37, 0.61, 3)
		dens *= 1 - math.Max(0, math.Min(1, d.Grain))*(0.5+0.5*ring)
	}
	// soft at each face, over the outer twentieth of the thickness
	return dens * maths.Smooth(0, 0.1*h, h-math.Abs(p.Y))
}

// profile is the density at t from zero at Inner to one at Outer. a t that is
// not a number, from a point that is none, is no density.
func (d Disc) profile(t float64) float64 {
	if math.IsNaN(t) {
		return 0
	}
	n := len(d.Profile)
	switch n {
	case 0:
		return 1
	case 1:
		return d.Profile[0]
	}
	x := math.Max(0, math.Min(1, t)) * float64(n-1)
	i := int(math.Floor(x))
	if i >= n-1 {
		return d.Profile[n-1]
	}
	f := x - float64(i)
	return d.Profile[i]*(1-f) + d.Profile[i+1]*f
}
