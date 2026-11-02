package film

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Laser is a lamp whose light is a narrow beam: from At along Aim, as
// wide as Width, with a soft edge a gaussian makes: Strength at its
// middle, e^-2 of it, about a seventh, a whole Width out, and none past
// three.
//
// Lit things show its spot; in a scattering gas the beam itself shows,
// step by step, which is the tracer's to gather. The beam is a bound
// volume, Bounds, so a tracer may skip the empty space round it.
type Laser struct {
	At       maths.Vec
	Aim      maths.Vec
	Width    float64
	Strength maths.Light
	// Length is how far the beam reaches, for its bounds; zero is a
	// thousand units.
	Length float64
}

// axis is the beam's unit direction.
func (l Laser) axis() maths.Vec {
	if l.Aim == (maths.Vec{}) {
		return maths.V(0, 0, -1)
	}
	return l.Aim.Unit()
}

// length is the beam's reach.
func (l Laser) length() float64 {
	if l.Length <= 0 {
		return 1000
	}
	return l.Length
}

// Light is the laser's light at a point: the beam's strength falling
// off across its width by a gaussian whose sigma is half the width,
// nothing behind the source or past its reach; toward is the way back
// to the source along the beam, and far how far that is.
func (l Laser) Light(p, _ maths.Vec) (maths.Vec, float64, maths.Light) {
	axis := l.axis()
	rel := p.Sub(l.At)
	along := rel.Dot(axis)
	if along <= 0 || along > l.length() {
		return axis.Neg(), along, maths.Light{}
	}
	off := rel.Sub(axis.Scale(along)).Len()
	if l.Width <= 0 || off > 3*l.Width {
		return axis.Neg(), along, maths.Light{}
	}
	sigma := l.Width / 2
	fall := math.Exp(-off * off / (2 * sigma * sigma))
	return axis.Neg(), along, l.Strength.Scale(fall)
}

// Bounds is the box the beam fills, three widths about its axis out to
// its reach.
func (l Laser) Bounds() maths.Box {
	axis := l.axis()
	end := l.At.Add(axis.Scale(l.length()))
	pad := 3 * l.Width
	lo := maths.V(math.Min(l.At.X, end.X)-pad,
		math.Min(l.At.Y, end.Y)-pad, math.Min(l.At.Z, end.Z)-pad)
	hi := maths.V(math.Max(l.At.X, end.X)+pad,
		math.Max(l.At.Y, end.Y)+pad, math.Max(l.At.Z, end.Z)+pad)
	return maths.Box{Min: lo, Max: hi}
}
