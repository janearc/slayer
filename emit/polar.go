package emit

import (
	"math"
	"time"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/observer"
)

// Polar is two beams out of a point, a magnetar's or a pulsar's: each
// pole its own direction, the second the first's opposite turned by
// Offset, since real poles are not always opposed.
//
// Width is how far off its pole a beam is at full strength, in
// radians, and Soft how much further it fades to nothing. Light is the
// spectrum along a pole at one unit of distance. Beyond is where the
// light starts, the star's surface.
type Polar struct {
	At          maths.Vec
	Light       bands.Light
	Beyond      float64
	Width, Soft float64
	Offset      float64
	// Tilt is how far the poles sit off the spin axis, and Spin how long
	// a turn about it takes; the spin axis leans Cone off Axis and goes
	// round it once a Precess. Noughts hold still.
	Tilt    float64
	Spin    time.Duration
	Axis    maths.Vec
	Cone    float64
	Precess time.Duration
}

// Poles are where the two beams point at a time. The spin axis wanders
// round a cone about Axis, once a Precess; the magnetic axis sits Tilt
// off the spin axis and goes round it once a Spin, a lighthouse; the
// second pole is the first's opposite turned by Offset.
func (polar Polar) Poles(at time.Duration) (maths.Vec, maths.Vec) {
	axis := polar.Axis.Unit()
	spin := tilted(axis, polar.Cone, share(at, polar.Precess))
	north := tilted(spin, polar.Tilt, share(at, polar.Spin))
	south := tilted(north.Neg(), polar.Offset, share(at, polar.Spin)+0.25)
	return north, south
}

// share is how far round a turn of a period a time is, as a fraction; a period
// of zero never turns
func share(at, period time.Duration) float64 {
	if period <= 0 {
		return 0
	}
	return float64(at) / float64(period)
}

// tilted is a direction leaned off an axis by an angle, toward a side that goes
// round the axis as turn goes from zero to one
func tilted(axis maths.Vec, angle, turn float64) maths.Vec {
	reference := maths.V(0, 1, 0)
	if math.Abs(axis.Y) > 0.9 {
		reference = maths.V(1, 0, 0)
	}
	across := axis.Cross(reference).Unit()
	side := maths.About(axis, 2*math.Pi*turn).Out(across)
	return axis.Scale(math.Cos(angle)).Add(side.Scale(math.Sin(angle)))
}

// Beams is the polar emitter as a thing.Lamp, read at whatever time
// Clock holds when it is asked: a thing.Lamp has no time of its own, so
// whoever draws a frame sets the clock first. it folds back into the lamp
// (film.Emitter, in ninjafilm) when lamps are given the picture's time.
func (polar Polar) Beams(clock *time.Duration) Beams {
	red, green, blue := srgb.Light(observer.Swatch(polar.Light))
	return Beams{Polar: polar, Clock: clock,
		strength: maths.RGB(red, green, blue)}
}

// Beams is a polar emitter on a clock, as a lamp
type Beams struct {
	Polar
	Clock    *time.Duration
	strength maths.Light
}

// Light is what reaches a point from the beams at the clock's time: the
// way toward the star, how far past its surface, and the light, by the
// inverse square and by how near the point is to either pole.
func (beams Beams) Light(at, _ maths.Vec) (maths.Vec, float64,
	maths.Light) {
	toward := beams.At.Sub(at)
	distance := toward.Len()
	if distance <= beams.Beyond {
		return maths.Vec{}, 0, maths.Light{}
	}
	out := toward.Scale(-1 / distance)
	north, south := beams.Poles(*beams.Clock)
	share := beams.along(out, north) + beams.along(out, south)
	falling := share / (distance * distance)
	return toward.Scale(1 / distance), distance - beams.Beyond,
		beams.strength.Scale(falling)
}

// along is how much of a beam goes out a way: all of it within Width of
// the pole, fading to none by Width and Soft further
func (beams Beams) along(out, pole maths.Vec) float64 {
	off := math.Acos(math.Max(-1, math.Min(1, out.Dot(pole))))
	return 1 - maths.Smooth(beams.Width, beams.Width+beams.Soft, off)
}
