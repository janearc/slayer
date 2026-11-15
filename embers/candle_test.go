package embers

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// aCandle is a candle with round numbers: a shot a second, at 30 m/s,
// burning two seconds, with a trail of 50 sparks a second.
func aCandle() Candle {
	return Candle{Shots: 6, Interval: time.Second, Drag: 2 * time.Second,
		Burn: 2 * time.Second, Speed: 30, Gravity: 10, Trail: 50, Carry: 0.2,
		Spread: 2, TrailLife: 500 * time.Millisecond,
		TrailDrag: 300 * time.Millisecond, Radius: measure.Metres(0.008),
		SparkRadius: measure.Metres(0.002), Hot: measure.Kelvin(1900),
		Cool: measure.Kelvin(1200), Seed: 3}
}

// stars is the particles that are stars, not trail sparks.
func stars(particles []Particle) []Particle {
	var only []Particle
	for _, particle := range particles {
		if particle.ID()%candleSpan == 0 {
			only = append(only, particle)
		}
	}
	return only
}

// With the air's hold all but gone, a star rises as a stone thrown up:
// 30 t - 5 t^2, 25 m after one second.
func TestACandlesStarWithoutAirRisesAsAThrownStone(t *testing.T) {
	candle := aCandle()
	candle.Drag = 1e6 * time.Second
	lit, err := NewRomanCandle(candle)
	if err != nil {
		t.Fatal(err)
	}
	first := stars(lit.At(time.Second))
	if len(first) != 2 || !near(first[0].Position().Y, 25, 1e-4) {
		t.Errorf("a second after lighting the stars are %v", first)
	}
}

// A shot a second, burning two: at 2.3 s the first star has burnt out
// and the second and third burn; the first's trail still falls, the
// sparks it shed in its last fifth of a second, at 50 a second.
func TestACandleThrowsAStarEachInterval(t *testing.T) {
	lit, err := NewRomanCandle(aCandle())
	if err != nil {
		t.Fatal(err)
	}
	at := 2300 * time.Millisecond
	burning := stars(lit.At(at))
	if len(burning) != 2 || burning[0].ID() != candleSpan ||
		burning[1].ID() != 2*candleSpan {
		t.Errorf("at 2.3 s the burning stars are %v", burning)
	}
	trailing := 0
	for _, particle := range lit.At(at) {
		if particle.ID()/candleSpan == 0 {
			trailing++
		}
	}
	if !(trailing >= 9 && trailing <= 10) {
		t.Errorf("the first star's trail is %d sparks at 2.3 s", trailing)
	}
	if got := lit.At(20 * time.Second); len(got) != 0 {
		t.Errorf("long after the last shot the candle is %d particles", len(got))
	}
	if got := lit.At(-time.Second); got != nil {
		t.Errorf("before it is lit the candle is %d particles", len(got))
	}
}

// A spark shed with all its star's velocity, thrown nowhere, slowed by
// the same drag, follows the very path its star does: on the embers'
// path a body's future depends only on where it is and how fast it
// goes. So every such spark sits on its star.
func TestASparkShedAtItsStarsVelocityRidesWithIt(t *testing.T) {
	candle := aCandle()
	candle.Carry, candle.Spread = 1, 0
	candle.TrailDrag = candle.Drag
	candle.Air = maths.V(3, 0, -1)
	lit, err := NewRomanCandle(candle)
	if err != nil {
		t.Fatal(err)
	}
	at := 1700 * time.Millisecond
	star := stars(lit.At(at))[0]
	for _, particle := range lit.At(at) {
		if particle.ID()/candleSpan != 0 {
			continue
		}
		if apart := particle.Position().Sub(star.Position()).Len(); !(apart <= 1e-9) {
			t.Fatalf("spark %d is %v m off its star", particle.ID(), apart)
		}
	}
}

// The star's velocity is how fast its position changes.
func TestACandleStarsVelocityIsItsPathsRate(t *testing.T) {
	candle := aCandle()
	candle.Air = maths.V(2, 0, 1)
	lit, err := NewRomanCandle(candle)
	if err != nil {
		t.Fatal(err)
	}
	launch, gravity := maths.V(0, 30, 0), maths.V(0, -10, 0)
	step := time.Microsecond
	for _, age := range []time.Duration{0, 300 * time.Millisecond, 1500 * time.Millisecond} {
		ahead := position(maths.Vec{}, launch, candle.Air, gravity, age+step, candle.Drag)
		behind := position(maths.Vec{}, launch, candle.Air, gravity, age, candle.Drag)
		rate := ahead.Sub(behind).Scale(1 / step.Seconds())
		if off := rate.Sub(lit.velocity(launch, gravity, age)).Len(); !(off <= 1e-3) {
			t.Errorf("at %v the star's velocity is %v off its path's rate", age, off)
		}
	}
}

// The same seed sheds the same trail; another seed, another.
func TestACandlesTrailIsTheSameEveryTime(t *testing.T) {
	one, _ := NewRomanCandle(aCandle())
	two, _ := NewRomanCandle(aCandle())
	other := aCandle()
	other.Seed = 4
	three, _ := NewRomanCandle(other)
	at := 1200 * time.Millisecond
	a, b, c := one.At(at), two.At(at), three.At(at)
	if len(a) < 3 || a[2].Position() != b[2].Position() {
		t.Error("one seed shed two different trails")
	}
	if a[2].Position() == c[2].Position() {
		t.Error("two seeds shed the same trail")
	}
}

// A candle that cannot be lit is refused, by each of its numbers; one
// with no trail at all is sound.
func TestARomanCandleRefusesWhatCannotBurn(t *testing.T) {
	broken := []func(*Candle){
		func(c *Candle) { c.Shots = 0 },
		func(c *Candle) { c.Shots = 1<<20 + 1 },
		func(c *Candle) { c.Interval = 0 },
		func(c *Candle) { c.Speed = math.NaN() },
		func(c *Candle) { c.Gravity = 0 },
		func(c *Candle) { c.Burn = -time.Second },
		func(c *Candle) { c.Trail = -1 },
		func(c *Candle) { c.Spread = math.Inf(1) },
		func(c *Candle) { c.TrailLife = 0 },
		func(c *Candle) { c.SparkRadius = measure.Metres(0) },
		func(c *Candle) { c.Air = maths.V(0, math.NaN(), 0) },
		func(c *Candle) { c.Cool = measure.Kelvin(2000) },
		func(c *Candle) { c.Trail = 1e5 },
	}
	for i, breaks := range broken {
		candle := aCandle()
		breaks(&candle)
		if _, err := NewRomanCandle(candle); err == nil {
			t.Errorf("broken candle %d was lit: %+v", i, candle)
		}
	}
	bare := aCandle()
	bare.Trail, bare.TrailLife, bare.TrailDrag = 0, 0, 0
	bare.SparkRadius = measure.Metres(0)
	lit, err := NewRomanCandle(bare)
	if err != nil {
		t.Fatalf("a candle with no trail was refused: %v", err)
	}
	if got := lit.At(1500 * time.Millisecond); len(got) != 2 {
		t.Errorf("a candle with no trail is %d particles, not its 2 stars",
			len(got))
	}
}
