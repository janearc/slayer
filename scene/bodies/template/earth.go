package template

import (
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/world"
)

// Earth is the terran world, with a thin blue air, clouds, cities at
// night, and the moon.
func Earth() bodies.Body {
	return bodies.Body{Name: "earth", Kind: bodies.Rock, Radius: 6371, Mass: 5.972e24,
		Day: hours(23.93), Tilt: deg(23.44),
		Ground: &bodies.Ground{
			Terrain: world.Terrain{Seed: 5, Ocean: 0.71, Plates: 12},
			Look:    terran(),
		},
		Air: &bodies.Air{Height: 8.5, Dense: 0.012,
			Colour: swatches.Light(swatches.EarthAir)},
		Clouds: &bodies.Clouds{Low: 2, High: 9, Cover: 0.55, Dense: 1, Seed: 5},
		// a city's light is a few thousandths of the sun's: a night
		// shot exposes for it, and by day it is nothing
		Cities: &bodies.Cities{Rungs: 3, Share: 0.12, Metro: 0.05,
			Glow: swatches.Light(swatches.City), Seed: 5},
		// the oval at sixty-seven degrees magnetic, eleven degrees off
		// the spin, a hundred to three hundred kilometres up, green
		// with a purple fringe and red at the top
		Aurora: &bodies.Aurora{Low: 100, High: 300, Lat: deg(67), Width: deg(4),
			Tilt: deg(11), Strength: 0.006, Curtains: 0.7, Seed: 5,
			Fringe: swatches.Light(swatches.AuroraPurple),
			Bottom: swatches.Light(swatches.AuroraGreen),
			Top:    swatches.Light(swatches.AuroraRed)},
		Moons: []bodies.Moon{{Body: bodies.Body{Name: "moon", Kind: bodies.Rock, Radius: 1737.4,
			Mass: 7.342e22, Ground: &bodies.Ground{
				Terrain: world.Terrain{Seed: 6, Ocean: 0.3, Plates: 5, Cold: 0.2},
				Look:    look(0.2, palette(swatches.Moon)),
			}}, Orbit: 384400}},
	}
}

// terran is the earth's look, from the paint box's terran set.
func terran() world.Look {
	return look(0.4, palette(swatches.Earth))
}
