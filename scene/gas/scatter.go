package gas

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// Through is how much of a band's light reaches a point from a lamp
// along way, a unit vector toward the lamp, far away: the tracer's
// march, handed in so this package owns the optics and the tracer the
// world.
//
// A sun's far is +Inf, and means from wherever the gas ends, which only
// the tracer knows.
type Through func(at, way maths.Vec, far float64, band maths.Band) float64

// Single is the light one step of a ray gathers at a point in a gas by
// single scattering: for each lamp, what reaches the point through the
// gas, times what the gas sends toward the eye of it by its phase at
// the lamp's angle to the eye, times the density, per unit of step.
//
// What survives from the point to the eye is the ray's own
// transmittance, which the tracer carries.
//
// A lamp's far is where it stands along its toward; a sun's is +Inf and
// is handed to Through as it is, since where the gas ends is the
// tracer's to know. A path of a billion metres stood in for that once,
// and through rayleigh's air that is exp(-11600) of the light: no sun.
func Single(g Gas, density float64, band maths.Band, lamps []thing.Lamp,
	p, toEye maths.Vec, through Through) maths.Light {
	var sum maths.Light
	if density <= 0 {
		return sum
	}
	eye := toEye.Unit()
	for _, l := range lamps {
		toward, far, light := l.Light(p, eye)
		if light == (maths.Light{}) {
			continue
		}
		way := toward.Unit()
		reach := through(p, way, far, band)
		// written so that a reach of no number, never through, is
		// passed over as none is
		if !(reach > 0) {
			continue
		}
		// the angle scattered through: from the lamp's way of travel,
		// which is minus toward, to the eye's
		cos := way.Neg().Dot(eye)
		lit := light.Scale(reach * density)
		sum = sum.Add(g.Inscatter(band, lit, cos))
	}
	return sum
}

// Uniform is a Through for a gas filling all space at one density, by beer's
// law along the straight line: the test's. A gas with no edge hides a sun,
// since the path to it is infinite; a test that wants a sun bounds the gas
// instead.
//
// a band the gas does not take, or a density of zero, passes all, a sun's
// endless path too, not zero times it.
func Uniform(g Gas, density float64) Through {
	return func(_, _ maths.Vec, far float64, band maths.Band) float64 {
		if e := g.Extinction(band); e == 0 || density == 0 {
			return 1
		}
		return math.Exp(-g.Extinction(band) * density * far)
	}
}
