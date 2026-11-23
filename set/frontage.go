package set

import (
	yaml "gopkg.in/yaml.v3"
)

// The two words a frontage says its side in: the items stand on the left
// of the channel's line or on its right, as one goes along it from its
// first point.
const (
	sideLeft  = "left"
	sideRight = "right"
)

// The most a setback or a gap may be, in metres. They bound a typing slip,
// not a street: a setback may be negative, for items that stand on the
// water side of a wall, as a boat does.
const (
	mostSetback = 1000
	mostGap     = 1000
)

// A frontage is the line of a channel that a stand's items stand along: the
// channel by name, the side of its line they stand on, how far from the
// channel's wall they stand, and the room between one run and the next.
type frontage struct {
	of           string
	right        bool
	setback, gap float64
}

// Frontage is the channel a stand stands along, whether its items stand on
// the right of its line, how far from the outer face of the channel's wall
// they stand, in metres, away from the water, and the room between runs,
// and whether the stand says one. Only a stand of the habit frontage does.
func (made Stand) Frontage() (of string, right bool, setback, gap float64,
	has bool) {
	if made.frontage == nil {
		return "", false, 0, 0, false
	}
	at := made.frontage
	return at.of, at.right, at.setback, at.gap, true
}

// frontage reads the line of a channel a stand stands along, every part
// said: the channel, the side, the setback from its wall and the gap
// between runs.
func (reading reader) frontage(into **frontage) step {
	return func(_, value *yaml.Node) error {
		var side string
		got := frontage{}
		err := reading.keys(value, map[string]step{
			"of":   reading.text(&got.of),
			"side": reading.word(&side, sideLeft, sideRight),
			"setback": reading.quantity(&got.setback, "m", -mostSetback,
				mostSetback),
			"gap": reading.quantity(&got.gap, "m", 0, mostGap),
		}, nil)
		if err == nil {
			err = reading.all(value, "a frontage", "of", "side", "setback",
				"gap")
		}
		if err != nil {
			return err
		}
		got.right = side == sideRight
		*into = &got
		return nil
	}
}
