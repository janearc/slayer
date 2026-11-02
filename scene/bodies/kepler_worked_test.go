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

// the published figures the orbits are held to: the astronomical unit,
// 149,597,870.7 km (iau 2012, resolution b2), and the sidereal year,
// 365.256363004 days at j2000 (simon et al., 1994).
//
// the geostationary radius is 42,164 km, and its period one sidereal day
// of 86,164.0905 s (iers). G is 6.6743e-11 (codata 2018), which is the
// one thing.G holds.
const (
	auKm         = 149597870.7
	siderealYear = 365.256363004 * 86400
	geoKm        = 42164.0
	siderealDay  = 86164.0905
	earthMass    = 5.972e24
)

// omega is kepler's rate worked the long way round: the period is
// T = 2 pi sqrt(r^3 / GM) and the rate is 2 pi / T, with r in kilometres.
func omega(mass, km float64) float64 {
	r := km * 1000
	period := 2 * math.Pi * math.Sqrt(r*r*r/(6.6743e-11*mass))
	return 2 * math.Pi / period
}

// massFor is the mass round which a circle of metres takes years julian
// years: kepler's third law solved for the mass, M = 4 pi^2 r^3 / (G T^2).
func massFor(years, metres float64) float64 {
	seconds := years * 365.25 * 86400
	return 4 * math.Pi * math.Pi * metres * metres * metres /
		(6.6743e-11 * seconds * seconds)
}

// Omega is 2 pi over kepler's period: the sun's mass at one astronomical
// unit comes round in a sidereal year, and the earth's at the
// geostationary radius in a sidereal day.
//
// the first is held to 5e-4 of the year, since the sun's mass here is
// 1.989e30 and the earth's own is left out; the second to 1e-4. a
// thousandth more on the radius is three parts in two thousand off in
// the rate, and shows.
func TestOmegaIsKeplersAtTheEarthsOrbitAndTheGeostationaryOne(t *testing.T) {
	sun := bodies.Body{Name: "sun", Mass: bodies.Sun}
	near(t, "the sun's omega at one au", sun.Omega(auKm), 2*math.Pi/siderealYear, 5e-4)
	earth := bodies.Body{Name: "earth", Mass: earthMass}
	near(t, "the earth's omega at 42,164 km", earth.Omega(geoKm), 2*math.Pi/siderealDay, 1e-4)
}

// Omega at other radii and masses is 2 pi / T, T from r^3 / GM, to the last few
// digits. a body of under a kilogram has one, and so has a radius of under a
// kilometre: they are rates, not zero.
func TestOmegaOfSmallMassesAndRadiiIsKeplersToo(t *testing.T) {
	near(t, "half a kilogram at a kilometre", (&bodies.Body{Mass: 0.5}).Omega(1),
		omega(0.5, 1), 1e-12)
	near(t, "a sun at half a kilometre", (&bodies.Body{Mass: bodies.Sun}).Omega(0.5),
		omega(bodies.Sun, 0.5), 1e-12)
	near(t, "the earth at ten thousand kilometres",
		(&bodies.Body{Mass: earthMass}).Omega(1e4), omega(earthMass, 1e4), 1e-12)
}

// Omega is zero where there is no orbit to go round: a body with no mass or
// less, or a radius of zero or less. a body of no mass at a radius so small its
// cube is under what a float holds is zero too, and not zero over zero.
func TestOmegaOfNothingIsNothing(t *testing.T) {
	for name, c := range map[string]struct{ mass, km float64 }{
		"no mass":          {0, 1000},
		"no mass, a speck": {0, 1e-120},
		"negative mass":    {-1, 1000},
		"no radius":        {bodies.Sun, 0},
		"negative radius":  {bodies.Sun, -1000},
		"neither":          {0, 0},
	} {
		if got := (&bodies.Body{Mass: c.mass}).Omega(c.km); got != 0 {
			t.Errorf("%s: omega is %v", name, got)
		}
	}
}

// the sun's schwarzschild radius, r_s = 2GM/c^2, is 2.95325 km by the
// iau's GM of 1.32712440018e20 m^3/s^2 (2015, resolution b3) and c of
// 299,792,458 m/s; the mass that GM is at G = 6.6743e-11 is 1.98847e30.
//
// the earth's is 8.87 mm (2 x 3.986004418e14 / c^2 = 8.870e-3 m). a
// thousandth off in the kilometres shows.
func TestSchwarzschildOfTheSunAndTheEarth(t *testing.T) {
	near(t, "the sun's schwarzschild radius", bodies.Schwarzschild(1.98847e30), 2.95325, 1e-4)
	near(t, "the earth's schwarzschild radius", bodies.Schwarzschild(5.9722e24), 8.870e-6, 5e-4)
}

// a body's moons are on orbits worked from its mass: the sun at one
// astronomical unit, tilted a quarter turn, with a moon started an
// eighth of a turn round.
//
// Orbit gives the point it is placed at as the centre, the radius in the
// world's units, the body's north (the tilt takes up to +z), kepler's
// period, the moon's phase, and locks it.
func TestAMoonsOrbitIsKeplersAboutTheBodysNorth(t *testing.T) {
	sun := bodies.Body{Name: "sun", Mass: bodies.Sun, Tilt: math.Pi / 2,
		Moons: []bodies.Moon{{Body: bodies.Body{Name: "earth"}, Orbit: auKm, Phase: 0.25 * math.Pi}}}
	at := maths.V(1, 2, 3)
	o := sun.Orbit(0, at, 0.5)
	if o.Centre != at {
		t.Errorf("the orbit is round %v, not %v", o.Centre, at)
	}
	near(t, "the orbit's radius", o.Radius, auKm*0.5, 1e-12)
	nearVec(t, "the orbit's north", o.Axis, maths.V(0, 0, 1), 1e-12)
	near(t, "the orbit's period", o.Period.Seconds(), siderealYear, 5e-4)
	if o.Phase != 0.25*math.Pi {
		t.Errorf("the moon starts at %v", o.Phase)
	}
	if !o.Locked {
		t.Error("the moon is not locked to the body")
	}
}

// a locked moon keeps one face to the body it goes round: the way to
// the body in the moon's own space is the same a quarter and a half
// of the way round as at the start. an unlocked one would not.
func TestALockedMoonKeepsOneFaceToItsBody(t *testing.T) {
	sun := bodies.Body{Name: "sun", Mass: bodies.Sun, Tilt: 0.3,
		Moons: []bodies.Moon{{Body: bodies.Body{Name: "earth"}, Orbit: auKm, Phase: 1.25}}}
	at := maths.V(1, 2, 3)
	o := sun.Orbit(0, at, 1e-8)
	face := func(t time.Duration) maths.Vec {
		s := o.At(t)
		return s.Orientation.In(at.Sub(s.Origin))
	}
	start := face(0)
	for _, quarter := range []time.Duration{o.Period / 4, o.Period / 2, 3 * o.Period / 4} {
		nearVec(t, "the way to the body", face(quarter), start, 1e-9)
	}
}

// the orbits go the way the package spins: prograde, from +x toward -z, round
// the body's north. a moon at phase zero is on +x, and a quarter of a period on
// it is on -z, the radius out; half a period on, -x.
func TestAMoonGoesRoundProgradeFromXTowardMinusZ(t *testing.T) {
	sun := bodies.Body{Name: "sun", Mass: bodies.Sun,
		Moons: []bodies.Moon{{Body: bodies.Body{Name: "earth"}, Orbit: auKm}}}
	at := maths.V(1, 2, 3)
	o := sun.Orbit(0, at, 1e-8)
	r := auKm * 1e-8
	nearVec(t, "the moon at the start", o.At(0).Origin, at.Add(maths.V(r, 0, 0)), 1e-9)
	nearVec(t, "a quarter on", o.At(o.Period/4).Origin, at.Add(maths.V(0, 0, -r)), 1e-9)
	nearVec(t, "half on", o.At(o.Period/2).Origin, at.Add(maths.V(-r, 0, 0)), 1e-9)
}

// a moon is placed on the orbit Orbit gives it, so the moon is as
// timed and as locked as that one is.
func TestPlaceMovesAMoonOnItsOrbit(t *testing.T) {
	sun := bodies.Body{Name: "sun", Mass: bodies.Sun, Tilt: 0.3,
		Moons: []bodies.Moon{{Body: bodies.Body{Name: "earth", Kind: bodies.Star}, Orbit: auKm, Phase: 1.25}}}
	at := maths.V(1, 2, 3)
	w := thing.NewWorld()
	if err := sun.Place(w, at, 1e-8, 2); err != nil {
		t.Fatal(err)
	}
	if got := solidNamed(t, w, "earth").Motion; got != thing.Motion(sun.Orbit(0, at, 1e-8)) {
		t.Errorf("the moon is on %v", got)
	}
}

// the limit on an orbit is 292 years, which a Duration holds: 291.999
// years round is made, and 292.001 is refused with the number named, for
// a moon and for a pair.
//
// the masses are kepler's third law solved for the year asked, so a
// thousandth more on the radius, a second more on the day, a different
// two or three in the 2 pi, or a limit of 293, shows.
func TestAnOrbitJustUnderTwoHundredNinetyTwoYearsIsMade(t *testing.T) {
	const km = 1e9
	for _, c := range []struct {
		years float64
		made  bool
	}{{291.999, true}, {292.001, false}} {
		total := massFor(c.years, km*1000)
		parent := bodies.Body{Name: "p", Kind: bodies.Star, Radius: 1, Mass: total,
			Moons: []bodies.Moon{{Body: bodies.Body{Name: "m", Kind: bodies.Star, Radius: 1}, Orbit: km}}}
		w := thing.NewWorld()
		err := parent.Place(w, maths.Vec{}, 1e-6, 2)
		half := bodies.Body{Name: "h", Kind: bodies.Star, Radius: 1, Mass: total / 2}
		pw := thing.NewWorld()
		perr := bodies.Binary(pw, half, half, maths.Vec{}, km, 1e-6, 2)
		if c.made {
			if err != nil || perr != nil {
				t.Fatalf("%v years: a moon is %v, a pair %v", c.years, err, perr)
			}
			want := c.years * 365.25 * 86400
			near(t, "a moon's period", solidNamed(t, w, "m").Motion.(thing.Orbit).Period.Seconds(), want, 1e-9)
			near(t, "a pair's period", pw.Things[0].Motion.(thing.Orbit).Period.Seconds(), want, 1e-9)
			continue
		}
		for _, e := range []error{err, perr} {
			if e == nil || !strings.Contains(e.Error(), "292") {
				t.Errorf("%v years is %v", c.years, e)
			}
		}
	}
}

// a parent of no mass has no period to refuse, and its moons are not
// timed.
//
// a parent of a kilogram or less, or a moon of a metre or less, is timed
// by its own numbers all the same: half a kilogram with a moon at 1e10 km,
// and a trillionth of a kilogram with a moon half a metre out
// (T = 2 pi sqrt(0.125 / (G 1e-12)), about 8,600 years), are refused.
func TestOrbitsOfNoMassAndOfSmallNumbers(t *testing.T) {
	moon := func(km float64) []bodies.Moon {
		return []bodies.Moon{{Body: bodies.Body{Name: "m", Kind: bodies.Star, Radius: 1}, Orbit: km}}
	}
	w := thing.NewWorld()
	none := bodies.Body{Name: "none", Kind: bodies.Star, Radius: 1, Moons: moon(1e10)}
	if err := none.Place(w, maths.Vec{}, 1e-6, 2); err != nil {
		t.Fatalf("a parent of no mass: %v", err)
	}
	if p := solidNamed(t, w, "m").Motion.(thing.Orbit).Period; p != 0 {
		t.Errorf("a moon of a parent of no mass goes round in %v", p)
	}
	for name, c := range map[string]struct {
		mass, km float64
	}{
		"half a kilogram": {0.5, 1e10},
		"half a metre":    {1e-12, 5e-4},
	} {
		b := bodies.Body{Name: "b", Kind: bodies.Star, Radius: 1, Mass: c.mass, Moons: moon(c.km)}
		if err := b.Place(thing.NewWorld(), maths.Vec{}, 1e-6, 2); err == nil ||
			!strings.Contains(err.Error(), "292") {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// an orbit of no radius is not timed, and is not refused, round the smallest
// mass a float holds: G times it is under what a float holds, so timing it
// would be zero over zero, no number, and refused.
func TestAnOrbitOfNoRadiusRoundASpeckIsNotRefused(t *testing.T) {
	speck := bodies.Body{Name: "speck", Kind: bodies.Star, Radius: 1,
		Mass:  math.SmallestNonzeroFloat64,
		Moons: []bodies.Moon{{Body: bodies.Body{Name: "m", Kind: bodies.Star, Radius: 1}}}}
	w := thing.NewWorld()
	if err := speck.Place(w, maths.Vec{}, 1e-6, 2); err != nil {
		t.Errorf("a moon at nought round a speck: %v", err)
	}
}
