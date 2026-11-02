package hud

import (
	"fmt"
	"image/color"

	"github.com/janearc/slayer/interfaces/terminal"
)

// Over is a terminal picture with the hud's cells over it: each cell
// the hud covers is its character in its colours, and every other cell
// is the picture's, untouched. a clear cell keeps the picture's colour
// behind its character. the picture's top left is the cover's.
//
// a picture not the cover's size is an error.
func (made Cover) Over(picture *terminal.Cells) (*terminal.Cells, error) {
	if picture.W != made.Rect.Dx() || picture.H != made.Rect.Dy() {
		return nil, fmt.Errorf("%w: a hud of %v over a picture of "+
			"%dx%d", ErrHud, made.Rect.Size(), picture.W, picture.H)
	}
	out := &terminal.Cells{W: picture.W, H: picture.H,
		Cells: append([]terminal.Cell(nil), picture.Cells...)}
	for at, cell := range made.cells {
		local := at.Sub(made.Rect.Min)
		under := &out.Cells[local.Y*out.W+local.X]
		bg := cell.Bg
		if cell.Clear {
			bg = behind(*under)
		}
		*under = terminal.Cell{Rune: cell.Rune, Fg: cell.Fg, Bg: bg}
	}
	return out, nil
}

// behind is the colour a clear hud cell keeps from the picture cell it
// covers: the colour of the larger part of the cell's six squares, the
// one the eye takes the cell to be.
func behind(cell terminal.Cell) color.RGBA {
	set := 0
	for bit := range 6 {
		if cell.Pattern&(1<<bit) != 0 {
			set++
		}
	}
	if set > 3 {
		return cell.Fg
	}
	return cell.Bg
}
