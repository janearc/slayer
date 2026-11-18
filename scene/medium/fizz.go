package medium

import (
	"math"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// Fizz is the haze of microbubbles over something dissolving: a column
// rising from its foot at the origin, thick where it is born, thinning
// and spreading as it climbs, ragged with the churn of the water. The
// bubbles big enough to see are not here; they are small spheres of a
// clear material, placed by the scene.
//
// This is only the mist of the ones too small to resolve, which
// scatters light the way a whitecap's foam does.
type Fizz struct {
	// Radius is the column's at its foot, and Height how high it
	// rises; by its top the column has spread to twice its radius.
	Radius, Height float64
	// Thick is the density at the middle of its foot.
	Thick float64
	// Rise is how fast the bubbles climb, in units a second. The
	// churn climbs with them.
	Rise float64
	// Churn is how much of the density the water's churn may take
	// away, nought to one, and Seed picks the churn.
	Churn float64
	Seed  uint64
	// risen is how far the pattern has climbed: set by Tick.
	risen float64
}

// spread is how many times its foot's radius the column holds at its
// top. It's picked: a glass's fizz widens but does not bloom.
const spread = 2

// Bounds is the box round the column at its widest.
func (f Fizz) Bounds() maths.Box {
	wide := f.Radius * spread
	return maths.Box{Min: maths.V(-wide, 0, -wide),
		Max: maths.V(wide, f.Height, wide)}
}

// Density is Thick at the middle of the foot, falling smoothly to
// nothing at the column's side and its top, the side widening as it
// climbs, thinned by the churn, which rises with the bubbles.
func (f Fizz) Density(at maths.Vec) float64 {
	if f.Radius <= 0 || f.Height <= 0 || f.Thick <= 0 {
		return 0
	}
	up := at.Y / f.Height
	if up < 0 || up > 1 {
		return 0
	}
	wide := f.Radius * (1 + (spread-1)*up)
	out := math.Hypot(at.X, at.Z) / wide
	if out > 1 {
		return 0
	}
	// thick at the foot, gone at the top; thick in the middle, gone
	// at the side
	density := f.Thick * (1 - maths.Smooth(0, 1, up)) *
		(1 - maths.Smooth(0, 1, out))
	if f.Churn <= 0 {
		return density
	}
	churned := at
	churned.Y -= f.risen
	scale := f.Radius / 2
	n := noise.Fractal(f.Seed, churned.X/scale, churned.Y/scale,
		churned.Z/scale, 4)
	thin := math.Min(f.Churn, 1)
	return density * (1 - thin*(0.5+0.5*n))
}

// Tick is the fizz as it is at a time: the churn risen with the
// bubbles, the column itself standing still.
func (f Fizz) Tick(at time.Duration) Medium {
	f.risen = f.Rise * at.Seconds()
	return f
}

// AdvectionDistance is the evaluated churn displacement along local +Y.
// Placement may rotate that local axis into an authored downstream flow.
func (f Fizz) AdvectionDistance() float64 { return f.risen }

// ExpansionFactor is the existing chosen top-to-foot radius ratio.
func (Fizz) ExpansionFactor() float64 { return spread }
