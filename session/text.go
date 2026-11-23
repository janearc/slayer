package session

import (
	"path/filepath"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// letters stands a text's letters on the ground where the set says, its
// baseline so far above it, facing its bearing and turning as it says,
// each letter a thing in the text's colour.
func (stage *staging) letters(each set.Text) error {
	east, north, up := each.At()
	place, err := stage.at(east, north)
	if err != nil {
		return err
	}
	place.Y += up
	stage.places[each.Name()] = place
	stage.note(each.Name(), "text", place)
	tally, err := textOf(each, place,
		filepath.Dir(stage.given.set)).Staged(stage.world)
	if err != nil {
		return err
	}
	stage.made.things += tally.Things
	stage.made.faces += tally.Faces
	stage.made.kept += tally.Kept
	return nil
}

// textOf is the manufactory's text of what the set says of one: its
// words in its font, found from dir, the set file's own directory, when
// the font is a file, standing at a place.
func textOf(each set.Text, place maths.Vec, dir string) manufacture.Text {
	size, depth, bevel, smooth, tolerance := each.Measures()
	red, green, blue, gloss := each.Colour()
	bearing, spin := each.Facing()
	return manufacture.Text{Name: each.Name(), Words: each.Words(),
		Font: each.Font(), Dir: dir, Size: size, Depth: depth,
		Bevel: bevel, Smooth: smooth * maths.Degree, Tolerance: tolerance,
		Colour: maths.RGB(red, green, blue), Gloss: gloss, At: place,
		Bearing: bearing, Spin: spin}
}
