package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/thing"
)

// the tests here hold that nothing is taken from where nothing is met, by
// giving the nothing an endless neighbour: a lamp of endless strength, a medium
// of endless thickness, a thing that returns endless.
//
// zero times endless is no number, so a step of zero, a face edge-on or a way
// of no length that is multiplied and not skipped makes the pixel no number.
// none of this happens in a picture; it is what the skips are for.
var endless = math.Inf(1)

// an edge-on face takes nothing from an endless sun: a floor, under a
// sun that grazes it along plus x, is dark.
func TestAFloorEdgeOnToAnEndlessSunIsDark(t *testing.T) {
	w := floor()
	w.Lamp(thing.Sun(maths.V(1, 0, 0), maths.Grey(endless)))
	if got := Trace(w, straightDown(), 0); got.R != 0 || got.G != 0 || got.B != 0 {
		t.Errorf("a floor edge-on to an endless sun is %+v", got)
	}
}

// a radar's echo takes nothing from what sends nothing back, from a radar that
// gives nothing, or from a face edge-on, though the other number be endless.
//
// the cases are a ball that returns zero, under a radar of endless strength at
// the eye; a ball that returns endless, under a radar of zero; and a ball that
// returns 0.8, under a radar of endless strength in the plane of the face the
// eye sees.
func TestAnEchoTakesNothingFromWhatIsNotThere(t *testing.T) {
	eye := maths.V(0, 0, 10)
	look := maths.Ray{At: eye, Dir: maths.V(0, 0, -1)}
	for _, c := range []struct {
		name string
		back float64
		at   maths.Vec
		beam float64
	}{
		{"a ball that returns nought", 0, eye, endless},
		{"a radar that gives nought", endless, eye, 0},
		{"a radar in the plane of the face", 0.8, maths.V(0, 8, 2), endless},
	} {
		w := thing.NewWorld()
		w.Add(radiantBall(0, 2, radiates{back: c.back}))
		w.Lamp(radar(c.at, c.beam))
		if got := inBand(bandStage(w), look, micro, passes); got.R != 0 {
			t.Errorf("%s: %v, want nought", c.name, got.R)
		}
	}
}

// a thing that passes none of a band shows what it gives off and takes
// nothing from what is behind it, though that be endless.
func TestAThingThatPassesNothingTakesNothingFromAnEndlessSourceBehindIt(t *testing.T) {
	w := thing.NewWorld()
	w.Add(radiantBall(-5, 1, radiates{gives: 2}))
	w.Add(radiantBall(-20, 1, radiates{gives: endless}))
	if got := inBand(bandStage(w), out, xray, passes); got.R != 2 {
		t.Errorf("an opaque thing before an endless source: %v, want 2", got.R)
	}
}

// a way of no length through a medium takes nothing from it, though the
// medium be endless in thickness: a medium that says it is crossed from 5
// to 5 is not crossed, for a painted volume, for a gas, and for the way
// from a point to a lamp, shadowed by a painted volume and by a gas.
func TestAWayOfNoLengthTakesNothingFromAnEndlessMedium(t *testing.T) {
	w := thing.NewWorld()
	w.Sky = even(maths.Grey(1))
	w.AddVolume(thing.Volume{Medium: endlessly{5, 5}, Material: glowOf(maths.Grey(0.5))})
	if got := Trace(w, along(0, 0), 0); got.R != 1 || got.Cover != 0 {
		t.Errorf("a painted volume of no way: %+v, want the sky", got)
	}

	w = thing.NewWorld()
	w.Sky = even(maths.Grey(1))
	w.AddVolume(thing.Volume{Medium: gas.Of{Shape: endlessly{5, 5}, In: tinted(redOptics, greenOptics, blueOptics), Unit: 1}})
	if got := Trace(w, along(0, 0), 0); got.R != 1 || got.Cover != 0 {
		t.Errorf("a gas of no way: %+v, want the sky", got)
	}

	up := maths.V(0, 1, 0)
	if got := clear(volumes(paintedBy(mediumOf(endlessly{5, 5}))), maths.Vec{}, up); got != 1 {
		t.Errorf("a shadow through a painted volume of no way: %v, want 1", got)
	}
	gassy := volumes(thing.Volume{Medium: gas.Of{Shape: endlessly{5, 5},
		In: tinted(redOptics, greenOptics, blueOptics), Unit: 1}})
	if got := gasClear(gassy, maths.Vec{}, up); got != maths.Grey(1) {
		t.Errorf("a shadow through a gas of no way: %v, want 1", got)
	}
}

// a step through a medium of no thickness takes nothing from it, though
// the lamp, or what the thing gives off, be endless.
//
// a ray through the corner of a box that holds a ball, in the box and
// out of the ball, finds nothing for a painted volume lit by an endless
// sun, for a gas lit by one, and, in a band, for a volume that gives off
// endless.
func TestAStepOfNoThicknessTakesNothingFromAnEndlessLight(t *testing.T) {
	corner := along(0.9, 0.9)
	sun := thing.Sun(maths.V(1, 0, 0), maths.Grey(endless))
	painted := thing.NewWorld()
	painted.AddVolume(thing.Volume{Medium: cornered{k: 1, radius: 1},
		Material: &thing.Material{Colour: maths.Grey(1)}})
	painted.Lamp(sun)
	if got := Trace(painted, corner, 0); got.R != 0 || got.G != 0 || got.B != 0 || got.Cover != 0 {
		t.Errorf("a painted volume the ray misses: %+v", got)
	}

	gassy := thing.NewWorld()
	gassy.AddVolume(thing.Volume{Medium: gas.Of{Shape: cornered{k: 1, radius: 1},
		In: tinted(redOptics, greenOptics, blueOptics), Unit: 1}})
	gassy.Lamp(sun)
	if got := Trace(gassy, corner, 0); got.R != 0 || got.G != 0 || got.B != 0 || got.Cover != 0 {
		t.Errorf("a gas the ray misses: %+v", got)
	}

	giving := thing.NewWorld()
	giving.AddVolume(thing.Volume{Medium: cornered{k: 1, radius: 1},
		Material: &thing.Material{Radiant: radiates{through: 1, gives: endless}}})
	if got := inBand(bandStage(giving), corner, xray, passes); got.R != 0 || got.Cover != 0 {
		t.Errorf("a volume the band misses: %+v", got)
	}
}
