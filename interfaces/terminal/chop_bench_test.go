package terminal

import (
	"image"
	"image/color"
	"math"
	"math/rand"
	"testing"

	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/render/colour"
)

func benchImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 160, 72))
	r := rand.New(rand.NewSource(1))
	for y := 0; y < 72; y++ {
		for x := 0; x < 160; x++ {
			img.Set(x, y, color.RGBA{uint8(r.Intn(256)), uint8(r.Intn(256)), uint8(r.Intn(256)), 255})
		}
	}
	return img
}

func BenchmarkChop(b *testing.B) {
	img := benchImage()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Chop(img)
	}
}

// oldEncode is colour.Encode as it was written, a power per call.
func oldEncode(v float64) uint8 {
	if math.IsNaN(v) {
		return 0
	}
	v = math.Max(0, math.Min(1, v))
	return uint8(math.Round(srgb.FromLinear(v) * 255))
}

// oldMean is mean with the old Encode.
func oldMean(px [6]color.RGBA, p uint8, set bool) color.RGBA {
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
	return color.RGBA{oldEncode(r * k), oldEncode(g * k), oldEncode(b * k), 255}
}

// oldSplit is split with the old mean.
func oldSplit(px [6]color.RGBA) Cell {
	best, bestErr := Cell{}, math.Inf(1)
	for p := uint8(0); p < 64; p += 2 {
		fg, bg := oldMean(px, p, true), oldMean(px, p, false)
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
		best.Fg = best.Bg
	}
	return best
}

// BenchmarkChopOld is Chop with the old Encode, for the same minute.
func BenchmarkChopOld(b *testing.B) {
	img := benchImage()
	bounds := img.Bounds()
	w, h := (bounds.Dx()+1)/2, (bounds.Dy()+2)/3
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out := &Cells{W: w, H: h, Cells: make([]Cell, w*h)}
		var px [6]color.RGBA
		for cy := 0; cy < h; cy++ {
			for cx := 0; cx < w; cx++ {
				for i := range px {
					x := min(bounds.Min.X+cx*2+i%2, bounds.Max.X-1)
					y := min(bounds.Min.Y+cy*3+i/2, bounds.Max.Y-1)
					px[i] = rgba(img.At(x, y))
				}
				out.Cells[cy*w+cx] = oldSplit(px)
			}
		}
	}
}

// the old split and the new agree on every cell of the picture.
func TestChopIsTheOldSplitCellForCell(t *testing.T) {
	img := benchImage()
	got := Chop(img)
	bounds := img.Bounds()
	for cy := 0; cy < got.H; cy++ {
		for cx := 0; cx < got.W; cx++ {
			var px [6]color.RGBA
			for i := range px {
				x := min(bounds.Min.X+cx*2+i%2, bounds.Max.X-1)
				y := min(bounds.Min.Y+cy*3+i/2, bounds.Max.Y-1)
				px[i] = rgba(img.At(x, y))
			}
			if got.Cells[cy*got.W+cx] != oldSplit(px) {
				t.Fatalf("cell %d,%d differs", cx, cy)
			}
		}
	}
}
