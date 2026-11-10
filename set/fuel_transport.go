package set

import (
	"math"

	yaml "gopkg.in/yaml.v3"
)

// HeatTransport is the supplied east/south velocity in horizontal model m/s. It
// is independent of prescribed mist wind and does not imply atmosphere
// coupling.
func (bed FuelBed) HeatTransport() (east, south float64) {
	return bed.transportEast, bed.transportSouth
}

// heatTransport requires both signed velocity components and their units.
func (reading reader) heatTransport(bed *FuelBed) step {
	return func(_, node *yaml.Node) error {
		err := reading.keys(node, map[string]step{
			"east":  reading.quantity(&bed.transportEast, "m/s", -math.MaxFloat64, math.MaxFloat64),
			"south": reading.quantity(&bed.transportSouth, "m/s", -math.MaxFloat64, math.MaxFloat64),
		}, nil)
		if err != nil {
			return err
		}
		return reading.all(node, "fuel heat transport", "east", "south")
	}
}
