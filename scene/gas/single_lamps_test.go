package gas

import (
	"math"
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// lampAt is a lamp that gives what it is told: a way, a distance, a light.
type lampAt struct {
	toward maths.Vec
	far    float64
	light  maths.Light
}

func (l lampAt) Light(_, _ maths.Vec) (maths.Vec, float64, maths.Light) {
	return l.toward, l.far, l.light
}

// asked is what a Through was asked.
type asked struct {
	at, way maths.Vec
	far     float64
	band    maths.Band
}

// reaching is a Through that records what it is asked and answers with the
// reach for the way, zero for a way it has no reach for.
func reaching(calls *[]asked, reach func(way maths.Vec) float64) Through {
	return func(at, way maths.Vec, far float64, band maths.Band) float64 {
		*calls = append(*calls, asked{at, way, far, band})
		return reach(way)
	}
}

// even scatters 0.01 a metre, absorbs 0.3 that scattering must not take,
// and throws light every way alike: rayleigh's lobe. test figures.
func even() Gas {
	return Gas{Name: "even", Ranges: map[radiation.Range]Optics{
		thing.Visible: {Absorb: 0.3, Scatter: 0.01}}}
}

var green = maths.Band{Nanometres: 550}

// a sun is asked for at its way, made a unit, and an endless distance, as
// it is. what comes of it is the sun's light times the reach, the density,
// the scattering and the phase at the angle.
//
// worked at (1, 2, 3), the sun toward (0, 3, 0), the eye toward (0, 0, 5):
// a right angle, so rayleigh's 3/(16 pi); with a reach of a half and a
// density of four, 0.5 x 4 x 0.01 x 3/(16 pi) of the light.
func TestASunScattersTowardTheEye(t *testing.T) {
	var calls []asked
	through := reaching(&calls, func(maths.Vec) float64 { return 0.5 })
	sun := film.Sun{Toward: maths.V(0, 3, 0), Strength: maths.RGB(2, 4, 8)}
	p := maths.V(1, 2, 3)
	got := Single(even(), 4, green, []thing.Lamp{sun}, p, maths.V(0, 0, 5), through)
	k := 0.5 * 4 * 0.01 * 3 / (16 * math.Pi)
	wantLight(t, "the sun's scattering", got, maths.RGB(2*k, 4*k, 8*k))
	if len(calls) != 1 {
		t.Fatalf("the sun asked the gas %d times, want once", len(calls))
	}
	c := calls[0]
	if c.at != p || c.way != maths.V(0, 1, 0) || !math.IsInf(c.far, 1) || c.band != green {
		t.Errorf("the gas was asked %v", c)
	}
}

// the angle scattered through is from the way the light travels, minus
// toward, to the way of the eye. a sun toward +x seen from -x has its light
// going on to the eye, forward, a cosine of one: (1+g)/(4 pi (1-g)^2) at
// g = 0.7.
//
// seen from +x, the light is thrown back, a cosine of minus one:
// (1-g)/(4 pi (1+g)^2). each at a reach of one, a density of one, 0.01.
func TestTheSunIsForwardFromBehindAndBackFromAhead(t *testing.T) {
	g := even()
	g.Phase = 0.7
	one := func(maths.Vec) float64 { return 1 }
	sun := film.Sun{Toward: maths.V(1, 0, 0), Strength: maths.Grey(1)}
	var calls []asked
	fwd := Single(g, 1, green, []thing.Lamp{sun}, maths.Vec{},
		maths.V(-1, 0, 0), reaching(&calls, one))
	wantLight(t, "forward", fwd, maths.Grey(0.01*1.7/(4*math.Pi*0.09)))
	back := Single(g, 1, green, []thing.Lamp{sun}, maths.Vec{},
		maths.V(1, 0, 0), reaching(&calls, one))
	wantLight(t, "back", back, maths.Grey(0.01*0.3/(4*math.Pi*2.89)))
}

// several lamps add: each by its own way, its own distance, its own reach.
//
// a sun toward +y (red, a reach of a half) and a lamp 10 off toward +z (green,
// a reach of a quarter), the eye toward +y, a density of two. the sun's light
// goes down, the eye is up, a cosine of minus one, rayleigh's 6/(16 pi); the
// lamp's goes across, 3/(16 pi).
//
// a dark lamp is not asked about. three more lamps have a reach of less than
// zero, of no number and of zero, and add nothing.
func TestSeveralLampsAddEachByItsOwnWay(t *testing.T) {
	var calls []asked
	reach := func(way maths.Vec) float64 {
		switch way {
		case maths.V(0, 1, 0):
			return 0.5
		case maths.V(0, 0, 1):
			return 0.25
		case maths.V(1, 0, 0):
			return -2
		case maths.V(-1, 0, 0):
			return math.NaN()
		}
		return 0
	}
	lamps := []thing.Lamp{
		film.Sun{Toward: maths.V(0, 1, 0), Strength: maths.RGB(1, 0, 0)},
		lampAt{toward: maths.V(0, 0, 2), far: 10, light: maths.RGB(0, 1, 0)},
		lampAt{toward: maths.V(0, 0, 1), far: 5, light: maths.Light{}},
		lampAt{toward: maths.V(1, 0, 0), far: 5, light: maths.Grey(1)},
		lampAt{toward: maths.V(-1, 0, 0), far: 5, light: maths.Grey(1)},
		lampAt{toward: maths.V(0, -1, 0), far: 5, light: maths.Grey(1)},
	}
	got := Single(even(), 2, green, lamps, maths.Vec{}, maths.V(0, 1, 0),
		reaching(&calls, reach))
	side := 3 / (16 * math.Pi)
	wantLight(t, "the lamps", got, maths.RGB(0.5*2*0.01*2*side, 0.25*2*0.01*side, 0))
	if len(calls) != 5 {
		t.Fatalf("the gas was asked %d times, want 5: the dark lamp is not asked", len(calls))
	}
	if calls[1].far != 10 || calls[1].way != maths.V(0, 0, 1) {
		t.Errorf("the second lamp was asked %v, want a unit way and its distance of 10", calls[1])
	}
	for i, c := range calls {
		if c.band != green || c.at != (maths.Vec{}) {
			t.Errorf("ask %d was %v", i, c)
		}
	}
}

// a gas of no density, or less, lights nothing and asks nothing: no lamp
// is called on and no path is walked.
func TestNoDensityAsksNothing(t *testing.T) {
	sun := film.Sun{Toward: maths.V(0, 1, 0), Strength: maths.Grey(1)}
	for _, d := range []float64{0, -1} {
		var calls []asked
		got := Single(even(), d, green, []thing.Lamp{sun}, maths.Vec{},
			maths.V(0, 0, 1), reaching(&calls, func(maths.Vec) float64 { return 1 }))
		if got != (maths.Light{}) || len(calls) != 0 {
			t.Errorf("a density of %v gave %v and %d asks", d, got, len(calls))
		}
	}
}

// a reach of zero means that none arrives, and a gas that scatters without end
// sends none of it: zero, not zero times endless. so for a reach of zero, and
// of less, and of no number.
func TestNoReachSendsNothingEvenFromAnEndlessScatterer(t *testing.T) {
	endless := Gas{Visible: func(float64) (absorb, scatter float64) {
		return 0, math.Inf(1)
	}}
	sun := film.Sun{Toward: maths.V(0, 1, 0), Strength: maths.Grey(1)}
	for _, r := range []float64{0, -1, math.NaN()} {
		var calls []asked
		got := Single(endless, 1, green, []thing.Lamp{sun}, maths.Vec{},
			maths.V(0, 0, 1), reaching(&calls, func(maths.Vec) float64 { return r }))
		if got != (maths.Light{}) {
			t.Errorf("a reach of %v gave %v", r, got)
		}
	}
}

// the density multiplies: a density of two sends twice what one does, and
// not a half of it.
func TestSingleScalesWithTheDensity(t *testing.T) {
	sun := film.Sun{Toward: maths.V(0, 1, 0), Strength: maths.Grey(1)}
	var calls []asked
	one := reaching(&calls, func(maths.Vec) float64 { return 1 })
	k := 0.01 * 3 / (16 * math.Pi)
	wantLight(t, "density two", Single(even(), 2, green, []thing.Lamp{sun},
		maths.Vec{}, maths.V(0, 0, 1), one), maths.Grey(2*k))
	wantLight(t, "density a half", Single(even(), 0.5, green, []thing.Lamp{sun},
		maths.Vec{}, maths.V(0, 0, 1), one), maths.Grey(0.5*k))
}

// the uniform path is beer's law over density times distance: with
// extinction 0.1 a metre, a density of two and three metres, e^-0.6; and a
// sun's endless path, to nothing.
func TestUniformIsBeersLawOverDensityTimesDistance(t *testing.T) {
	g := Gas{Ranges: map[radiation.Range]Optics{
		thing.Visible: {Absorb: 0.04, Scatter: 0.06}}}
	wantClose(t, "uniform", Uniform(g, 2)(maths.Vec{}, maths.Up, 3, green), math.Exp(-0.6))
	if got := Uniform(g, 2)(maths.Vec{}, maths.Up, math.Inf(1), green); got != 0 {
		t.Errorf("uniform to the sun is %v", got)
	}
}
