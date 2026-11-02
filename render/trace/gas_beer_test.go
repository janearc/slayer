package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// three bands of one gas, red, green and blue, each with an absorption
// and a scattering per metre. their sums, the extinctions, are 0.9, 0.5
// and 0.2, so the red goes first and the blue last.
var (
	redOptics   = gas.Optics{Absorb: 0.5, Scatter: 0.4}
	greenOptics = gas.Optics{Absorb: 0.3, Scatter: 0.2}
	blueOptics  = gas.Optics{Absorb: 0.1, Scatter: 0.1}
)

func extinct(o gas.Optics) float64 { return o.Absorb + o.Scatter }

// gasBlock is a block of a gas, of density k and half-width half, set at
// a place at a scale.
func gasBlock(k, half float64, g gas.Gas, unit float64, origin maths.Vec,
	scale float64) thing.Volume {
	return thing.Volume{Medium: gas.Of{Shape: block{k: k, half: half}, In: g,
		Unit: unit}, Motion: at(origin, scale)}
}

// a gas takes each of the eye's three bands down by its own extinction,
// by beer's law: what is left of a channel is exp(-tau), with tau the
// extinction per metre in that band, times the metres to a unit, times
// the density, times the way through in the volume's own units.
//
// against a sky of one that is the pixel, and the pixel covers by one less
// what the most open channel keeps. worked by hand: the block of
// half-width 1 is 2 across, so tau is e times 2 times 0.75 times 2, which
// is 2.7, 1.5 and 0.6; and the volume scaled up keeps its look.
func TestAGasTakesEachBandByItsOwnExtinction(t *testing.T) {
	g := tinted(redOptics, greenOptics, blueOptics)
	for _, scale := range []float64{1, 2} {
		w := thing.NewWorld()
		w.Sky = even(maths.Grey(1))
		w.AddVolume(gasBlock(0.75, 1, g, 2, maths.Vec{}, scale))
		got := Trace(w, along(0, 0), 0)
		tau := func(o gas.Optics) float64 { return extinct(o) * 2 * 0.75 * 2 }
		keep := maths.RGB(math.Exp(-tau(redOptics)), math.Exp(-tau(greenOptics)),
			math.Exp(-tau(blueOptics)))
		colours(t, "a gas", got, keep, 1e-6)
		if !within(got.Cover, 1-keep.B, 1e-6) {
			t.Errorf("at scale %v the gas covers %v, want %v", scale, got.Cover, 1-keep.B)
		}
	}
}

// a march stops when nothing is left to show: when no channel keeps more
// than a thousandth. it is not stopped by a channel that has gone while
// another has not, and it goes on as long as any one is left.
//
// the gas is 2 across in 48 steps, a 24th of a metre each, so a channel
// of extinction e keeps exp(-e n / 24) after n steps. the dead channels
// are the ones of 6 and 5, under a thousandth by the 28th and 34th
// steps; the live one of 1 keeps 0.135 at the end.
//
// with all three of 6 the march is over at the 28th, where each keeps
// 0.0009. the walk is worked step by step.
func TestAGasMarchStopsWhenNoChannelShows(t *testing.T) {
	for _, c := range []struct {
		name string
		ext  [3]float64
	}{
		{"red is the last to go", [3]float64{1, 6, 5}},
		{"green is the last to go", [3]float64{5, 1, 6}},
		{"blue is the last to go", [3]float64{6, 5, 1}},
		{"all go together", [3]float64{6, 6, 6}},
		{"none goes", [3]float64{1, 1, 1}},
	} {
		g := tinted(gas.Optics{Absorb: c.ext[0]}, gas.Optics{Absorb: c.ext[1]},
			gas.Optics{Absorb: c.ext[2]})
		w := thing.NewWorld()
		w.Sky = even(maths.Grey(1))
		w.AddVolume(gasBlock(1, 1, g, 1, maths.Vec{}, 1))
		got := Trace(w, along(0, 0), 0)

		keep := [3]float64{1, 1, 1}
		for n := 0; n < 48; n++ {
			if !(keep[0] > 1e-3 || keep[1] > 1e-3 || keep[2] > 1e-3) {
				break
			}
			for i := range keep {
				keep[i] *= math.Exp(-c.ext[i] / 24)
			}
		}
		colours(t, c.name, got, maths.RGB(keep[0], keep[1], keep[2]), 1e-7)
		if want := 1 - math.Max(keep[0], math.Max(keep[1], keep[2])); !within(got.Cover, want, 1e-6) {
			t.Errorf("%s: covers %v, want %v", c.name, got.Cover, want)
		}
	}
}

// warm is the sun the scattering tests are lit by, a little warm.
var warm = maths.RGB(1, 0.8, 0.6)

// a gas lit by a sun scatters it toward the eye. in each channel it is
// the sun times what the gas between lets through, times the phase for
// the angle, times the gas's scattering, times the thickness of the
// step, summed down the ray over what is left of the sky behind.
//
// a sun along plus x and an eye along minus z are a right angle apart,
// where rayleigh's phase is 3 over 16 pi; every point is 1 from the
// gas's edge on the sun's side, so the gas lets exp(-e) through.
//
// the sum is worked step by step over the 48 steps of the march, and
// is within two parts in a hundred of the integral, in each channel.
func TestAGasScattersASunByItsPhase(t *testing.T) {
	g := tinted(redOptics, greenOptics, blueOptics)
	w := thing.NewWorld()
	w.AddVolume(gasBlock(1, 1, g, 1, maths.Vec{}, 1))
	w.Lamp(thing.Sun(maths.V(1, 0, 0), warm))
	got := Trace(w, along(0, 0), 0)

	phase := 3 / (16 * math.Pi)
	var want, integral maths.Light
	for c, o := range []gas.Optics{redOptics, greenOptics, blueOptics} {
		e := extinct(o)
		in := channel(warm, c) * math.Exp(-e) * phase * o.Scatter
		thick, keep, sum := 1.0/24, 1.0, 0.0
		for i := 0; i < 48; i++ {
			sum += in * thick * keep
			keep *= math.Exp(-e * thick)
		}
		put(&want, c, sum)
		put(&integral, c, in*(1-math.Exp(-2*e))/e)
	}
	colours(t, "a gas lit by a sun", got, want, 1e-6)
	for c, p := range []float64{got.Light().R, got.Light().G, got.Light().B} {
		if math.Abs(p/channel(integral, c)-1) > 0.02 {
			t.Errorf("channel %d is %v, the integral is %v", c, p, channel(integral, c))
		}
	}
}

// channel and put read and write one channel of a light by its number.
func channel(l maths.Light, c int) float64 {
	return [3]float64{l.R, l.G, l.B}[c]
}

func put(l *maths.Light, c int, v float64) {
	switch c {
	case 0:
		l.R = v
	case 1:
		l.G = v
	default:
		l.B = v
	}
}

// a lamp that lights from everywhere at once, with no direction to give,
// lights a gas by half its light and no more: no phase, no shadow, and
// nothing taken by the gas on the way to the lamp.
//
// a lamp of no light is not asked where it is: a direction that is no
// number, from a lamp that gives nothing, makes no number of the pixel.
func TestAGasIsLitEvenlyByALampFromEverywhere(t *testing.T) {
	g := tinted(redOptics, greenOptics, blueOptics)
	w := thing.NewWorld()
	w.AddVolume(gasBlock(1, 1, g, 1, maths.Vec{}, 1))
	w.Lamp(&steady{far: math.Inf(1), light: warm})
	w.Lamp(&steady{toward: maths.V(math.NaN(), 0, 0), far: math.Inf(1)})
	got := Trace(w, along(0, 0), 0)

	var want maths.Light
	for c, o := range []gas.Optics{redOptics, greenOptics, blueOptics} {
		e := extinct(o)
		in := channel(warm, c) * 0.5 * o.Scatter
		thick, keep, sum := 1.0/24, 1.0, 0.0
		for i := 0; i < 48; i++ {
			sum += in * thick * keep
			keep *= math.Exp(-e * thick)
		}
		put(&want, c, sum)
	}
	colours(t, "a gas lit from everywhere", got, want, 1e-6)
}

// a surface that stands between a gas and the sun leaves the gas dark.
func TestASurfaceShadesAGasFromTheSun(t *testing.T) {
	g := tinted(redOptics, greenOptics, blueOptics)
	w := thing.NewWorld()
	w.AddVolume(gasBlock(1, 1, g, 1, maths.Vec{}, 1))
	w.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: &thing.Material{Name: "coal"},
		Motion: at(maths.V(4, 0, 0), 1)})
	w.Lamp(thing.Sun(maths.V(1, 0, 0), warm))
	if got := Trace(w, along(0, 0), 0); got.Light() != (maths.Light{}) {
		t.Errorf("the gas in the shade of a ball is lit: %v", got.Light())
	}
}

// one step of a ray in a gas gathers its light by the step's gain: twice
// the gain is twice what is gathered, and the gain does not touch what the
// step keeps.
//
// worked at the middle of the block, with a sun along plus x that has 1 of
// gas to cross: the light gathered by a step of thickness 0.1 is the sun,
// times exp(-e), times 3 over 16 pi, times the scattering, times 0.1,
// times the gain; and what the step keeps is exp(-0.1 e).
func TestAStepOfGasGathersByItsGain(t *testing.T) {
	g := tinted(redOptics, greenOptics, blueOptics)
	w := thing.NewWorld()
	w.AddVolume(gasBlock(1, 1, g, 1, maths.Vec{}, 1))
	w.Lamp(thing.Sun(maths.V(1, 0, 0), warm))
	sc := stage(w, 0)
	r := along(0, 0)
	c := crossings(sc.vs, r, math.Inf(1))[0]
	for _, gain := range []float64{1, 2.5} {
		keep, added := gasStep(sc, c, maths.Vec{}, maths.Vec{}, r.Dir, 0.1, gain,
			maths.Grey(1), maths.Light{})
		var wantKeep, wantAdded maths.Light
		for ch, o := range []gas.Optics{redOptics, greenOptics, blueOptics} {
			e := extinct(o)
			put(&wantKeep, ch, math.Exp(-0.1*e))
			put(&wantAdded, ch, channel(warm, ch)*math.Exp(-e)*3/(16*math.Pi)*
				o.Scatter*0.1*gain)
		}
		colours(t, "what a step keeps", fixed(keep), wantKeep, 1e-12)
		colours(t, "what a step gathers", fixed(added), wantAdded, 1e-12)
	}
}

// fixed is a light that can be checked as a pixel is.
type fixed maths.Light

func (f fixed) Light() maths.Light { return maths.Light(f) }

// a volume that is painted and lit by a lamp scatters what a gas in the
// way lets through. the sun along plus x has two gas blocks of density
// 1.5 between it and the painted one: a small one 2 across at 3 along
// plus x, and a large one round the painted one, 3 to its edge.
//
// together they take exp(-e 1.5 (2 + 3)) of each channel.
//
// the painted volume is white, 0.6 thick and 2 across, and lights half of
// the sun, less its own 1 of way out: exp(-0.6). a lamp that lights from
// everywhere is not taken by the gas at all.
func TestAPaintedVolumeIsLitThroughAGasInTheWay(t *testing.T) {
	g := tinted(redOptics, greenOptics, blueOptics)
	w := thing.NewWorld()
	w.AddVolume(thing.Volume{Medium: block{k: 0.6, half: 1},
		Material: &thing.Material{Colour: maths.Grey(1)}})
	w.AddVolume(gasBlock(1.5, 1, g, 1, maths.V(3, 0, 0), 1))
	w.AddVolume(gasBlock(1.5, 3, g, 1, maths.V(0, 0, 0), 1))
	w.Lamp(thing.Sun(maths.V(1, 0, 0), warm))
	sc := stage(w, 0)
	r := along(0, 0)
	here := maths.V(0, 0, 0.3)
	var painted crossing
	for _, c := range crossings(sc.vs, r, math.Inf(1)) {
		if c.v.gas == nil {
			painted = c
		}
	}
	got := scattered(sc, painted, here, here, r.Dir, true)
	var want maths.Light
	for ch, o := range []gas.Optics{redOptics, greenOptics, blueOptics} {
		e := extinct(o)
		put(&want, ch, channel(warm, ch)*0.5*math.Exp(-0.6)*
			math.Exp(-e*1.5*2)*math.Exp(-e*1.5*3))
	}
	colours(t, "a painted volume under a sun through a gas", fixed(got), want, 1e-12)

	w.Lamps = []thing.Lamp{&steady{far: math.Inf(1), light: warm}}
	got = scattered(stage(w, 0), painted, here, here, r.Dir, true)
	colours(t, "a painted volume lit from everywhere", fixed(got), warm.Scale(0.5), 1e-12)
}

// gasMedium is a gas that does not say where a ray crosses it but by its
// box, so the tracer asks the box.
type gasMedium struct {
	block
	g gas.Gas
}

func (m gasMedium) Gas() gas.Gas { return m.g }

var _ medium.Medium = gasMedium{}
