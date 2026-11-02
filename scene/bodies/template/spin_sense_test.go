package template

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/thing"
)

// durationOf is a count of seconds as a Duration.
func durationOf(seconds float64) time.Duration {
	return time.Duration(seconds * float64(time.Second))
}

// quarterDayOn is where +x goes in a quarter of a day of so many
// seconds, forward in time whichever way the day runs. the body is placed as a bare star of that day and the tilt,
// which is all Place reads for the turning, so the rest of a template's
// parts are not built.
func quarterDayOn(t *testing.T, day, tilt float64) maths.Vec {
	t.Helper()
	spinner := bodies.Body{Name: "spinner", Kind: bodies.Star, Radius: 1,
		Day: durationOf(day), Tilt: tilt}
	w := thing.NewWorld()
	if err := spinner.Place(w, maths.Vec{}, 1, 2); err != nil {
		t.Fatal(err)
	}
	quarter := spinner.Day / 4
	if quarter < 0 {
		quarter = -quarter
	}
	return w.Things[0].At(quarter).Orientation.Out(maths.XAxis)
}

// physics, worked by hand: slayer turns a body prograde, +x toward -z
// about its north, which a tilt of theta about x carries to
// (0, cos theta, sin theta).
//
// a quarter day on, the body's +x, which is on the tilt's axis and so
// does not move with it, has turned a quarter round that north, to
// north cross +x = (0, sin theta, -cos theta).
//
// so a body with no tilt takes +x to -z, and one tipped past ninety
// degrees, whose north has swung under the orbit's, takes it to +z.
//
// venus, uranus and pluto are the three that turn backward, by a tilt
// past ninety degrees and a day that is positive; the others turn
// forward. the vector is held to a millionth, so the tilt's size is
// held too, near enough to tell a mutant from it.
func TestEachBodyTurnsTheWayItsTiltSays(t *testing.T) {
	backward := map[string]bool{"venus": true, "uranus": true, "pluto": true}
	for _, b := range []bodies.Body{Sol(), Mercury(), Venus(), Earth(), Mars(),
		Jupiter(), Saturn(), Uranus(), Neptune(), Pluto(), Charon(),
		Magnetar(), Pulsar(), HotJupiter()} {
		if b.Day <= 0 {
			t.Errorf("%s has a day of %v; a spinning body's is positive", b.Name, b.Day)
			continue
		}
		got := quarterDayOn(t, b.Day.Seconds(), b.Tilt)
		want := maths.V(0, math.Sin(b.Tilt), -math.Cos(b.Tilt))
		if d := got.Sub(want).Len(); !(d < 1e-6) {
			t.Errorf("%s: +x a quarter day on is %v, want %v", b.Name, got, want)
		}
		if turnsBackward := got.Z > 0; turnsBackward != backward[b.Name] {
			t.Errorf("%s turns backward: %v, want %v (+x went to z = %v)",
				b.Name, turnsBackward, backward[b.Name], got.Z)
		}
	}
}

// physics, worked by hand: with no tilt a quarter day takes +x to -z,
// the convention itself, and a quarter day backward takes it to +z.
// this holds the helper above to the convention, so the test over it
// can not agree with a wrong sense.
func TestAQuarterDayTurnsPlusXTowardMinusZWhenThereIsNoTilt(t *testing.T) {
	got := quarterDayOn(t, 4*3600, 0)
	if d := got.Sub(maths.V(0, 0, -1)).Len(); !(d < 1e-12) {
		t.Errorf("+x a quarter day on is %v, want (0, 0, -1)", got)
	}
	back := quarterDayOn(t, -4*3600, 0)
	if d := back.Sub(maths.V(0, 0, 1)).Len(); !(d < 1e-12) {
		t.Errorf("+x a quarter of a negative day on is %v, want (0, 0, 1)", back)
	}
}

// physics, worked by hand: the comets and the hole do not turn. a body
// with no day stands still, and +x stays +x whatever the time.
func TestBodiesWithNoDayDoNotTurn(t *testing.T) {
	for _, b := range []bodies.Body{HaleBopp(), Hyakutake(), Quasar()} {
		w := thing.NewWorld()
		still := bodies.Body{Name: b.Name, Kind: bodies.Star, Radius: 1,
			Day: b.Day, Tilt: b.Tilt}
		if err := still.Place(w, maths.Vec{}, 1, 2); err != nil {
			t.Fatal(err)
		}
		if b.Day != 0 || b.Tilt != 0 {
			t.Errorf("%s has a day of %v and a tilt of %v, want none", b.Name, b.Day, b.Tilt)
		}
		if _, spins := w.Things[0].Motion.(thing.Spin); spins {
			t.Errorf("%s is carried by a spin", b.Name)
		}
	}
}
