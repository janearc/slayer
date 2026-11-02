package film

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/data"
)

// what a laser counts in, each defined
const (
	// twice is the width of the gaussian in sigmas. It is defined by
	// the gaussian's exponent, minus the square of the offset over two
	// sigma squared.
	twice = 2
	// back is the way a beam of no aim points, down minus z. It is
	// defined as the way a sensor looks.
	back = -1
)

var (
	// cutoff is how many widths from its axis a laser's light is cut off.
	cutoff = data.Picked("laser-cutoff-widths")
	// reach is how far a laser reaches when none is given.
	reach = data.Picked("laser-reach")
)

// Laser is a lamp whose light is a narrow beam. It runs from its place
// along its aim and is as wide as its width. Its soft edge is a gaussian.
// The strength is at its middle. About a seventh of it is left a width
// out. That is e to the minus two. None is left past three.
//
// Lit things show its spot. In a scattering gas the beam itself shows,
// step by step. Gathering that is the tracer's job. The beam is a bound
// volume called Bounds. So a tracer may skip the empty space round it.
//
// Its length is how far the beam reaches, for its bounds. Nought is a
// thousand units. Zero aims are down minus z.
//
//ninja:vow practice slayer's scenes make their laser here
type Laser struct {
	at, aim  maths.Vec
	width    float64
	strength maths.Light
	length   float64
}

// NewLaser is a laser at a place and aimed. It has a width, a strength
// and a reach.
//
//ninja:vow practice slayer's scenes make their laser here
func NewLaser(at, aim maths.Vec, width float64, strength maths.Light,
	length float64) Laser {
	return Laser{at, aim, width, strength, length}
}

// axis is the beam's unit direction.
func (laser Laser) axis() maths.Vec {
	if laser.aim == (maths.Vec{}) {
		return maths.V(none, none, back)
	}
	return laser.aim.Unit()
}

// span is the beam's reach.
func (laser Laser) span() float64 {
	if laser.length <= none {
		return reach
	}
	return laser.length
}

// Light is the light of the laser at a point. It is the strength of the
// beam falling off across its width by a gaussian. The sigma is half the
// width. There is nothing behind the source or past its reach.
//
// The toward is the way back to the source along the beam. The far is
// how far that is.
func (laser Laser) Light(point, _ maths.Vec) (toward maths.Vec,
	far float64, light maths.Light) {
	axis := laser.axis()
	rel := point.Sub(laser.at)
	along := rel.Dot(axis)
	if along <= none || along > laser.span() {
		return axis.Neg(), along, maths.Light{}
	}
	off := rel.Sub(axis.Scale(along)).Len()
	if laser.width <= none || off > cutoff*laser.width {
		return axis.Neg(), along, maths.Light{}
	}
	sigma := laser.width / pair
	fall := math.Exp(-off * off / (twice * sigma * sigma))
	return axis.Neg(), along, laser.strength.Scale(fall)
}

// Bounds is the box the beam fills. It is three widths about its axis
// out to its reach.
//
//ninja:vow practice slayer's tracer skips the empty space round a beam
func (laser Laser) Bounds() maths.Box {
	axis := laser.axis()
	run := axis.Scale(laser.span())
	start, end := laser.at, laser.at.Add(run)
	pad := cutoff * laser.width
	lowX := math.Min(start.X, end.X) - pad
	lowY := math.Min(start.Y, end.Y) - pad
	lowZ := math.Min(start.Z, end.Z) - pad
	highX := math.Max(start.X, end.X) + pad
	highY := math.Max(start.Y, end.Y) + pad
	highZ := math.Max(start.Z, end.Z) + pad
	low := maths.V(lowX, lowY, lowZ)
	high := maths.V(highX, highY, highZ)
	return maths.Box{Min: low, Max: high}
}

// Lasers are emitters.
var _ Emitter = Laser{}
