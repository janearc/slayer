package hud

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Picture draws the hud on a picture: laid out in cells, rows of them
// down the picture and each cell half as wide as it is tall, as a
// terminal's are, and each cell the hud covers drawn there. what it
// does not cover is left as it was.
//
// the cells are whole pixels, so a picture whose size does not divide
// has a strip at its right and its foot that no cell reaches.
func (made Hud) Picture(img *image.RGBA, rows int, at time.Duration,
	theme Theme, view View) error {
	if rows < 1 || img.Bounds().Dy()/rows < 2 {
		return fmt.Errorf("%w: %d rows on a picture %d high", ErrHud,
			rows, img.Bounds().Dy())
	}
	high := img.Bounds().Dy() / rows
	wide := high / 2
	cols := img.Bounds().Dx() / wide
	cover, err := made.Cover(image.Rect(0, 0, cols, rows), at, theme, view)
	if err != nil {
		return err
	}
	face, err := mono(high)
	if err != nil {
		return err
	}
	for _, place := range cover.Mask() {
		cell, _ := cover.At(place.X, place.Y)
		corner := image.Pt(place.X*wide, place.Y*high)
		min := img.Bounds().Min.Add(corner)
		paint(img, image.Rectangle{min, min.Add(image.Pt(wide, high))},
			cell, face)
	}
	return nil
}

// mono is go's mono font at a size that fills a cell so high.
func mono(high int) (font.Face, error) {
	parsed, err := opentype.Parse(gomono.TTF)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrHud, err)
	}
	return opentype.NewFace(parsed, &opentype.FaceOptions{
		Size: float64(high) * 0.8, DPI: 72, Hinting: font.HintingFull})
}

// paint draws one cell in its rectangle: its ground unless it is clear,
// then its character. a block that fills eighths of a cell, a bar's, is
// filled exactly; a frame's lines are drawn through the cell's middle;
// every other character is the font's.
func paint(img *image.RGBA, at image.Rectangle, cell Cell, face font.Face) {
	if !cell.Clear {
		fill(img, at, cell.Bg)
	}
	if share := eighthsOf(cell.Rune); share >= 0 {
		right := at.Min.X + (share*at.Dx()+4)/8
		filled := image.Rect(at.Min.X, at.Min.Y, right, at.Max.Y)
		fill(img, filled, cell.Fg)
		return
	}
	if lines(img, at, cell) {
		return
	}
	metrics := face.Metrics()
	base := at.Min.Y + (at.Dy()+(metrics.Ascent-metrics.Descent).Round())/2
	writer := font.Drawer{Dst: img, Src: image.NewUniform(cell.Fg),
		Face: face, Dot: fixed.P(at.Min.X, base)}
	writer.DrawString(string(cell.Rune))
}

// eighthsOf is how many eighths of a cell a block character fills from
// its left, or minus one for any other character. a space fills none.
func eighthsOf(letter rune) int {
	for share, block := range eighths {
		if block == letter {
			return share
		}
	}
	return -1
}

// strokes is, for each of a frame's characters, which way its lines go
// from the cell's middle: left, right, up, down.
var strokes = map[rune][4]bool{
	'─': {true, true, false, false}, '│': {false, false, true, true},
	'┌': {false, true, false, true}, '┐': {true, false, false, true},
	'└': {false, true, true, false}, '┘': {true, false, true, false},
}

// lines draws a frame's character as lines from the cell's middle, a
// twelfth of the cell high thick and at least a pixel, and is whether
// the character was one.
func lines(img *image.RGBA, at image.Rectangle, cell Cell) bool {
	ways, ok := strokes[cell.Rune]
	if !ok {
		return false
	}
	thick := max(1, at.Dy()/12)
	mid := at.Min.Add(at.Size().Div(2)).Sub(image.Pt(thick/2, thick/2))
	across := image.Rect(mid.X, mid.Y, mid.X+thick, mid.Y+thick)
	left, right := across, across
	left.Min.X, right.Max.X = at.Min.X, at.Max.X
	up, down := across, across
	up.Min.Y, down.Max.Y = at.Min.Y, at.Max.Y
	for way, to := range [4]image.Rectangle{left, right, up, down} {
		if ways[way] {
			fill(img, to, cell.Fg)
		}
	}
	return true
}

// fill paints a rectangle of the picture in one colour.
func fill(img *image.RGBA, at image.Rectangle, colour color.RGBA) {
	draw.Draw(img, at, image.NewUniform(colour), image.Point{}, draw.Src)
}
