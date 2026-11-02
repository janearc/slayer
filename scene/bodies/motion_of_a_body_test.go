package bodies_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/thing"
)

// a body turns once a Day, prograde: from +x toward -z, about its north. four
// hours of a four hour day is a full turn, one hour a quarter of one, which
// takes +x to -z, and a negative day turns the other way, +x to +z.
//
// a day of zero is no turn, and a nanosecond is a day all the same.
func TestABodyTurnsProgradeOnceADayAndBackwardOnANegativeOne(t *testing.T) {
	place := func(day time.Duration, tilt float64) *thing.Thing {
		b := bodies.Body{Name: "s", Kind: bodies.Star, Radius: 8, Day: day, Tilt: tilt}
		w := thing.NewWorld()
		if err := b.Place(w, maths.Vec{}, 0.5, 2); err != nil {
			t.Fatal(err)
		}
		return w.Things[0]
	}
	turned := func(th *thing.Thing, at time.Duration, v maths.Vec) maths.Vec {
		return th.At(at).Orientation.Out(v)
	}
	four := 4 * time.Hour
	fwd, back := place(four, 0), place(-four, 0)
	nearVec(t, "+x a quarter of a day on", turned(fwd, time.Hour, maths.XAxis), maths.V(0, 0, -1), 1e-12)
	nearVec(t, "+x half a day on", turned(fwd, 2*time.Hour, maths.XAxis), maths.V(-1, 0, 0), 1e-12)
	nearVec(t, "+x a day on", turned(fwd, four, maths.XAxis), maths.V(1, 0, 0), 1e-12)
	nearVec(t, "+x a quarter of a negative day on", turned(back, time.Hour, maths.XAxis), maths.V(0, 0, 1), 1e-12)
	if sp := back.Motion.(thing.Spin); sp.Period != -four {
		t.Errorf("a negative day turns with a period of %v", sp.Period)
	}

	still := place(0, 0)
	if _, spins := still.Motion.(thing.Spin); spins {
		t.Error("a body with no day spins")
	}
	nearVec(t, "+x with no day", turned(still, time.Hour, maths.XAxis), maths.XAxis, 1e-12)
	short := place(1, 0)
	if sp, spins := short.Motion.(thing.Spin); !spins || sp.Period != 1 {
		t.Errorf("a body with a day of a nanosecond is on %#v", short.Motion)
	}

	// tilted a quarter turn about x its north is +z: the body's +y stays
	// there as it turns, and its +x, on the tilt's axis, goes round to +y
	tilted := place(four, math.Pi/2)
	nearVec(t, "the tilted north", turned(tilted, time.Hour, maths.Up), maths.V(0, 0, 1), 1e-12)
	nearVec(t, "the tilted +x", turned(tilted, time.Hour, maths.XAxis), maths.V(0, 1, 0), 1e-12)
}

// PlaceOn puts the body on the motion it is given, not on one of its own:
// the day is not added and the moons are not placed, and every part of
// it, the thing and the volumes, is carried by that motion.
func TestPlaceOnCarriesTheBodyOnTheMotionItIsGiven(t *testing.T) {
	b := bodies.Body{Name: "s", Kind: bodies.Star, Radius: 8, Day: 4 * time.Hour,
		Tilt: 0.5, Mass: bodies.Sun,
		Air:   &bodies.Air{Height: 6, Dense: 0.25},
		Moons: []bodies.Moon{{Body: bodies.Body{Name: "m", Kind: bodies.Star, Radius: 1}, Orbit: 1e6}}}
	track := thing.Orbit{Centre: maths.V(1, 2, 3), Radius: 10, Period: 8 * time.Hour}
	w := thing.NewWorld()
	if err := b.PlaceOn(w, track, 0.5, 2); err != nil {
		t.Fatal(err)
	}
	if len(w.Things) != 1 || len(w.Volumes) != 1 {
		t.Fatalf("it placed %d things and %d volumes, want the star and its air",
			len(w.Things), len(w.Volumes))
	}
	if got := w.Things[0].Motion; got != thing.Motion(track) {
		t.Errorf("the star is carried by %#v, not the track", got)
	}
	if got := w.Volumes[0].Motion; got != thing.Motion(track) {
		t.Errorf("the air is carried by %#v, not the track", got)
	}
	// and no turn of its own: an hour on, a quarter of its day, its +x is
	// where the track leaves it, which is +x
	if o := w.Things[0].At(time.Hour).Orientation.Out(maths.XAxis); o != maths.XAxis {
		t.Errorf("the star turns, its +x at %v", o)
	}
}

// PlaceOn refuses as Place does, by name: a scale that is no size.
func TestPlaceOnRefusesWithTheReason(t *testing.T) {
	track := thing.Orbit{Radius: 10, Period: time.Hour}
	b := bodies.Body{Name: "s", Kind: bodies.Star, Radius: 8}
	if err := b.PlaceOn(thing.NewWorld(), track, 0, 2); err == nil || !strings.Contains(err.Error(), "scale") {
		t.Errorf("a scale of nought: %v", err)
	}
	if err := b.PlaceOn(thing.NewWorld(), track, math.Inf(1), 2); err == nil || !strings.Contains(err.Error(), "scale") {
		t.Errorf("an endless scale: %v", err)
	}
}
