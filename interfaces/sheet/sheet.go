// Package sheet assembles labelled images into a contact sheet, like a whole
// roll printed on one page for comparison.
//
// Tests and commands add renders and write one PNG. Images are enlarged in
// whole-pixel blocks using file.Block, preserving the approximate aspect ratios
// of terminal sextants and square raster pixels.
package sheet

import (
	"image"
	"image/color"
	"image/draw"
	"reflect"
	"strconv"

	"github.com/janearc/slayer/interfaces/file"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// the sheet's own look: a darkroom grey behind the frames, and the labels
// in a paler grey, so nothing on the sheet competes with the pictures
var (
	ground = color.RGBA{38, 38, 40, 255}
	ink    = color.RGBA{200, 200, 196, 255}
)

const (
	gap   = 12 // between frames, and around the edge
	label = 18 // the strip under each frame for its label
)

// Sheet is pictures waiting to be printed, in the order they were added.
type Sheet struct {
	columns int
	frames  []frame
}

type frame struct {
	label string
	img   *image.RGBA
}

// New is an empty sheet that prints columns frames to a row.
func New(columns int) *Sheet {
	return &Sheet{columns: max(columns, 1)}
}

// Add puts a picture on the sheet with its label. pixel is its pixels'
// width over their height, as file.PNG takes it: 1 for square, 0.75 for a
// sextant. a nil picture, or a nil pointer of any image type, panics
// with its label.
func (s *Sheet) Add(name string, img image.Image, pixel float64) {
	if v := reflect.ValueOf(img); !v.IsValid() ||
		v.Kind() == reflect.Pointer && v.IsNil() {
		panic("sheet: no picture to add as " + strconv.Quote(name))
	}
	bw, bh := file.Block(pixel)
	s.frames = append(s.frames, frame{name, file.Grow(img, bw, bh)})
}

// Len is how many pictures are on the sheet.
func (s *Sheet) Len() int { return len(s.frames) }

// Image is the sheet printed: every frame in a cell the size of the
// largest, left to right and then down, each with its label beneath. a
// Sheet made without New prints one frame to a row.
func (s *Sheet) Image() *image.RGBA {
	cw, ch := 0, 0
	for _, f := range s.frames {
		cw = max(cw, f.img.Bounds().Dx())
		ch = max(ch, f.img.Bounds().Dy())
	}
	cols := min(max(s.columns, 1), max(len(s.frames), 1))
	rows := (len(s.frames) + cols - 1) / cols
	w := gap + cols*(cw+gap)
	h := gap + rows*(ch+label+gap)
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(out, out.Bounds(), image.NewUniform(ground), image.Point{},
		draw.Src)
	for i, f := range s.frames {
		x := gap + (i%cols)*(cw+gap)
		y := gap + (i/cols)*(ch+label+gap)
		at := image.Rect(x, y, x+f.img.Bounds().Dx(),
			y+f.img.Bounds().Dy())
		draw.Draw(out, at, f.img, image.Point{}, draw.Src)
		write(out, f.label, x, y+ch+label-5)
	}
	return out
}

// PNG prints the sheet to path, as file.PNG writes: whole, beside path
// first, so a sheet that cannot be written leaves what was there.
func (s *Sheet) PNG(path string) error {
	return file.PNG(path, s.Image(), 1)
}

// write is a label in the sheet's ink, its baseline at y.
func write(img *image.RGBA, text string, x, y int) {
	d := font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(ink),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(text)
}
