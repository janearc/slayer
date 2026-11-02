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

// an eye inside a volume sees the way from the eye to the far side, not
// the way from a unit out: a block 2 across and a ball 2 across, thick
// 0.5, from 0.4 in front of the middle, are 1.4 deep to the back, so tau
// is 0.7 and the glow is shown by 1 - exp(-0.7).
func TestAnEyeInsideAVolumeSeesFromTheEye(t *testing.T) {
	glow := maths.RGB(0.9, 0.5, 0.2)
	for _, m := range []medium.Medium{block{k: 0.5, half: 1}, ball{k: 0.5, radius: 1}} {
		w := thing.NewWorld()
		w.AddVolume(thing.Volume{Medium: m, Material: glowOf(glow)})
		got := Trace(w, maths.Ray{At: maths.V(0, 0, 0.4), Dir: maths.V(0, 0, -1)}, 0)
		colours(t, "an eye inside", got, glow.Scale(1-math.Exp(-0.7)), 2e-6)
	}
}

// a medium that says it was not crossed is not drawn, whatever enter and
// leave it also gives: a block of half-width 100 round the ray, thick
// enough to be seen, that says it was not crossed.
//
// a block that says it was crossed from 9 to 2, backward, and one from
// 2 to 2, a way of no length, show nothing either.
func TestAVolumeThatWasNotCrossedIsNotDrawn(t *testing.T) {
	for _, m := range []told{
		{block{k: 1, half: 100}, 2, 9, false},
		{block{k: 1, half: 100}, 9, 2, true},
		{block{k: 1, half: 100}, 2, 2, true},
	} {
		w := thing.NewWorld()
		w.AddVolume(thing.Volume{Medium: m, Material: glowOf(maths.Grey(1))})
		if got := Trace(w, along(0, 0), 0); got.R != 0 || got.Cover != 0 {
			t.Errorf("a %v that was not crossed gives %+v", m, got)
		}
	}
}

// what is hidden is not drawn and does not shade: a hidden ball over a
// floor under a sun leaves the floor as lit as an empty sky would, and a
// hidden or a mediumless volume adds nothing to a ray, nor stops one.
func TestWhatIsHiddenIsLeftOut(t *testing.T) {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Colour: maths.Grey(1)}})
	w.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: &thing.Material{Colour: maths.Grey(1)},
		Motion: at(maths.V(0, 3, 0), 1), Attrs: thing.Attrs{Hidden: true}})
	w.AddVolume(thing.Volume{Medium: block{k: 100, half: 2}, Material: glowOf(maths.Grey(1)),
		Motion: at(maths.V(0, 3, 0), 1), Attrs: thing.Attrs{Hidden: true}})
	w.AddVolume(thing.Volume{Material: glowOf(maths.Grey(1))})
	w.Lamp(thing.Sun(maths.V(0, 1, 0), maths.Grey(1)))
	got := Trace(w, maths.Ray{At: maths.V(0, 10, 0), Dir: maths.V(0, -1, 0)}, 0)
	if got.R != 1 || got.Cover != 1 || got.Depth != 10 {
		t.Errorf("the floor under hidden things is %+v, want lit 1 at 10", got)
	}
	if n := len(stage(w, 0).ps) + len(stage(w, 0).vs); n != 1 {
		t.Errorf("%d things and volumes are placed, want the floor alone", n)
	}
}

// a gas that does not say where a ray crosses it but by its box has the
// box asked, from a point inside it as from outside: a block 1 across,
// seen from its middle, is left 0.5 on, and not before the first unit.
func TestGasClearFromInsideAGasThatIsToldByItsBox(t *testing.T) {
	g := tinted(redOptics, greenOptics, blueOptics)
	v := thing.Volume{Medium: gasMedium{block{k: 1, half: 0.5}, g}}
	got := gasClear(volumes(v), maths.Vec{}, maths.V(0, 1, 0))
	want := maths.RGB(math.Exp(-0.9*0.5), math.Exp(-0.5*0.5), math.Exp(-0.2*0.5))
	if !near(got.R, want.R, 1e-12) || !near(got.G, want.G, 1e-12) || !near(got.B, want.B, 1e-12) {
		t.Errorf("from the middle of a gas 1 across: %v, want %v", got, want)
	}
	var _ gas.Optical = gasMedium{}
}
