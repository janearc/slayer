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

// The numbers an orbit is worked in, each true by definition.
const (
	circuitNought = 0                   // defined: no amount
	circuitUnity  = 1                   // defined: one whole
	circuitTurn   = 2 * math.Pi         // defined: radians in one full turn
	circuitDay    = 86400               // defined: seconds in a day
	circuitYear   = 365.25 * circuitDay // defined: seconds in a julian year
)

// circuitAlong is how near along the orbit's north the first axis of the
// plane may lie before the plane takes z for it instead: the dot product
// of the two.
var circuitAlong = pick("orbit-along-limit")

// circuitUnseen is the scale of a growing thing before it is born.
var circuitUnseen = pick("orbit-unseen-scale")

// Period is how long a circular orbit takes round a mass, by Kepler's
// third law: the mass in kilograms and the orbit's radius in metres, the
// time it takes to go round once. A moon's clock is its parent's mass;
// its own is left out, so the moon's month is 27.45 days, not 27.32.
//
// an orbit longer than a Duration holds, 292 years, or of no number, is
// refused by name: Sedna's 11,400 years would otherwise be 292.
func Period(mass, radius float64) time.Duration {
	if mass <= circuitNought || radius <= circuitNought {
		return circuitNought
	}
	seconds := circuitTurn * math.Sqrt(radius*radius*radius/(G*mass))
	if !(seconds < longest) {
		tooLong(seconds)
	}
	return time.Duration(seconds * float64(time.Second))
}

// longest is the most seconds a Duration holds, a little under; defined by
// the largest count of nanoseconds a Duration is.
const longest = float64(math.MaxInt64/int64(time.Second)) - circuitUnity

// tooLong refuses an orbit by name.
func tooLong(seconds float64) {
	panic(fmt.Sprintf("thing: an orbit of %v years is past the 292 a "+
		"Duration holds, or is no number", seconds/circuitYear))
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

// circuitUpward is the axis, or up when none was given: the zero vector.
func circuitUpward(axis maths.Vec) maths.Vec {
	var none maths.Vec
	if axis == none {
		return maths.Up
	}
	return axis
}

// circuitSized is a subspace at the origin, unturned, of a scale.
func circuitSized(scale float64) maths.Subspace {
	var origin maths.Vec
	var orientation maths.Orientation
	return maths.NewSubspace(origin, orientation, scale)
}

// north is the orbit's axis, one unit long, up when none was given.
func (orbit Orbit) north() maths.Vec {
	return circuitUpward(orbit.Axis).Unit()
}

// axes is the orbit's plane: two axes square to its north.
func (orbit Orbit) axes() (maths.Vec, maths.Vec) {
	north := orbit.north()
	first := maths.XAxis
	if math.Abs(north.Dot(first)) > circuitAlong {
		first = maths.ZAxis
	}
	shift := north.Scale(north.Dot(first))
	first = first.Sub(shift).Unit()
	return first, north.Cross(first)
}

// Angle is how far round the orbit is at elapsed, in radians.
func (orbit Orbit) Angle(elapsed time.Duration) float64 {
	if orbit.Period <= circuitNought {
		return orbit.Phase
	}
	return orbit.Phase + circuitTurn*float64(elapsed)/float64(orbit.Period)
}

// At is where the orbiting thing's subspace is at elapsed.
func (orbit Orbit) At(elapsed time.Duration) maths.Subspace {
	first, second := orbit.axes()
	angle := orbit.Angle(elapsed)
	across := first.Scale(orbit.Radius * math.Cos(angle))
	along := second.Scale(orbit.Radius * math.Sin(angle))
	at := orbit.Centre.Add(across).Add(along)
	facing := orbit.Facing
	if orbit.Locked {
		turn := maths.Rotation(orbit.north(), angle-orbit.Phase)
		facing = turn.Then(orbit.Facing)
	}
	return maths.NewSubspace(at, facing, orbit.Scale)
}

// Nested is a motion inside another: a subspace carried along by its
// parent's motion, so a plume on a moon goes round with the moon and
// an eye on the moon looks up from where it stood.
type Nested struct {
	Of Motion
	In maths.Subspace
}

// At is the child's subspace in the world at elapsed: inside its parent's.
func (nested Nested) At(elapsed time.Duration) maths.Subspace {
	if nested.Of == nil {
		return nested.In
	}
	return nested.In.Within(nested.Of.At(elapsed))
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

// At is the subspace turned as far as elapsed takes it.
func (spin Spin) At(elapsed time.Duration) maths.Subspace {
	angle := spin.Phase
	if spin.Period != circuitNought {
		angle += circuitTurn * float64(elapsed) / float64(spin.Period)
	}
	axis := circuitUpward(spin.Axis)
	turned := maths.Rotation(axis, angle).Then(spin.In.Orientation)
	return maths.NewSubspace(spin.In.Origin, turned, spin.In.Scale)
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

// At is the subspace at elapsed, grown as far as it takes it, in its
// parent's.
func (growing Growing) At(elapsed time.Duration) maths.Subspace {
	scale := growing.Rate * (elapsed - growing.From).Seconds()
	if scale <= circuitNought {
		scale = circuitUnseen
	}
	sized := circuitSized(scale)
	if growing.Of == nil {
		return sized
	}
	return sized.Within(growing.Of.At(elapsed))
}
