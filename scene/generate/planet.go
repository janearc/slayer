package generate

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/texture"
)

// Makeup is what a planet is made of, as shares of its mass. The four
// shares are each between zero and one and add to one.
type Makeup struct {
	// Metal is iron and nickel, mostly in a core.
	Metal float64
	// Rock is silicates.
	Rock float64
	// Ice is water and other volatiles, frozen or not.
	Ice float64
	// Gas is hydrogen and helium.
	Gas float64
}

// Years is a span of time in Julian years. A time.Duration ends at about
// 292 years, short of a planet's age or the year of a wide orbit.
type Years float64

// Sun is what a planet needs of the star it goes round.
type Sun struct {
	// Mass sets the planet's year from its distance.
	Mass measure.Mass
	// Luminosity sets how warm the planet is at its distance, as a
	// multiple of the Sun's.
	Luminosity float64
}

// Orbit is where a body goes round its primary. One of Distance, Year or
// Flux is given; the others follow, and when Distance and Year are both
// given they must agree.
type Orbit struct {
	// Distance is the semi-major axis.
	Distance measure.Length
	// Year is the time once round.
	Year Years
	// Flux is the light the planet gets from its sun, in the Earth's: it
	// stands where its sun's light comes to that, and is given in place
	// of a Distance, never beside one. See Planet.Distance.
	Flux float64
	// Eccentricity is the orbit's stretch: zero a circle, under one an
	// ellipse.
	Eccentricity float64
	// Inclination is the tilt of the orbit against the system's plane.
	Inclination measure.Angle
}

// Planet is what a caller gives to make a planet.
type Planet struct {
	Name   string
	Mass   measure.Mass
	Makeup Makeup
	// Day is how long it takes to turn once; Tilt is its axis against its
	// orbit.
	Day  time.Duration
	Tilt measure.Angle
	// Locked is a planet whose day is its year, one face always toward
	// its sun; see Turn. Its Day follows from its orbit.
	Locked bool
	// Clouds overrides the derived cloud layer as an artistic choice. A nil
	// value keeps the default; zero Cover explicitly gives a clear sky.
	Clouds *CloudLook
	// AirLook is an authored appearance for the existing atmospheric shell;
	// nil retains the derived default.
	AirLook *AirLook
	// GiantLook is an authored, made-up gas appearance. Nil keeps the
	// generated default. Its seed is the planet's seed.
	GiantLook *texture.Banded
	// GiantGas is an authored override for the existing giant volume's depth
	// and opacity; nil retains the template defaults.
	GiantGas *GiantGasLook
	// GroundLook overrides a rocky world's look with invented artistic values.
	GroundLook *GroundLook
	// Sun and Orbit place it. A rogue planet has no sun: a zero Sun.
	Sun   Sun
	Orbit Orbit
	// Age is how long since it formed.
	Age Years
	// Seed picks everything the numbers leave open: its ground, its
	// weather, where its cities are.
	Seed uint64
}

// CloudLook describes a single procedural cloud shell. Heights are
// kilometres above the surface, Cover is a share, and Loss is extinction
// per kilometre. These are authored appearance values, not predictions of
// cloud chemistry or weather.
type CloudLook struct {
	Base, Top   float64
	Cover, Loss float64
	// Weather is empty for the body's usual pattern, or "bands" for
	// latitude-based weather regardless of the body's spin or orbit;
	// "soft-bands" also leaves fine billows unturned by storms.
	Weather string
	// StormTurn, when given, is the artistic turn at a storm's heart in
	// radians. Nil retains the existing weather's turn.
	StormTurn *float64
}

// AirLook describes an artistic single-shell atmosphere. Height is its
// scale height in kilometres, Loss the extinction per kilometre at the
// surface, and Colour its linear RGB scattering tint. It is not a spectral
// gas composition or a separate Rayleigh/Mie model.
type AirLook struct {
	Height float64
	Loss   float64
	Colour maths.Light
}

// GiantGasLook is an artistic control for the existing giant density ramp.
// Depth is a share of the body's radius; Opacity is the constructor's
// dimensionless density control. It is not a spectral atmosphere model.
type GiantGasLook struct {
	Depth, Opacity float64
}

// GroundLook controls the coarse ocean share and two painted detail shares.
// Ridge contrast is colour only, not displaced mountain geometry.
type GroundLook struct{ Ocean, Variation, RidgeContrast float64 }

// Water is the state a planet's water is in at its surface.
type Water int

// The states of water, by what the surface temperature and pressure allow.
const (
	Dry Water = iota
	Ice
	Ocean
	Steam
)

// Derived is what follows from a Planet's numbers.
type Derived struct {
	Radius bodies.Span
	// Gravity is the pull at the surface, in metres per second squared.
	Gravity float64
	// Temperature is the planet's equilibrium temperature in kelvin, from
	// its sun's light at its distance.
	Temperature float64
	// Air is whether it keeps an atmosphere, and Water what state its
	// water is in.
	Air   bool
	Water Water
	// Kind is the body kind it is drawn as: Rock or Giant.
	Kind bodies.Kind
}

// EarthMass is the Earth's mass in kilograms, the unit a set says a
// planet's mass in (NASA's Earth fact sheet, Williams, NSSDCA).
const EarthMass = 5.9722e24

// AstronomicalUnit is the astronomical unit in metres, exact by IAU 2012
// Resolution B2, the unit a set says a planet's distance in.
const AstronomicalUnit = 1.495978707e11

// Derive works out what follows from a planet's numbers, or says which
// number is out of range.
func Derive(planet Planet) (Derived, error) {
	if err := checked(planet); err != nil {
		return Derived{}, err
	}
	metres, err := distance(planet)
	if err != nil {
		return Derived{}, err
	}
	kind := kindOf(planet.Makeup)
	radius := radiusOf(planet, metres, kind)
	temperature := equilibrium(planet.Sun.Luminosity, metres, albedo(kind))
	air := keepsAir(planet.Mass, radius, temperature, kind)
	return Derived{
		Radius:      bodies.Kilometres(radius / metresPerKilometre),
		Gravity:     gravity(planet.Mass, radius),
		Temperature: temperature,
		Air:         air,
		Water:       waterOf(planet.Makeup.Ice, temperature, air),
		Kind:        kind,
	}, nil
}

// checked refuses a planet whose numbers cannot make one: no mass, a
// makeup whose shares are out of range or do not add to one, or an orbit
// that cannot be placed.
func checked(planet Planet) error {
	if !planet.Mass.Positive() {
		return fmt.Errorf("generate: %s has no mass", planet.Name)
	}
	makeup := planet.Makeup
	total := 0.0
	for _, share := range []float64{makeup.Metal, makeup.Rock, makeup.Ice,
		makeup.Gas} {
		if !(share >= 0 && share <= 1) {
			return fmt.Errorf("generate: %s's makeup has a share of %v, "+
				"outside nought to one", planet.Name, share)
		}
		total += share
	}
	if math.Abs(total-1) > makeupSlack {
		return fmt.Errorf("generate: %s's makeup adds to %v, not one",
			planet.Name, total)
	}
	if err := checkedOrbit(planet); err != nil {
		return err
	}
	_, err := Turn(planet)
	return err
}

// checkedOrbit refuses an orbit that cannot be placed: a negative
// distance or year, a lit planet with neither, or a year with no sun's
// mass to turn it into a distance.
func checkedOrbit(planet Planet) error {
	orbit, sun := planet.Orbit, planet.Sun
	switch {
	case orbit.Distance.InMetres() < 0 || orbit.Year < 0 || orbit.Flux < 0:
		return fmt.Errorf("generate: %s's orbit is less than nothing",
			planet.Name)
	case sun.Luminosity < 0:
		return fmt.Errorf("generate: %s's sun gives less than no light",
			planet.Name)
	case orbit.Year > 0 && !sun.Mass.Positive():
		return fmt.Errorf("generate: %s has a year and no sun's mass to "+
			"go round", planet.Name)
	case sun.Luminosity > 0 && orbit.Distance.InMetres() == 0 &&
		orbit.Year == 0 && orbit.Flux == 0:
		return fmt.Errorf("generate: %s has a sun and no distance, year "+
			"or light it gets",
			planet.Name)
	}
	return nil
}

// Make makes a planet's body: its core from Derive, and the parts its
// numbers call for.
func Make(planet Planet) (bodies.Body, error) {
	derived, err := Derive(planet)
	if err != nil {
		return bodies.Body{}, err
	}
	if derived.Kind == bodies.Giant && planet.Clouds != nil {
		return bodies.Body{}, fmt.Errorf("generate: %s: a cloud shell is "+
			"not supported on a gas giant", planet.Name)
	}
	if derived.Kind == bodies.Giant && planet.AirLook != nil {
		return bodies.Body{}, fmt.Errorf("generate: %s: authored air requires a rocky planet", planet.Name)
	}
	if derived.Kind == bodies.Giant && planet.GroundLook != nil {
		return bodies.Body{}, fmt.Errorf("generate: %s: ground look requires a rocky planet", planet.Name)
	}
	if planet.GroundLook != nil {
		for _, value := range []float64{planet.GroundLook.Ocean, planet.GroundLook.Variation, planet.GroundLook.RidgeContrast} {
			if !(value >= 0 && value <= 1) {
				return bodies.Body{}, fmt.Errorf("generate: %s: ground look shares must be from nought to one", planet.Name)
			}
		}
	}
	if derived.Kind != bodies.Giant && planet.GiantLook != nil {
		return bodies.Body{}, fmt.Errorf("generate: %s: giant appearance requires a gas giant", planet.Name)
	}
	if planet.GiantGas != nil {
		gas := planet.GiantGas
		if derived.Kind != bodies.Giant {
			return bodies.Body{}, fmt.Errorf("generate: %s: giant gas requires a gas giant", planet.Name)
		}
		if !(gas.Depth > 0 && gas.Depth <= 1) || math.IsNaN(gas.Opacity) || math.IsInf(gas.Opacity, 0) || gas.Opacity < 0 || gas.Opacity > 100000 {
			return bodies.Body{}, fmt.Errorf("generate: %s: giant gas depth must be above zero and at most one, and opacity from zero to 100000", planet.Name)
		}
	}
	if planet.Clouds != nil && planet.Clouds.Weather != "" &&
		planet.Clouds.Weather != "bands" && planet.Clouds.Weather != "soft-bands" {
		return bodies.Body{}, fmt.Errorf("generate: %s: unknown cloud weather %q",
			planet.Name, planet.Clouds.Weather)
	}
	if planet.Clouds != nil && planet.Clouds.StormTurn != nil {
		turn := *planet.Clouds.StormTurn
		if math.IsNaN(turn) || math.IsInf(turn, 0) || turn < 0 || turn > 10 {
			return bodies.Body{}, fmt.Errorf("generate: %s: cloud storm turn "+
				"must be from 0 to 10 radians", planet.Name)
		}
	}
	parts, err := partsOf(planet, derived)
	if err != nil {
		return bodies.Body{}, fmt.Errorf("generate: %s: %w", planet.Name,
			err)
	}
	body, err := bodies.NewBody(planet.Name, derived.Kind, derived.Radius,
		planet.Mass, parts...)
	if err != nil {
		return bodies.Body{}, err
	}
	turn, err := Turn(planet)
	if err != nil {
		return bodies.Body{}, err
	}
	return body.Turning(turn, planet.Tilt), nil
}
