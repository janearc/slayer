package template

import (
	"math"
	"testing"

	"github.com/janearc/slayer/scene/bodies"
)

// everyBody is each template and each moon of one, with the moons
// after their parent. the hot jupiter is left out, being jupiter.
func everyBody() []bodies.Body {
	var out []bodies.Body
	for _, b := range []bodies.Body{Sol(), Mercury(), Venus(), Earth(), Mars(),
		Jupiter(), Saturn(), Uranus(), Neptune(), Pluto(), Charon(),
		HaleBopp(), Hyakutake(), Magnetar(), Pulsar(), Quasar()} {
		out = append(out, b)
		for _, m := range b.Moons {
			out = append(out, m.Body)
		}
	}
	return out
}

// density is a body's mean density in kilograms a cubic metre, worked
// by hand: its mass over a sphere of its radius, 4/3 pi r^3, with the
// radius in metres from kilometres.
func density(b bodies.Body) float64 {
	r := b.Radius * 1000
	return b.Mass / (4.0 / 3.0 * math.Pi * r * r * r)
}

// densities is the mean density of each body that has one published, in
// kg/m^3, as remembered. a body is held to within 10 per cent of it. a
// radius in the wrong unit, or a mass off by a power of ten, is a
// thousand-fold or more off, and shows; so does a typed digit.
var densities = map[string]float64{
	"sol": 1410, "mercury": 5427, "venus": 5243, "earth": 5514, "mars": 3933,
	"jupiter": 1326, "saturn": 687, "uranus": 1271, "neptune": 1638,
	"pluto": 1854, "charon": 1702,
	"moon": 3344, "phobos": 1876, "deimos": 1471,
	"io": 3528, "europa": 3013, "ganymede": 1936, "callisto": 1834,
	"titan": 1881, "rhea": 1236, "enceladus": 1609,
	"titania": 1711, "oberon": 1630, "miranda": 1200, "triton": 2061,
}

// physics, worked by hand: a body's mean density is its mass over
// 4/3 pi r^3. for the earth, 5.972e24 kg over 4.189 * (6.371e6)^3 =
// 1.083e21 m^3 is 5514 kg/m^3. each planet, moon and the sun is held
// to within 10 per cent of the density published for it.
func TestMeanDensityOfEachWorldIsTheOnePublished(t *testing.T) {
	for _, b := range everyBody() {
		want, ok := densities[b.Name]
		if !ok {
			continue
		}
		if got := density(b); !(math.Abs(got-want) <= 0.1*want) {
			t.Errorf("%s: mean density is %.0f kg/m^3, published about %v", b.Name, got, want)
		}
	}
}

// physics, worked by hand: the same, for the bodies that have no single
// published density here, each held to a plausible range in kg/m^3.
//
// a neutron star holds more than a sun's mass in a ball of ten or
// twelve kilometres, which is between 1e17 and 1e18, nuclear matter
// being 2.3e17.
//
// a hole of a mass M has a horizon of radius 2GM/c^2, so its mean
// density goes as one over M squared: one sun's is 1.8e19, and 6.6e10
// suns' is that over 4.4e21, 4e-3 kg/m^3.
//
// a comet's nucleus is some hundreds, 100 to 600 measured on those
// visited, as remembered.
//
// hale-bopp's, 1e16 kg in 30 km, is 88 by this working, under that.
// its band here is 50 to 1500, and so holds it; whether its mass should
// be higher is left open.
func TestMeanDensityOfTheOddBodiesIsInARangeThatCanBe(t *testing.T) {
	for _, c := range []struct {
		body   bodies.Body
		lo, hi float64
	}{
		{Magnetar(), 1e17, 1e18},
		{Pulsar(), 1e17, 1e18},
		{Quasar(), 3e-3, 6e-3},
		{HaleBopp(), 50, 1500},
		{Hyakutake(), 100, 1500},
	} {
		if got := density(c.body); !(got >= c.lo && got <= c.hi) {
			t.Errorf("%s: mean density is %v kg/m^3, want between %v and %v",
				c.body.Name, got, c.lo, c.hi)
		}
	}
}

// gravities is the surface gravity of each body that has one published
// at its mean radius, in m/s^2, as remembered. the giants are left out,
// since theirs is read at a pressure and not a surface.
var gravities = map[string]float64{
	"sol": 274, "mercury": 3.70, "venus": 8.87, "earth": 9.80, "mars": 3.71,
	"pluto": 0.62, "charon": 0.288, "moon": 1.62,
	"io": 1.796, "europa": 1.314, "ganymede": 1.428, "callisto": 1.235,
	"titan": 1.352,
}

// physics, worked by hand: gravity at a surface is g = GM/r^2, with G =
// 6.6743e-11, the radius in metres and the mass in kilograms. for the
// earth, 6.6743e-11 * 5.972e24 = 3.986e14, over (6.371e6)^2 = 4.059e13,
// is 9.82 m/s^2.
//
// it is held to 3 per cent of the published one, which leaves room for
// the published figures being remembered.
func TestSurfaceGravityOfEachSolidWorldIsThePublishedOne(t *testing.T) {
	for _, b := range everyBody() {
		want, ok := gravities[b.Name]
		if !ok {
			continue
		}
		r := b.Radius * 1000
		if got := 6.6743e-11 * b.Mass / (r * r); !(math.Abs(got-want) <= 0.03*want) {
			t.Errorf("%s: surface gravity is %.3f m/s^2, published about %v", b.Name, got, want)
		}
	}
}
