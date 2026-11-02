package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// an eye inside the field, looking straight out, is let go at once, and
// gathers on its straight way out to the edge of the reach the light of
// the glowing volumes along it, shifted.
//
// the shift is the fourth power of a still eye's ratio to a still
// emitter's: the root of one less 2 over r, for the emitter at r, over
// the root of one less 2 over 20, for the eye at 20 M. beyond the reach
// it sees the plain picture, with no shift.
//
// the eye is at 20 M, looking along minus z, and sees a white block at
// 30 M, tau 1, in the reach, a red block at 60 M, tau 1, past it, and
// then the sky, which is the way it looked, (0, 0, -1). every number is
// the same in a world twice the size, the field and the blocks.
//
// worked: the white block gives off 1 - exp(-1) times the gain at 30 M,
// ((1 - 2/30)/(1 - 2/20))^2, and lets exp(-1) through; the red one gives
// 1 - exp(-1) of red and lets the same through; the sky is behind both.
func TestALeavingRayGathersItsLegShifted(t *testing.T) {
	keep := math.Exp(-1)
	gain := math.Pow((1-2.0/30)/(1-2.0/20), 2)
	inside := (1 - keep) * gain
	for _, scale := range []float64{1, 2} {
		w := hole(thing.Still(maths.Subspace{Scale: scale}))
		w.AddVolume(thing.Volume{Medium: block{k: 0.5, half: 1}, Material: glowOf(maths.Grey(1)),
			Motion: at(maths.V(0, 0, -30*scale), scale)})
		w.AddVolume(thing.Volume{Medium: block{k: 0.5, half: 1}, Material: glowOf(maths.RGB(1, 0, 0)),
			Motion: at(maths.V(0, 0, -60*scale), scale)})
		got := Trace(w, maths.Ray{At: maths.V(0, 0, -20*scale), Dir: maths.V(0, 0, -1)}, 0)
		want := maths.RGB(inside+keep*((1-keep)+0), inside, inside-keep*keep)
		colours(t, "a ray out of the field", got, want, 1e-6)
		if want := 1 - keep*keep; !within(got.Cover, want, 1e-6) {
			t.Errorf("scale %v: covers %v, want %v", scale, got.Cover, want)
		}
		if !math.IsInf(float64(got.Depth), 1) {
			t.Errorf("scale %v: the sky is at %v", scale, got.Depth)
		}
	}
}

// a ray out of the field to a surface covers its pixel, and the depth it hands
// back is greater than zero and short of the way: a white wall across the way,
// 100 past the eye, behind a block that lets half through.
//
// the tracer, as it was first written, leaves the straight leg from the edge of
// the bending out of the depth, which this does not hold.
func TestALeavingRayThatMeetsASurfaceCoversTheWay(t *testing.T) {
	w := hole(nil)
	w.AddVolume(thing.Volume{Medium: block{k: math.Ln2 / 2, half: 1}, Material: glowOf(maths.Grey(0)),
		Motion: at(maths.V(0, 0, -30), 1)})
	w.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Glow: maths.Grey(1)},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 0, -100),
			Orientation: maths.Orientation{X: maths.XAxis, Y: maths.ZAxis, Z: maths.YAxis.Neg()}})})
	got := Trace(w, maths.Ray{At: maths.V(0, 0, -20), Dir: maths.V(0, 0, -1)}, 0)
	colours(t, "a wall seen through half", got, maths.Grey(0.5), 1e-6)
	if got.Cover != 1 {
		t.Errorf("the wall covers %v", got.Cover)
	}
	if !(got.Depth > 0 && got.Depth <= 80) {
		t.Errorf("the wall is at %v, want something short of 80 and above nought", got.Depth)
	}
}

// before is a world with a hole of mass 1 and a reach of 40, a compass
// sky, and a glowing ball of a given radius at a given place.
func before(at maths.Vec, radius float64) *thing.World {
	w := hole(nil)
	w.Add(thing.Thing{Shape: shape.Sphere{Radius: radius},
		Material: &thing.Material{Glow: maths.Grey(0.6)},
		Motion:   thing.Still(maths.Subspace{Origin: at})})
	return w
}

// what stands before the field is seen as it always was, and what is
// within is hidden by the hole when it is behind it. a glowing ball 55
// out is before the reach, which begins 60 along; the eye at 100 sees it
// as it is.
//
// one 20 beyond the hole, on the line through it, is behind the hole,
// and the ray into the hole is black.
//
// a wall exactly where the field is entered is before it, and seen; and a
// ball a tenth of a unit before an eye that is half a unit from the reach
// is seen, as the field is not yet entered.
func TestWhatLiesBeforeTheFieldIsSeenAsItIsAndWhatLiesBehindTheHoleIsHidden(t *testing.T) {
	ahead := maths.Ray{At: maths.V(0, 0, -100), Dir: maths.V(0, 0, 1)}
	colours(t, "a ball before the field", Trace(before(maths.V(0, 0, -55), 1), ahead, 0),
		maths.Grey(0.6), 1e-6)
	colours(t, "a ball behind the hole", Trace(before(maths.V(0, 0, 20), 1), ahead, 0),
		maths.Light{}, 0)

	w := hole(nil)
	w.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Glow: maths.Grey(0.6)},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 0, -40),
			Orientation: maths.Orientation{X: maths.XAxis, Y: maths.ZAxis, Z: maths.YAxis.Neg()}})})
	colours(t, "a wall at the reach", Trace(w, ahead, 0), maths.Grey(0.6), 1e-6)

	near := maths.Ray{At: maths.V(0, 0, -40.5), Dir: maths.V(0, 0, 1)}
	colours(t, "a ball by the eye", Trace(before(maths.V(0, 0, -40.3), 0.1), near, 0),
		maths.Grey(0.6), 1e-6)
}

// what lies between the eye and the field is seen through, as it is
// anywhere: a glowing block, tau 1, 70 out before a reach 60 along,
// gives off 1 - exp(-1) of its glow before the ray goes in, and the ray
// that goes into the hole is black beyond it.
//
// so is a block in the half unit between an eye 40.5 out and the reach,
// of tau 0.2.
func TestWhatLiesBetweenTheEyeAndTheFieldIsSeenThrough(t *testing.T) {
	into := maths.Ray{At: maths.V(0, 0, -100), Dir: maths.V(0, 0, 1)}
	w := hole(nil)
	w.AddVolume(thing.Volume{Medium: block{k: 0.5, half: 1}, Material: glowOf(maths.RGB(1, 0.5, 0.25)),
		Motion: at(maths.V(0, 0, -70), 1)})
	colours(t, "a block in front of the hole", Trace(w, into, 0),
		maths.RGB(1, 0.5, 0.25).Scale(1-math.Exp(-1)), 1e-6)

	w = hole(nil)
	w.AddVolume(thing.Volume{Medium: block{k: 0.5, half: 0.2}, Material: glowOf(maths.RGB(1, 0.5, 0.25)),
		Motion: at(maths.V(0, 0, -40.25), 1)})
	colours(t, "a block in the gap by the reach", Trace(w, maths.Ray{At: maths.V(0, 0, -40.5), Dir: maths.V(0, 0, 1)}, 0),
		maths.RGB(1, 0.5, 0.25).Scale(1-math.Exp(-0.2)), 1e-6)
}

// a field of a mass so small it bends nothing a picture shows is a
// window: every ray goes straight through it, and the picture is the
// one a hand can work. a ray at 3 from the line through the hole, 100
// out, runs past glowing blocks.
//
// a red one is before the field, at 70 out, a green one in it on the way
// in, at 20, a blue one in it on the way out, 20 past the hole, and a
// yellow one beyond the reach, 70 past it.
//
// the sky is the way it went, (0, 0, 1). each block is tau 1, and lets
// exp(-1) through what is behind it, so red is 1 - e, green e (1 - e),
// blue e^2 (1 - e), yellow e^3 (1 - e) of red and green, and the sky e^4
// of blue.
//
// a ball 10 before the hole, 0.5 of grey, in the way in, takes the place
// of what lies beyond it: its glow behind red and green, times e^2; and
// the depth to it is the way to the first step that meets it, within a
// step of 89.
//
// that the blocks on the way out are seen is the leg the ray gathers once
// it is let go at the hole, which a ball there would not be: the tracer
// gathers volumes on the leg and not surfaces.
func TestAFieldOfNoMassIsAWindow(t *testing.T) {
	const small = 1e-6
	window := func() *thing.World {
		w := thing.NewWorld()
		w.Lamps = nil
		w.Sky = compass{}
		w.AddField(thing.Field{Mass: small, Reach: 40})
		for _, v := range []struct {
			z     float64
			glows maths.Light
		}{{-70, maths.RGB(1, 0, 0)}, {-20, maths.RGB(0, 1, 0)}} {
			w.AddVolume(thing.Volume{Medium: block{k: 0.5, half: 1}, Material: glowOf(v.glows),
				Motion: at(maths.V(0, 3, v.z), 1)})
		}
		return w
	}
	ray := maths.Ray{At: maths.V(0, 3, -100), Dir: maths.V(0, 0, 1)}
	e := math.Exp(-1)

	w := window()
	w.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: &thing.Material{Glow: maths.Grey(0.5)},
		Motion: at(maths.V(0, 3, -10), 1)})
	got := Trace(w, ray, 0)
	colours(t, "red, green and a ball", got, maths.RGB((1-e)+e*e*0.5, e*(1-e)+e*e*0.5, e*e*0.5), 1e-4)
	if got.Cover != 1 || !(got.Depth > 87 && got.Depth <= 89.01) {
		t.Errorf("the ball covers %v at %v, want 1 at about 89", got.Cover, got.Depth)
	}

	w = window()
	w.AddVolume(thing.Volume{Medium: block{k: 0.5, half: 1}, Material: glowOf(maths.RGB(0, 0, 1)),
		Motion: at(maths.V(0, 3, 20), 1)})
	w.AddVolume(thing.Volume{Medium: block{k: 0.5, half: 1}, Material: glowOf(maths.RGB(1, 1, 0)),
		Motion: at(maths.V(0, 3, 70), 1)})
	got = Trace(w, ray, 0)
	colours(t, "red, green, blue, yellow and the sky", got,
		maths.RGB((1-e)*(1+e*e*e), (1-e)*(e+e*e*e), e*e*(1-e)+e*e*e*e), 1e-4)
	if want := 1 - e*e*e*e; !within(got.Cover, want, 1e-4) {
		t.Errorf("covers %v, want %v", got.Cover, want)
	}
}

// a march ends when what lies behind is gone: through a block on the way
// in so thick it lets under a thousandth through, a ray in the field
// stops, and the colour is the block's glow and nothing of the sky
// behind it.
//
// the sky is blue only in its blue, so the red and the blue of the pixel
// are the same only if the sky is not added.
func TestAMarchEndsWhenNothingBehindShows(t *testing.T) {
	w := thing.NewWorld()
	w.Lamps = nil
	w.Sky = compass{}
	w.AddField(thing.Field{Mass: 1e-6, Reach: 40})
	w.AddVolume(thing.Volume{Medium: block{k: 4, half: 1}, Material: glowOf(maths.Grey(1)),
		Motion: at(maths.V(0, 3, -20), 1)})
	got := Trace(w, maths.Ray{At: maths.V(0, 3, -100), Dir: maths.V(0, 0, 1)}, 0)
	if got.R != got.B || !(got.R > 0.999 && got.R < 1) {
		t.Errorf("through the thick block: %+v, want grey just under 1", got)
	}
}
