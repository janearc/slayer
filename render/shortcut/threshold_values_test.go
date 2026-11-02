package shortcut

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
)

// clipHand is the threshold of the clip tone at a top of one, worked by hand. a
// grey pixel shows when its byte is greater than zero. the byte is round(255 *
// 12.92 * l) for a light l under the sRGB toe (0.0031308), and it is greater
// than zero when 255 * 12.92 * l reaches a half.
//
// so the light at which it first shows is 0.5 / 255 / 12.92 = 1.5176e-4. the
// bisection answers the largest light that does not show, within 2^-40 of the
// edge.
//
// the pixel holds its light as a float32 whose step here is 2^-36, so the edge
// may sit up to half of that, 7.3e-12, from the hand value. holding the answer
// to 1e-11 leaves the float32 its room and no more.
const clipHand = 0.5 / 255 / 12.92

// shown is whether the clip tone develops the grey light l into a pixel
// above black.
func shown(tone colour.Tone, l float64) bool {
	f := frame.New(1, 1)
	f.Set(0, 0, frame.Pixel{R: float32(l), G: float32(l), B: float32(l), Cover: 1})
	c := tone.Develop(f).RGBAAt(0, 0)
	return c.R > 0 || c.G > 0 || c.B > 0
}

// the clip tone's threshold is the light that first shows, 1.5176e-4.
func TestTheClipThresholdIsWhereALightFirstShows(t *testing.T) {
	got := thresholdIn(t, colour.Tone{}, 1)
	if !closeTo(got, clipHand, 1e-11) {
		t.Errorf("the clip threshold is %.12g, want %.12g", got, clipHand)
	}
	if shown(colour.Tone{}, got) {
		t.Errorf("a light of the threshold, %.12g, shows", got)
	}
	// the edge is within a bisection's last step: a light that much
	// above the threshold shows
	if !shown(colour.Tone{}, got+0x1p-40) {
		t.Errorf("a light a last step above the threshold, %.12g, does not show", got+0x1p-40)
	}
}

// the answer is one number, and it is worked by hand. the pixel holds
// float32(mid), and the float32 step from 2^-13 to 2^-12 is 2^-36.
// 0.5 / 255 / 12.92 is 10429107.74 of those steps, so the first light
// that shows is 10429108 steps.
//
// a mid shows when it rounds to that step or above, which is from half
// a step under it, 10429107.5 steps (a tie goes to the even 10429108),
// = 166865720 * 2^-40. the bisection keeps lo under that edge and hi
// at or over it, 2^-40 apart after 40 passes, so lo is 166865719 * 2^-40.
//
// a pass more or fewer lands on another multiple of its own step.
func TestTheClipThresholdIsOneNumber(t *testing.T) {
	const want = 166865719.0 / (1 << 40)
	if got := thresholdIn(t, colour.Tone{}, 1); got != want {
		t.Errorf("the clip threshold is %.20g, want %.20g", got, want)
	}
}

// the threshold is a transmittance, so what it lets through is it times the
// top: a top of 100 makes it a hundredth of the hand value (1.5176e-6); a top
// of 4 makes it a quarter; a top at or under zero is taken as one.
func TestTheThresholdIsInTransmittance(t *testing.T) {
	for _, c := range []struct {
		top  float64
		want float64
		tol  float64
	}{
		{100, clipHand / 100, 2e-12},
		{4, clipHand / 4, 1e-11},
		{0.5, clipHand * 2, 2e-11},
		{0, clipHand, 1e-11},
		{-3, clipHand, 1e-11},
	} {
		if got := thresholdIn(t, colour.Tone{}, c.top); !closeTo(got, c.want, c.tol) {
			t.Errorf("at a top of %v the threshold is %.12g, want %.12g", c.top, got, c.want)
		}
	}
}

// a light so dim that even all of it, a keep of one, does not show has
// a threshold of one: nothing it could be cut for. a top of 1e-5 is a
// 15th of the 1.5176e-4 that first shows.
//
// a keep of one that does show, a top of 1.6e-4 just over the edge,
// bisects to the edge over the top: 1.5176e-4 / 1.6e-4 = 0.9485.
func TestADimTopNeverShows(t *testing.T) {
	if got := thresholdIn(t, colour.Tone{}, 1e-5); got != 1 {
		t.Errorf("a top of 1e-5 gives %v, want 1", got)
	}
	got := thresholdIn(t, colour.Tone{}, 1.6e-4)
	if want := clipHand / 1.6e-4; !closeTo(got, want, 1e-7) || !(got < 1) {
		t.Errorf("a top of 1.6e-4 gives %.9g, want %.9g", got, want)
	}
}

// the tone moves the threshold as it should. two stops up is exposure
// 4, which multiplies the light by 4, so the light that first shows is
// a quarter of the hand value.
//
// a knee at a half leaves a light under the knee as it was, and 1.5e-4
// is far under, so the threshold is the hand value.
func TestTheToneMovesTheThreshold(t *testing.T) {
	if got := thresholdIn(t, colour.Tone{Exposure: 4}, 1); !closeTo(got, clipHand/4, 1e-11) {
		t.Errorf("at exposure 4 the threshold is %.12g, want %.12g", got, clipHand/4)
	}
	if got := thresholdIn(t, colour.Tone{Exposure: 0.5}, 1); !closeTo(got, clipHand*2, 2e-11) {
		t.Errorf("at exposure 0.5 the threshold is %.12g, want %.12g", got, clipHand*2)
	}
	if got := thresholdIn(t, colour.Tone{Knee: 0.5}, 1); !closeTo(got, clipHand, 1e-11) {
		t.Errorf("with a knee the threshold is %.12g, want %.12g", got, clipHand)
	}
}

// the remainder is keep times top: not the sum, the quotient or either alone.
// 0.5 * 8 = 4, 2 * 3 = 6, and a keep of zero leaves nothing.
func TestTheRemainderIsKeepTimesTop(t *testing.T) {
	for _, c := range []struct{ keep, top, want float64 }{
		{0.5, 8, 4}, {2, 3, 6}, {0, 7, 0}, {7, 0, 0}, {0.25, 0.25, 0.0625},
	} {
		if got := Remainder(c.keep, c.top); got != c.want {
			t.Errorf("Remainder(%v, %v) is %v, want %v", c.keep, c.top, got, c.want)
		}
	}
}

// the brightest light is the largest channel of any lamp, whichever lamp and
// whichever channel; with no lamp, or none greater than zero, it is one.
func TestTheBrightestIsTheLargestChannel(t *testing.T) {
	for _, c := range []struct {
		name  string
		lamps []maths.Light
		want  float64
	}{
		{"red alone", []maths.Light{maths.RGB(5, 1, 2)}, 5},
		{"green alone", []maths.Light{maths.RGB(1, 5, 2)}, 5},
		{"blue alone", []maths.Light{maths.RGB(1, 2, 5)}, 5},
		{"the second lamp", []maths.Light{maths.RGB(3, 2, 1), maths.RGB(1, 2, 9)}, 9},
		{"the first lamp", []maths.Light{maths.RGB(9, 2, 1), maths.RGB(3, 2, 1)}, 9},
		{"under one", []maths.Light{maths.RGB(0.25, 0.5, 0.125)}, 0.5},
		{"none", nil, 1},
		{"a dark lamp", []maths.Light{maths.RGB(0, 0, 0)}, 1},
		{"a lamp under nought", []maths.Light{maths.RGB(-2, -1, -3)}, 1},
	} {
		if got := Brightest(c.lamps); got != c.want {
			t.Errorf("%s: the brightest is %v, want %v", c.name, got, c.want)
		}
	}
}
