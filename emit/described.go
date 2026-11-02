package emit

import (
	"errors"
	"math"
	"time"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// Described is a light somebody else made, as emit reads one: what its light is
// made of, how strong it is along its beam in watts per steradian, the half
// angle of its beam and the soft edge past it, and how much of itself it is at
// a time. the manufactory's lights answer this.
//
// it makes the description and never calls emit; emit reads it here.
type Described interface {
	Bands() bands.Light
	Strength() float64
	Angles() (half, soft measure.Angle)
	Share(at time.Duration) float64
}

var errDescribed = errors.New("emit: an emitter is made of a " +
	"description with some light and some strength, a way to shine, " +
	"and an exposure above nought")

// summed is all the power of a spectrum, band by band.
func summed(light bands.Light) float64 {
	first, last := light.Span()
	sum := 0.0
	for band := first; band <= last; band++ {
		sum += light.Power(band)
	}
	return sum
}

// Of is a point emitter made of a description, at a place, shining a way.
//
// the description's spectrum says what the light is made of and not how much of
// it there is, so it is scaled until, added up over every band, it is the
// description's strength times the exposure: the exposure is what a watt a
// steradian is in a picture's light, and is the picture's to choose.
//
// a beam half a turn wide is every way, and has no width.
func Of(lamp Described, at, toward maths.Vec,
	exposure float64) (Point, error) {
	if lamp == nil || !(exposure > 0) || math.IsInf(exposure, 0) ||
		!(toward.Len() > 0) {
		return Point{}, errDescribed
	}
	light, strength := lamp.Bands(), lamp.Strength()
	all := summed(light)
	if !(all > 0) || !(strength > 0) {
		return Point{}, errDescribed
	}
	half, soft := lamp.Angles()
	width := half.InRadians()
	if width >= math.Pi {
		width = 0
	}
	return Point{At: at, Light: light.Scale(strength * exposure / all),
		Pulse: lamp.Share, Toward: toward.Unit(), Width: width,
		Soft: soft.InRadians()}, nil
}
