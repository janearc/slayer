package colour

import (
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
)

// a picture cut from a larger one keeps its own pixels through Masked
// and Grained: each pixel of a ten by four picture is its own x and y,
// and the six by two cut at (2, 1) of it is (x, y) at (x, y), not the
// larger picture's rows run together.
func TestAPictureCutFromALargerOneCopiesWhole(t *testing.T) {
	big := image.NewRGBA(image.Rect(0, 0, 10, 4))
	for y := range 4 {
		for x := range 10 {
			big.SetRGBA(x, y, color.RGBA{uint8(x), uint8(y), 7, 255})
		}
	}
	sub := big.SubImage(image.Rect(2, 1, 8, 3)).(*image.RGBA)
	for name, out := range map[string]*image.RGBA{
		"masked":  Masked(sub, 3, 1),
		"grained": Grained(sub, 0, 0, 1),
	} {
		for y := 1; y < 3; y++ {
			for x := 2; x < 8; x++ {
				if got, want := out.RGBAAt(x, y), (color.RGBA{uint8(x), uint8(y), 7, 255}); got != want {
					t.Errorf("%s at %d, %d is %v, want %v", name, x, y, got, want)
				}
			}
		}
	}
}

// light without bound, through a knee, is the shoulder's end: white, not the
// black that zero times infinity made of it. a part that is no number is black
// by rule.
func TestEndlessLightIsWhiteAndNoNumberIsBlack(t *testing.T) {
	px := Tone{Knee: 0.6}.pixel(maths.RGB(math.Inf(1), 0.3, 0.3))
	if px != (color.RGBA{255, 255, 255, 255}) {
		t.Errorf("endless red through the knee is %v", px)
	}
	if got := Encode(math.NaN()); got != 0 {
		t.Errorf("no number encodes as %d", got)
	}
}

// a bloom wider than any picture, or endless, is refused by name; one so fine
// its radius squares to zero leaves a star as it was, its middle weighed one.
func TestABloomIsASize(t *testing.T) {
	for _, r := range []float64{math.Inf(1), 1e9} {
		func() {
			defer func() {
				got, _ := recover().(string)
				if !strings.Contains(got, "bloom") {
					t.Errorf("a bloom of %v: %q", r, got)
				}
			}()
			bloomed([]maths.Light{maths.Grey(5)}, 1, 1, 1, r)
		}()
	}
	star := []maths.Light{maths.Grey(0), maths.Grey(4), maths.Grey(0)}
	got := bloomed(star, 3, 1, 1, 1e-200)
	// the spill is a third of the three past white in a light of four,
	// one, all of it kept at the middle: 4 + 1
	if got[1] != maths.Grey(5) || got[0] != (maths.Light{}) {
		t.Errorf("a star under the finest bloom is %v", got)
	}
}

// a composite's layers are one size: one that is not is refused by name,
// not left out of the picture without a word.
func TestALayerOfAnotherSizeIsRefused(t *testing.T) {
	defer func() {
		got, _ := recover().(string)
		if !strings.Contains(got, "layer 1 is 3 by 3") {
			t.Errorf("a 3 by 3 layer on a 2 by 2: %q", got)
		}
	}()
	False([]Layer{{Frame: frame.New(2, 2)}, {Frame: frame.New(3, 3)}})
}
