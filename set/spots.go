package set

import (
	"math"

	"github.com/janearc/slayer/maths"
	yaml "go.yaml.in/yaml/v3"
)

// A Spotlight is a lamp kept to a cone: its place, the way it aims, the
// cone's half-angle and how much of it the light takes to fade at the
// edge, both in degrees, and its light, a black body's colour at a
// temperature, as libtheme gives it, so many times over.
type Spotlight struct {
	name                  string
	at, aim               [3]float64
	cone, soft            float64
	temperature, strength float64
}

// Name is the spot's name.
func (made Spotlight) Name() string { return made.name }

// At is where it stands, in the world's space.
func (made Spotlight) At() maths.Vec { return fromMap(made.at) }

// Aim is the way it points, in the world's space, as long as written.
func (made Spotlight) Aim() maths.Vec { return fromMap(made.aim) }

// Cone is the cone's half-angle and its soft edge, in degrees.
func (made Spotlight) Cone() (cone, soft float64) { return made.cone, made.soft }

// Shine is its temperature in kelvin and how many times its colour its
// light is, one unit away down the middle.
func (made Spotlight) Shine() (temperature, strength float64) {
	return made.temperature, made.strength
}

// spotlights reads each spot, by its name.
func (reading reader) spotlights(_, value *yaml.Node) error {
	return reading.each(value, func(name, block *yaml.Node) error {
		made := Spotlight{name: name.Value}
		var source string
		err := reading.keys(block, map[string]step{
			"at":  reading.three(&made.at, -farthest, farthest),
			"aim": reading.three(&made.aim, -farthest, farthest),
			"cone": reading.number(&made.cone, math.SmallestNonzeroFloat64,
				180),
			"soft":        reading.number(&made.soft, 0, 180),
			"temperature": reading.number(&made.temperature, 1, hottest),
			"strength":    reading.number(&made.strength, 0, brightest),
			"source":      reading.source(&source),
		}, nil)
		if err == nil {
			err = reading.all(block, "a spot", "at", "aim", "cone", "soft",
				"temperature", "strength", "source")
		}
		if err == nil && made.aim == [3]float64{} {
			err = reading.refuse(block, "spot %s aims at nought, nought "+
				"and nought, which is no way", made.name)
		}
		reading.set.world.spots = append(reading.set.world.spots, made)
		return err
	})
}
