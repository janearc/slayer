// Package hud overlays text, frames, bars, and labels after scene rendering.
// Elements occupy screen space, so scene lighting does not affect them. Labels
// anchored in world space can optionally request occlusion checks.
//
// Layout uses terminal-style cells within a supplied rectangle. A drawer
// changes only the cells its elements cover; all other pixels remain the
// scene's. Terminals display text as glyphs over sextants, while raster output
// draws the same cell layout.
//
// The rectangle-and-coverage contract follows Carrie's drawer model so HUD
// coverage can be passed through without translation.
package hud

import (
	"image"
	"image/color"
	"slices"
	"time"
)

// Cell is one cell a hud covers: the character in it, the colour it is
// drawn in, and the colour behind it. a clear cell has no colour of its
// own behind it: the picture's shows through, as it does behind a
// frame's lines and a label's words.
type Cell struct {
	Rune   rune
	Fg, Bg color.RGBA
	Clear  bool
}

// Cover is a hud laid out in a rectangle of cells: the rectangle, and
// the cells in it the hud covers, by their place. a cell it does not
// hold is the picture's.
type Cover struct {
	Rect  image.Rectangle
	cells map[image.Point]Cell
}

// At is the cell at column x, row y, and whether the hud covers it.
func (made Cover) At(x, y int) (Cell, bool) {
	cell, ok := made.cells[image.Pt(x, y)]
	return cell, ok
}

// Mask is every cell the hud covers, row by row from the top left.
func (made Cover) Mask() []image.Point {
	mask := make([]image.Point, 0, len(made.cells))
	for at := range made.cells {
		mask = append(mask, at)
	}
	slices.SortFunc(mask, func(a, b image.Point) int {
		if a.Y != b.Y {
			return a.Y - b.Y
		}
		return a.X - b.X
	})
	return mask
}

// set covers the cell at, inside the rectangle, with a cell; one outside
// it is left out, so an element past the edge is cut there.
func (made Cover) set(at image.Point, cell Cell) {
	if at.In(made.Rect) {
		made.cells[at] = cell
	}
}

// Element is one thing a hud draws. each is laid out in the cells of
// the rectangle it is given, at the frame's time.
type Element interface {
	lay(on *grid) error
}

// Hud is the elements drawn over a picture, each over the ones before.
type Hud struct {
	Elements []Element
}

// grid is a hud being laid out: the cover it fills, the frame's time,
// the theme it is drawn in, and the view the picture under it was seen
// through.
type grid struct {
	cover Cover
	at    time.Duration
	theme Theme
	view  View
}

// Cover is the hud laid out in a rectangle of cells at a frame's time,
// in a theme, over a picture seen through a view.
func (made Hud) Cover(rect image.Rectangle, at time.Duration, theme Theme,
	view View) (Cover, error) {
	on := &grid{cover: Cover{Rect: rect, cells: map[image.Point]Cell{}},
		at: at, theme: theme, view: view}
	for _, each := range made.Elements {
		if err := each.lay(on); err != nil {
			return Cover{}, err
		}
	}
	return on.cover, nil
}

// Point is a place on the screen, as shares of its width from the left and its
// height from the top: zero, zero is the top left corner, one, one the bottom
// right.
type Point struct{ Across, Down float64 }

// Box is a part of the screen, its top left and bottom right as shares.
type Box struct{ Left, Top, Right, Bottom float64 }

// cell is the cell a point is in.
func (on *grid) cell(at Point) image.Point {
	r := on.cover.Rect
	return image.Pt(r.Min.X+int(at.Across*float64(r.Dx())),
		r.Min.Y+int(at.Down*float64(r.Dy())))
}

// cells is the cells a box takes, each edge at its nearest cell edge,
// and never less than one cell either way.
func (on *grid) cells(box Box) image.Rectangle {
	r := on.cover.Rect
	edge := func(share float64, from, size int) int {
		return from + int(share*float64(size)+0.5)
	}
	left, top := edge(box.Left, r.Min.X, r.Dx()), edge(box.Top, r.Min.Y,
		r.Dy())
	right, bottom := edge(box.Right, r.Min.X, r.Dx()),
		edge(box.Bottom, r.Min.Y, r.Dy())
	return image.Rect(left, top, max(right, left+1), max(bottom, top+1))
}
