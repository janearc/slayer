package medium

import (
	"math"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// Jet is a pair of jets, one up y and one down, from the origin out to Length:
// a cone each, Base across at the origin and spreading by Spread of the
// distance out, Thick at the heart of the base and thinning to zero at the end,
// gaussian across so the edges are soft.
//
// From is where each starts, out from the origin: a hole's jets are launched
// from the disc, not from the shadow.
//
// Knots is a dressing: the jet bunched along its length into bright knots by
// noise, as m87's is in the pictures, zero smooth and one all knots; Seed makes
// a jet's own.
type Jet struct {
	Length, Base, Spread float64
	Thick                float64
	From                 float64
	Knots                float64
	Seed                 uint64
	// Flow is how fast the knots move out along the jet, in base widths a
	// second, and Time when the jet is seen, in seconds: zero for knots
	// that stand still.
	Flow, Time float64
}

// Bounds is the box both cones fit in.
func (j Jet) Bounds() maths.Box {
	r := j.Base/2 + j.Length*j.Spread
	return maths.Box{Min: maths.V(-r, -j.Length, -r),
		Max: maths.V(r, j.Length, r)}
}

// Span is where a ray is in the jets: the slab and the cylinder they
// fit in, then probed in from both ends to where there is gas.
func (j Jet) Span(r maths.Ray, near, far float64) (float64, float64, bool) {
	enter, leave, ok := slab(j.Length, r, near, far)
	if !ok {
		return 0, 0, false
	}
	wide := j.Base/2 + j.Length*j.Spread
	enter, leave, ok = cylinder(wide, r, enter, leave)
	if !ok {
		return 0, 0, false
	}
	// probing no coarser than the narrowest the jet gets, so a ray
	// across the base does not step over it
	finest := j.Base / 2
	if finest <= 0 {
		finest = j.Length * j.Spread / 8
	}
	return probed(j, r, enter, leave, finest)
}

// Density is the cone's, thinning outward and soft across.
func (j Jet) Density(p maths.Vec) float64 {
	if j.Length <= 0 {
		return 0
	}
	y := math.Abs(p.Y)
	if y >= j.Length || y < j.From {
		return 0
	}
	w := j.Base/2 + y*j.Spread
	if w <= 0 {
		return 0
	}
	d := math.Hypot(p.X, p.Z) / w
	if d > 2.5 {
		// past two and a half widths there is nothing, so a span
		// can find the edge
		return 0
	}
	density := j.Thick * (1 - y/j.Length) * math.Exp(-2*d*d)
	if j.Knots > 0 {
		// knots a few base widths apart along the jet, the same up
		// and down a side, each side its own
		side := 0.0
		if p.Y < 0 {
			side = 7
		}
		along := (y - j.Flow*j.Time*j.Base) / (3 * j.Base)
		n := noise.Fractal(j.Seed, side, along, 0, 2)
		density *= math.Max(0, 1+j.Knots*2*n)
	}
	return density
}

// slab is where a ray is between y of minus and plus half, within near
// and far.
func slab(half float64, r maths.Ray, near, far float64) (float64, float64,
	bool) {
	if r.Dir.Y == 0 {
		if math.Abs(r.At.Y) > half {
			return 0, 0, false
		}
		return near, far, near < far
	}
	a, b := (-half-r.At.Y)/r.Dir.Y, (half-r.At.Y)/r.Dir.Y
	if a > b {
		a, b = b, a
	}
	enter, leave := math.Max(near, a), math.Min(far, b)
	return enter, leave, enter < leave
}

// cylinder is where a ray is inside a cylinder of a radius round y,
// within near and far.
func cylinder(radius float64, r maths.Ray, near, far float64) (float64,
	float64, bool) {
	a := r.Dir.X*r.Dir.X + r.Dir.Z*r.Dir.Z
	b := 2 * (r.At.X*r.Dir.X + r.At.Z*r.Dir.Z)
	c := r.At.X*r.At.X + r.At.Z*r.At.Z - radius*radius
	if a < 1e-12 {
		if c > 0 {
			return 0, 0, false
		}
		return near, far, near < far
	}
	disc := b*b - 4*a*c
	if disc < 0 {
		return 0, 0, false
	}
	s := math.Sqrt(disc)
	t1, t2 := (-b-s)/(2*a), (-b+s)/(2*a)
	enter, leave := math.Max(near, t1), math.Min(far, t2)
	return enter, leave, enter < leave
}

// probed is a span pulled in from both ends to where the medium has
// any density at all, in steps no coarser than finest and at least
// forty-eight of them, so a march starts on the gas and not on the box
// round it; a ray that finds none has no span.
func probed(m Medium, r maths.Ray, enter, leave, finest float64) (float64,
	float64, bool) {
	probes := 48
	if finest > 0 {
		steps := int(math.Ceil((leave - enter) / finest))
		probes = max(probes, min(2048, steps))
	}
	step := (leave - enter) / float64(probes)
	first, last := -1, -1
	for i := 0; i < probes; i++ {
		if m.Density(r.Point(enter+(float64(i)+0.5)*step)) > 0 {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		return 0, 0, false
	}
	return enter + float64(first)*step, enter + float64(last+1)*step, true
}

// Tick is the jet as it is at a time: the same jet, its knots moved
// along.
func (j Jet) Tick(at time.Duration) Medium {
	j.Time = at.Seconds()
	return j
}
