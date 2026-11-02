package shortcut

import (
	"image"
	"image/color"
	"testing"

	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
)

// the distances below are oklab distances from black, which is the origin of
// oklab, so each is the length of the pixel's own oklab vector. a byte under
// 0.04045 of 255 is linear at byte / 255 / 12.92.
//
// a grey of byte 2 is 2 / 255 / 12.92 = 6.0705e-4 linear. oklab takes the cube
// root of the light in each cone and mixes the three with rows that sum to one
// for lightness and to zero for the opponent axes, so a grey is cbrt(6.0705e-4)
// = 0.08467 from black.
//
// a grey of byte 4 is twice the light, so 2^(1/3) times as far, 0.08467 *
// 1.25992 = 0.10668.
//
// the pure channels at byte 2 go through the cone mix and the opponent rows,
// which is three matrices of arithmetic. red was worked by hand to 0.0575.
//
// the figures here, red 0.05747, green 0.07749 and blue 0.04656, are the same
// sum done once with the published oklab matrices, apart from libtheme and from
// the code under test.
const (
	grey2  = 0.08467
	grey4  = 0.10668
	red2   = 0.05747
	green2 = 0.07749
	blue2  = 0.04656
)

// pic is a picture of one row, a pixel to each colour.
func pic(cs ...color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, len(cs), 1))
	for i, c := range cs {
		img.SetRGBA(i, 0, c)
	}
	return img
}

func grey(v uint8) color.RGBA { return color.RGBA{v, v, v, 255} }

// Apart on four pixels: one the same, one a byte apart in every channel
// (the same, by its documented leniency), one two bytes apart and one
// four. the worst is the four, grey4.
//
// the mean is over all four pixels, the same ones included:
// (0 + 0 + 0.08467 + 0.10668) / 4 = 0.04784. swapped, the pictures are
// as far apart.
func TestApartHoldsItsLeniencyAndItsMean(t *testing.T) {
	a := pic(color.RGBA{10, 20, 30, 255}, grey(0), grey(0), grey(0))
	b := pic(color.RGBA{10, 20, 30, 255}, grey(1), grey(2), grey(4))
	wantMean := (grey2 + grey4) / 4
	for _, pair := range [][2]*image.RGBA{{a, b}, {b, a}} {
		worst, mean := apartIn(t, pair[0], pair[1])
		if !closeTo(worst, grey4, 3e-5) || !closeTo(mean, wantMean, 3e-5) {
			t.Errorf("apart is %.5f worst, %.5f mean; want %.5f, %.5f", worst, mean, grey4, wantMean)
		}
	}
}

// a byte apart in every channel is the same, whichever picture is the
// brighter, at the dark end and at the bright: 200 and 201 are a byte
// apart, and so are 255 and 254, and so are channels that are a byte
// apart each in a different direction.
func TestApartTakesAByteAsTheSame(t *testing.T) {
	pairs := [][2]color.RGBA{
		{grey(0), grey(1)}, {grey(1), grey(0)},
		{grey(200), grey(201)}, {grey(201), grey(200)},
		{grey(255), grey(254)},
		{color.RGBA{200, 100, 50, 255}, color.RGBA{201, 99, 50, 255}},
		{color.RGBA{99, 101, 50, 255}, color.RGBA{100, 100, 51, 255}},
	}
	for _, p := range pairs {
		if worst, mean := apartIn(t, pic(p[0]), pic(p[1])); worst != 0 || mean != 0 {
			t.Errorf("%v against %v is %v, %v; want nothing", p[0], p[1], worst, mean)
		}
	}
}

// two bytes apart in a single channel is not the same, however near the
// other two are: black against a pixel of byte 2 in red alone, in green
// alone and in blue alone, with the other channels the same or a byte
// out. a single pixel's mean is its worst.
func TestApartSeesTwoBytesInOneChannel(t *testing.T) {
	for _, c := range []struct {
		name string
		to   color.RGBA
		want float64
	}{
		{"red", color.RGBA{2, 0, 0, 255}, red2},
		{"green", color.RGBA{0, 2, 0, 255}, green2},
		{"blue", color.RGBA{0, 0, 2, 255}, blue2},
	} {
		worst, mean := apartIn(t, pic(grey(0)), pic(c.to))
		if !closeTo(worst, c.want, 2e-4) || !closeTo(mean, c.want, 2e-4) {
			t.Errorf("black against %s is %.5f worst, %.5f mean; want %.5f", c.name, worst, mean, c.want)
		}
	}
	// a byte out in the two other channels does not hide the third
	worst, _ := apartIn(t, pic(grey(0)), pic(color.RGBA{2, 1, 1, 255}))
	if !(worst > 0.03) {
		t.Errorf("two bytes in red with a byte in the others is %.5f apart", worst)
	}
}

// the worst is the largest, wherever it falls in the row, and the mean is over
// every pixel: two pixels, grey2 and grey4 in either order, are grey4 at worst
// and (grey2 + grey4) / 2 = 0.09568 on average. a picture of no pixels is
// nothing apart, not a quotient of zero.
func TestApartWorstAndMeanAndNone(t *testing.T) {
	for _, order := range [][]uint8{{2, 4}, {4, 2}} {
		a := pic(grey(0), grey(0))
		b := pic(grey(order[0]), grey(order[1]))
		worst, mean := apartIn(t, a, b)
		if !closeTo(worst, grey4, 3e-5) || !closeTo(mean, (grey2+grey4)/2, 3e-5) {
			t.Errorf("%v: worst %.5f, mean %.5f; want %.5f, %.5f",
				order, worst, mean, grey4, (grey2+grey4)/2)
		}
	}
	empty := image.NewRGBA(image.Rect(0, 0, 0, 0))
	if worst, mean := apartIn(t, empty, empty); worst != 0 || mean != 0 {
		t.Errorf("no pixels are %v, %v apart", worst, mean)
	}
}

// Apart looks at the pictures' own bounds, not at the corner: a window
// of 2 by 2 at (1, 1) in a picture 4 by 3 has four pixels, and the
// picture outside it, which differs by the whole of black against white
// everywhere, is not looked at.
//
// one pixel of the window is grey2 apart, so the worst is grey2 and the
// mean is grey2 / 4 = 0.02117.
func TestApartLooksAtTheBoundsItIsGiven(t *testing.T) {
	a := image.NewRGBA(image.Rect(0, 0, 4, 3))
	b := image.NewRGBA(image.Rect(0, 0, 4, 3))
	for y := 0; y < 3; y++ {
		for x := 0; x < 4; x++ {
			a.SetRGBA(x, y, grey(0))
			b.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
		}
	}
	window := image.Rect(1, 1, 3, 3)
	wa, wb := a.SubImage(window).(*image.RGBA), b.SubImage(window).(*image.RGBA)
	for y := 1; y < 3; y++ {
		for x := 1; x < 3; x++ {
			wb.SetRGBA(x, y, grey(0))
		}
	}
	wb.SetRGBA(2, 2, grey(2))
	worst, mean := apartIn(t, wa, wb)
	if !closeTo(worst, grey2, 3e-5) || !closeTo(mean, grey2/4, 1e-5) {
		t.Errorf("the window is %.5f worst, %.5f mean; want %.5f, %.5f", worst, mean, grey2, grey2/4)
	}
}

// Error develops both frames through the tone before it compares them:
// at exposure 2 a light of 2 / 255 / 12.92 is the byte 4 and black is
// black, so a frame of one such pixel and one black against two black
// is grey4 at worst and half that on average.
//
// at an exposure of 1 the same light is the byte 2, grey2 at worst.
func TestErrorDevelopsBothThroughTheTone(t *testing.T) {
	light := float32(2.0 / 255 / 12.92)
	exact := frame.New(2, 1)
	taken := frame.New(2, 1)
	taken.Set(0, 0, frame.Pixel{R: light, G: light, B: light, Cover: 1})
	worst, mean := Error(exact, taken, colour.Tone{Exposure: 2})
	if !closeTo(worst, grey4, 3e-5) || !closeTo(mean, grey4/2, 3e-5) {
		t.Errorf("at exposure 2 the error is %.5f, %.5f; want %.5f, %.5f", worst, mean, grey4, grey4/2)
	}
	worst, mean = Error(exact, taken, colour.Tone{})
	if !closeTo(worst, grey2, 3e-5) || !closeTo(mean, grey2/2, 3e-5) {
		t.Errorf("at exposure 1 the error is %.5f, %.5f; want %.5f, %.5f", worst, mean, grey2, grey2/2)
	}
}

// a shortcut is honest when its worst is no more than its budget, which
// is 0.02 for each of the five, and a name with no budget is never
// honest, whatever its worst.
func TestHonestIsWithinABudget(t *testing.T) {
	names := []string{"baked", "skip", "distance", "once", "cutoff"}
	if len(Budgets) != 5 {
		t.Errorf("there are %d budgets, want 5", len(Budgets))
	}
	for _, name := range names {
		if Budgets[name] != 0.02 {
			t.Errorf("the budget for %s is %v, want 0.02", name, Budgets[name])
		}
		if !Honest(name, 0) || !Honest(name, 0.01) || !Honest(name, 0.02) {
			t.Errorf("%s at or under its budget is not honest", name)
		}
		if Honest(name, 0.0200001) || Honest(name, 0.03) || Honest(name, 1) {
			t.Errorf("%s over its budget is honest", name)
		}
	}
	for _, name := range []string{"", "bake", "Baked", "fast", "all", "none"} {
		if Honest(name, 0) || Honest(name, 0.01) || Honest(name, 5) {
			t.Errorf("%q has no budget and is honest", name)
		}
	}
}
