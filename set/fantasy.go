package set

import (
	"math"
	"strconv"

	yaml "gopkg.in/yaml.v3"
)

// the fantasy keys of the set, each a departure from what the sky and
// the ground measure or derive; the names are the hillside flags' own
const (
	fantasyLake      = "lake"
	fantasyLakeGlow  = "lakeglow"
	fantasyMoon      = "moon"
	fantasyMoonGlow  = "moonglow"
	fantasyGiants    = "giants"
	fantasyClose     = "close"
	fantasyCloseGlow = "closeglow"
)

// noElevation is the lake's zero, which is no lake: true by definition,
// since still water to no elevation floods nothing.
const noElevation = 0.0

// Departure is one thing a scene states that the physics does not say:
// a key of the fantasy block, and its value, a number or a switch.
type Departure struct {
	key    string
	number float64
	on     bool
	isFlag bool
}

// Key is the departure's name in the fantasy block.
func (made Departure) Key() string { return made.key }

// Number is the departure's value, when it is a number.
func (made Departure) Number() float64 { return made.number }

// On is the departure's value, when it is a switch.
func (made Departure) On() bool { return made.on }

// Value is the departure's value as written in the set.
func (made Departure) Value() string {
	if made.isFlag {
		return strconv.FormatBool(made.on)
	}
	return strconv.FormatFloat(made.number, 'g', -1, 64)
}

// Fantasy is every departure the set's fantasy block states, in the
// order written; none when it has no block.
func (made Set) Fantasy() []Departure {
	return append([]Departure(nil), made.fantasy...)
}

// fantasy reads the block of departures. Each key takes the value its
// hillside flag takes: lake an elevation in metres from zero, the
// scales and the glow a number from zero or above zero, the masks a
// switch. Any other key is refused, with the ones that are known.
func (reading reader) fantasy(_, value *yaml.Node) error {
	scale := reading.departure(math.SmallestNonzeroFloat64)
	return reading.keys(value, map[string]step{
		fantasyLake:      reading.departure(noElevation),
		fantasyLakeGlow:  reading.switched(),
		fantasyMoon:      scale,
		fantasyMoonGlow:  reading.departure(noElevation),
		fantasyGiants:    scale,
		fantasyClose:     scale,
		fantasyCloseGlow: reading.switched(),
	}, nil)
}

// departureStep reads a finite number, from least up, as a departure.
type departureStep struct {
	reading reader
	least   float64
}

// read reads the number and keeps it as a departure of the key's name.
func (from departureStep) read(name, value *yaml.Node) error {
	var amount float64
	err := from.reading.number(&amount, from.least, math.MaxFloat64)(name, value)
	if err == nil {
		from.reading.set.fantasy = append(from.reading.set.fantasy,
			Departure{key: name.Value, number: amount})
	}
	return err
}

// departure is a step that reads a finite number, from least up, as a
// departure.
func (reading reader) departure(least float64) step {
	return departureStep{reading: reading, least: least}.read
}

// switchStep reads true or false as a departure.
type switchStep struct {
	reading reader
}

// read reads the switch and keeps it as a departure of the key's name.
func (from switchStep) read(name, value *yaml.Node) error {
	var on bool
	err := from.reading.yes(&on)(name, value)
	if err == nil {
		from.reading.set.fantasy = append(from.reading.set.fantasy,
			Departure{key: name.Value, on: on, isFlag: true})
	}
	return err
}

// switched is a step that reads true or false as a departure.
func (reading reader) switched() step {
	return switchStep{reading: reading}.read
}
