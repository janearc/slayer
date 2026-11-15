package template

import (
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/world"
)

// the held tests spell the published figures at their call sites and
// build the want through the same doors the templates speak. the doors
// themselves are held by hand next door, in the bodies suite, so a
// door cannot quietly bend both sides the same way without that suite
// seeing it. colours are not held by apart, so a want wears none.

// its unwraps a template the test trusts; a refusal fails the test.
func its(t *testing.T, make func() (bodies.Body, error)) bodies.Body {
	t.Helper()
	body, err := make()
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// want is a held figure: a body of the spelled radius and mass,
// wearing the spelled parts.
func want(t *testing.T, name string, kind bodies.Kind, radiusKm,
	massKg float64, parts ...bodies.Part) bodies.Body {
	t.Helper()
	body, err := bodies.NewBody(name, kind, bodies.Kilometres(radiusKm),
		measure.Kilograms(massKg), parts...)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// groundOf is the body's ground, which these tests require it to wear.
func groundOf(t *testing.T, b bodies.Body) bodies.Ground {
	t.Helper()
	ground, worn := b.Ground()
	if !worn {
		t.Fatalf("%s wears no ground", b.Name())
	}
	return ground
}

// wantComa is a held coma figure.
func wantComa(t *testing.T, radiusKm, dense float64) bodies.Coma {
	t.Helper()
	coma, err := bodies.NewComa(bodies.Kilometres(radiusKm),
		bodies.PerKilometre(dense), maths.Light{}, maths.Light{})
	if err != nil {
		t.Fatal(err)
	}
	return coma
}

// wantTail is a held tail figure.
func wantTail(t *testing.T, lengthKm, baseKm, spread, curve, dense,
	wisp float64, seed uint64) bodies.Tail {
	t.Helper()
	tail, err := bodies.NewTail(bodies.Kilometres(lengthKm),
		bodies.Kilometres(baseKm), spread, curve,
		bodies.PerKilometre(dense), maths.Light{}, maths.Light{})
	if err != nil {
		t.Fatal(err)
	}
	if wisp == 0 {
		return tail
	}
	tail, err = tail.Wisped(maths.Share(wisp), seed)
	if err != nil {
		t.Fatal(err)
	}
	return tail
}

// wantJets is a held jets figure.
func wantJets(t *testing.T, lengthKm, baseKm, spread, fromKm, dense,
	tiltRad, speed float64) bodies.Jets {
	t.Helper()
	jets, err := bodies.NewJets(bodies.Kilometres(lengthKm),
		bodies.Kilometres(baseKm), spread, bodies.Kilometres(fromKm),
		bodies.PerKilometre(dense), measure.Radians(tiltRad),
		maths.Share(speed), maths.Light{})
	if err != nil {
		t.Fatal(err)
	}
	return jets
}

// wantGround is a held ground figure; what else it wears hangs by its
// own methods at the call site.
func wantGround(t *testing.T, terrain world.Terrain,
	worn world.Look) bodies.Ground {
	t.Helper()
	ground, err := bodies.NewGround(terrain, worn)
	if err != nil {
		t.Fatal(err)
	}
	return ground
}

// wantAir is a held air figure.
func wantAir(t *testing.T, heightKm, dense float64) bodies.Air {
	t.Helper()
	air, err := bodies.NewAir(bodies.Kilometres(heightKm),
		bodies.PerKilometre(dense), maths.Light{})
	if err != nil {
		t.Fatal(err)
	}
	return air
}

// wantAurora is a held aurora figure.
func wantAurora(t *testing.T, lowKm, highKm, latRad, widthRad, tiltRad,
	strength, curtains float64, seed uint64) bodies.Aurora {
	t.Helper()
	aurora, err := bodies.NewAurora(
		bodies.NewShell(bodies.Kilometres(lowKm),
			bodies.Kilometres(highKm)),
		measure.Radians(latRad), measure.Radians(widthRad),
		measure.Radians(tiltRad), maths.Share(strength),
		maths.Share(curtains), seed,
		bodies.NewAuroraColours(maths.Light{}, maths.Light{},
			maths.Light{}))
	if err != nil {
		t.Fatal(err)
	}
	return aurora
}

// wantRings is a held rings figure.
func wantRings(t *testing.T, innerTimes, outerTimes, thickKm, dense float64,
	profile []float64, grain, ringlets float64) bodies.Rings {
	t.Helper()
	rings, err := bodies.NewRings(innerTimes, outerTimes,
		bodies.Kilometres(thickKm), bodies.PerKilometre(dense),
		maths.Light{})
	if err != nil {
		t.Fatal(err)
	}
	return rings.Profiled(profile).Grained(grain, ringlets)
}

// wantGas is a held gas figure.
func wantGas(t *testing.T, depth, opacity float64,
	bands texture.Banded) bodies.Gas {
	t.Helper()
	gas, err := bodies.NewGas(maths.Share(depth), opacity, bands)
	if err != nil {
		t.Fatal(err)
	}
	return gas
}

// wantPlume is a held plume figure.
func wantPlume(t *testing.T, toward maths.Vec, heightKm, widthKm,
	dense float64) bodies.Plume {
	t.Helper()
	plume, err := bodies.NewPlume(toward, bodies.Kilometres(heightKm),
		bodies.Kilometres(widthKm), bodies.PerKilometre(dense),
		maths.Light{})
	if err != nil {
		t.Fatal(err)
	}
	return plume
}

// wantMoon is a held moon figure.
func wantMoon(t *testing.T, body bodies.Body, orbitKm,
	phaseRad float64) bodies.Moon {
	t.Helper()
	moon, err := bodies.NewMoon(body, bodies.Kilometres(orbitKm),
		measure.Radians(phaseRad))
	if err != nil {
		t.Fatal(err)
	}
	return moon
}

// wantDisc is a held disc figure.
func wantDisc(t *testing.T, innerKm, outerKm, thickKm, dense, hotKelvin,
	bright, speed, streaks, shear float64) bodies.Disc {
	t.Helper()
	disc, err := bodies.NewDisc(bodies.Kilometres(innerKm),
		bodies.Kilometres(outerKm), bodies.Kilometres(thickKm),
		bodies.PerKilometre(dense), measure.Kelvin(hotKelvin), bright)
	if err != nil {
		t.Fatal(err)
	}
	disc, err = disc.Beamed(maths.Share(speed))
	if err != nil {
		t.Fatal(err)
	}
	return disc.Dressed(streaks, shear)
}
