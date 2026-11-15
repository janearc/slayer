package embers

import (
	"errors"
	"math"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// Candle is a roman candle: a tube that throws Shots stars straight up,
// one every Interval, each at Speed. A star follows the embers' path,
// linear drag over Drag toward the Air's velocity, under Gravity, and
// burns for Burn from Hot down to Cool.
//
// As it burns, each star sheds a trail: Trail sparks a second, each
// born where the star is, carried along at Carry of the star's velocity
// and thrown at up to Spread in any direction, falling back on the
// embers' path with TrailDrag for TrailLife, from the star's heat at
// its birth down to Cool. A Trail of zero sheds none.
type Candle struct {
	Shots                int
	Interval, Drag, Burn time.Duration
	Speed, Gravity       float64
	Trail, Carry, Spread float64
	TrailLife, TrailDrag time.Duration
	Radius, SparkRadius  measure.Length
	Hot, Cool            measure.Temperature
	Air                  maths.Vec
	Seed                 uint64
}

// RomanCandle is a candle checked and ready to light. Its At is the
// burning stars and their trails at a time since it was lit. It is a
// Source.
type RomanCandle struct{ candle Candle }

// candleSpan is how many particle ids each shot takes: its star, and
// every spark its trail can shed.
const candleSpan = 1 << 24

// NewRomanCandle checks a candle: one shot to a million; every time, speed,
// size and temperature greater than zero and finite, but Trail, Carry,
// Spread and the trail's times, which may be zero when there is no
// trail; Cool no hotter than Hot; and no more living particles than an
// emitter may hold.
func NewRomanCandle(candle Candle) (RomanCandle, error) {
	positive := []float64{candle.Interval.Seconds(), candle.Drag.Seconds(),
		candle.Burn.Seconds(), candle.Speed, candle.Gravity,
		candle.Radius.InMetres(), candle.Hot.InKelvin(), candle.Cool.InKelvin()}
	for _, value := range positive {
		if !(value > 0) || math.IsInf(value, 1) {
			return RomanCandle{}, errors.New("embers: a candle's times, " +
				"speed, gravity, star size and temperatures must be greater " +
				"than zero and finite")
		}
	}
	trail := []float64{candle.Trail, candle.Carry, candle.Spread,
		candle.TrailLife.Seconds(), candle.TrailDrag.Seconds(),
		candle.SparkRadius.InMetres()}
	for _, value := range trail {
		if !(value >= 0) || math.IsInf(value, 1) {
			return RomanCandle{}, errors.New("embers: a candle's trail " +
				"must be zero or more and finite")
		}
	}
	if candle.Trail > 0 && !(candle.TrailLife > 0 && candle.TrailDrag > 0 &&
		candle.SparkRadius.InMetres() > 0) {
		return RomanCandle{}, errors.New("embers: a candle's trail sparks " +
			"must live, slow and have a size")
	}
	for _, value := range []float64{candle.Air.X, candle.Air.Y, candle.Air.Z} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return RomanCandle{}, errors.New("embers: a candle's air must " +
				"be finite")
		}
	}
	if !(candle.Cool.InKelvin() <= candle.Hot.InKelvin()) {
		return RomanCandle{}, errors.New("embers: a candle's stars must " +
			"cool, not warm")
	}
	// a shot shows for its burn and its last sparks' life after it
	showing := math.Ceil((candle.Burn + candle.TrailLife).Seconds() /
		candle.Interval.Seconds())
	shedding := math.Ceil(candle.Trail*candle.TrailLife.Seconds()) + 2
	if candle.Shots < 1 || candle.Shots > 1<<20 ||
		showing*(1+shedding) > mostLiving ||
		candle.Trail*candle.Burn.Seconds() >= candleSpan {
		return RomanCandle{}, errors.New("embers: a candle throws one shot " +
			"or more, and no more living stars and sparks than an emitter " +
			"may hold")
	}
	return RomanCandle{candle}, nil
}

// At is every star still burning, and every spark of its trail still
// living, at a time since the candle was lit.
func (lit RomanCandle) At(when time.Duration) []Particle {
	candle := lit.candle
	if when < 0 {
		return nil
	}
	gravity := maths.V(0, -candle.Gravity, 0)
	hot, cool := candle.Hot.InKelvin(), candle.Cool.InKelvin()
	var living []Particle
	for shot := range int64(candle.Shots) {
		age := when - time.Duration(shot)*candle.Interval
		if age < 0 {
			break
		}
		if age >= candle.Burn+candle.TrailLife {
			continue
		}
		launch := maths.V(0, candle.Speed, 0)
		if age < candle.Burn {
			at := position(maths.Vec{}, launch, candle.Air, gravity, age, candle.Drag)
			heat := cool + (hot-cool)*(1-age.Seconds()/candle.Burn.Seconds())
			living = append(living, Particle{id: shot * candleSpan,
				at: at, radius: candle.Radius, heat: measure.Kelvin(heat),
				age: age})
		}
		living = append(living, lit.trail(shot, age, launch, gravity)...)
	}
	return living
}

// trail is a shot's sparks still living when the star is age old:
// each born at one Trail-th of a second apart, while the star burns,
// and no longer ago than a spark lives.
func (lit RomanCandle) trail(shot int64, age time.Duration, launch,
	gravity maths.Vec) []Particle {
	candle := lit.candle
	if candle.Trail <= 0 {
		return nil
	}
	hot, cool := candle.Hot.InKelvin(), candle.Cool.InKelvin()
	newest := math.Min(age.Seconds(), candle.Burn.Seconds())
	oldest := math.Max(0, age.Seconds()-candle.TrailLife.Seconds())
	var sparks []Particle
	first := int64(math.Ceil(oldest * candle.Trail))
	for n := first; float64(n)/candle.Trail < newest; n++ {
		born := float64(n) / candle.Trail
		spark := age - seconds(born)
		if spark < 0 || spark >= candle.TrailLife {
			continue
		}
		from := position(maths.Vec{}, launch, candle.Air, gravity,
			seconds(born), candle.Drag)
		id := shot*candleSpan + 1 + n
		jitter := maths.V(2*share(candle.Seed, id, 6)-1,
			2*share(candle.Seed, id, 7)-1, 2*share(candle.Seed, id, 8)-1)
		velocity := lit.velocity(launch, gravity, seconds(born)).
			Scale(candle.Carry).Add(jitter.Scale(candle.Spread))
		at := position(from, velocity, candle.Air, gravity, spark, candle.TrailDrag)
		starHeat := cool + (hot-cool)*(1-born/candle.Burn.Seconds())
		spent := spark.Seconds() / candle.TrailLife.Seconds()
		heat := starHeat + (cool-starHeat)*spent
		sparks = append(sparks, Particle{id: id, at: at,
			radius: candle.SparkRadius, heat: measure.Kelvin(heat), age: spark})
	}
	return sparks
}

// velocity is a star's velocity age after it leaves the tube: the
// embers' path's own, the terminal velocity the air and gravity draw it
// to, plus what is left of its launch's difference from that.
func (lit RomanCandle) velocity(launch, gravity maths.Vec,
	age time.Duration) maths.Vec {
	tau := lit.candle.Drag.Seconds()
	terminal := lit.candle.Air.Add(gravity.Scale(tau))
	return terminal.Add(launch.Sub(terminal).Scale(math.Exp(-age.Seconds() / tau)))
}
