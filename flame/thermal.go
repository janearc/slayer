package flame

import (
	"errors"
	"math"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/texture"
)

// Thermal is hot gas let go at once, a fireball, as it is let go: its
// Radius in metres and its centre's Height above the ground; its gas at
// Hottest in Air at Air's temperature; its soot as thick to green light
// as Soot, per metre at 550 nm, as a Fire's Absorption is.
//
// Spread is the n of Scorer's law below, zero read as his 4; Stem, the
// soot of the column it draws up behind it as a share of its own, zero
// for none; Start, when it is let go on the picture's clock; and Seed,
// for its eddies.
type Thermal struct {
	Radius, Height float64
	Hottest, Air   measure.Temperature
	Soot           float64
	Spread, Stem   float64
	Start          time.Duration
	Seed           uint64
}

// A Fireball is a thermal in space, rising, cooling and rolling, as a
// medium and its own glow, the way a flame's Body is. Its laws, cited
// in the vanguard's plans/2026-11-10-pyro-sources.md:
//
//   - Scorer (1957): a thermal's height above its virtual origin is n
//     times its radius, n about 4 (Lecoanet and Jeevanjee 2018 find
//     6.4 for a turbulent one), and it rises at w = C sqrt(g' r), C
//     about 1.2, g' the gravity its lightness feels.
//   - Its lightness times its volume holds as it takes in air, so g'
//     falls as the cube of its radius, and so does its excess heat.
//   - Together: r^2 = r0^2 + k t, with k = (2C/n) r0^(3/2) sqrt(g d),
//     d the share by which it is lighter than the air when let go,
//     1 - Air/Hottest. It rises as the root of time.
//   - Inside, it turns as Hill's spherical vortex does: on its axis the
//     gas goes up at 3/2 of the rise, about a ring of still gas r/sqrt2
//     out. So it has turned (3n/sqrt2) ln(r/r0) radians about that
//     ring: once for about every doubling in size. Its soot gathers
//     into the ring as it grows, the mushroom's cap.
//
// Its soot thins as its volume grows. All of this is Boussinesq, for
// gas a little lighter than the air; a fireball at 1,800 K is six
// times lighter, so its first moments are outside the laws, and are
// drawn by them anyway. The air does not thin with height and holds no
// layers, so the thermal never stops rising and never flattens.
//
// In its own space its centre rises along y from its Height, and a
// unit is a metre. Constructed by NewFireball.
type Fireball struct {
	thermal Thermal
	at      time.Duration
	rate    float64
	origin  float64
	n       float64
	glows   glowing
}

// Scorer's constants, gravity, and the look's own, which are picked:
// how far past its radius the fireball's box reaches, how thick the
// ring of the cap is, how many radii across its eddies are, and how
// much they take from its soot.
const (
	scorerC         = 1.2
	scorerN         = 4
	thermalGravity  = 9.80665
	thermalReach    = 1.25
	ringWidth       = 0.15
	thermalEddies   = 3
	thermalOctaves  = 5
	thermalClumping = 0.45
)

// NewFireball is the fireball a thermal makes, its glow in a picture's
// light at an exposure, as NewBody's is. The radius, the soot and the
// two temperatures must be greater than zero and finite, the gas hotter
// than the air; the height zero or more; the spread, when given, one or
// more; and the stem from zero to one.
func NewFireball(thermal Thermal, exposure float64) (*Fireball, error) {
	hot, air := thermal.Hottest.InKelvin(), thermal.Air.InKelvin()
	positive := []float64{thermal.Radius, thermal.Soot, hot, air}
	for _, value := range positive {
		if !(value > 0) || math.IsInf(value, 1) {
			return nil, errors.New("flame: a thermal's radius, soot and " +
				"temperatures must be greater than zero and finite")
		}
	}
	if !(hot > air) {
		return nil, errors.New("flame: a thermal must be hotter than its air")
	}
	if !(thermal.Height >= 0) || math.IsInf(thermal.Height, 1) {
		return nil, errors.New("flame: a thermal's height must be zero or " +
			"more and finite")
	}
	n := thermal.Spread
	if n == 0 {
		n = scorerN
	}
	if !(n >= 1) || math.IsInf(n, 1) {
		return nil, errors.New("flame: a thermal's spread must be one or " +
			"more and finite")
	}
	if !(thermal.Stem >= 0 && thermal.Stem <= 1) {
		return nil, errors.New("flame: a thermal's stem is a share of its " +
			"soot, from zero to one")
	}
	glows, err := Colours(ColourSpec{Coolest: thermal.Air,
		Hottest: thermal.Hottest, Exposure: exposure})
	if err != nil {
		return nil, err
	}
	lighter := 1 - air/hot
	r0 := thermal.Radius
	rate := 2 * scorerC / n * math.Pow(r0, 1.5) * math.Sqrt(thermalGravity*lighter)
	return &Fireball{thermal: thermal, rate: rate,
		origin: thermal.Height - n*r0, n: n, glows: glows}, nil
}

// age is how long it has been rising at its time: negative before it
// is let go.
func (ball *Fireball) age() time.Duration {
	return ball.at - ball.thermal.Start
}

// radius is the fireball's radius at its time, in metres: its radius
// when let go until then.
func (ball *Fireball) radius() float64 {
	t := math.Max(0, ball.age().Seconds())
	r0 := ball.thermal.Radius
	return math.Sqrt(r0*r0 + ball.rate*t)
}

// centre is its centre's height at its time, in metres.
func (ball *Fireball) centre() float64 {
	return ball.origin + ball.n*ball.radius()
}

// excess is how much hotter than the air its heart is at its time.
func (ball *Fireball) excess() float64 {
	shrink := ball.thermal.Radius / ball.radius()
	return (ball.thermal.Hottest.InKelvin() - ball.thermal.Air.InKelvin()) *
		shrink * shrink * shrink
}

// turned is how far it has turned about its ring by its time, in
// radians: (3n/sqrt2) ln(r/r0).
func (ball *Fireball) turned() float64 {
	return 3 * ball.n / math.Sqrt2 * math.Log(ball.radius()/ball.thermal.Radius)
}

// Tick is the fireball at a time on the picture's clock: before its
// Start it is not there.
func (ball *Fireball) Tick(at time.Duration) medium.Medium {
	ticked := *ball
	ticked.at = at
	return &ticked
}

// Bounds is the box the fireball and its stem keep inside at its time.
func (ball *Fireball) Bounds() maths.Box {
	r := ball.radius()
	wide := thermalReach * r
	bottom := ball.centre() - wide
	if ball.thermal.Stem > 0 {
		bottom = 0
	}
	return maths.Box{Min: maths.V(-wide, math.Max(0, bottom), -wide),
		Max: maths.V(wide, ball.centre()+wide, wide)}
}

// Finest is the smallest eddy it draws at its time, in metres, as a
// Body's is: a grid that holds it wants cells no wider than half this.
func (ball *Fireball) Finest() float64 {
	return ball.radius() / thermalEddies / math.Pow(2, thermalOctaves-1)
}

// Density is its soot at p: the cap, a ball gathering into its ring,
// and the stem below it, each clumped by eddies that turn with it.
// Outside its box it is none, though the ring's and the stem's edges
// would reach a little past.
func (ball *Fireball) Density(p maths.Vec) float64 {
	box := ball.Bounds()
	if ball.age() < 0 || p.X < box.Min.X || p.X > box.Max.X || p.Y < box.Min.Y ||
		p.Y > box.Max.Y || p.Z < box.Min.Z || p.Z > box.Max.Z {
		return 0
	}
	r := ball.radius()
	shrink := ball.thermal.Radius / r
	soot := ball.thermal.Soot * shrink * shrink * shrink
	centre := maths.V(0, ball.centre(), 0)
	q := p.Sub(centre).Scale(1 / r)
	return soot * (ball.cap(q) + ball.stem(p, r)) * ball.clumps(q)
}

// cap is the thermal's own soot at q, in its radii from its centre: a
// ball, thickest at its heart, that gathers into the ring Hill's vortex
// turns about as it grows, all of it by the time it is three times its
// first size.
func (ball *Fireball) cap(q maths.Vec) float64 {
	grown := math.Log(ball.radius() / ball.thermal.Radius)
	gathered := math.Min(1, grown/math.Log(3))
	whole := math.Max(0, 1-q.Dot(q))
	across := math.Hypot(q.X, q.Z) - 1/math.Sqrt2
	ring := math.Exp(-(across*across + q.Y*q.Y) / (2 * ringWidth * ringWidth))
	return (1-gathered)*whole + gathered*ring
}

// stem is the column the thermal draws up behind it, at p in metres: a
// share of its soot, from the ground to its centre, a fifth of its
// radius wide at its foot and a third at its top.
func (ball *Fireball) stem(p maths.Vec, r float64) float64 {
	top := ball.centre()
	if ball.thermal.Stem <= 0 || p.Y < 0 || p.Y > top {
		return 0
	}
	wide := r * (0.2 + (1.0/3-0.2)*p.Y/top)
	out := math.Hypot(p.X, p.Z) / wide
	return ball.thermal.Stem * math.Exp(-out*out)
}

// clumps is how much of the soot its eddies leave at q, from
// 1 - thermalClumping to one: noise, taken where q was before the
// thermal turned it about its ring, so the clumps roll with the gas.
func (ball *Fireball) clumps(q maths.Vec) float64 {
	around := math.Hypot(q.X, q.Z)
	ring := 1 / math.Sqrt2
	out, up := around-ring, q.Y
	turn := -ball.turned()
	cos, sin := math.Cos(turn), math.Sin(turn)
	out, up = out*cos-up*sin, out*sin+up*cos
	way := maths.V(1, 0, 0)
	if around > 0 {
		way = maths.V(q.X/around, 0, q.Z/around)
	}
	was := way.Scale(ring + out).Add(maths.V(0, up, 0)).Scale(thermalEddies)
	// five octaves of noise stay within about 0.65 of zero: as a share
	// from zero to one
	lumpy := noise.Fractal(ball.thermal.Seed, was.X, was.Y, was.Z, thermalOctaves)
	share := math.Max(0, math.Min(1, 0.5+lumpy/1.3))
	return 1 - thermalClumping*(1-share)
}

// Heat is its gas's temperature at p: the air's, and at its heart the
// excess heat it has left, falling away to the air's at its edge. The
// stem is the air's own.
func (ball *Fireball) Heat(p maths.Vec) measure.Temperature {
	air := ball.thermal.Air.InKelvin()
	if ball.age() < 0 {
		return measure.Kelvin(air)
	}
	centre := maths.V(0, ball.centre(), 0)
	q := p.Sub(centre).Scale(1 / ball.radius())
	excess := ball.excess()
	whole := math.Max(0, 1-q.Dot(q))
	return measure.Kelvin(air + excess*whole)
}

// Tint is soot's tilt, as a flame's Body's is: the fireball's soot is
// the flame's.
func (ball *Fireball) Tint() maths.Light { return Tilt() }

// fireballGlow is the fireball's glow, as the texture its volume's
// material shines with.
type fireballGlow struct{ ball *Fireball }

// Shine is the fireball's glow, for its volume's material.
func (ball *Fireball) Shine() texture.Timed { return fireballGlow{ball} }

// At is the thermal source at the point's heat.
func (glow fireballGlow) At(p maths.Vec) maths.Light {
	return glow.ball.glows.at(glow.ball.Heat(p))
}

// Tick is the glow at a time, the fireball's own.
func (glow fireballGlow) Tick(at time.Duration) texture.Texture {
	return fireballGlow{glow.ball.Tick(at).(*Fireball)}
}
