package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// a floor that casts no shadow, under a sun beneath it, has none of its
// light: a surface turned away from a lamp is dark, and not lit by less
// than nothing.
func TestASurfaceTurnedFromALampIsDarkNotNegative(t *testing.T) {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Colour: chalk},
		Attrs: thing.Attrs{Clear: true}})
	w.Lamp(thing.Sun(maths.V(0, -1, 0), maths.Grey(1)))
	colours(t, "a floor turned from the sun", Trace(w, straightDown(), 0), maths.Light{}, 0)
}

// a ray that touches a ball only at its edge sees a surface that faces
// it side-on, and the side it faces is the one the ray came from.
//
// a ray along plus x over the top of a unit ball, touching it at
// (0, 1, 0), meets a normal straight up, which is square on to a flood
// from above, and the flood lights it by all it has.
func TestARayThatTouchesABallAtItsEdgeSeesItsOutside(t *testing.T) {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: &thing.Material{Colour: chalk}})
	w.Lamp(film.Flood{Strength: maths.Grey(1)})
	got := Trace(w, maths.Ray{At: maths.V(-5, 1, 0), Dir: maths.V(1, 0, 0)}, 0)
	colours(t, "a ball touched at its top", got, chalk, 2e-6)
	if got.Depth != 5 {
		t.Errorf("the ball is touched at %v, want 5", got.Depth)
	}
}

// a floor under a gas is lit by what the gas lets through. a floor of
// chalk under a sun overhead, with a block of gas that only absorbs, the
// whole 2 above it, is dimmed twice, once from the sun to the floor and
// once from the floor to the eye, each time by exp(-2e).
//
// the light is chalk times sun times exp(-4e) in each channel, with e
// 0.9, 0.5 and 0.2 per metre.
func TestAFloorUnderAGasIsLitThroughIt(t *testing.T) {
	smoke := tinted(gas.Optics{Absorb: 0.9}, gas.Optics{Absorb: 0.5}, gas.Optics{Absorb: 0.2})
	w := floor()
	w.AddVolume(gasBlock(1, 1, smoke, 1, maths.V(0, 1, 0), 1))
	sun := maths.RGB(1, 0.8, 0.5)
	w.Lamp(thing.Sun(maths.V(0, 1, 0), sun))
	want := chalk.Times(sun).Times(maths.RGB(math.Exp(-4*0.9), math.Exp(-4*0.5), math.Exp(-4*0.2)))
	// the shadow ray starts a hair above the floor, which is a hair less
	// than 2 of gas
	colours(t, "a floor under a gas", Trace(w, straightDown(), 0), want, 1e-3)
}

// a gas thin enough to take nothing of the light behind it, its density
// being under what a double can take from one, still gives off what it
// scatters.
//
// the light is the sun times the phase times the scattering times the
// thickness of each step, summed over the 48 steps.
func TestAThinGasStillShines(t *testing.T) {
	g := tinted(gas.Optics{Scatter: 1e-9}, gas.Optics{Scatter: 1e-9}, gas.Optics{Scatter: 1e-9})
	w := thing.NewWorld()
	w.AddVolume(gasBlock(1e-9, 1, g, 1, maths.Vec{}, 1))
	w.Lamp(thing.Sun(maths.V(1, 0, 0), maths.Grey(1)))
	got := Trace(w, along(0, 0), 0)
	want := 3 / (16 * math.Pi) * 1e-9 * (1e-9 * 2)
	for _, p := range []float32{got.R, got.G, got.B} {
		if p == 0 || math.Abs(float64(p)/want-1) > 1e-6 {
			t.Errorf("the thin gas shines %v, want %v", p, want)
		}
	}
}

// negative is a sky whose red is minus zero, and is otherwise dim.
type negative struct{}

func (negative) Light(maths.Vec) maths.Light {
	return maths.Light{R: math.Copysign(0, -1), G: 0.2, B: 0.3}
}

// a ray that meets no volume comes back as the plain picture gives it, bit for
// bit, down to the sign of a zero; and a picture sampled at one across is the
// plain picture bit for bit as well.
func TestWhatNoVolumeTouchesIsUntouchedToTheBit(t *testing.T) {
	w := thing.NewWorld()
	w.Sky = negative{}
	w.AddVolume(thing.Volume{Medium: block{k: 1, half: 1}, Material: glowOf(maths.Grey(1)),
		Motion: at(maths.V(50, 0, 0), 1)})
	got := Trace(w, along(0, 0), 0)
	if !math.Signbit(float64(got.R)) || got.G != 0.2 {
		t.Errorf("a ray that meets no volume comes back %+v", got)
	}

	plain := thing.NewWorld()
	plain.Sky = negative{}
	from := maths.V(0, 2, 6)
	eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, centre)}
	lens := film.Pinhole{Wide: 60 * maths.Degree}
	a, b := Render(plain, eye, lens, 0, 3, 3), Sampled(plain, eye, lens, 0, 3, 3, 1)
	for i := range a.Pix {
		if !samePixel(a.Pix[i], b.Pix[i]) {
			t.Fatalf("pixel %d is %+v sampled once, %+v plain", i, b.Pix[i], a.Pix[i])
		}
	}
}

// samePixel is whether two pixels are the same bit for bit.
func samePixel(a, b frame.Pixel) bool {
	same := func(x, y float32) bool { return math.Float32bits(x) == math.Float32bits(y) }
	return same(a.R, b.R) && same(a.G, b.G) && same(a.B, b.B) &&
		same(a.Cover, b.Cover) && same(a.Depth, b.Depth)
}
