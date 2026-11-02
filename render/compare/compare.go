// Package compare checks a terminal rendering against the raster rendering of
// the same scene. smokecheck uses it to verify that terminal output represents
// the same image at a coarser resolution.
package compare

import (
	"image"
	"image/color"
	"math"

	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/render/colour"
)

// Report is what one comparison found.
type Report struct {
	Pixels   int     // terminal pixels in the picture
	Compared int     // how many were away from an edge, and so compared
	Worst    float64 // the largest oklab distance among those compared
	Mean     float64 // the mean oklab distance among those compared
	At       image.Point
}

// Honest is whether every compared pixel is within what an eye can see.
func (r Report) Honest() bool { return r.Worst <= ok.Eye }

// Cells compares text, the terminal's picture at one pixel per sextant, with
// real, the same scene rendered bw by bh real pixels to each of text's.
//
// Both are averaged the same way a screen mixes light, in linear light. A
// terminal cell is two sextants across and three down.
//
// A cell whose real pixels are not all within an eye of their own mean
// holds an edge, and is left out, since two colours a cell cannot draw an
// edge and was never meant to.
func Cells(real, text image.Image, bw, bh int) Report {
	tb := text.Bounds()
	r := Report{Pixels: tb.Dx() * tb.Dy()}
	var sum float64
	for cy := 0; cy+3 <= tb.Dy(); cy += 3 {
		for cx := 0; cx+2 <= tb.Dx(); cx += 2 {
			block := image.Rect(cx*bw, cy*bh, (cx+2)*bw, (cy+3)*bh)
			if spread(real, block) > ok.Eye {
				continue
			}
			for y := cy; y < cy+3; y++ {
				for x := cx; x < cx+2; x++ {
					d := apart(real, text, x, y, bw, bh)
					r.Compared++
					sum += d
					if d > r.Worst {
						r.Worst = d
						r.At = image.Pt(x, y)
					}
				}
			}
		}
	}
	if r.Compared > 0 {
		r.Mean = sum / float64(r.Compared)
	}
	return r
}

// apart is how far terminal pixel x, y is from the real pixels behind it.
func apart(real, text image.Image, x, y, bw, bh int) float64 {
	behind := image.Rect(x*bw, y*bh, (x+1)*bw, (y+1)*bh)
	at := text.Bounds().Min
	return ok.Distance(lab(mean(real, behind)),
		lab(text.At(at.X+x, at.Y+y)))
}

// mean is the colour of a rectangle of pixels, averaged as light.
func mean(img image.Image, rect image.Rectangle) color.RGBA {
	var rs, gs, bs float64
	n := 0
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			c := bytes(img.At(x, y))
			rs += colour.Decode(c.R)
			gs += colour.Decode(c.G)
			bs += colour.Decode(c.B)
			n++
		}
	}
	k := 1 / float64(n)
	return color.RGBA{colour.Encode(rs * k), colour.Encode(gs * k),
		colour.Encode(bs * k), 255}
}

// spread is how far the furthest pixel in a rectangle is from its mean.
func spread(img image.Image, rect image.Rectangle) float64 {
	m := lab(mean(img, rect))
	worst := 0.0
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			d := ok.Distance(m, lab(img.At(x, y)))
			worst = math.Max(worst, d)
		}
	}
	return worst
}

// lab is a pixel as libtheme sees it.
func lab(c color.Color) ok.OKLab {
	b := bytes(c)
	return ok.FromSwatch(srgb.RGB8(b.R, b.G, b.B).Swatch())
}

// bytes is any colour as eight bits a channel.
func bytes(c color.Color) color.RGBA {
	r, g, b, _ := c.RGBA()
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 255}
}
