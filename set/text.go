package set

import (
	"math"
	"slices"

	yaml "go.yaml.in/yaml/v3"
)

// A Text is a line of solid letters on the set, by its name: its words,
// the font they are made in, how big, deep and bevelled they are, what
// they are made of, where they stand, which way they face, and how fast
// they turn.
type Text struct {
	name, words, font  string
	size, depth, bevel float64
	smooth, tolerance  float64
	colour             [3]float64
	gloss              float64
	east, north, up    float64
	facing, spin       float64
}

// Name is the text's name, which cameras use to find it.
func (made Text) Name() string { return made.name }

// Words is the line the letters spell.
func (made Text) Words() string { return made.words }

// Font is the font the letters are made in: a font file, its path from
// the set file's own directory, or one of go's own fonts by name, as
// typography.Named reads it.
func (made Text) Font() string { return made.font }

// Measures is an em, the depth and the bevel, in metres; the angle in degrees
// below which the walls of a letter blend as one curve; and the tolerance a
// curve is flattened to, in the font's units, or zero for typography's own.
func (made Text) Measures() (size, depth, bevel, smooth,
	tolerance float64) {
	return made.size, made.depth, made.bevel, made.smooth, made.tolerance
}

// Colour is how much of the red, green and blue that lands on the letters they
// give back, each from zero to one, and how much of a mirror they are.
func (made Text) Colour() (red, green, blue, gloss float64) {
	return made.colour[0], made.colour[1], made.colour[2], made.gloss
}

// At is where the line's middle stands, an easting and a northing, and
// how many metres above the ground there its baseline is.
func (made Text) At() (east, north, up float64) {
	return made.east, made.north, made.up
}

// Facing is the bearing the letters' fronts face, in degrees clockwise from
// north, and how many seconds they take to turn once about up: zero is still,
// and less than zero turns the other way.
func (made Text) Facing() (bearing, spin float64) {
	return made.facing, made.spin
}

// Texts is every text on the set, in the order written.
func (made Set) Texts() []Text { return slices.Clone(made.texts) }

// texts reads each text, by its name.
func (reading reader) texts(_, value *yaml.Node) error {
	return reading.each(value, func(name, block *yaml.Node) error {
		made := Text{name: name.Value, facing: 180,
			colour: [3]float64{0.8, 0.8, 0.8}}
		var colour *[3]float64
		err := reading.keys(block, map[string]step{
			"words":     reading.text(&made.words),
			"font":      reading.text(&made.font),
			"size":      reading.number(&made.size, 1e-3, 1e4),
			"depth":     reading.number(&made.depth, 1e-4, 1e4),
			"bevel":     reading.number(&made.bevel, 0, 1e4),
			"smooth":    reading.number(&made.smooth, 0, 180),
			"tolerance": reading.number(&made.tolerance, 0, 1e4),
			"colour": reading.trio(&colour, "red", "green", "blue",
				[3][2]float64{{0, 1}, {0, 1}, {0, 1}}),
			"gloss":  reading.number(&made.gloss, 0, 1),
			"east":   reading.number(&made.east, 0, math.Inf(1)),
			"north":  reading.number(&made.north, 0, math.Inf(1)),
			"up":     reading.number(&made.up, -1e4, 1e4),
			"facing": reading.number(&made.facing, 0, 360),
			"spin":   reading.number(&made.spin, -1e6, 1e6),
		}, nil)
		if colour != nil {
			made.colour = *colour
		}
		if err == nil {
			err = reading.complete(block, made)
		}
		reading.set.texts = append(reading.set.texts, made)
		return err
	})
}

// complete refuses a text that leaves out what letters cannot be made
// without, or whose bevel is half its depth or more.
func (reading reader) complete(block *yaml.Node, made Text) error {
	switch {
	case made.words == "" || made.font == "" || made.size == 0 ||
		made.depth == 0:
		return reading.refuse(block, "text %s says its words, font, "+
			"size and depth, all of them", made.name)
	case 2*made.bevel >= made.depth:
		return reading.refuse(block, "text %s is bevelled %v, which "+
			"is half its depth of %v or more", made.name,
			made.bevel, made.depth)
	}
	return nil
}
