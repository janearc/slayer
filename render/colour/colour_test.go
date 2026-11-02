package colour

import (
	"image/color"
	"math"
	"testing"

	"github.com/janearc/slayer/render/frame"
)

func TestHalfTheLightIs188NotHalfTheByte(t *testing.T) {
	for _, c := range []struct {
		in   float64
		want uint8
	}{{0, 0}, {0.5, 188}, {1, 255}, {3, 255}, {-1, 0}} {
		if got := Encode(c.in); got != c.want {
			t.Errorf("Encode(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestDecodeUndoesEncode(t *testing.T) {
	for b := 0; b < 256; b++ {
		if got := Encode(Decode(uint8(b))); got != uint8(b) {
			t.Fatalf("byte %d came back as %d", b, got)
		}
	}
}

func TestExposureScalesBeforeTheSqueeze(t *testing.T) {
	f := frame.New(1, 1)
	f.Set(0, 0, frame.Pixel{R: 0.25, G: 0.25, B: 0.25, Cover: 1})
	if got := Develop(f, 2).RGBAAt(0, 0).R; got != 188 {
		t.Fatalf("a quarter of the light at twice the exposure is %d, want 188", got)
	}
}

// A tone with no knee is Develop; with a knee, light far above white
// lands under white, keeps its hue, and light under the knee is as it
// was.
func TestTheShoulderFitsBrightLightAndKeepsHue(t *testing.T) {
	f := frame.New(3, 1)
	f.Set(0, 0, frame.Pixel{R: 0.3, G: 0.2, B: 0.1, Cover: 1})
	f.Set(1, 0, frame.Pixel{R: 40, G: 20, B: 10, Cover: 1})
	f.Set(2, 0, frame.Pixel{R: 0.9, G: 0.9, B: 0.9, Cover: 1})
	plain, same := Develop(f, 1), Tone{}.Develop(f)
	for x := 0; x < 3; x++ {
		if plain.RGBAAt(x, 0) != same.RGBAAt(x, 0) {
			t.Fatalf("a tone with no knee differs from develop at %d", x)
		}
	}
	if plain.RGBAAt(1, 0) != (color.RGBA{255, 255, 255, 255}) {
		t.Fatalf("the clamp left %v", plain.RGBAAt(1, 0))
	}
	shoulder := Tone{Knee: 0.6}.Develop(f)
	if shoulder.RGBAAt(0, 0) != plain.RGBAAt(0, 0) {
		t.Errorf("under the knee the shoulder changed %v to %v", plain.RGBAAt(0, 0), shoulder.RGBAAt(0, 0))
	}
	bright := shoulder.RGBAAt(1, 0)
	if bright.R == 255 && bright.G == 255 {
		t.Errorf("forty times white is still white: %v", bright)
	}
	// the brightest part on white and the rest under it, the order
	// kept, and the chroma's direction kept: the gaps between the parts
	// stay in the same proportion
	r, g, b := Decode(bright.R), Decode(bright.G), Decode(bright.B)
	if bright.G == 255 || !(r > g && g > b) {
		t.Errorf("the bright light clipped or turned: %v %v %v", r, g, b)
	}
	if math.Abs((r-g)/(g-b)-2) > 0.2 {
		t.Errorf("the hue moved: %v %v %v", r, g, b)
	}
}

// Bloom spreads the light a pixel holds beyond white to its neighbours:
// a lone pixel at twenty times white lights the pixels round it, and a
// picture with nothing beyond white is untouched.
func TestBloomSpreadsWhatIsBeyondWhite(t *testing.T) {
	f := frame.New(9, 9)
	for y := 0; y < 9; y++ {
		for x := 0; x < 9; x++ {
			f.Set(x, y, frame.Pixel{R: 0.1, G: 0.1, B: 0.1, Cover: 1})
		}
	}
	f.Set(4, 4, frame.Pixel{R: 20, G: 20, B: 20, Cover: 1})
	plain := Tone{}.Develop(f)
	bloomed := Tone{Bloom: 1.2}.Develop(f)
	if plain.RGBAAt(5, 4) != plain.RGBAAt(0, 0) {
		t.Fatal("without bloom the neighbour changed")
	}
	near, far := bloomed.RGBAAt(5, 4).R, bloomed.RGBAAt(0, 0).R
	if !(near > plain.RGBAAt(5, 4).R+20 && far == plain.RGBAAt(0, 0).R) {
		t.Errorf("the neighbour is %d, the corner %d, plain %d", near, far, plain.RGBAAt(0, 0).R)
	}
	f.Set(4, 4, frame.Pixel{R: 0.9, G: 0.9, B: 0.9, Cover: 1})
	if (Tone{Bloom: 1.2}).Develop(f).RGBAAt(5, 4) != plain.RGBAAt(5, 4) {
		t.Error("a picture with nothing beyond white bloomed")
	}
}
