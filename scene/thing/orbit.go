package thing

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
)

// G is the gravitational constant, in metres cubed a kilogram a second
// squared: the one number gravity needs.
const G = 6.67430e-11

// Period is how long a circular orbit takes round a mass, by Kepler's
// third law: the mass in kilograms and the orbit's radius in metres, the
// time it takes to go round once. A moon's clock is its parent's mass;
// its own is left out, so the moon's month is 27.45 days, not 27.32.
//
// an orbit longer than a Duration holds, 292 years, or of no number, is
// refused by name: Sedna's 11,400 years would otherwise be 292.
func Period(mass, radius float64) time.Duration {
	if mass <= 0 || radius <= 0 {
		return 0
	}
	seconds := 2 * math.Pi * math.Sqrt(radius*radius*radius/(G*mass))
	if !(seconds < longest) {
		tooLong(seconds)
	}
	return time.Duration(seconds * float64(time.Second))
}

// longest is the most seconds a Duration holds, a little under.
const longest = float64(math.MaxInt64/int64(time.Second)) - 1

// tooLong refuses an orbit by name.
func tooLong(seconds float64) {
	panic(fmt.Sprintf("thing: an orbit of %v years is past the 292 a "+
		"Duration holds, or is no number", seconds/(365.25*86400)))
}

// Orbit is a motion round a centre: a circle of Radius, in the plane
// square to Axis, once every Period, starting Phase radians round from
// the plane's first axis. It is the simplest orbit, a circle, and it is
// timed by gravity when Period comes from Period.
type Orbit struct {
	// Centre is what is orbited, in the parent's space.
	Centre maths.Vec
	// Radius is the orbit's, in the parent's units.
	Radius float64
	// Axis is the orbit's north; zero means up. Going round is
	// counterclockwise seen from the axis's end, the right-hand rule.
	Axis maths.Vec
	// Period is once round; zero or less never moves.
	Period time.Duration
	// Phase is where on the circle it starts, in radians.
	Phase float64
	// Facing is the thing's own orientation, kept as it goes round.
	Facing maths.Orientation
	// Locked turns the thing once a circuit as well, so it keeps one face
	// to the centre, as every big moon does; Facing is then its facing
	// where it starts, at Phase, not at phase zero.
	Locked bool
	// Scale is the thing's own scale, as a subspace's.
	Scale float64
}

// north is the orbit's axis, one unit long, up when none was given.
func (o Orbit) north() maths.Vec {
	n := o.Axis
	if n == (maths.Vec{}) {
		n = maths.Up
	}
	return n.Unit()
}

// axes is the orbit's plane: two axes square to its north.
func (o Orbit) axes() (maths.Vec, maths.Vec) {
	n := o.north()
	e := maths.XAxis
	if math.Abs(n.Dot(e)) > 0.9 {
		e = maths.ZAxis
	}
	e = e.Sub(n.Scale(n.Dot(e))).Unit()
	return e, n.Cross(e)
}

// Angle is how far round the orbit is at t, in radians.
func (o Orbit) Angle(t time.Duration) float64 {
	if o.Period <= 0 {
		return o.Phase
	}
	return o.Phase + 2*math.Pi*float64(t)/float64(o.Period)
}

// At is where the orbiting thing's subspace is at t.
func (o Orbit) At(t time.Duration) maths.Subspace {
	e, f := o.axes()
	a := o.Angle(t)
	at := o.Centre.Add(e.Scale(o.Radius * math.Cos(a))).
		Add(f.Scale(o.Radius * math.Sin(a)))
	facing := o.Facing
	if o.Locked {
		facing = maths.Rotation(o.north(), a-o.Phase).Then(o.Facing)
	}
	return maths.Subspace{Origin: at, Orientation: facing, Scale: o.Scale}
}

// Nested is a motion inside another: a subspace carried along by its
// parent's motion, so a plume on a moon goes round with the moon and
// an eye on the moon looks up from where it stood.
type Nested struct {
	Of Motion
	In maths.Subspace
}

// At is the child's subspace in the world at t: inside its parent's.
func (n Nested) At(t time.Duration) maths.Subspace {
	if n.Of == nil {
		return n.In
	}
	return n.In.Within(n.Of.At(t))
}

// Spin is a thing turning in place: its subspace, the axis it turns
// about in the parent's space, and how long once round takes, the
// other way round when Period is negative. Nought never turns. Phase
// is where the turn starts, in radians.
type Spin struct {
	In     maths.Subspace
	Axis   maths.Vec
	Period time.Duration
	Phase  float64
}

// At is the subspace turned as far as t takes it.
func (s Spin) At(t time.Duration) maths.Subspace {
	a := s.Phase
	if s.Period != 0 {
		a += 2 * math.Pi * float64(t) / float64(s.Period)
	}
	axis := s.Axis
	if axis == (maths.Vec{}) {
		axis = maths.Up
	}
	return maths.Subspace{Origin: s.In.Origin,
		Orientation: maths.Rotation(axis, a).Then(s.In.Orientation),
		Scale:       s.In.Scale}
}

// Growing is a subspace that grows with time, carried by a motion: a
// fireball. From is when it is born, and after that it is Rate units
// across a second, in the parent's units; before, it is as good as
// nothing, a scale too small to see.
type Growing struct {
	Of   Motion
	From time.Duration
	Rate float64
}

// At is the subspace at t, grown as far as t takes it, in its parent's.
func (g Growing) At(t time.Duration) maths.Subspace {
	scale := g.Rate * (t - g.From).Seconds()
	if scale <= 0 {
		scale = 1e-12
	}
	s := maths.Subspace{Scale: scale}
	if g.Of == nil {
		return s
	}
	return s.Within(g.Of.At(t))
}
