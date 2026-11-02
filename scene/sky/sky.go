// Package sky describes light arriving from infinitely far away, indexed by
// direction. A sky has no intersectable geometry. A ray that misses the scene
// samples the sky, or returns black when the world has none.
package sky

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Sky is light arriving from infinitely far away. Light is what arrives
// from the direction toward, in the world's space.
type Sky interface {
	Light(toward maths.Vec) maths.Light
}

// Gradient is a sky that is one colour at the horizon and another straight
// up, blending between them by height above the horizon. Below the horizon
// it is Ground, for the rays that can reach it past the edge of the world.
//
// its horizon is level with the eye, which is true of a flat world only.
// on a sphere the true horizon dips below level with height, 10 degrees
// at 100 km on the earth: a Gradient there draws a painted horizon.
type Gradient struct {
	Horizon, Zenith, Ground maths.Light
	// Up is where straight up is; zero means the world's own up.
	Up maths.Vec
}

// Light is the light arriving from the direction toward.
func (g Gradient) Light(toward maths.Vec) maths.Light {
	up := g.Up
	if up == (maths.Vec{}) {
		up = maths.V(0, 1, 0)
	}
	if !measurable(toward) || !measurable(up) {
		return g.unmeasured(toward, up)
	}
	// below is as it was, line for line: changed at all, the compiler
	// joins its multiplies and adds otherwise, and a tenth of the sky
	// moves a last bit
	h := toward.Unit().Dot(up.Unit())
	if h < 0 {
		return g.Ground
	}
	return g.Horizon.Scale(1 - h).Add(g.Zenith.Scale(h))
}

// unmeasured is Light for a way or an up too long or too short for Unit,
// or pointing nowhere: measured at its own size, or no number.
func (g Gradient) unmeasured(toward, up maths.Vec) maths.Light {
	h := direction(toward).Dot(direction(up))
	if h < 0 {
		return g.Ground
	}
	return g.Horizon.Scale(1 - h).Add(g.Zenith.Scale(h))
}

// direction is v one unit long. a v too long or too short for Unit to measure
// is measured at its own size first; one that points nowhere, zero, endless or
// no number, is no number, so its light is too and the fault shows, rather than
// passing as the horizon.
func direction(v maths.Vec) maths.Vec {
	if measurable(v) {
		return v.Unit()
	}
	s := math.Max(math.Abs(v.X), math.Max(math.Abs(v.Y), math.Abs(v.Z)))
	if s == 0 || math.IsInf(s, 0) || math.IsNaN(s) {
		nan := math.NaN()
		return maths.V(nan, nan, nan)
	}
	return maths.V(v.X/s, v.Y/s, v.Z/s).Unit()
}

// measurable is whether Unit measures v truly, by its largest part: a
// length is no less than that and no more than root three of it, so a
// largest part from 1e-150 to 5e149 keeps it where Unit is true.
//
// it does not ask v.Len(): shared with Unit, the compiler would join
// Unit's multiplies and adds otherwise, and the sky move a last bit.
func measurable(v maths.Vec) bool {
	s := math.Max(math.Abs(v.X), math.Max(math.Abs(v.Y), math.Abs(v.Z)))
	return s > 1e-150 && s < 5e149
}
