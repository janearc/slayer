package bodies_test

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// Aim sets the sun and the trail on every tail the body has, and no
// other part of it.
func TestAimSetsEveryTail(t *testing.T) {
	b := bodies.Body{Name: "c", Tails: []bodies.Tail{{Length: 1}, {Length: 2}, {Length: 3}}}
	b.Aim(maths.V(1, 2, 3), maths.V(4, 5, 6))
	for i, tail := range b.Tails {
		if tail.Sun != maths.V(1, 2, 3) || tail.Trail != maths.V(4, 5, 6) {
			t.Errorf("tail %d is aimed at %v along %v", i, tail.Sun, tail.Trail)
		}
		if tail.Length != float64(i+1) {
			t.Errorf("tail %d is %v long", i, tail.Length)
		}
	}
}

// two tails of a body of 8 km at half a unit a kilometre: 64 km long is
// 32 units, 8 km across the base is 4, 0.125 a kilometre is 0.25 a unit;
// the second is ragged, in wisps of three base widths, 3 x 8 x 0.5 = 12
// units, with its own seed. the smooth one is a plain tail.
func TestTailsAreCutToTheScale(t *testing.T) {
	smooth := bodies.Tail{Length: 64, Base: 8, Spread: 0.125, Curve: 0.25, Dense: 0.125,
		Colour: maths.Light{R: 0.5, G: 0.25, B: 0.125}}
	ragged := smooth
	ragged.Wisp, ragged.Seed = 0.25, 21
	b := bodies.Body{Name: "c", Kind: bodies.Star, Radius: 8, Tails: []bodies.Tail{smooth, ragged}}
	b.Aim(maths.V(-3, 0, 0), maths.V(1, 0, 1))
	w := thing.NewWorld()
	if err := b.Place(w, maths.V(0, 4, 0), 0.5, 2); err != nil {
		t.Fatal(err)
	}
	plain := medium.Tail{Length: 32, Base: 4, Spread: 0.125, Curve: 0.25, Thick: 0.25}
	same(t, "the smooth tail", w.Volumes[0].Medium, plain)
	same(t, "the ragged tail", w.Volumes[1].Medium,
		medium.Wispy{Of: plain, Seed: 21, Scale: 12, Amount: 0.25})
	same(t, "the tail's colour", w.Volumes[0].Material.Colour, maths.Light{R: 0.5, G: 0.25, B: 0.125})
}

// a tail streams away from the sun and bends back along the trail, worked
// by hand: a body at (0, 4, 0) and a sun at (-3, 0, 0) are (3, 4, 0)
// apart and 5 long, so the way away is (0.6, 0.8, 0).
//
// the trail (1, 0, 1) reversed is (-1, 0, -1), and its part along the way
// away is -0.6, so taking that out leaves (-1, 0, -1) + 0.6 (0.6, 0.8, 0)
// = (-0.64, 0.48, -1), 1.64 long squared, which is the tail's x.
//
// its z is x cross y, (0.8, -0.6, -0.8) over the same root.
func TestATailIsAwayFromTheSunAndBackAlongTheTrail(t *testing.T) {
	b := bodies.Body{Name: "c", Kind: bodies.Star, Radius: 8,
		Tails: []bodies.Tail{{Length: 64, Base: 8, Spread: 0.125, Dense: 0.125}}}
	b.Aim(maths.V(-3, 0, 0), maths.V(1, 0, 1))
	w := thing.NewWorld()
	at := maths.V(0, 4, 0)
	if err := b.Place(w, at, 0.5, 2); err != nil {
		t.Fatal(err)
	}
	s := w.Volumes[0].At(0)
	root := math.Sqrt(1.64)
	nearVec(t, "the tail's origin", s.Origin, at, 1e-12)
	nearVec(t, "the tail's y, away", s.Orientation.Y, maths.V(0.6, 0.8, 0), 1e-12)
	nearVec(t, "the tail's x, back", s.Orientation.X, maths.V(-0.64/root, 0.48/root, -1/root), 1e-12)
	nearVec(t, "the tail's z", s.Orientation.Z, maths.V(0.8/root, -0.6/root, -0.8/root), 1e-12)
}

// a trail along the way away, or against it, leaves nothing to bend back
// along, and the tail takes the first axis of a frame standing on the
// way away.
//
// with a sun at (0, -5, 0) and the body at (0, 5, 0) the way away is +y,
// and that frame's x is +x.
func TestATrailAlongTheTailTakesTheFramesFirstAxis(t *testing.T) {
	for _, trail := range []maths.Vec{maths.V(0, 2, 0), maths.V(0, -2, 0)} {
		b := bodies.Body{Name: "c", Kind: bodies.Star, Radius: 8,
			Tails: []bodies.Tail{{Length: 64, Base: 8, Dense: 0.125}}}
		b.Aim(maths.V(0, -5, 0), trail)
		w := thing.NewWorld()
		if err := b.Place(w, maths.V(0, 5, 0), 0.5, 2); err != nil {
			t.Fatal(err)
		}
		o := w.Volumes[0].At(0).Orientation
		nearVec(t, "the tail's y", o.Y, maths.V(0, 1, 0), 1e-12)
		nearVec(t, "the tail's x", o.X, maths.V(1, 0, 0), 1e-12)
		nearVec(t, "the tail's z", o.Z, maths.V(0, 0, 1), 1e-12)
	}
}

// a tail is aimed once, from where the body is at time zero: on a track that
// takes the body round, the tail goes along with the body and does not turn.
//
// a body at (10, 0, 0) at zero and at (0, 0, -10) a nanosecond on (a track of
// four nanoseconds, prograde), with a sun at (-100, 0, 0), streams along +x at
// both times.
func TestATailIsAimedWhereTheBodyIsAtNought(t *testing.T) {
	b := bodies.Body{Name: "c", Kind: bodies.Star, Radius: 8,
		Tails: []bodies.Tail{{Length: 64, Base: 8, Dense: 0.125}}}
	b.Aim(maths.V(-100, 0, 0), maths.V(0, 0, 1))
	w := thing.NewWorld()
	track := thing.Orbit{Radius: 10, Period: 4 * time.Nanosecond}
	if err := b.PlaceOn(w, track, 0.5, 2); err != nil {
		t.Fatal(err)
	}
	v := w.Volumes[0]
	nearVec(t, "the tail's y at nought", v.At(0).Orientation.Y, maths.V(1, 0, 0), 1e-12)
	later := v.At(time.Nanosecond)
	nearVec(t, "the tail's y a nanosecond on", later.Orientation.Y, maths.V(1, 0, 0), 1e-12)
	nearVec(t, "the tail's place a nanosecond on", later.Origin, maths.V(0, 0, -10), 1e-9)
}
