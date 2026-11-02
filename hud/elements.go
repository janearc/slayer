package hud

import (
	"fmt"
	"image"
	"math"

	"github.com/janearc/slayer/maths"
)

// Align is which part of a line of words stands at its place: its start,
// its middle or its end.
type Align int

// the ways a line of words is aligned on its place
const (
	Left Align = iota
	Centre
	Right
)

// Text is words on the screen: what they say, where, aligned how, in
// the ink role, on the ground role. no ground is clear: the picture
// shows behind the words.
type Text struct {
	Words       Wording
	At          Point
	Align       Align
	Ink, Ground string
}

// lay sets the words' cells, one character a cell along the row.
func (made Text) lay(on *grid) error {
	return on.words(made.Words.At(on.at), on.cell(made.At), made.Align,
		made.Ink, made.Ground)
}

// words sets a line of words' cells, aligned on the cell at: its start,
// middle or end there. a character a cell, so a character a terminal
// draws two cells wide, as many of the far east's are, is set too close.
func (on *grid) words(words string, at image.Point, align Align, ink,
	ground string) error {
	cell, err := on.inked(ink, ground)
	if err != nil {
		return err
	}
	letters := []rune(words)
	start := at.X - len(letters)*int(align)/2
	for i, letter := range letters {
		cell.Rune = letter
		on.cover.set(image.Pt(start+i, at.Y), cell)
	}
	return nil
}

// inked is a cell in an ink role on a ground role, clear with no ground.
func (on *grid) inked(ink, ground string) (Cell, error) {
	fg, err := on.theme.Colour(ink)
	if err != nil || ground == "" {
		return Cell{Fg: fg, Clear: true}, err
	}
	bg, err := on.theme.Colour(ground)
	return Cell{Fg: fg, Bg: bg}, err
}

// Frame is a box's edge drawn in lines, in the ink role, the picture
// showing inside it and behind its lines.
type Frame struct {
	Box Box
	Ink string
}

// lay sets the cells of the box's edge: a line along its top and bottom
// rows and down its first and last columns, a corner at each corner.
func (made Frame) lay(on *grid) error {
	box := on.cells(made.Box)
	if box.Dx() < 2 || box.Dy() < 2 {
		return fmt.Errorf("%w: a frame takes two cells each way, "+
			"not %v", ErrHud, box.Size())
	}
	cell, err := on.inked(made.Ink, "")
	if err != nil {
		return err
	}
	last := box.Max.Sub(image.Pt(1, 1))
	for x := box.Min.X; x <= last.X; x++ {
		on.edge(cell, '─', image.Pt(x, box.Min.Y), image.Pt(x, last.Y))
	}
	for y := box.Min.Y; y <= last.Y; y++ {
		on.edge(cell, '│', image.Pt(box.Min.X, y), image.Pt(last.X, y))
	}
	on.edge(cell, '┌', box.Min)
	on.edge(cell, '┐', image.Pt(last.X, box.Min.Y))
	on.edge(cell, '└', image.Pt(box.Min.X, last.Y))
	on.edge(cell, '┘', last)
	return nil
}

// edge sets each place with the cell, its character a line.
func (on *grid) edge(cell Cell, line rune, at ...image.Point) {
	cell.Rune = line
	for _, each := range at {
		on.cover.set(each, cell)
	}
}

// Bar is a gauge from zero to one shown as a share of a box filled from the
// left, the full part in the fill role, the rest in the empty role, to an
// eighth of a cell.
type Bar struct {
	Box         Box
	Value       Gauge
	Fill, Empty string
}

// eighths are the characters that fill a cell from its left, by how many
// eighths of it: zero is a space, eight the whole block.
var eighths = [9]rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉', '█'}

// lay sets every cell of the box, each row filled alike.
func (made Bar) lay(on *grid) error {
	value := made.Value.At(on.at)
	if !(value >= 0 && value <= 1) {
		return fmt.Errorf("%w: a bar shows nought to one, not %v",
			ErrHud, value)
	}
	cell, err := on.inked(made.Fill, made.Empty)
	if err != nil {
		return err
	}
	box := on.cells(made.Box)
	full := int(math.Round(value * float64(box.Dx()*8)))
	for x := box.Min.X; x < box.Max.X; x++ {
		cell.Rune = eighths[max(0, min(8, full-(x-box.Min.X)*8))]
		for y := box.Min.Y; y < box.Max.Y; y++ {
			on.cover.set(image.Pt(x, y), cell)
		}
	}
	return nil
}

// Label is words pinned to a point of the world: set where the view's
// camera puts the point, aligned on it, and not set when the point is
// behind the camera, or, if Occluded, when the scene stands between.
type Label struct {
	Point       maths.Vec
	Words       Wording
	Align       Align
	Ink, Ground string
	Occluded    bool
}

// lay sets the label's words where its point falls, if it is seen. its
// roles are asked of the theme whether it is seen or not, so a role the
// theme has not is found the first frame and not when the label comes
// into view.
func (made Label) lay(on *grid) error {
	if _, err := on.inked(made.Ink, made.Ground); err != nil {
		return err
	}
	across, down, ahead, err := on.view.Project(made.Point)
	if err != nil || !ahead || made.Occluded && on.view.hidden(made.Point) {
		return err
	}
	at := on.cell(Point{Across: across, Down: down})
	return on.words(made.Words.At(on.at), at, made.Align, made.Ink,
		made.Ground)
}
