package embers

import (
	"errors"
	"math"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// Shell is a mortar's aerial shell and the stars it throws, in SI units.
//
// The shell leaves the mortar straight up at Lift and climbs against
// gravity and the air's drag, which grows as the square of its speed.
// Terminal is the speed at which that drag would hold the shell's
// weight. After Fuse the shell bursts wherever it is, rising or
// falling, and throws Stars stars outward on a sphere, each at between
// nine tenths and all of Throw, on top of the shell's own velocity.
//
// A star follows the same path as any ember: linear drag over StarDrag
// toward the Air's velocity, under Gravity. It burns for Burn, from Hot
// down to Cool, and is then gone. Before the burst the shell itself is
// one particle at Cool, the glow of its burning time fuse.
type Shell struct {
	Lift, Terminal, Throw, Gravity float64
	Fuse, StarDrag, Burn           time.Duration
	Stars                          int
	Radius                         measure.Length
	Hot, Cool                      measure.Temperature
	Air                            maths.Vec
	Seed                           uint64
}

// Mortar is a shell checked and ready to fire. Its At is the shell, or
// its stars, at a time since it was fired. It is a Source.
type Mortar struct{ shell Shell }

// NewMortar checks a shell: every speed, time and size greater than
// zero and finite (Throw may be zero, for stars that only fall), Cool
// no hotter than Hot, and no more stars than an emitter may hold.
func NewMortar(shell Shell) (Mortar, error) {
	positive := []float64{shell.Lift, shell.Terminal, shell.Gravity,
		shell.Fuse.Seconds(), shell.StarDrag.Seconds(), shell.Burn.Seconds(),
		shell.Radius.InMetres(), shell.Hot.InKelvin(), shell.Cool.InKelvin()}
	for _, value := range positive {
		if !(value > 0) || math.IsInf(value, 1) {
			return Mortar{}, errors.New("embers: a shell's speeds, times, " +
				"star size and temperatures must be greater than zero and " +
				"finite")
		}
	}
	finite := []float64{shell.Throw, shell.Air.X, shell.Air.Y, shell.Air.Z}
	for _, value := range finite {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return Mortar{}, errors.New("embers: a shell's throw and air " +
				"must be finite")
		}
	}
	if shell.Throw < 0 {
		return Mortar{}, errors.New("embers: a shell's throw must be zero " +
			"or more")
	}
	if !(shell.Cool.InKelvin() <= shell.Hot.InKelvin()) {
		return Mortar{}, errors.New("embers: a shell's stars must cool, " +
			"not warm")
	}
	if shell.Stars < 1 || shell.Stars > mostLiving {
		return Mortar{}, errors.New("embers: a shell throws from one star " +
			"to as many as an emitter may hold")
	}
	return Mortar{shell}, nil
}

// Burst is when the shell bursts, after it is fired, and how high.
func (mortar Mortar) Burst() (time.Duration, measure.Length) {
	height, _ := mortar.shell.rise(mortar.shell.Fuse.Seconds())
	return mortar.shell.Fuse, measure.Metres(height)
}

// At is the shell before its burst, its burning stars after it, and
// nothing before it is fired or once the stars have burnt out.
func (mortar Mortar) At(when time.Duration) []Particle {
	shell := mortar.shell
	if when < 0 {
		return nil
	}
	if when < shell.Fuse {
		height, _ := shell.rise(when.Seconds())
		return []Particle{{id: int64(shell.Stars), at: maths.V(0, height, 0),
			radius: shell.Radius, heat: shell.Cool, age: when}}
	}
	age := when - shell.Fuse
	if age >= shell.Burn {
		return nil
	}
	height, speed := shell.rise(shell.Fuse.Seconds())
	burst, carried := maths.V(0, height, 0), maths.V(0, speed, 0)
	hot, cool := shell.Hot.InKelvin(), shell.Cool.InKelvin()
	heat := measure.Kelvin(cool + (hot-cool)*(1-age.Seconds()/shell.Burn.Seconds()))
	stars := make([]Particle, 0, shell.Stars)
	for id := range int64(shell.Stars) {
		velocity := carried.Add(shell.thrown(id))
		at := position(burst, velocity, shell.Air, maths.V(0, -shell.Gravity, 0),
			age, shell.StarDrag)
		stars = append(stars, Particle{id: id, at: at, radius: shell.Radius,
			heat: heat, age: age})
	}
	return stars
}

// thrown is a star's velocity out of the burst: a direction even over
// the sphere (a height on it and a bearing, from two draws), at a speed
// from a third.
func (shell Shell) thrown(id int64) maths.Vec {
	up := 2*share(shell.Seed, id, 4) - 1
	turn := math.Pi * (2*share(shell.Seed, id, 5) - 1)
	across := math.Sqrt(1 - up*up)
	speed := shell.Throw * (0.9 + 0.1*share(shell.Seed, id, 3))
	return maths.V(across*math.Cos(turn), up, across*math.Sin(turn)).Scale(speed)
}

// rise is the shell's height and upward speed t seconds after it is
// fired, under gravity g and drag that holds its weight at the terminal
// speed vt.
//
// Going up, with theta the angle whose tangent is Lift over
// vt, the speed is vt tan(theta - g t/vt) and the height (vt^2/g)
// ln(cos(theta - g t/vt) / cos theta), until the apex at t = vt theta/g.
// Coming down, s seconds past the apex, the speed is -vt tanh(g s/vt)
// and the fall from the apex (vt^2/g) ln cosh(g s/vt).
//
// Each is written so that a shell the air hardly slows, whose vt is far
// above its Lift, keeps its digits: with a = g t/vt, the cosine ratio
// is 1 + (Lift/vt) sin a - 2 sin^2(a/2), taken through log1p; the speed
// is (Lift - vt tan a) / (1 + (Lift/vt) tan a); and the apex's height is
// half of (vt^2/g) log1p((Lift/vt)^2).
func (shell Shell) rise(t float64) (height, speed float64) {
	g, vt, lift := shell.Gravity, shell.Terminal, shell.Lift
	ratio := lift / vt
	scale := vt * vt / g
	apex := vt * math.Atan(ratio) / g
	if t <= apex {
		a := g * t / vt
		half := math.Sin(a / 2)
		height = scale * math.Log1p(ratio*math.Sin(a)-2*half*half)
		return height, (lift - vt*math.Tan(a)) / (1 + ratio*math.Tan(a))
	}
	top := scale / 2 * math.Log1p(ratio*ratio)
	s := g * (t - apex) / vt
	return top - scale*logCosh(s), -vt * math.Tanh(s)
}

// logCosh is ln cosh s for s of zero or more. Up to one it is
// log1p(2 sinh^2(s/2)), which keeps its digits where s is small; past
// one it is s + log1p(e^-2s) - ln 2, which does not overflow where cosh
// does, past s of about 710.
func logCosh(s float64) float64 {
	if s <= 1 {
		half := math.Sinh(s / 2)
		return math.Log1p(2 * half * half)
	}
	return s + math.Log1p(math.Exp(-2*s)) - math.Ln2
}
