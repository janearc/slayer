package terminal

import (
	"image/color"
	"math"
	"math/rand/v2"
	"testing"

	"github.com/janearc/slayer/render/colour"
)

// lightAverage is the colour of the pixels on one side of pattern p, set
// or unset, averaged as light with render/colour's Decode and Encode, and
// black for a side with none. written here, not taken from mean.
func lightAverage(px [6]color.RGBA, p int, set bool) color.RGBA {
	var r, g, b float64
	n := 0
	for i, c := range px {
		if (p>>i&1 == 1) == set {
			r += colour.Decode(c.R)
			g += colour.Decode(c.G)
			b += colour.Decode(c.B)
			n++
		}
	}
	if n == 0 {
		return color.RGBA{A: 255}
	}
	return color.RGBA{colour.Encode(r / float64(n)), colour.Encode(g / float64(n)), colour.Encode(b / float64(n)), 255}
}

// spread is dist's own sum as the doc gives it: the squared difference
// of each part on the srgb bytes, weighted 0.3, 0.59 and 0.11.
func spread(a, b color.RGBA) float64 {
	dr, dg, db := float64(a.R)-float64(b.R), float64(a.G)-float64(b.G), float64(a.B)-float64(b.B)
	return 0.3*dr*dr + 0.59*dg*dg + 0.11*db*db
}

// errorOf is how far six pixels are from the cell that pattern p draws
// of them, each side in its own average.
func errorOf(px [6]color.RGBA, p int) float64 {
	fg, bg := lightAverage(px, p, true), lightAverage(px, p, false)
	e := 0.0
	for i, c := range px {
		if p>>i&1 == 1 {
			e += spread(c, fg)
		} else {
			e += spread(c, bg)
		}
	}
	return e
}

// split's pick is the closest of all 64 patterns, tried by brute force
// here: every pattern's error, by dist, on its own two averages.
//
// a pattern and its opposite draw the same cell, so the pick is the one
// of the pair with the top left unset, held to the glyph and the two
// colours the brute force gives it. a trial whose best is within a
// millionth of another pair's is left out, and the run counts them.
func TestSplitPicksTheClosestOfAll64(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	shade := func() color.RGBA {
		return color.RGBA{uint8(rng.IntN(256)), uint8(rng.IntN(256)), uint8(rng.IntN(256)), 255}
	}
	picked := map[uint8]bool{}
	left := 0
	for trial := range 600 {
		var px [6]color.RGBA
		switch trial % 3 {
		case 0: // six colours of anything
			for i := range px {
				px[i] = shade()
			}
		case 1: // two colours, split any way, a little off
			a, b, cut := shade(), shade(), rng.IntN(64)
			for i := range px {
				c := a
				if cut>>i&1 == 1 {
					c = b
				}
				off := func(v uint8) uint8 { return uint8(min(255, max(0, int(v)+rng.IntN(9)-4))) }
				px[i] = color.RGBA{off(c.R), off(c.G), off(c.B), 255}
			}
		default: // three colours, which ties more often
			palette := [3]color.RGBA{shade(), shade(), shade()}
			for i := range px {
				px[i] = palette[rng.IntN(3)]
			}
		}
		var errs [64]float64
		best := 0
		for p := range errs {
			errs[p] = errorOf(px, p)
			if errs[p] < errs[best] {
				best = p
			}
		}
		pair := [2]int{best, best ^ 63}
		plain := true
		for p, e := range errs {
			if p != pair[0] && p != pair[1] && e <= errs[best]+1e-6 {
				plain = false
			}
		}
		if !plain {
			left++
			continue
		}
		want := pair[0]
		if want&1 == 1 {
			want = pair[1]
		}
		wantFg, wantBg := lightAverage(px, want, true), lightAverage(px, want, false)
		if want == 0 {
			wantFg = wantBg
		}
		var got Cell
		ends(t, "split", func() { got = split(px) })
		wantCell := Cell{Rune: sextantOf(want), Fg: wantFg, Bg: wantBg, Pattern: uint8(want)}
		if got != wantCell {
			t.Fatalf("trial %d, pixels %v: split gave %+v, want %+v (errors %.3f and %.3f)",
				trial, px, got, wantCell, errs[want], errs[got.Pattern])
		}
		picked[got.Pattern] = true
	}
	if left > 60 {
		t.Errorf("%d of 600 trials were left out as too close to call", left)
	}
	if len(picked) < 20 {
		t.Errorf("only %d different patterns were picked: the trials are too alike to hold split", len(picked))
	}
}

// two exact colours, black and white, in any of the 64 arrangements, come
// back as the arrangement itself or its opposite, whichever has the top
// left unset, at no error at all: nothing else fits exactly.
//
// worked by hand: pattern p lays white on its set bits. if p is odd the
// cell is the opposite, 63-p, with the colours swapped, so its fg is
// black. a cell of one colour, p of 0 or 63, is pattern 0 and a space.
func TestSplitOfBlackAndWhiteFindsEachArrangement(t *testing.T) {
	white, black := color.RGBA{255, 255, 255, 255}, color.RGBA{0, 0, 0, 255}
	for p := range 64 {
		var px [6]color.RGBA
		for i := range px {
			px[i] = black
			if p>>i&1 == 1 {
				px[i] = white
			}
		}
		var want Cell
		switch {
		case p == 0:
			want = Cell{Rune: ' ', Fg: black, Bg: black}
		case p == 63:
			want = Cell{Rune: ' ', Fg: white, Bg: white}
		case p&1 == 0:
			want = Cell{Rune: sextantOf(p), Fg: white, Bg: black, Pattern: uint8(p)}
		default:
			want = Cell{Rune: sextantOf(63 - p), Fg: black, Bg: white, Pattern: uint8(63 - p)}
		}
		if got := split(px); got != want {
			t.Errorf("pattern %06b: got %+v, want %+v", p, got, want)
		}
	}
}

// six pixels of one colour are a space on that colour, with nothing set
// and the same colour for the foreground: nothing is drawn in it, and
// the cell says so. worked by hand, for any colour at all.
func TestSixPixelsOfOneColourAreASpaceOfThatColour(t *testing.T) {
	for _, c := range []color.RGBA{
		{90, 90, 90, 255}, {200, 10, 90, 255}, {0, 0, 0, 255},
		{255, 255, 255, 255}, {1, 254, 128, 255},
	} {
		px := [6]color.RGBA{c, c, c, c, c, c}
		if got, want := split(px), (Cell{Rune: ' ', Fg: c, Bg: c, Pattern: 0}); got != want {
			t.Errorf("six of %v: got %+v, want %+v", c, got, want)
		}
	}
}

// mean is the average of the pixels on one side of a pattern, as light:
// the set bits of p for set, the unset ones for the other.
//
// worked by hand. Encode's own doc gives half the light as the byte 188,
// and light of 0 and 1 are the bytes 0 and 255, so black and white
// average to 188, and red, red, black, black to 188 red. one pixel is
// itself, and a side with none is black.
func TestMeanIsTheAverageOfASideAsLight(t *testing.T) {
	black, white, red := color.RGBA{0, 0, 0, 255}, color.RGBA{255, 255, 255, 255}, color.RGBA{255, 0, 0, 255}
	grey := color.RGBA{12, 34, 56, 255}
	for _, w := range []struct {
		what string
		px   [6]color.RGBA
		p    uint8
		set  bool
		want color.RGBA
	}{
		{"set: black and white", [6]color.RGBA{black, white, grey, grey, grey, grey}, 0b000011, true, color.RGBA{188, 188, 188, 255}},
		{"unset: red, red, black, black", [6]color.RGBA{grey, grey, red, red, black, black}, 0b000011, false, color.RGBA{188, 0, 0, 255}},
		{"one pixel is itself", [6]color.RGBA{black, black, black, black, black, grey}, 0b100000, true, grey},
		{"no pixels set is black", [6]color.RGBA{grey, grey, grey, grey, grey, grey}, 0, true, black},
		{"no pixels unset is black", [6]color.RGBA{grey, grey, grey, grey, grey, grey}, 63, false, black},
	} {
		if got := mean(w.px, w.p, w.set); got != w.want {
			t.Errorf("%s: got %v, want %v", w.what, got, w.want)
		}
	}
}

// dist is the weighted sum of the squared byte differences, 0.3 for red,
// 0.59 for green and 0.11 for blue, whichever way round the two come.
//
// worked by hand: the parts differ by 20, 30 and 40, so
// 0.3*400 + 0.59*900 + 0.11*1600 = 120 + 531 + 176 = 827. then each
// weight alone: a difference of 10 in one part is 0.3*100 = 30,
// 0.59*100 = 59 and 0.11*100 = 11.
func TestDistWeighsRedGreenAndBlue(t *testing.T) {
	for _, w := range []struct {
		a, b color.RGBA
		want float64
	}{
		{color.RGBA{30, 40, 50, 255}, color.RGBA{10, 10, 10, 255}, 827},
		{color.RGBA{10, 10, 10, 255}, color.RGBA{30, 40, 50, 255}, 827},
		{color.RGBA{10, 0, 0, 255}, color.RGBA{0, 0, 0, 255}, 30},
		{color.RGBA{0, 10, 0, 255}, color.RGBA{0, 0, 0, 255}, 59},
		{color.RGBA{0, 0, 10, 255}, color.RGBA{0, 0, 0, 255}, 11},
		{color.RGBA{0, 0, 10, 255}, color.RGBA{0, 0, 10, 255}, 0},
	} {
		if got := dist(w.a, w.b); math.Abs(got-w.want) > 1e-9 {
			t.Errorf("dist(%v, %v) = %v, want %v", w.a, w.b, got, w.want)
		}
	}
}
