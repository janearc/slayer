package set

import yaml "gopkg.in/yaml.v3"

// WorldPosition is an explicit absolute scene coordinate in metres. East
// and north use the map frame; up is measured from the scene's origin.
type WorldPosition struct {
	east, north, up float64
}

// East is the absolute easting in metres.
func (made WorldPosition) East() float64 { return made.east }

// North is the absolute northing in metres.
func (made WorldPosition) North() float64 { return made.north }

// Up is the absolute elevation in metres.
func (made WorldPosition) Up() float64 { return made.up }

// worldPosition reads an explicitly named absolute coordinate.
func (reading reader) worldPosition(into **WorldPosition) step {
	return func(_, value *yaml.Node) error {
		made := &WorldPosition{}
		err := reading.keys(value, map[string]step{
			"east":  reading.quantity(&made.east, "m", -mostSpaceCoordinate, mostSpaceCoordinate),
			"north": reading.quantity(&made.north, "m", -mostSpaceCoordinate, mostSpaceCoordinate),
			"up":    reading.quantity(&made.up, "m", -mostSpaceCoordinate, mostSpaceCoordinate),
		}, nil)
		if err == nil {
			err = reading.all(value, "an absolute world position", "east", "north", "up")
		}
		if err == nil {
			*into = made
		}
		return err
	}
}
