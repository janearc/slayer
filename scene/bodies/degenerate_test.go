package bodies_test

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
)

// eitherInfinity is defined by math.IsInf: a sign of zero asks for
// infinity of either sign.
const eitherInfinity = 0

// fixtureSpan is the fixture's length of that name, in kilometres.
func fixtureSpan(t *testing.T, name string) bodies.Span {
	t.Helper()
	return bodies.Kilometres(picked(t, name))
}

// glowing is a light of the fixture's share in every band.
func glowing(t *testing.T) maths.Light {
	t.Helper()
	return maths.Grey(picked(t, "degenerate-light"))
}

// share is the fixture's share of a whole.
func share(t *testing.T) maths.Portion {
	t.Helper()
	return maths.Share(picked(t, "degenerate-share"))
}

// degenerateBody is a rock of the fixture's radius and mass.
func degenerateBody(t *testing.T) bodies.Body {
	t.Helper()
	radius := fixtureSpan(t, "degenerate-radius")
	mass := measure.Kilograms(picked(t, "degenerate-mass"))
	made, err := bodies.NewBody("a rock", bodies.Rock, radius, mass)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// An orbit's rate round a radius that is no number is zero, as it is for
// a radius of zero or less, and not NaN.
func TestOmegaOfNoNumberIsZero(t *testing.T) {
	body := degenerateBody(t)
	for name, radius := range map[string]float64{
		"NaN": math.NaN(), "zero": nought, "below zero": -unity,
	} {
		if got := body.Omega(bodies.Kilometres(radius)); got != nought {
			t.Errorf("Omega at a radius of %s is %v; want zero", name, got)
		}
	}
}

// Anti-test: a real radius gives a rate above zero and finite.
func TestOmegaOfARealRadiusIsAboveZero(t *testing.T) {
	body := degenerateBody(t)
	got := body.Omega(fixtureSpan(t, "degenerate-orbit"))
	if !(got > nought) || math.IsInf(got, eitherInfinity) {
		t.Errorf("Omega at a real radius is %v; want above zero, finite",
			got)
	}
}

// jetsOf is a pair of jets whose knots are what the test says.
func jetsOf(t *testing.T, knots float64) bodies.Jets {
	t.Helper()
	nothing := bodies.PerKilometre(nought)
	length := fixtureSpan(t, "degenerate-orbit")
	base := fixtureSpan(t, "degenerate-radius")
	none := bodies.Kilometres(nought)
	jets, err := bodies.NewJets(length, base, nought, none, nothing,
		measure.Radians(nought), share(t), glowing(t))
	if err != nil {
		t.Fatal(err)
	}
	return jets.Knotted(knots)
}

// holeWearing is a hole of the fixture's radius and mass wearing a part,
// or the refusal.
func holeWearing(t *testing.T, part bodies.Part) error {
	t.Helper()
	radius := fixtureSpan(t, "degenerate-radius")
	mass := measure.Kilograms(picked(t, "degenerate-mass"))
	_, err := bodies.NewBody("a hole", bodies.Hole, radius, mass, part)
	return err
}

// Knots above one or below zero are not a share, and a body wearing them
// is refused, by the verb's name.
func TestKnotsOutsideAShareAreRefused(t *testing.T) {
	for name, knots := range map[string]float64{
		"over":  picked(t, "degenerate-knots-over"),
		"under": picked(t, "degenerate-knots-under"),
	} {
		err := holeWearing(t, jetsOf(t, knots))
		if err == nil || !strings.Contains(err.Error(), "Knotted") {
			t.Errorf("knots %s a share: %v; want a refusal by Knotted",
				name, err)
		}
	}
}

// Anti-test: the ends of a share, and the middle, are taken.
func TestKnotsInsideAShareAreTaken(t *testing.T) {
	for _, knots := range []float64{nought, picked(t, "degenerate-share"),
		unity} {
		if err := holeWearing(t, jetsOf(t, knots)); err != nil {
			t.Errorf("knots of %v refused: %v", knots, err)
		}
	}
}

// A count of rungs past what a tiling can be split is refused, as it
// would otherwise be walked one rung at a time for ever.
func TestTooManyRungsAreRefused(t *testing.T) {
	for _, rungs := range []uint{pickedCount(t, "degenerate-rungs-over"),
		pickedCount(t, "degenerate-rungs-huge")} {
		_, err := bodies.NewFineness(rungs, share(t), share(t))
		if err == nil || !strings.Contains(err.Error(), "rungs") {
			t.Errorf("NewFineness of %d rungs: %v; want a refusal", rungs,
				err)
		}
		_, err = bodies.NewCities(rungs, share(t), share(t), glowing(t),
			nought)
		if err == nil || !strings.Contains(err.Error(), "rungs") {
			t.Errorf("NewCities of %d rungs: %v; want a refusal", rungs,
				err)
		}
	}
}

// Anti-test: none, and the most, are taken, and read back as they were
// said.
func TestTheMostRungsAreTaken(t *testing.T) {
	for _, rungs := range []uint{nought,
		pickedCount(t, "degenerate-rungs-most")} {
		fine, err := bodies.NewFineness(rungs, share(t), share(t))
		if err != nil || fine.Rungs() != int(rungs) {
			t.Errorf("NewFineness of %d rungs: %v, reads %d", rungs, err,
				fine.Rungs())
		}
		cities, err := bodies.NewCities(rungs, share(t), share(t),
			glowing(t), nought)
		if err != nil || cities.Rungs() != int(rungs) {
			t.Errorf("NewCities of %d rungs: %v, reads %d", rungs, err,
				cities.Rungs())
		}
	}
}
