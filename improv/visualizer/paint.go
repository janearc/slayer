package main

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// The palette: a vaporwave shop. A dark violet ground, neon pink, cyan
// and purple marks, and a pale ink held well clear of the ground so the
// numbers read first.
var (
	groundTop    = color.RGBA{0x1c, 0x0a, 0x36, 0xff}
	groundBottom = color.RGBA{0x0b, 0x02, 0x1c, 0xff}
	laneGround   = color.RGBA{0x24, 0x0e, 0x44, 0xff}
	ink          = color.RGBA{0xfb, 0xee, 0xff, 0xff}
	faint        = color.RGBA{0x9a, 0x7c, 0xc4, 0xff}
	pink         = color.RGBA{0xff, 0x2a, 0x9d, 0xff}
	cyan         = color.RGBA{0x00, 0xf0, 0xff, 0xff}
	purple       = color.RGBA{0xb9, 0x67, 0xff, 0xff}
	sunny        = color.RGBA{0xff, 0xfb, 0x96, 0xff}
	mint         = color.RGBA{0x05, 0xff, 0xa1, 0xff}
	unlit        = color.RGBA{0x34, 0x18, 0x5c, 0xff}
	// laneColours are given to the lanes in the order they are named.
	laneColours = []color.RGBA{pink, cyan, purple, sunny, mint}
)

// The text roles, each a face at a size in reference pixels (a frame
// 1080 high); the writer scales them to the frame.
const (
	textReadout = iota
	textName
	textRow
	textNumber
)

// textSizes are the reference sizes of the text roles, in pixels.
var textSizes = map[int]float64{
	textReadout: 200, textName: 34, textRow: 22, textNumber: 30,
}

// textMono is which roles use the mono face, so changing figures keep
// their places.
var textMono = map[int]bool{textReadout: true, textNumber: true}

// writer draws text in the film's faces, made once for the frame size.
type writer struct{ faces map[int]font.Face }

// newWriter parses the two Go faces and makes each role's size.
func newWriter(unit float64) (*writer, error) {
	mono, err := opentype.Parse(gomonobold.TTF)
	if err != nil {
		return nil, err
	}
	bold, err := opentype.Parse(gobold.TTF)
	if err != nil {
		return nil, err
	}
	w := &writer{faces: map[int]font.Face{}}
	for role, size := range textSizes {
		parsed := bold
		if textMono[role] {
			parsed = mono
		}
		w.faces[role], err = opentype.NewFace(parsed, &opentype.FaceOptions{
			Size: size * unit, DPI: 72, Hinting: font.HintingFull})
		if err != nil {
			return nil, err
		}
	}
	return w, nil
}

// left draws a line starting at x with its baseline at y.
func (w *writer) left(img *image.RGBA, role, x, y int, c color.RGBA,
	s string) {
	d := font.Drawer{Dst: img, Src: image.NewUniform(c), Face: w.faces[role],
		Dot: fixed.P(x, y)}
	d.DrawString(s)
}

// centre draws a line centred on x with its baseline at y.
func (w *writer) centre(img *image.RGBA, role, x, y int, c color.RGBA,
	s string) {
	wide := font.MeasureString(w.faces[role], s).Round()
	w.left(img, role, x-wide/2, y, c, s)
}

// right draws a line ending at x with its baseline at y.
func (w *writer) right(img *image.RGBA, role, x, y int, c color.RGBA,
	s string) {
	wide := font.MeasureString(w.faces[role], s).Round()
	w.left(img, role, x-wide, y, c, s)
}

// fill paints a rectangle one colour.
func fill(img *image.RGBA, r image.Rectangle, c color.RGBA) {
	blend(img, r, c, 1)
}

// blend lays a colour over a rectangle at a strength from 0 to 1.
func blend(img *image.RGBA, r image.Rectangle, c color.RGBA,
	strength float64) {
	r = r.Intersect(img.Bounds())
	tone := [3]float64{float64(c.R), float64(c.G), float64(c.B)}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		row := img.Pix[y*img.Stride : (y+1)*img.Stride]
		for x := r.Min.X; x < r.Max.X; x++ {
			for k := range 3 {
				was := float64(row[4*x+k])
				row[4*x+k] = uint8(was + (tone[k]-was)*strength)
			}
			row[4*x+3] = 0xff
		}
	}
}

// glowRings is how many widening halos a glow lays, and glowStrength
// how much each one tints; where they overlap near the mark, the glow
// is brightest.
const (
	glowRings    = 4
	glowStrength = 0.12
)

// glow paints a mark with a soft halo of its colour, reach pixels a
// ring, kept inside clip.
func glow(img *image.RGBA, r, clip image.Rectangle, c color.RGBA,
	reach int) {
	for ring := glowRings; ring >= 1; ring-- {
		blend(img, r.Inset(-ring*reach).Intersect(clip), c, glowStrength)
	}
	fill(img, r.Intersect(clip), c)
}

// mix is a colour moved from off toward on by a level from 0 to 1.
func mix(off, on color.RGBA, level float64) color.RGBA {
	level = math.Max(0, math.Min(1, level))
	step := func(a, b uint8) uint8 {
		return uint8(float64(a) + (float64(b)-float64(a))*level)
	}
	return color.RGBA{step(off.R, on.R), step(off.G, on.G),
		step(off.B, on.B), 0xff}
}

// line draws a straight line two pixels thick at a strength.
func line(img *image.RGBA, x0, y0, x1, y1 float64, c color.RGBA,
	strength float64) {
	steps := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))) + 1
	for i := 0; i <= steps; i++ {
		f := float64(i) / float64(steps)
		x := int(x0 + (x1-x0)*f)
		y := int(y0 + (y1-y0)*f)
		blend(img, image.Rect(x, y, x+2, y+2), c, strength)
	}
}
