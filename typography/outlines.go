package typography

import (
	"fmt"

	"github.com/janearc/slayer/maths"
)

// Outline is one letter of a line as the rings of its outline, flat: each
// ring a closed run of straight steps, in scene units, with y up and z
// nought, relative to the letter's own zero. The first ring of a piece of
// the letter turns counterclockwise, and the holes inside it clockwise, so
// that the letter is always on a ring's left. It is what a thing that
// follows a letter's edge, a tube bent to its shape, is built from.
type Outline struct {
	Letter rune
	At     maths.Vec
	Rings  [][]maths.Vec
}

// Outlines is the outlines of the letters of a line, laid out as Make lays
// them: each letter's place in the line along x, and the line's advance
// centred on x zero. Depth and Bevel are not read, a flat outline having
// neither. A letter with no outline, a space, has no rings. A glyph whose
// contours cross is refused, as Make refuses it.
func Outlines(face *Font, text Text) ([]Outline, error) {
	asked := text
	asked.Depth, asked.Bevel = 1, 0
	if err := asked.check(); err != nil {
		return nil, err
	}
	line, err := Layout(face, text.Words)
	if err != nil {
		return nil, err
	}
	tolerance := text.Tolerance
	if tolerance == 0 {
		tolerance = face.Units() / 1000
	}
	scale := text.Size / face.Units()
	made, kept := []Outline{}, map[Glyph][][]maths.Vec{}
	for _, each := range line.Letters {
		rings, found := kept[each.Glyph]
		if !found {
			if rings, err = face.ringsOf(each, tolerance, scale); err != nil {
				return nil, err
			}
			kept[each.Glyph] = rings
		}
		made = append(made, Outline{Letter: each.Letter, Rings: rings,
			At: maths.V((each.Pen-line.Width/2)*scale, 0, 0)})
	}
	return made, nil
}

// ringsOf is the rings of a placed letter's glyph, scaled to scene units.
func (face *Font) ringsOf(each placed, tolerance,
	scale float64) ([][]maths.Vec, error) {
	solids, err := face.solids(each.Glyph, tolerance)
	if err != nil {
		return nil, fmt.Errorf("%w: %q: %w", ErrType, each.Letter, err)
	}
	var rings [][]maths.Vec
	for _, one := range solids {
		for _, round := range one.rings() {
			scaled := make([]maths.Vec, len(round))
			for index, at := range round {
				scaled[index] = maths.V(at.x*scale, at.y*scale, 0)
			}
			rings = append(rings, scaled)
		}
	}
	return rings, nil
}
