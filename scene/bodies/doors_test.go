package bodies_test

import (
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
)

// made is a door-made body the test trusts: radius in kilometres, mass
// in kilograms, wearing the parts. A refusal fails the test, since
// these are the suite's own known-good figures.
func made(t *testing.T, name string, kind bodies.Kind, radiusKm,
	massKg float64, parts ...bodies.Part) bodies.Body {
	t.Helper()
	body, err := bodies.NewBody(name, kind, bodies.Kilometres(radiusKm),
		measure.Kilograms(massKg), parts...)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// temp is a template the test trusts; a refusal fails the test.
func temp(t *testing.T, make func() (bodies.Body, error)) bodies.Body {
	t.Helper()
	body, err := make()
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// every constructor argument is read back from its own reader: each is
// given a different number, written here as a literal, so a constructor
// that swapped two of its fields reads one back as the other.
func TestNewBodyReadsBackEveryCoreArgumentFromItsOwnReader(t *testing.T) {
	body, err := bodies.NewBody("vesta", bodies.Giant,
		bodies.Kilometres(262.7), measure.Kilograms(2.59e20))
	if err != nil {
		t.Fatal(err)
	}
	if body.Name() != "vesta" {
		t.Errorf("the name is %q, want vesta", body.Name())
	}
	if body.Kind() != bodies.Giant {
		t.Errorf("the kind is %v, want Giant", body.Kind())
	}
	near(t, "the radius in kilometres", body.Radius(), 262.7, 1e-12)
	near(t, "the mass in kilograms", body.Mass(), 2.59e20, 1e-12)
}

// a clouds layer, a tail and an aurora are each made from arguments that
// differ from one another, and each reader gives back its own.
func TestPartConstructorsReadBackDistinctArgumentsFromTheirOwnReaders(t *testing.T) {
	clouds, err := bodies.NewClouds(
		bodies.NewShell(bodies.Kilometres(3), bodies.Kilometres(9)),
		maths.Share(0.4), bodies.PerKilometre(0.05), 7)
	if err != nil {
		t.Fatal(err)
	}
	near(t, "the clouds' low edge", clouds.Low(), 3, 1e-12)
	near(t, "the clouds' high edge", clouds.High(), 9, 1e-12)
	near(t, "the clouds' cover", clouds.Cover(), 0.4, 1e-12)
	near(t, "the clouds' density", clouds.Dense(), 0.05, 1e-12)
	if clouds.Seed() != 7 {
		t.Errorf("the clouds' seed is %d, want 7", clouds.Seed())
	}

	tail, err := bodies.NewTail(bodies.Kilometres(700), bodies.Kilometres(40),
		0.15, 0.35, bodies.PerKilometre(0.002), maths.RGB(0.1, 0.2, 0.3),
		maths.RGB(0.4, 0.5, 0.6))
	if err != nil {
		t.Fatal(err)
	}
	tail, err = tail.Wisped(maths.Share(0.6), 11)
	if err != nil {
		t.Fatal(err)
	}
	near(t, "the tail's length", tail.Length(), 700, 1e-12)
	near(t, "the tail's base", tail.Base(), 40, 1e-12)
	near(t, "the tail's spread", tail.Spread(), 0.15, 1e-12)
	near(t, "the tail's curve", tail.Curve(), 0.35, 1e-12)
	near(t, "the tail's density", tail.Dense(), 0.002, 1e-12)
	near(t, "the tail's wisp", tail.Wisp(), 0.6, 1e-12)
	if tail.Colour() != maths.RGB(0.1, 0.2, 0.3) ||
		tail.Glow() != maths.RGB(0.4, 0.5, 0.6) || tail.Seed() != 11 {
		t.Errorf("the tail's colour, glow and seed are %v, %v, %d: want "+
			"(0.1, 0.2, 0.3), (0.4, 0.5, 0.6), 11", tail.Colour(),
			tail.Glow(), tail.Seed())
	}

	aurora, err := bodies.NewAurora(
		bodies.NewShell(bodies.Kilometres(80), bodies.Kilometres(500)),
		measure.Radians(0.3), measure.Radians(0.04), measure.Radians(0.2),
		maths.Share(0.7), maths.Share(0.25), 13,
		bodies.NewAuroraColours(maths.RGB(1, 0, 0), maths.RGB(0, 1, 0),
			maths.RGB(0, 0, 1)))
	if err != nil {
		t.Fatal(err)
	}
	near(t, "the aurora's low edge", aurora.Low(), 80, 1e-12)
	near(t, "the aurora's high edge", aurora.High(), 500, 1e-12)
	near(t, "the aurora's latitude", aurora.Lat(), 0.3, 1e-12)
	near(t, "the aurora's width", aurora.Width(), 0.04, 1e-12)
	near(t, "the aurora's tilt", aurora.Tilt(), 0.2, 1e-12)
	near(t, "the aurora's strength", aurora.Strength(), 0.7, 1e-12)
	near(t, "the aurora's curtains", aurora.Curtains(), 0.25, 1e-12)
	colours := aurora.Colours()
	if aurora.Seed() != 13 || colours.Fringe() != maths.RGB(1, 0, 0) ||
		colours.Bottom() != maths.RGB(0, 1, 0) ||
		colours.Top() != maths.RGB(0, 0, 1) {
		t.Errorf("the aurora's seed and colours are %d, %v, %v, %v: want "+
			"13, red, green, blue", aurora.Seed(), colours.Fringe(),
			colours.Bottom(), colours.Top())
	}
}
