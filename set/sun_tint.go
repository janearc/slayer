package set

import yaml "gopkg.in/yaml.v3"

// The RGB channel order and count are defined by this file format.
const (
	sunRed = iota
	sunGreen
	sunBlue
	sunChannels
)

// sunUnit is defined as full transmission of one linear RGB channel.
const sunUnit = 1

// Tint supplies an authored linear RGB filter on the direct daylight lamp.
// Unspecified tint preserves white daylight; this is not atmospheric transfer.
func (made Sun) Tint() (red, green, blue float64) {
	if made.tint == nil {
		return sunUnit, sunUnit, sunUnit
	}
	return made.tint[sunRed], made.tint[sunGreen], made.tint[sunBlue]
}

type sunTintReader struct {
	reading reader
	sun     *Sun
}

// read applies the same finite channel bounds through both scene readers.
func (input sunTintReader) read(name, block *yaml.Node) error {
	var tint [sunChannels]float64
	if err := input.reading.three(&tint, sceneScalarZero,
		sunUnit)(name, block); err != nil {
		return err
	}
	input.sun.tint = &tint
	return nil
}
