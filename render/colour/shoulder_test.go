package colour

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// a knee outside 0 through 1 is no knee: light of any size comes back as it
// came, whether the knee is zero, below it, exactly one, or above it. at one
// the roll-off has no room to roll in, and would be a zero divided into a
// number.
func TestNoKneeOrOneOutOfRangeIsTheClamp(t *testing.T) {
	for _, knee := range []float64{0, -1, 1, 1.5, 3} {
		l := maths.RGB(5, 3, 1)
		if got := (Tone{Knee: knee}).squeeze(l); got != l {
			t.Errorf("a knee of %v squeezed %v to %v", knee, l, got)
		}
	}
}

// light at or under the knee is as it is, even a colour already past
// white. two of red has a luminance of 2 * 0.2126 = 0.4252, so a knee
// of exactly that has the light on it, and the shoulder would otherwise
// draw it toward white.
func TestLightOnTheKneeIsAsItIs(t *testing.T) {
	if got := (Tone{Knee: 0.5}).squeeze(maths.Grey(0.4)); got != maths.Grey(0.4) {
		t.Errorf("under the knee a grey became %v", got)
	}
	red := maths.RGB(2, 0, 0)
	if got := (Tone{Knee: red.Luminance()}).squeeze(red); got != red {
		t.Errorf("red on the knee became %v", got)
	}
}

// above the knee light rolls off by the logarithm:
//
//	knee + room*ln(1 + (y - knee)/room) / ln(1 + (top - knee)/room)
//
// with room the distance from the knee to white, so that top lands on
// white and a light halfway there is further than halfway up. each
// case is worked from that, for a grey light, whose parts are all its
// luminance.
func TestTheShoulderRollsOffByTheLogarithm(t *testing.T) {
	for _, c := range []struct {
		name string
		tone Tone
		in   float64
		want float64
	}{
		// room 0.5; y - knee = 1 so 1 + 2 = 3; top - knee = 2 so 1 + 4 = 5
		{"top 2.5", Tone{Knee: 0.5, Top: 2.5}, 1.5, 0.5 + 0.5*math.Log(3)/math.Log(5)},
		// the top is white
		{"top itself", Tone{Knee: 0.5, Top: 2.5}, 2.5, 1},
		// top 2: top - knee = 1.5 so 1 + 3 = 4
		{"top 2", Tone{Knee: 0.5, Top: 2}, 1.5, 0.5 + 0.5*math.Log(3)/math.Log(4)},
		// no top, or one of one or less, is a hundred: 1 + 99.5/0.5 = 200
		{"no top", Tone{Knee: 0.5}, 1.5, 0.5 + 0.5*math.Log(3)/math.Log(200)},
		{"top of one", Tone{Knee: 0.5, Top: 1}, 1.5, 0.5 + 0.5*math.Log(3)/math.Log(200)},
		{"top of a half", Tone{Knee: 0.5, Top: 0.5}, 1.5, 0.5 + 0.5*math.Log(3)/math.Log(200)},
		// knee 0.6, room 0.4; y - knee = 39.4 so 1 + 98.5; top - knee =
		// 99.4 so 1 + 248.5
		{"a brighter light, a higher knee", Tone{Knee: 0.6}, 40, 0.6 + 0.4*math.Log(99.5)/math.Log(249.5)},
	} {
		got := c.tone.squeeze(maths.Grey(c.in))
		if !near(got, maths.Grey(c.want)) {
			t.Errorf("%s: %v lands on %v, want %v", c.name, c.in, got, c.want)
		}
	}
}

// light past the top, and endless light, are white and not more. endless light
// is drawn white by rule: zero times infinity is no number, so it is not left
// to the arithmetic.
func TestLightPastTheTopIsWhite(t *testing.T) {
	tone := Tone{Knee: 0.5, Top: 2}
	for _, l := range []maths.Light{maths.Grey(4), maths.Grey(1e6), maths.RGB(math.Inf(1), 0.3, 0.3), maths.Grey(math.Inf(1))} {
		if got := tone.squeeze(l); !near(got, maths.Grey(1)) {
			t.Errorf("%v squeezed to %v, want white", l, got)
		}
	}
}

// a colour too saturated to fit is drawn toward white along its hue, until its
// brightest part is exactly white and the luminance is what the roll-off gave
// it. four times red through a knee of a half: the luminance is 4 * 0.2126 =
// 0.8504, and the roll-off puts that at
//
//	s = 0.5 + 0.5*ln(1 + 0.3504/0.5) / ln(200)
//
// the red part is then 1 and the green and blue, which were equal and zero, are
// equal and share the rest: 0.2126 + 0.7874*g = s.
func TestASaturatedColourBleachesAlongItsHue(t *testing.T) {
	s := 0.5 + 0.5*math.Log(1+0.3504/0.5)/math.Log(200)
	g := (s - 0.2126) / (0.7152 + 0.0722)
	got := (Tone{Knee: 0.5}).squeeze(maths.RGB(4, 0, 0))
	if !near(got, maths.RGB(1, g, g)) {
		t.Errorf("four times red is %v, want %v", got, maths.RGB(1, g, g))
	}
	if !(g > 0 && g < s) {
		t.Fatalf("the test's own green, %v, is not under the luminance %v", g, s)
	}
}
