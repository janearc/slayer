package shortcut

import (
	"image"

	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/libtheme-css/spaces/srgb"

	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
)

// Error is how far a picture with a shortcut is from the exact one:
// the worst and the mean oklab distance over the pixels, both frames
// developed the same way, as smokecheck judges a terminal against the real.
func Error(exact, taken *frame.Frame, tone colour.Tone) (worst, mean float64) {
	a, b := tone.Develop(exact), tone.Develop(taken)
	return Apart(a, b)
}

// Apart is the worst and mean oklab distance between two pictures of
// one size. Two pixels a single eight-bit step apart in every channel
// count as the same: that is the encoding's own step, and in the dark
// the eye's cube root would otherwise make a sin of black against one.
func Apart(a, b *image.RGBA) (worst, mean float64) {
	bounds := a.Bounds()
	n := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			p, q := a.RGBAAt(x, y), b.RGBAAt(x, y)
			if step(p.R, q.R) && step(p.G, q.G) && step(p.B, q.B) {
				n++
				continue
			}
			d := ok.Distance(lab(p.R, p.G, p.B), lab(q.R, q.G, q.B))
			mean += d
			if d > worst {
				worst = d
			}
			n++
		}
	}
	if n > 0 {
		mean /= float64(n)
	}
	return worst, mean
}

// lab is a pixel's eight bits a channel as oklab, through libtheme.
func lab(r, g, b uint8) ok.OKLab {
	return ok.FromSwatch(srgb.RGB8(r, g, b).Swatch())
}

// Honest is whether a shortcut's error is within its budget.
func Honest(name string, worst float64) bool {
	budget, ok := Budgets[name]
	return ok && worst <= budget
}

// step is whether two channel values are within one step of each other.
func step(a, b uint8) bool {
	if a > b {
		a, b = b, a
	}
	return b-a <= 1
}
