package rings

import (
	"fmt"
	"math"

	kerr "github.com/janearc/celestial/hole"
)

// Lens is a hole that does not spin, of mass one, and an eye held still
// beside it: at a distance from its centre in masses, leaning above the
// disc's plane by an angle. The disc's axis is the hole's z, and the eye
// is in the x z plane, so every ray it sends toward the hole's axis
// stays there.
type Lens struct {
	hole     kerr.Hole
	distance float64
	lean     float64
	eye      kerr.Vec3
	frame    kerr.Frame
}

// NewLens is an eye at a distance from a still hole of mass one,
// leaning above the disc's plane by lean radians. It refuses a distance
// that is not a number or is no further out than the photon sphere,
// three masses, from where no eye sees a shadow's edge to follow; and a
// lean that is not a number or reaches the axis.
func NewLens(distance, lean float64) (Lens, error) {
	if !(distance > photonSphere) || math.IsInf(distance, 0) {
		return Lens{}, fmt.Errorf("rings: an eye %v masses out is not "+
			"outside the photon sphere: give a finite distance past %v",
			distance, photonSphere)
	}
	if !(math.Abs(lean) < halfTurn/two) {
		return Lens{}, fmt.Errorf("rings: a lean of %v radians does "+
			"not leave the axis: give one between minus and plus a "+
			"quarter turn", lean)
	}
	hole, err := kerr.NewHole(stillMass, stillSpin)
	if err != nil {
		return Lens{}, fmt.Errorf("rings: %w", err)
	}
	eye := kerr.NewVec3(distance*math.Cos(lean), none,
		distance*math.Sin(lean))
	return Lens{hole: *hole, distance: distance, lean: lean, eye: eye,
		frame: hole.FrameOf(eye, hole.Static(eye))}, nil
}

// photonSphere is the radius, in masses, at which light goes round a
// still hole: three, defined by the schwarzschild metric.
const photonSphere = 3

// Distance is the eye's distance from the hole's centre, in masses.
func (lens Lens) Distance() float64 { return lens.distance }

// Lean is the eye's height above the disc's plane, as an angle.
func (lens Lens) Lean() float64 { return lens.lean }

// heading is the way the eye looks at an angle off the hole's centre,
// in the plane of the eye and the disc's axis: toward the centre, turned
// up toward the axis by the angle. It is in the hole's own axes, which
// the eye's frame takes as its own, as the tracer hands them.
func (lens Lens) heading(angle float64) kerr.Vec3 {
	toward := [3]float64{-math.Cos(lens.lean), none, -math.Sin(lens.lean)}
	up := [3]float64{-math.Sin(lens.lean), none, math.Cos(lens.lean)}
	cos, sin := math.Cos(angle), math.Sin(angle)
	return kerr.NewVec3(cos*toward[0]+sin*up[0], none,
		cos*toward[2]+sin*up[2])
}

// Follow is the ray the eye sees at an angle off the hole's centre,
// positive above it: marched back from the eye until it leaves past the
// eye, comes from the hole, or runs out of steps.
func (lens Lens) Follow(angle float64) Path {
	ray := lens.hole.Arriving(lens.eye, lens.frame, lens.heading(angle))
	walk := newWalk(lens, ray)
	for i := 0; i < steps && !walk.done; i++ {
		walk.step()
	}
	return walk.path()
}

// Impact is the impact parameter, in masses, of the ray the eye sees at
// an angle: its angular momentum over its energy, as celestial says.
func (lens Lens) Impact(angle float64) float64 {
	ray := lens.hole.Arriving(lens.eye, lens.frame, lens.heading(angle))
	return lens.hole.ImpactParameter(ray)
}

// Shadow is the impact parameter, in masses, at which a ray is just
// captured: root twenty-seven, as celestial says.
func (lens Lens) Shadow() float64 { return lens.hole.Shadow() }

// Edge is the angle off the hole's centre, above it, at which the eye
// sees the shadow's edge: the least angle whose ray escapes, halved
// toward from a captured ray at the centre and an escaping one far out.
func (lens Lens) Edge() float64 {
	in, out := float64(none), halfTurn/two
	for range halvings {
		mid := (in + out) / two
		if lens.Follow(mid).Captured() {
			in = mid
			continue
		}
		out = mid
	}
	return out
}
