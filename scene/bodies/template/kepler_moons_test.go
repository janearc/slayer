package template

import (
	"math"
	"testing"

	"github.com/janearc/slayer/scene/bodies"
)

// keplerDays is the period of a circle of km kilometres round a mass
// in kilograms, in days, worked the long way: T = 2 pi sqrt(r^3 / GM),
// with G = 6.6743e-11 m^3 / (kg s^2) and r in metres.
func keplerDays(mass, km float64) float64 {
	r := km * 1000
	return 2 * math.Pi * math.Sqrt(r*r*r/(6.6743e-11*mass)) / 86400
}

// published is each moon's sidereal period in days, as remembered, to five
// figures where sure.
//
// the 1 per cent allowed below is wider than the gap between a point-mass
// circle and the real orbit for every moon here: measured when this was
// written, the widest gaps were the moon's, 0.48 per cent, and miranda's, 0.52.
var published = map[string]float64{
	"moon": 27.322, "phobos": 0.31891, "deimos": 1.2624,
	"io": 1.7691, "europa": 3.5512, "ganymede": 7.1546, "callisto": 16.689,
	"titan": 15.945, "rhea": 4.5175, "enceladus": 1.3702,
	"titania": 8.7059, "oberon": 13.463, "miranda": 1.4135,
	"triton": 5.8769,
}

// physics, worked by hand: a moon goes round its parent in kepler's
// period, T = 2 pi sqrt(r^3 / GM), from the parent's mass alone.
//
// for the moon, r = 3.844e8 m and GM = 6.6743e-11 * 5.972e24 =
// 3.986e14, so r^3 is 5.68e25, over GM 1.425e11, a root of 3.775e5,
// times 2 pi is 2.372e6 s, 27.45 days.
//
// the published sidereal period is 27.32: the point-mass circle leaves
// out the moon's own mass, as thing.Period says.
//
// each of the template's moons, worked so from its parent's mass and
// its orbit, is held to within 1 per cent of the period published for
// it. the direction of triton's orbit is not held here, only its
// period.
func TestEachMoonsKeplerPeriodIsThePublishedOne(t *testing.T) {
	seen := map[string]bool{}
	for _, parent := range []bodies.Body{Earth(), Mars(), Jupiter(), Saturn(),
		Uranus(), Neptune()} {
		for _, m := range parent.Moons {
			name := m.Body.Name
			want, ok := published[name]
			if !ok {
				t.Errorf("%s has no published period here to hold it to", name)
				continue
			}
			seen[name] = true
			got := keplerDays(parent.Mass, m.Orbit)
			if !(math.Abs(got-want) <= 0.01*want) {
				t.Errorf("%s round %s: kepler gives %.5f days, published %v, "+
					"more than 1 per cent apart", name, parent.Name, got, want)
			}
		}
	}
	for name := range published {
		if !seen[name] {
			t.Errorf("a published period is listed for %s, which no template has", name)
		}
	}
}

// physics, worked by hand: the worked period of the moon, above, is
// 27.45 days, and is held to that to the hundredth, so a change to the
// earth's mass or the moon's orbit shows as a day and not as a
// fraction of a per cent.
func TestTheMoonsWorkedPeriodIsTwentySevenPointFourFiveDays(t *testing.T) {
	e := Earth()
	if got := keplerDays(e.Mass, e.Moons[0].Orbit); math.Abs(got-27.45) > 0.01 {
		t.Errorf("the moon's period is %v days, want 27.45", got)
	}
}
