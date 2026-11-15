package template

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/world"
)

// Earth is the terran world, with a thin blue air, clouds, cities at
// night, and the moon.
func Earth() (bodies.Body, error) {
	ground, err := bodies.NewGround(
		world.Terrain{Seed: 5, Ocean: 0.71, Plates: 12}, terran())
	if err != nil {
		return bodies.Body{}, fmt.Errorf("earth: %w", err)
	}
	air, err := bodies.NewAir(km(8.5), bodies.PerKilometre(0.012),
		swatches.Light(swatches.EarthAir))
	if err != nil {
		return bodies.Body{}, fmt.Errorf("earth: %w", err)
	}
	clouds, err := bodies.NewClouds(bodies.NewShell(km(2), km(9)),
		maths.Percent(55), bodies.PerKilometre(1), 5)
	if err != nil {
		return bodies.Body{}, fmt.Errorf("earth: %w", err)
	}
	// a city's light is a few thousandths of the sun's: a night
	// shot exposes for it, and by day it is nothing
	cities, err := bodies.NewCities(3, maths.Percent(12),
		maths.Percent(5), swatches.Light(swatches.City), 5)
	if err != nil {
		return bodies.Body{}, fmt.Errorf("earth: %w", err)
	}
	// the oval at sixty-seven degrees magnetic, eleven degrees off
	// the spin, a hundred to three hundred kilometres up, green
	// with a purple fringe and red at the top
	aurora, err := bodies.NewAurora(bodies.NewShell(km(100), km(300)),
		deg(67), deg(4), deg(11), maths.Share(0.006),
		maths.Percent(70), 5,
		bodies.NewAuroraColours(
			swatches.Light(swatches.AuroraPurple),
			swatches.Light(swatches.AuroraGreen),
			swatches.Light(swatches.AuroraRed)))
	if err != nil {
		return bodies.Body{}, fmt.Errorf("earth: %w", err)
	}
	moon, err := theMoon()
	if err != nil {
		return bodies.Body{}, err
	}
	earth, err := bodies.NewBody("earth", bodies.Rock, km(6371),
		kg(5.972e24), ground, air, clouds, cities, aurora, moon)
	if err != nil {
		return bodies.Body{}, err
	}
	return earth.Turning(hours(23.93), deg(23.44)), nil
}

// theMoon is the earth's moon on its orbit.
func theMoon() (bodies.Moon, error) {
	ground, err := bodies.NewGround(
		world.Terrain{Seed: 6, Ocean: 0.3, Plates: 5, Cold: 0.2},
		look(0.2, palette(swatches.Moon)))
	if err != nil {
		return bodies.Moon{}, fmt.Errorf("the moon: %w", err)
	}
	body, err := bodies.NewBody("moon", bodies.Rock, km(1737.4),
		kg(7.342e22), ground)
	if err != nil {
		return bodies.Moon{}, err
	}
	return bodies.NewMoon(body, km(384400), deg(0))
}

// terran is the earth's look, from the paint box's terran set.
func terran() world.Look {
	return look(0.4, palette(swatches.Earth))
}
