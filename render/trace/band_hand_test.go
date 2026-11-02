package trace

import (
	"math"
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	tracer "github.com/janearc/slayer/trace"
)

// radiates is a radiant with one number for every band: it lets so much of a
// unit through, gives off so much, and sends so much of a radar back.
type radiates struct{ through, gives, back float64 }

func (r radiates) Through(maths.Band) float64  { return r.through }
func (r radiates) Emits(maths.Band) float64    { return r.gives }
func (r radiates) Scatters(maths.Band) float64 { return r.back }

// lightLed is a radiant that glows where the lamps reach it, by the
// lamplight to the power of its excitation.
type lightLed struct {
	radiates
	excitation float64
}

func (l lightLed) Excitation(maths.Band) float64 { return l.excitation }

// xray is a band of x-rays, and micro one of microwaves, 12 and a half
// centimetres, which is radio as libtheme draws the ranges.
var (
	xray  = maths.Band{Nanometres: 1}
	micro = maths.Band{Nanometres: 1.26e8}
)

// bandStage is a world as a picture in a band sees it.
func bandStage(w *thing.World) staged {
	return staged{w: w, ps: place(w, 0), vs: placeVolumes(w, 0),
		spread: tracer.Spread(w, 0)}
}

// out is a ray from the origin along minus z.
var out = maths.Ray{Dir: maths.V(0, 0, -1)}

// radiantBall is a ball z along the line of sight that does what a
// radiant says.
func radiantBall(z, radius float64, r thing.Radiant) thing.Thing {
	return thing.Thing{Shape: shape.Sphere{Radius: radius}, Material: &thing.Material{Radiant: r},
		Motion: at(maths.V(0, 0, z), 1)}
}

// in a band a thing gives off what it gives off, whole where the ray
// meets it, and covers its pixel, at the distance it is. a ball of radius
// 1 at 20 gives 5 at 19; one with its near face a half unit from the
// ray's start, which is under a unit, gives it at a half.
//
// a ray that meets nothing is dark and covers nothing, at no distance.
// a volume in the way thins what is behind it by exp(-tau), tau 1 for a
// block of thickness 0.5 and 2 across, and covers by what it took,
// 1 - exp(-1), against the sky.
func TestABandSeesTheFirstThingThatGivesItOff(t *testing.T) {
	gives := radiates{gives: 5}
	for _, c := range []struct {
		name  string
		w     func() *thing.World
		ray   maths.Ray
		light float64
		cover float32
		depth float64
	}{
		{"a ball", func() *thing.World {
			w := thing.NewWorld()
			w.Add(radiantBall(-20, 1, gives))
			return w
		}, out, 5, 1, 19},
		{"a ball a half unit away", func() *thing.World {
			w := thing.NewWorld()
			w.Add(radiantBall(-1.5, 1, gives))
			return w
		}, out, 5, 1, 0.5},
		{"nothing", thing.NewWorld, out, 0, 0, math.Inf(1)},
		{"a ball behind a block", func() *thing.World {
			w := thing.NewWorld()
			w.Add(radiantBall(-20, 1, gives))
			w.AddVolume(thing.Volume{Medium: block{k: 0.5, half: 1}, Motion: at(maths.V(0, 0, -10), 1)})
			return w
		}, out, 5 * math.Exp(-1), 1, 19},
		{"the sky behind a block", func() *thing.World {
			w := thing.NewWorld()
			w.AddVolume(thing.Volume{Medium: block{k: 0.5, half: 1}, Motion: at(maths.V(0, 0, -10), 1)})
			return w
		}, out, 0, float32(1 - math.Exp(-1)), math.Inf(1)},
	} {
		got := inBand(bandStage(c.w()), c.ray, xray, passes)
		if !within(got.R, c.light, 1e-6) || got.R != got.G || got.G != got.B ||
			!within(got.Cover, float64(c.cover), 1e-6) || float64(got.Depth) != c.depth {
			t.Errorf("%s: %+v, want %v covering %v at %v", c.name, got, c.light, c.cover, c.depth)
		}
	}
}

// a ray in a band passes through eight things that pass the band and
// sees the ninth, the number it began with, and no more. a row of balls, each
// passing the whole of the band, with the last giving off 3, is seen
// when it is the ninth and not when it is the tenth.
//
// what is behind eight things is taken as dark. this is through a whole
// picture of one pixel.
func TestABandPassesEightThingsAndSeesTheNinth(t *testing.T) {
	row := func(n int) float32 {
		w := thing.NewWorld()
		for i := 1; i <= n; i++ {
			r := radiates{through: 1}
			if i == n {
				r.gives = 3
			}
			w.Add(radiantBall(-2*float64(i), 0.4, r))
		}
		eye := maths.Subspace{}
		return RenderIn(w, eye, film.Pinhole{Wide: 10 * maths.Degree}, 0, 1, 1, xray).At(0, 0).R
	}
	for _, c := range []struct {
		things int
		want   float32
	}{{1, 3}, {8, 3}, {9, 3}, {10, 0}, {12, 0}} {
		if got := row(c.things); got != c.want {
			t.Errorf("a row of %d: %v, want %v", c.things, got, c.want)
		}
	}
}

// what passes a band passes it by its number to the power of the way
// through, in its own units. a ball of radius 1 scaled by 2, which
// passes a half of a unit of the band, is 2 across in its own units and
// lets 0.25 through, whatever it is in the world.
//
// an eye inside a ball of radius a half sees the way out, a half, and
// 0.5 to the half through it, and not nothing, though it is under a unit.
//
// the source behind gives 5, at 20. a second ball touching the first,
// which passes half a unit of the band and is 2 across, is entered where
// the first is left, and takes the rest by its own way: 0.5 to the 2.
func TestAThingPassesABandByItsWayThrough(t *testing.T) {
	source := radiates{gives: 5}
	scaled := func() *thing.World {
		w := thing.NewWorld()
		w.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: &thing.Material{Radiant: radiates{through: 0.5}},
			Motion: at(maths.V(0, 0, -10), 2)})
		w.Add(radiantBall(-20, 1, source))
		return w
	}
	got := inBand(bandStage(scaled()), out, xray, passes)
	if !within(got.R, 5*0.25, 1e-6) || !within(got.Depth, 8, 1e-5) {
		t.Errorf("through a ball doubled: %+v, want 1.25 at 8", got)
	}

	inside := thing.NewWorld()
	inside.Add(radiantBall(0, 0.5, radiates{through: 0.5}))
	inside.Add(radiantBall(-20, 1, source))
	got = inBand(bandStage(inside), out, xray, passes)
	if want := 5 * math.Sqrt(0.5); !within(got.R, want, 1e-6) || !within(got.Depth, 0.5, 1e-6) {
		t.Errorf("from inside a ball: %+v, want %v at a half", got, want)
	}

	touching := thing.NewWorld()
	touching.Add(radiantBall(0, 1, radiates{through: 0.5}))
	touching.Add(radiantBall(-2, 1, radiates{through: 0.5}))
	touching.Add(radiantBall(-20, 1, source))
	got = inBand(bandStage(touching), maths.Ray{At: maths.V(0, 0, 5), Dir: maths.V(0, 0, -1)}, xray, passes)
	if want := 5 * 0.25 * 0.25; !within(got.R, want, 1e-3*want) {
		t.Errorf("through two balls touching: %v, want %v", got.R, want)
	}
}

// a thing that has a radiant and lets none of the band through is opaque
// to it: what it gives off is all there is. the source behind gives 5.
func TestAThingThatPassesNothingShowsOnlyItself(t *testing.T) {
	w := thing.NewWorld()
	w.Add(radiantBall(-5, 1, radiates{gives: 2}))
	w.Add(radiantBall(-20, 1, radiates{gives: 5}))
	if got := inBand(bandStage(w), out, xray, passes); got.R != 2 {
		t.Errorf("an opaque thing in front gives %v, want 2", got.R)
	}
}

// a volume in a band thins the band by beer's law as it thins light when
// it has nothing to say of the band, or has no material, or is a gas
// with a number for the band.
//
// for a block of density 0.5, 2 across, tau is 1 and exp(-1) is left; the
// same for a block twice the size, which is as thick through the middle
// in its own units.
//
// the gas has an absorption of 0.3 and a scattering of 0.1 in x-rays a
// metre, and 2 metres to a unit, so an extinction of 0.8 a unit, tau 0.8.
// a block that passes half a unit of the band, of density 1, 2 across,
// lets half to the 2 through. what is behind is a ball that gives 5.
func TestAVolumeThinsABandByBeersLaw(t *testing.T) {
	g := gas.Gas{Name: "x", Ranges: map[radiation.Range]gas.Optics{radiation.XRay(): {Absorb: 0.3, Scatter: 0.1}}}
	for _, c := range []struct {
		name string
		v    thing.Volume
		tau  float64
	}{
		{"a block with no material", thing.Volume{Medium: block{k: 0.5, half: 1}}, 1},
		{"a block that says nothing of the band", thing.Volume{Medium: block{k: 0.5, half: 1},
			Material: &thing.Material{Colour: maths.Grey(1)}}, 1},
		{"the same doubled", thing.Volume{Medium: block{k: 0.5, half: 1}, Motion: at(maths.Vec{}, 2)}, 1},
		{"a gas", thing.Volume{Medium: gas.Of{Shape: block{k: 0.5, half: 1}, In: g, Unit: 2}}, 0.8},
		{"a gas doubled", thing.Volume{Medium: gas.Of{Shape: block{k: 0.5, half: 1}, In: g, Unit: 2},
			Motion: at(maths.Vec{}, 2)}, 0.8},
		{"a block that passes a band by 0.5 a unit", thing.Volume{Medium: block{k: 1, half: 1},
			Material: &thing.Material{Radiant: radiates{through: 0.5}}}, 2 * math.Ln2},
		{"the same doubled", thing.Volume{Medium: block{k: 1, half: 1},
			Material: &thing.Material{Radiant: radiates{through: 0.5}}, Motion: at(maths.Vec{}, 2)}, 2 * math.Ln2},
	} {
		w := thing.NewWorld()
		w.Add(radiantBall(-20, 1, radiates{gives: 5}))
		v := c.v
		if v.Motion == nil {
			v.Motion = at(maths.V(0, 0, -10), 1)
		} else {
			v.Motion = at(maths.V(0, 0, -10), 2)
		}
		w.AddVolume(v)
		got := inBand(bandStage(w), out, xray, passes)
		if want := 5 * math.Exp(-c.tau); !within(got.R, want, 1e-6) {
			t.Errorf("%s: %v, want %v", c.name, got.R, want)
		}
	}
}

// a volume that gives off a band gives off as much as it would absorb of
// light, step by step, of what has been let through before it.
//
// a block of density 1 and 2 across in 48 steps of a 24th, of a band it
// passes 0.9 of a unit of and gives off 1: each step gives 1 - exp(-1/24)
// of what is left, and leaves 0.9 to the 1/24 of it. the sum is worked
// step by step, and is the same at twice the size.
func TestAVolumeGivesOffABandAsItAbsorbsLight(t *testing.T) {
	for _, scale := range []float64{1, 2} {
		w := thing.NewWorld()
		w.AddVolume(thing.Volume{Medium: block{k: 1, half: 1},
			Material: &thing.Material{Radiant: radiates{through: 0.9, gives: 1}},
			Motion:   at(maths.V(0, 0, -10), scale)})
		keep, added := 1.0, 0.0
		for i := 0; i < 48; i++ {
			added += keep * (1 - math.Exp(-1.0/24))
			keep *= math.Pow(0.9, 1.0/24)
		}
		got := inBand(bandStage(w), out, xray, passes)
		if !within(got.R, added, 1e-6) || !within(got.Cover, 1-keep, 1e-6) {
			t.Errorf("scale %v: %v covering %v, want %v covering %v", scale, got.R, got.Cover, added, 1-keep)
		}
	}
}

// radar is a radar of microwaves from a place.
func radar(at maths.Vec, strength float64) film.Radar {
	return film.Radar{At: at, Range: radiation.Radio(), Strength: strength}
}

// a radar's echo is what the thing sends back of it: how squarely the
// beam lands, how rough the thing is, and how much it returns, times the
// radar's strength, and what the beam kept on the way.
//
// worked, at the middle of a ball of radius 2 that sends back 0.8, with a
// radar at the eye, square on, of strength 2: 0.8 by 2, 1.6.
//
// with a block of density 1, 2 across, that passes half a unit of
// microwave between the radar and the ball, the beam keeps 0.25 each way,
// and the picture keeps 0.25 again of the echo it sees: 1.6 times 0.25
// times 0.25.
//
// a sun in the world, which is no radar, is not asked for a beam. a radar
// behind the ball, a radar that gives nothing or less than nothing, and
// a radar whose beam is stopped by a ball, light nothing; and a thing
// that sends back less than nothing sends back nothing.
func TestARadarEchoesByItsStrengthAndWhatItKept(t *testing.T) {
	eye := maths.V(0, 0, 10)
	look := maths.Ray{At: eye, Dir: maths.V(0, 0, -1)}
	world := func(back float64) *thing.World {
		w := thing.NewWorld()
		w.Add(radiantBall(0, 2, radiates{back: back}))
		w.Lamp(thing.Sun(maths.V(0, 1, 0), maths.Grey(1)))
		return w
	}
	pixel := func(w *thing.World) float32 {
		return inBand(bandStage(w), look, micro, passes).R
	}
	w := world(0.8)
	w.Lamp(radar(eye, 2))
	if got := pixel(w); !within(got, 1.6, 1e-6) {
		t.Errorf("a radar of 2 at the eye: %v, want 1.6", got)
	}

	// a block between the radar and the ball
	w = world(0.8)
	w.Lamp(radar(eye, 2))
	w.AddVolume(thing.Volume{Medium: block{k: 1, half: 1},
		Material: &thing.Material{Radiant: radiates{through: 0.5}}, Motion: at(maths.V(0, 0, 5), 1)})
	if got := pixel(w); !within(got, 1.6*0.25*0.25, 1e-6) {
		t.Errorf("a radar through a block: %v, want %v", got, 1.6*0.25*0.25)
	}

	// radars that light nothing: behind, giving nothing, giving less
	// than nothing, shadowed by a wall that is not in the way of the eye
	w = world(0.8)
	w.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.5}, Motion: at(maths.V(0, 5.66, 7.66), 1)})
	w.Lamp(radar(maths.V(0, 0, -10), 2))
	w.Lamp(radar(eye, 0))
	w.Lamp(radar(eye, -3))
	w.Lamp(radar(maths.V(0, 8, 10), 2))
	if got := pixel(w); got != 0 {
		t.Errorf("radars that light nothing: %v", got)
	}

	// a thing that sends back less than nothing sends back nothing
	w = world(-0.5)
	w.Lamp(radar(eye, 2))
	if got := pixel(w); got != 0 {
		t.Errorf("a thing that returns less than nothing: %v", got)
	}
}

// a radar lights the side of a thing that faces it, and the inside of a
// hollow one from within, and the side of a thing it only touches that
// the radar is on. an eye inside a ball of radius 2 with a radar at the
// eye sees the inner wall square on, 0.8 of 2.
//
// a radar at (-5, 8, 0) over a ball of radius 1 that a ray touches at its
// top, (0, 1, 0), sees the top at the cosine of the angle between the
// beam and straight up: the beam is (-5, 7, 0), so 7 over root 74, times
// 0.8.
func TestARadarLightsTheSideTheEyeIsOn(t *testing.T) {
	hollow := thing.NewWorld()
	hollow.Add(thing.Thing{Shape: shape.Sphere{Radius: 2}, Material: &thing.Material{Radiant: radiates{back: 0.8}},
		Attrs: thing.Attrs{Clear: true}})
	hollow.Lamp(radar(maths.Vec{}, 2))
	if got := inBand(bandStage(hollow), out, micro, passes).R; !within(got, 1.6, 1e-6) {
		t.Errorf("the inside of a ball: %v, want 1.6", got)
	}

	top := thing.NewWorld()
	top.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: &thing.Material{Radiant: radiates{back: 0.8}}})
	top.Lamp(radar(maths.V(-5, 8, 0), 1))
	touch := maths.Ray{At: maths.V(-5, 1, 0), Dir: maths.V(1, 0, 0)}
	// the beam from the top (0, 1, 0) to the radar is (-5, 7, 0), of length root 74
	if got, want := inBand(bandStage(top), touch, micro, passes).R, 0.8*7/math.Sqrt(74); !within(got, want, 1e-6) {
		t.Errorf("a ball touched at its top: %v, want %v", got, want)
	}
}

// a radar on the far side of a thing from the eye, one that casts no
// shadow, lights nothing the eye sees of it: the beam lands on its other
// face, and a face turned away is dark, not negative.
func TestARadarBehindAThingLightsNothingOfItsNearFace(t *testing.T) {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: shape.Sphere{Radius: 2}, Material: &thing.Material{Radiant: radiates{back: 0.8}},
		Attrs: thing.Attrs{Clear: true}})
	w.Lamp(radar(maths.V(0, 0, -10), 2))
	look := maths.Ray{At: maths.V(0, 0, 10), Dir: maths.V(0, 0, -1)}
	if got := inBand(bandStage(w), look, micro, passes).R; got != 0 {
		t.Errorf("a radar behind a ball: %v, want dark", got)
	}
}

// what an excited gas gives off is the light that reaches it to the power
// of its excitation, found by the lamps' luminance past what shades it:
// a surface in the way shades it all, and a gas in the way shades it by
// the luminance of what it lets through.
//
// a block of density 0.2, 2 across, of excitation one, passing all of the
// band and giving off 1, is lit by a sun of 0.8 from along plus x that
// has 1 of the block to cross itself: each step is lit by 0.8 exp(-0.2),
// gives 1 - exp(-0.2/24) of that, and 48 steps of them are the pixel.
//
// a gas block 2 across between the sun and the block, at 3 along plus x,
// of density 1.5 and the tinted extinctions, takes the luminance of
// exp(-e k 2) across the channels on top.
func TestExcitedGasGlowsByTheLightThatReachesIt(t *testing.T) {
	g := tinted(redOptics, greenOptics, blueOptics)
	glowing := func() *thing.World {
		w := thing.NewWorld()
		w.AddVolume(thing.Volume{Medium: block{k: 0.2, half: 1},
			Material: &thing.Material{Radiant: lightLed{radiates{through: 1, gives: 1}, 1}}})
		w.Lamp(thing.Sun(maths.V(1, 0, 0), maths.Grey(0.8)))
		return w
	}
	sum := func(light float64) float64 {
		return 48 * (1 - math.Exp(-0.2/24)) * light
	}
	pixel := func(w *thing.World) float32 {
		return inBand(bandStage(w), maths.Ray{At: maths.V(0, 0, 10), Dir: maths.V(0, 0, -1)}, xray, passes).R
	}
	if got, want := pixel(glowing()), sum(0.8*math.Exp(-0.2)); !within(got, want, 1e-6) {
		t.Errorf("a lit block: %v, want %v", got, want)
	}

	shaded := glowing()
	shaded.Add(thing.Thing{Shape: shape.Sphere{Radius: 1.5}, Motion: at(maths.V(3, 0, 0), 1)})
	if got := pixel(shaded); got != 0 {
		t.Errorf("a block behind a ball from the sun: %v, want dark", got)
	}

	veiled := glowing()
	veiled.AddVolume(gasBlock(1.5, 1, g, 1, maths.V(3, 0, 0), 1))
	across := maths.RGB(math.Exp(-0.9*1.5*2), math.Exp(-0.5*1.5*2), math.Exp(-0.2*1.5*2))
	if got, want := pixel(veiled), sum(0.8*math.Exp(-0.2)*across.Luminance()); !within(got, want, 1e-6) {
		t.Errorf("a block behind a gas from the sun: %v, want %v", got, want)
	}
}

// a picture in the zero band is the whole visible, which is Render's, lit
// as Render lights it: a floor under a sun is lit in the zero band, and
// is dark in a band without a lamp for it. and every pixel of a picture
// in a band is drawn.
func TestAPictureInABandIsDrawnWholeAndTheZeroBandIsRender(t *testing.T) {
	w := floor()
	w.Lamp(thing.Sun(maths.V(0, 1, 0), maths.Grey(1)))
	from := maths.V(0, 10, 0)
	eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.Vec{})}
	lens := film.Pinhole{Wide: 60 * maths.Degree}
	if got, want := RenderIn(w, eye, lens, 0, 4, 3, maths.Band{}), Render(w, eye, lens, 0, 4, 3); got.At(1, 1) != want.At(1, 1) || got.At(0, 0).R == 0 {
		t.Errorf("the zero band is %+v, Render's is %+v", got.At(1, 1), want.At(1, 1))
	}

	glowy := thing.NewWorld()
	glowy.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Radiant: radiates{gives: 1}}})
	f := RenderIn(glowy, eye, lens, 0, 4, 3, xray)
	for y := 0; y < 3; y++ {
		for x := 0; x < 4; x++ {
			if p := f.At(x, y); p.R != 1 || p.Cover != 1 {
				t.Errorf("the pixel at %d, %d is %+v", x, y, p)
			}
		}
	}
}
