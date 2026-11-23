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

// Bounds is the box round the column at its widest: chosen.fizzSpread times
// the foot's radius.
func (fizz Fizz) Bounds() maths.Box {
	wide := fizz.Radius * chosen.fizzSpread
	return corners(maths.V(-wide, nought, -wide),
		maths.V(wide, fizz.Height, wide))
}

// Density is Thick at the middle of the foot, falling smoothly to
// nothing at the column's side and its top, the side widening as it
// climbs, thinned by the churn, which rises with the bubbles. The column is
// cut flat at its foot, where it is as dense as it is anywhere.
func (fizz Fizz) Density(point maths.Vec) float64 {
	if fizz.Radius <= nought || fizz.Height <= nought ||
		fizz.Thick <= nought {
		return nought
	}
	climb := point.Y / fizz.Height
	if climb < nought || climb > unity {
		return nought
	}
	wide := fizz.Radius * (unity + (chosen.fizzSpread-unity)*climb)
	across := math.Hypot(point.X, point.Z) / wide
	if across > unity {
		return nought
	}
	// thick at the foot, gone at the top; thick in the middle, gone
	// at the side
	density := fizz.Thick * (unity - maths.Smooth(nought, unity, climb)) *
		(unity - maths.Smooth(nought, unity, across))
	if fizz.Churn <= nought {
		return density
	}
	churned := point
	churned.Y -= fizz.risen
	scale := fizz.Radius / chosen.fizzGrain
	grain := noise.Fractal(fizz.Seed, churned.X/scale, churned.Y/scale,
		churned.Z/scale, chosen.fizzOctaves)
	thin := math.Min(fizz.Churn, unity)
	return density * (unity - thin*(halfway+halfway*grain))
}

// Tick is the fizz as it is at a time: the churn risen with the
// bubbles, the column itself standing still.
func (fizz Fizz) Tick(at time.Duration) Medium {
	fizz.risen = fizz.Rise * at.Seconds()
	return fizz
}

// AdvectionDistance is the evaluated churn displacement along local +Y.
// Placement may rotate that local axis into an authored downstream flow.
func (fizz Fizz) AdvectionDistance() float64 { return fizz.risen }

// ExpansionFactor is the chosen ratio of the column's radius at the top to
// its radius at the foot.
func (Fizz) ExpansionFactor() float64 { return chosen.fizzSpread }
