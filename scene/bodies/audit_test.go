package bodies

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// star is a minimal door-made body for the refusal tests.
func star(t *testing.T) Body {
	t.Helper()
	made, err := NewBody("a", Star, Kilometres(1),
		measure.Kilograms(1))
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// a part that cannot be described is refused at its own constructor,
// by name: a gas of no depth, a shell upside down.
func TestAPartThatCannotBeIsRefusedAtItsDoor(t *testing.T) {
	if _, err := NewGas(maths.Share(0), 1, texture.Banded{}); err == nil ||
		!strings.Contains(err.Error(), "depth") {
		t.Errorf("a gas of no depth: %v", err)
	}
	_, err := NewClouds(NewShell(Kilometres(5), Kilometres(5)),
		maths.Share(0.5), PerKilometre(1), 1)
	if err == nil || !strings.Contains(err.Error(), "shell") {
		t.Errorf("an upside-down shell: %v", err)
	}
	// the aurora's own door refuses a shell upside down, by the aurora's
	// name: its high edge is no higher than its low one
	_, err = NewAurora(NewShell(Kilometres(5), Kilometres(5)),
		measure.Radians(0), measure.Radians(0), measure.Radians(0),
		maths.Share(0.5), maths.Share(0.5), 1, AuroraColours{})
	if err == nil || !strings.Contains(err.Error(), "an aurora") ||
		!strings.Contains(err.Error(), "shell") {
		t.Errorf("an upside-down aurora: %v", err)
	}
}

// a body placed at a scale that is no size, a hole spinning at its
// mass, or a tail at its own sun, is refused by name at Place, not put
// into the world as infinity or nil.
func TestABodyThatCannotBePlacedIsRefused(t *testing.T) {
	for name, scale := range map[string]float64{
		"no scale": 0, "NaN scale": math.NaN(),
		"endless scale": math.Inf(1),
	} {
		err := star(t).Place(thing.NewWorld(), maths.Vec{}, scale, 2)
		if err == nil || !strings.Contains(err.Error(), "scale") {
			t.Errorf("%s: %v", name, err)
		}
	}
	hole, err := NewBody("a", Hole, Kilometres(1), measure.Kilograms(Sun))
	if err != nil {
		t.Fatal(err)
	}
	err = hole.Spinning(1).Place(thing.NewWorld(), maths.Vec{}, 1, 2)
	if err == nil || !strings.Contains(err.Error(), "horizon") {
		t.Errorf("spun out: %v", err)
	}
	tail, err := NewTail(Kilometres(1), Kilometres(1), 0, 0,
		PerKilometre(0), maths.Light{}, maths.Light{})
	if err != nil {
		t.Fatal(err)
	}
	aimless, err := star(t).With(tail)
	if err != nil {
		t.Fatal(err)
	}
	err = aimless.Place(thing.NewWorld(), maths.Vec{}, 1, 2)
	if err == nil || !strings.Contains(err.Error(), "tail") {
		t.Errorf("tail at its sun: %v", err)
	}
}

// an orbit past what a Duration holds is refused as an error, before
// thing.Period panics: by hand, a moon 1e10 km round the earth, or two
// suns 1e10 km apart, take hundreds of years and more.
func TestAnOrbitTooLongIsAnError(t *testing.T) {
	centre, err := NewBody("earth", Star, Kilometres(6371),
		measure.Kilograms(5.972e24))
	if err != nil {
		t.Fatal(err)
	}
	far, err := NewMoon(star(t), Kilometres(1e10), measure.Radians(0))
	if err != nil {
		t.Fatal(err)
	}
	withFar, err := centre.With(far)
	if err != nil {
		t.Fatal(err)
	}
	err = withFar.Place(thing.NewWorld(), maths.Vec{}, 1e-3, 2)
	if err == nil || !strings.Contains(err.Error(), "292") {
		t.Errorf("a moon at 1e10 km: %v", err)
	}
	sun, err := NewBody("sun", Rock, Kilometres(1), measure.Kilograms(Sun))
	if err != nil {
		t.Fatal(err)
	}
	err = Binary(thing.NewWorld(), sun, sun, maths.Vec{}, Kilometres(1e10), 1e-3, 2)
	if err == nil || !strings.Contains(err.Error(), "292") {
		t.Errorf("two suns 1e10 km apart: %v", err)
	}
	// a massless body cannot pass the door, so the inner guard is
	// reached by the package's own hand
	none := Body{made: true, name: "none", kind: Rock, radius: 1}
	err = Binary(thing.NewWorld(), none, none, maths.Vec{}, Kilometres(10), 1, 2)
	if err == nil || !strings.Contains(err.Error(), "no mass") {
		t.Errorf("two bodies of no mass: %v", err)
	}
}

// Omega is zero where there is no orbit to go round: a body with no mass or
// less, or a radius of zero or less. a body of no mass at a radius so small its
// cube is under what a float holds is zero too, and not zero over zero. the
// door refuses a body of no mass, so these are made here by hand.
func TestOmegaOfNothingIsNothing(t *testing.T) {
	for name, c := range map[string]struct{ mass, km float64 }{
		"no mass":          {0, 1000},
		"no mass, a speck": {0, 1e-120},
		"negative mass":    {-1, 1000},
		"no radius":        {Sun, 0},
		"negative radius":  {Sun, -1000},
		"neither":          {0, 0},
	} {
		body := Body{mass: c.mass}
		if got := body.Omega(Kilometres(c.km)); got != 0 {
			t.Errorf("%s: omega is %v", name, got)
		}
	}
}

// a parent of no mass has no period to refuse, and its moons are not timed:
// a moon 1e10 km out would take longer than a Duration holds round any
// real mass. the door refuses a body of no mass, so the parent is made here
// by hand.
func TestAMoonOfAParentOfNoMassIsNotTimed(t *testing.T) {
	far, err := NewMoon(star(t), Kilometres(1e10), measure.Radians(0))
	if err != nil {
		t.Fatal(err)
	}
	none := Body{made: true, name: "none", kind: Star, radius: 1,
		moons: []Moon{far}}
	if err := none.Place(thing.NewWorld(), maths.Vec{}, 1e-6, 2); err != nil {
		t.Fatalf("a parent of no mass: %v", err)
	}
	orbit, err := none.Orbit(0, maths.Vec{}, 1e-6)
	if err != nil {
		t.Fatal(err)
	}
	if orbit.Period != 0 {
		t.Errorf("a moon of a parent of no mass goes round in %v, want 0",
			orbit.Period)
	}
}
