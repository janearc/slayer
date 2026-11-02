package typography

import "fmt"

// Placed is one letter set on a line: the letter, the glyph that draws
// it, and where the pen stands when it is drawn, in the font's units
// from the start of the line.
type Placed struct {
	Letter rune
	Glyph  Glyph
	Pen    float64
}

// Line is a line of letters laid out, and how far the pen went: its
// advance, the width the line takes on the page, in the font's units.
type Line struct {
	Letters []Placed
	Width   float64
}

// Layout sets a line's letters on its baseline: each where the one
// before it leaves the pen, moved by the kerning between the two.
func Layout(font Metrics, words string) (Line, error) {
	var line Line
	var before Glyph
	for i, letter := range []rune(words) {
		glyph, kern, advance, err := measured(font, letter, before,
			i == 0)
		if err != nil {
			return Line{}, err
		}
		line.Width += kern
		line.Letters = append(line.Letters,
			Placed{Letter: letter, Glyph: glyph, Pen: line.Width})
		line.Width += advance
		before = glyph
	}
	return line, nil
}

// measured is what the font says of a letter set after the glyph before it: the
// glyph that draws it, the kerning between the two, zero for a line's first
// letter, and how far it moves the pen.
func measured(font Metrics, letter rune, before Glyph,
	first bool) (glyph Glyph, kern, advance float64, err error) {
	if glyph, err = font.Index(letter); err != nil {
		return 0, 0, 0, err
	}
	if !first {
		if kern, err = font.Kern(before, glyph); err != nil {
			return 0, 0, 0, fmt.Errorf("%w: kerning %q: %w",
				ErrType, letter, err)
		}
	}
	if advance, err = font.Advance(glyph); err != nil {
		return 0, 0, 0, fmt.Errorf("%w: %q: %w", ErrType, letter, err)
	}
	return glyph, kern, advance, nil
}
