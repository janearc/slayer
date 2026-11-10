package set

import (
	yaml "gopkg.in/yaml.v3"
	"slices"
)

// FuelBed describes an experimental horizontal surface-fuel layer.
// The selected proxy does not model crown ignition or atmospheric feedback.
type FuelBed struct {
	recipe, surface                                                   string
	width, depth, cell, step, east, north, radius, hot, ambient, fuel float64
	transportEast, transportSouth                                     float64
}

// FuelBeds returns a copy of the configured surface experiments.
func (s Set) FuelBeds() []FuelBed { return slices.Clone(s.beds) }

// Recipe names the authored gas-flame display.
func (b FuelBed) Recipe() string { return b.recipe }

// Surface chooses fuel-fraction paint or the terrain's own appearance.
func (b FuelBed) Surface() string { return b.surface }

// Width is the number of cells along east-west.
func (b FuelBed) Width() int { return int(b.width) }

// Depth is the number of cells along north-south.
func (b FuelBed) Depth() int { return int(b.depth) }

// Cell is the horizontal spacing in metres.
func (b FuelBed) Cell() float64 { return b.cell }

// Step is the fixed simulation tick in seconds.
func (b FuelBed) Step() float64 { return b.step }

// East is the bed centre in terrain easting metres.
func (b FuelBed) East() float64 { return b.east }

// North is the bed centre in terrain northing metres.
func (b FuelBed) North() float64 { return b.north }

// Radius is the Gaussian ignition width in metres.
func (b FuelBed) Radius() float64 { return b.radius }

// Hot is the central initial layer temperature in kelvin.
func (b FuelBed) Hot() float64 { return b.hot }

// Ambient is the surrounding layer temperature in kelvin.
func (b FuelBed) Ambient() float64 { return b.ambient }

// Fuel is the initial remaining-fuel mass fraction.
func (b FuelBed) Fuel() float64 { return b.fuel }

// fuelBeds reads the experimental layer through checked YAML inputs.
func (reading reader) fuelBeds(_, node *yaml.Node) error {
	return reading.each(node, func(name, block *yaml.Node) error {
		b := FuelBed{surface: "fuel"}
		if err := reading.keys(block, map[string]step{
			"heat-transport": reading.heatTransport(&b),
			"surface":        reading.word(&b.surface, "fuel", "terrain"),
			"recipe":         reading.text(&b.recipe), "model": reading.only("mandel-grass"),
			"east": reading.number(&b.east, -1e8, 1e8), "north": reading.number(&b.north, -1e8, 1e8),
			"width": reading.whole(&b.width, 8, 32), "depth": reading.whole(&b.depth, 8, 32),
			"cell": reading.quantity(&b.cell, "m", .25, 2), "step": reading.quantity(&b.step, "s", .001, .1),
			"ignition-radius":      reading.quantity(&b.radius, "m", .01, 100),
			"ignition-temperature": reading.quantity(&b.hot, "K", 300, 3000),
			"ambient-temperature":  reading.quantity(&b.ambient, "K", 200, 400),
			"fuel":                 reading.number(&b.fuel, 0, 1),
		}, nil); err != nil {
			return err
		}
		if err := reading.all(block, "fuel bed", "recipe", "model", "east", "north", "width", "depth", "cell", "step", "ignition-radius", "ignition-temperature", "ambient-temperature", "fuel"); err != nil {
			return err
		}
		if int(b.width)%8 != 0 || int(b.depth)%8 != 0 || b.hot <= b.ambient {
			return reading.refuse(name, "bed dimensions are multiples of eight and ignition is hotter than ambient")
		}
		reading.set.beds = append(reading.set.beds, b)
		return nil
	})
}

// foundBeds refuses unsupported recipes and missing terrain.
func (reading reader) foundBeds() error {
	if len(reading.set.beds) > 1 {
		return reading.refuse(&yaml.Node{}, "at most one experimental fuel bed")
	}
	for _, b := range reading.set.beds {
		row, ok := reading.set.Recipe(b.recipe)
		spec, fire := row.Fire()
		if !ok || !fire || spec.Layout.Pattern != "" || spec.Smoke != nil || spec.Sparks != nil {
			return reading.refuse(&yaml.Node{}, "fuel bed needs a single-patch fire recipe without smoke or sparks")
		}
		if reading.set.terrain == nil {
			return reading.refuse(&yaml.Node{}, "fuel bed needs ground scenery")
		}
	}
	return nil
}
