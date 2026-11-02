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

// of is a frame of one pixel with the three parts given.
func of(r, g, b float32) *frame.Frame {
	f := frame.New(1, 1)
	f.Set(0, 0, frame.Pixel{R: r, G: g, B: b, Cover: 1})
	return f
}

// a visible layer adds as its own colour, all three parts: a half, a
// quarter and an eighth of the light come out as the bytes the curve
// gives them, 188 for the half.
func TestAVisibleLayerIsItsOwnColour(t *testing.T) {
	got := False([]Layer{{Frame: of(0.5, 0.25, 0.125)}}).RGBAAt(0, 0)
	want := color.RGBA{Encode(0.5), Encode(0.25), Encode(0.125), 255}
	if got != want || got.R != 188 {
		t.Errorf("a half, a quarter and an eighth came out %v, want %v", got, want)
	}
}

// a gain is how strongly a layer shows, a multiple: half the gain is
// half the light, four times the gain is four times. a visible layer
// scales all of its colour; a band's, only the red, which is all it
// reads, and the green and blue of its pixel do not count.
func TestAGainScalesALayer(t *testing.T) {
	got := False([]Layer{{Frame: of(1, 0.5, 0.25), Gain: 0.5}}).RGBAAt(0, 0)
	want := color.RGBA{Encode(0.5), Encode(0.25), Encode(0.125), 255}
	if got != want {
		t.Errorf("a visible layer at half gain came out %v, want %v", got, want)
	}
	long := maths.Band{Nanometres: 8e-4}
	stand := Stands(long)
	for _, c := range []struct{ red, gain, scale float64 }{
		{0.1, 0.5, 0.05}, // half the gain on a tenth of red
		{0.05, 4, 0.2},   // four times the gain on a twentieth
		{0.1, 0, 0.1},    // no gain is one
	} {
		// the pixel's green and blue are full, and not read
		f := of(float32(c.red), 1, 1)
		got := False([]Layer{{Band: long, Frame: f, Gain: c.gain}}).RGBAAt(0, 0)
		light := stand.Scale(c.scale)
		want := color.RGBA{Encode(light.R), Encode(light.G), Encode(light.B), 255}
		if got != want {
			t.Errorf("a band at gain %v on red %v came out %v, want %v", c.gain, c.red, got, want)
		}
	}
}

// layers add before the tone: two eighths of light at twice the
// exposure are a half, which is 188. a bright layer through a knee is
// rolled off and not clipped.
func TestAToneIsAppliedToTheSumOfTheLayers(t *testing.T) {
	two := []Layer{{Frame: of(0.125, 0.125, 0.125)}, {Frame: of(0.125, 0.125, 0.125)}}
	got := Tone{Exposure: 2}.False(two).RGBAAt(0, 0)
	if got != (color.RGBA{188, 188, 188, 255}) {
		t.Errorf("two eighths at twice the exposure came out %v, want 188", got)
	}
	// grey 40 through a knee of 0.6 and the default top of 100: 0.6 +
	// 0.4*ln(99.5)/ln(249.5), about 0.933, which is under white
	rolled := 0.6 + 0.4*math.Log(99.5)/math.Log(249.5)
	bright := Tone{Knee: 0.6}.False([]Layer{{Frame: of(40, 40, 40)}}).RGBAAt(0, 0)
	if want := Encode(rolled); bright.R != want || bright.R == 255 {
		t.Errorf("forty times white through the knee is %v, want %d", bright, want)
	}
}

// layers of nothing are a picture of nothing, of no size, and not a
// panic on the first layer that is not there.
func TestNoLayersIsNoPicture(t *testing.T) {
	for name, img := range map[string]*image.RGBA{
		"false":      False(nil),
		"tone false": Tone{Exposure: 2}.False([]Layer{}),
	} {
		if img == nil || img.Bounds() != image.Rect(0, 0, 0, 0) || len(img.Pix) != 0 {
			t.Errorf("%s of no layers is %v", name, img)
		}
	}
}

// layers are one size, in width and in height: either alone is enough
// to refuse one, by name.
func TestALayerOfAnotherWidthOrHeightIsRefused(t *testing.T) {
	for _, c := range []struct {
		other *frame.Frame
		want  string
	}{
		{frame.New(3, 2), "layer 1 is 3 by 2"},
		{frame.New(2, 3), "layer 1 is 2 by 3"},
	} {
		said := refusal(func() { False([]Layer{{Frame: frame.New(2, 2)}, {Frame: c.other}}) })
		if !strings.Contains(said, c.want) {
			t.Errorf("a layer of %d by %d on a 2 by 2: %q", c.other.W, c.other.H, said)
		}
	}
}

// every pixel is its own place in a picture of several rows: of a
// three by two, the pixel at (2, 1) is a half and the one at (1, 0) a
// quarter, and the rest are dark: 188 and 137 for the two.
func TestTheLayersPixelsKeepTheirPlaces(t *testing.T) {
	f := frame.New(3, 2)
	f.Set(2, 1, frame.Pixel{R: 0.5, G: 0.5, B: 0.5, Cover: 1})
	f.Set(1, 0, frame.Pixel{R: 0.25, G: 0.25, B: 0.25, Cover: 1})
	img := False([]Layer{{Frame: f}})
	if img.Bounds() != image.Rect(0, 0, 3, 2) {
		t.Fatalf("a 3 by 2 layer made a picture %v", img.Bounds())
	}
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			want := color.RGBA{0, 0, 0, 255}
			switch {
			case x == 2 && y == 1:
				want = color.RGBA{188, 188, 188, 255}
			case x == 1 && y == 0:
				want = color.RGBA{137, 137, 137, 255}
			}
			if got := img.RGBAAt(x, y); got != want {
				t.Errorf("(%d, %d) is %v, want %v", x, y, got, want)
			}
		}
	}
}
