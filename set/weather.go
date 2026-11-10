package set

import (
	"math"

	"gopkg.in/yaml.v3"
)

// Weather is a planet's weather as a fluid makes it, where a set asks
// for it in place of painted storms.
//
// it says how many cells round its grid, how many seconds its day is,
// how long its friction and its bands take, its storms, a seed, the
// moment of its weather a shot shows, in seconds since the weather
// began, and how often to keep it on the way.
type Weather struct {
	cells, day, drag, relax float64
	rate, wide, strength    float64
	seed, moment, every     float64
}

// Weather is the planet's weather, and whether the set gives it any.
func (made Planet) Weather() (Weather, bool) {
	if made.weather == nil {
		return Weather{}, false
	}
	return *made.weather, true
}

// Grid is how many cells round the weather's grid.
func (made Weather) Grid() int { return int(made.cells) }

// Turning is the planet's day in seconds, its friction's time and its
// bands' time, in seconds.
func (made Weather) Turning() (day, drag, relax float64) {
	return made.day, made.drag, made.relax
}

// Storms is how many storms break out a second over a square metre, how
// wide each is in metres, and how fast each spins, radians a second.
func (made Weather) Storms() (rate, wide, strength float64) {
	return made.rate, made.wide, made.strength
}

// Seed is the weather's seed.
func (made Weather) Seed() uint64 { return uint64(made.seed) }

// Moment is the moment of the weather a shot shows, and how often it is
// kept on the way, both in seconds.
func (made Weather) Moment() (moment, every float64) {
	return made.moment, made.every
}

// Heat is the temperature of the planet's cloud tops and the hottest a
// clearing shows, in kelvin, and whether the set says.
func (made Planet) Heat() (top, hottest float64, has bool) {
	if made.heat == nil {
		return 0, 0, false
	}
	return made.heat[0], made.heat[1], true
}

// heated reads a planet's heat, both parts said.
func (reading reader) heated(into **[2]float64) step {
	return func(_, value *yaml.Node) error {
		heat := [2]float64{math.NaN(), math.NaN()}
		err := reading.keys(value, map[string]step{
			"tops":    reading.number(&heat[0], 1, 1e5),
			"hottest": reading.number(&heat[1], 1, 1e5),
		}, nil)
		if err != nil {
			return err
		}
		if math.IsNaN(heat[0]) || math.IsNaN(heat[1]) || heat[1] < heat[0] {
			return reading.refuse(value, "heat says its tops and its "+
				"hottest, no cooler than its tops")
		}
		*into = &heat
		return nil
	}
}

// Cooking is the column whose haze fully cooked air bears and the
// seconds over which the air cooks, and whether the set says.
func (made Planet) Cooking() (column string, seconds float64, has bool) {
	if made.cooking == nil {
		return "", 0, false
	}
	return made.cooking.column, made.cooking.seconds, true
}

// cooking is how a planet's air cooks: the column whose haze cooked air
// bears, and over how many seconds.
type cooking struct {
	column  string
	seconds float64
}

// cooked reads how a planet's air cooks, both parts said.
func (reading reader) cooked(into **cooking) step {
	return func(_, value *yaml.Node) error {
		made := cooking{seconds: math.NaN()}
		err := reading.keys(value, map[string]step{
			"column": reading.text(&made.column),
			"time":   reading.number(&made.seconds, 1, 1e12),
		}, nil)
		if err != nil {
			return err
		}
		if made.column == "" || math.IsNaN(made.seconds) {
			return reading.refuse(value, "cooking says its column and time")
		}
		*into = &made
		return nil
	}
}

// Airglow is the planet's sodium airglow looking straight down, watts a
// square metre a steradian, the band it glows in, degrees either side of
// the equator, the steepest share of the band taken for its fronts, and
// whether the set says.
func (made Planet) Airglow() (radiance, band, fronts float64, has bool) {
	if made.airglow == nil {
		return 0, 0, 0, false
	}
	return made.airglow[0], made.airglow[1], made.airglow[2], true
}

// fronts is the share of a band taken for its fronts when the set says
// none: a fifth.
const fronts = 0.2

// glowing reads a planet's airglow, its radiance and band said, its
// fronts said or a fifth.
func (reading reader) glowing(into **[3]float64) step {
	return func(_, value *yaml.Node) error {
		glow := [3]float64{math.NaN(), math.NaN(), fronts}
		err := reading.keys(value, map[string]step{
			"radiance": reading.number(&glow[0], 0, 1e9),
			"band":     reading.number(&glow[1], 0.1, 90),
			"fronts":   reading.number(&glow[2], 0.001, 0.999),
		}, nil)
		if err != nil {
			return err
		}
		if math.IsNaN(glow[0]) || math.IsNaN(glow[1]) {
			return reading.refuse(value, "airglow says its radiance and band")
		}
		*into = &glow
		return nil
	}
}

// weathered reads a planet's weather, every part said.
func (reading reader) weathered(into **Weather) step {
	return func(_, value *yaml.Node) error {
		made := Weather{cells: math.NaN(), day: math.NaN(), drag: math.NaN(),
			relax: math.NaN(), rate: math.NaN(), wide: math.NaN(),
			strength: math.NaN(), seed: math.NaN(), moment: math.NaN(),
			every: math.NaN()}
		err := reading.keys(value, map[string]step{
			"cells": reading.whole(&made.cells, 16, 2048),
			"day":   reading.number(&made.day, 60, 1e9),
			"drag":  reading.number(&made.drag, 1, 1e12),
			"relax": reading.number(&made.relax, 1, 1e12),
			"storms": reading.block(map[string]step{
				"rate":     reading.number(&made.rate, 1e-30, 1),
				"wide":     reading.number(&made.wide, 1, 1e9),
				"strength": reading.number(&made.strength, 1e-12, 1)}),
			"seed":   reading.whole(&made.seed, 0, mostSeed),
			"moment": reading.number(&made.moment, 0, 1e12),
			"every":  reading.number(&made.every, 1, 1e12),
		}, nil)
		if err != nil {
			return err
		}
		for _, part := range []float64{made.cells, made.day, made.drag,
			made.relax, made.rate, made.wide, made.strength, made.seed,
			made.moment, made.every} {
			if math.IsNaN(part) {
				return reading.refuse(value, "weather says its cells, day, "+
					"drag, relax, storms, seed, moment and every")
			}
		}
		if int(made.cells)&(int(made.cells)-1) != 0 {
			return reading.refuse(value, "weather's cells are a power of two")
		}
		*into = &made
		return nil
	}
}
