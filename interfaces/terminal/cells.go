package terminal

// how a terminal shows a picture. a cell is two pixels across and three
// down, the six squares of a sextant, and it can show only two colours: one
// for the squares its glyph fills and one for the rest. for each cell, Chop
// tries every split of its six pixels into two and keeps the closest.

import (
	"image"
	"image/color"
	"math"

	"github.com/janearc/slayer/render/colour"
)

// Cell is one terminal cell: a glyph, the colour of the squares it fills,
// and the colour of the rest.
type Cell struct {
	Rune    rune
	Fg, Bg  color.RGBA
	Pattern uint8 // which of the six squares are Fg; see Glyph
}

// Cells is a grid of cells, row by row from the top left.
type Cells struct {
	W, H  int
	Cells []Cell
}

// At is the cell at column x, row y.
func (c *Cells) At(x, y int) Cell { return c.Cells[y*c.W+x] }

// Picture is what the terminal will show, as a picture: each cell as its
// six squares, in the two colours it actually got, two pixels across and
// three down a cell. It is Chop's inverse as far as a cell can say.
func Picture(c *Cells) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, c.W*2, c.H*3))
	for cy := range c.H {
		for cx := range c.W {
			cell := c.At(cx, cy)
			for i := range 6 {
				col := cell.Bg
				if cell.Pattern&(1<<i) != 0 {
					col = cell.Fg
				}
				out.SetRGBA(cx*2+i%2, cy*3+i/2, col)
			}
		}
	}
	return out
}

// Glyph is the character that fills the squares set in pattern. Bit 0 is
// the top left, bit 1 the top right, then the middle pair, then the bottom.
//
// Unicode's sextants run from U+1FB00 in pattern order, skipping the four
// that already had characters: empty, full, and the left and right halves.
func Glyph(pattern uint8) rune {
	switch p := rune(pattern & 63); {
	case p == 0:
		return ' '
	case p == 63:
		return '█'
	case p == 21: // the left column
		return '▌'
	case p == 42: // the right column
		return '▐'
	case p < 21:
		return 0x1FB00 + p - 1
	case p < 42:
		return 0x1FB00 + p - 2
	default:
		return 0x1FB00 + p - 3
	}
}

// Chop cuts img into cells. A picture that does not divide evenly is padded
// by repeating its last column and row.
func Chop(img image.Image) *Cells {
	b := img.Bounds()
	w, h := (b.Dx()+1)/2, (b.Dy()+2)/3
	out := &Cells{W: w, H: h, Cells: make([]Cell, w*h)}
	var px [6]color.RGBA
	for cy := 0; cy < h; cy++ {
		for cx := 0; cx < w; cx++ {
			for i := range px {
				x := min(b.Min.X+cx*2+i%2, b.Max.X-1)
				y := min(b.Min.Y+cy*3+i/2, b.Max.Y-1)
				px[i] = rgba(img.At(x, y))
			}
			out.Cells[cy*w+cx] = split(px)
		}
	}
	return out
}

// rgba is any colour as eight bits a part, its alpha dropped: a colour
// that is not opaque comes out as its premultiplied value, darker.
func rgba(c color.Color) color.RGBA {
	r, g, b, _ := c.RGBA()
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 255}
}

// split finds the two-colour cell closest to six pixels, by dist. Each
// group's colour is averaged in linear light, not in srgb bytes, which
// would make every edge too dark; that mean is not dist's own closest.
func split(px [6]color.RGBA) Cell {
	best, bestErr := Cell{}, math.Inf(1)
	// patterns and their opposites draw the same cell with the colours
	// swapped, so trying the 32 with the top left unset covers all 64
	for p := uint8(0); p < 64; p += 2 {
		fg, bg := mean(px, p, true), mean(px, p, false)
		e := 0.0
		for i, c := range px {
			if p&(1<<i) != 0 {
				e += dist(c, fg)
			} else {
				e += dist(c, bg)
			}
		}
		if e < bestErr {
			best = Cell{Rune: Glyph(p), Fg: fg, Bg: bg, Pattern: p}
			bestErr = e
		}
	}
	if best.Pattern == 0 {
		// one colour: nothing to draw in the foreground
		best.Fg = best.Bg
	}
	return best
}

// mean is the average colour of the pixels on one side of pattern p, the
// set side or the unset side, averaged as light.
func mean(px [6]color.RGBA, p uint8, set bool) color.RGBA {
	var r, g, b float64
	n := 0
	for i, c := range px {
		if (p&(1<<i) != 0) == set {
			r += colour.Decode(c.R)
			g += colour.Decode(c.G)
			b += colour.Decode(c.B)
			n++
		}
	}
	if n == 0 {
		return color.RGBA{A: 255}
	}
	k := 1 / float64(n)
	return color.RGBA{colour.Encode(r * k), colour.Encode(g * k),
		colour.Encode(b * k), 255}
}

// dist is how different two colours look, measured on the srgb bytes, which
// are much closer to how different they look than linear light is. its
// weights are near rec. 601's luma, chosen, not measured.
func dist(a, b color.RGBA) float64 {
	dr := float64(a.R) - float64(b.R)
	dg := float64(a.G) - float64(b.G)
	db := float64(a.B) - float64(b.B)
	return 0.3*dr*dr + 0.59*dg*dg + 0.11*db*db
}
