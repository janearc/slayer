package set

import (
	"math"

	"github.com/janearc/slayer/maths"
	yaml "gopkg.in/yaml.v3"
)

// A Sphere is a ball of a material, its middle at a place, so big; and,
// where the set says, carried by a motion, which then says where it is.
type Sphere struct {
	name, material, motion string
	middle                 [3]float64
	radius                 float64
}

// Name is the sphere's name.
func (made Sphere) Name() string { return made.name }

// Material is the name of the material it wears.
func (made Sphere) Material() string { return made.material }

// Middle is where its middle stands before any motion, in the world's
// space.
func (made Sphere) Middle() maths.Vec { return fromMap(made.middle) }

// Radius is its radius, in the world's units.
func (made Sphere) Radius() float64 { return made.radius }

// Motion is the name of the motion that carries it, and whether one
// does.
func (made Sphere) Motion() (string, bool) {
	return made.motion, made.motion != ""
}

// A Wisp is a ball of gas made ragged by noise: thick at its middle and
// thinning to nothing at its radius, and the colour it gives back of the
// light that crosses it.
//
// its noise has a seed that makes its wisps its own, a size for them,
// and how much of the density they may take.
type Wisp struct {
	name                         string
	colour                       Colour
	middle                       [3]float64
	radius, thick, scale, amount float64
	seed                         float64
}

// Name is the wisp's name.
func (made Wisp) Name() string { return made.name }

// Colour is what it gives back of the light.
func (made Wisp) Colour() Colour { return made.colour }

// Middle is where its middle stands, in the world's space.
func (made Wisp) Middle() maths.Vec { return fromMap(made.middle) }

// Ball is its radius, in the world's units, and its density at the
// middle.
func (made Wisp) Ball() (radius, thick float64) {
	return made.radius, made.thick
}

// Ragged is its seed, how big its wisps are, in the world's units, and
// how much of the density they may take.
func (made Wisp) Ragged() (seed uint64, scale, amount float64) {
	return uint64(made.seed), made.scale, made.amount
}

// spheres reads each sphere, by its name.
func (reading reader) spheres(_, value *yaml.Node) error {
	return reading.each(value, func(name, block *yaml.Node) error {
		made := Sphere{name: name.Value}
		var source string
		err := reading.keys(block, map[string]step{
			"material": reading.text(&made.material),
			"middle":   reading.three(&made.middle, -farthest, farthest),
			"radius": reading.number(&made.radius,
				math.SmallestNonzeroFloat64, farthest),
			"motion": reading.text(&made.motion),
			"source": reading.source(&source),
		}, nil)
		if err == nil {
			err = reading.all(block, "a sphere", "material", "middle",
				"radius", "source")
		}
		reading.set.world.spheres = append(reading.set.world.spheres, made)
		return err
	})
}

// wisps reads each wisp, by its name.
func (reading reader) wisps(_, value *yaml.Node) error {
	return reading.each(value, func(name, block *yaml.Node) error {
		made := Wisp{name: name.Value}
		var source string
		over := math.SmallestNonzeroFloat64
		err := reading.keys(block, map[string]step{
			"colour": reading.colour(&made.colour),
			"middle": reading.three(&made.middle, -farthest, farthest),
			"radius": reading.number(&made.radius, over, farthest),
			"thick":  reading.number(&made.thick, over, brightest),
			"seed":   reading.whole(&made.seed, 0, mostSeed),
			"scale":  reading.number(&made.scale, over, farthest),
			"amount": reading.number(&made.amount, over, 1),
			"source": reading.source(&source),
		}, nil)
		if err == nil {
			err = reading.all(block, "a wisp", "colour", "middle", "radius",
				"thick", "seed", "scale", "amount", "source")
		}
		reading.set.world.wisps = append(reading.set.world.wisps, made)
		return err
	})
}
