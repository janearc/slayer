package colour

import (
	"image/color"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The shoulder with its solved scale, checked inside the package that
// is its one definition. These came from render/colour, which held a
// copy of the shoulder until 2026-11-15.

// How the shoulder is checked.
const (
	// quotientStep is the step of the difference quotients either side
	// of the knee. The shoulder's own bend over it is the step over twice
	// its scale, under 1e-6 for the least scale here, about 0.05; its
	// roundings over it are about 1e-8.
	quotientStep = 1e-8 // picked: see above
	// slopeShare is how near the quotients either side must agree.
	slopeShare = 1e-6 // defined: the brief's
	// whiteShare is how near a light the shoulder is worked for must
	// land on what it is worked to: a few roundings.
	whiteShare = 1e-12 // picked: see above
	// scaleShare is how near scaleOf's scale must be to Newton's, as a
	// share of it: a few roundings, where scaleOf is exact to its last bit.
	scaleShare = 1e-12 // picked: see above
	// newtonSteps is how many steps Newton's method takes, far more than
	// the dozen or so it needs from newtonStart.
	newtonSteps = 100 // picked: see above
	// newtonStart is where Newton's method starts: below every scale here,
	// where the white it reaches is short of the room.
	newtonStart = 1e-6 // picked: see above
	// risingSteps is how many lights from the knee to past the top the
	// rise is checked at.
	risingSteps = 10000 // picked: finer than any bend of the curve
	// riseShare is how far a value may fall and not be a fall: a rounding
	// or two of a number near one.
	riseShare = 1e-15 // picked: see above
	// pastTop is how far past the top the rise is checked, as a multiple.
	pastTop = 2 // picked: a little past white
	// belowTop is a light a part in a million under the top, short of
	// white on the shoulder.
	belowTop = 1 - 1e-6 // picked: see above
	// aboveKnee is a light a tenth of the room past the knee, where the
	// old shoulder and the new one part.
	aboveKnee = 0.1 // picked: see above
	// lightShare is how near two lights' parts must be to be one light:
	// a part in a billion.
	lightShare = 1e-9 // picked: a few roundings of a chain of steps
)

// shoulderTones are the tones the shoulder is checked at: the session's knee
// and top, the same knee with the top read as a hundred, and two others.
var shoulderTones = []Tone{
	NewTone(none, 0.7, 8, none),    // defined: hillside/run.go's tone
	NewTone(none, 0.7, none, none), // defined: the same knee, no top
	NewTone(none, 0.5, 2.5, none),  // picked: a low knee and a near top
	NewTone(none, 0.6, none, none), // picked: a knee colour_test uses
}

// belowKnee is lights of every kind of hue, each to be scaled to a
// luminance at or under a knee: greys, a red past one as FIX-REPORT found
// on Venus's sky, a deep blue past one, and a pale warm colour.
var belowKnee = []maths.Light{
	maths.Grey(1),            // picked: a grey
	maths.RGB(1.2, 0.6, 0.1), // picked: a red part past one
	maths.RGB(0.05, 0.1, 2),  // picked: a blue part past one
	maths.RGB(0.9, 0.8, 0.7), // picked: a pale warm colour
	maths.RGB(0, 1, 0),       // picked: green alone
}

// kneeShares are the shares of the knee the lights below it are scaled
// to: nothing, a little, half and nearly all, far enough under it that
// no rounding of the scaling lifts one over. The knee itself is
// TestLightOnTheKneeIsAsItIs's.
var kneeShares = []float64{0, 0.01, 0.5, 0.999} // picked: see above

// lightsAgree is whether two lights agree, part by part, to lightShare.
func lightsAgree(got, want maths.Light) bool {
	return math.Abs(got.R-want.R) < lightShare &&
		math.Abs(got.G-want.G) < lightShare &&
		math.Abs(got.B-want.B) < lightShare
}

// shoulderOf is the luminance a grey light of y comes out of a tone at,
// read as its red part, which a grey shares.
func shoulderOf(tone Tone, y float64) float64 {
	return tone.Squeeze(maths.Grey(y)).R
}

// oldShoulder is the shoulder as it was before 2026-11-15: the light as
// it is to the knee, and above it the room to white times ln(1 + (y -
// knee)/room) over ln(1 + (top - knee)/room). Its slope just above the
// knee is one over that last logarithm, about 0.17 at a top of a hundred.
func oldShoulder(knee, top, y float64) float64 {
	if y <= knee {
		return y
	}
	room := 1 - knee
	return knee + room*math.Log1p((y-knee)/room)/math.Log1p((top-knee)/room)
}

// sides is a curve's difference quotients just below and just above a
// place.
func sides(curve func(float64) float64, at float64) (below, above float64) {
	below = (curve(at) - curve(at-quotientStep)) / quotientStep
	above = (curve(at+quotientStep) - curve(at)) / quotientStep
	return below, above
}

// newtonScale is the scale a at which knee + a ln(1 + (top - knee)/a) is
// white, by Newton's method on a ln(1 + d/a) - room, d being top - knee,
// apart from scaleOf's bisection. That rises with a and bends down, so
// from below the root every step stays below it and nears it.
func newtonScale(knee, top float64) float64 {
	span, room := top-knee, 1-knee
	scale := newtonStart
	for range newtonSteps {
		value := scale*math.Log1p(span/scale) - room
		slope := math.Log1p(span/scale) - span/(scale+span)
		scale -= value / slope
	}
	return scale
}

// rises is whether a curve's value never falls, by more than a rounding,
// over risingSteps even steps from one place to another, and where it
// first does. Past the top a grey comes out white within a rounding that
// changes with the light, and that is not a fall.
func rises(curve func(float64) float64, from, to float64) (bool, float64) {
	last := curve(from)
	for step := 1; step <= risingSteps; step++ {
		at := from + (to-from)*float64(step)/risingSteps
		got := curve(at)
		if got < last-riseShare {
			return false, at
		}
		last = got
	}
	return true, to
}

// a knee outside 0 through 1 is no knee: light of any size comes back as it
// came, whether the knee is zero, below it, exactly one, or above it. at one
// the roll-off has no room to roll in, and would be a zero divided into a
// number.
func TestNoKneeOrOneOutOfRangeIsTheClamp(t *testing.T) {
	for _, knee := range []float64{0, -1, 1, 1.5, 3} { // picked: see above
		l := maths.RGB(5, 3, 1) // picked: past white in every part
		if got := NewTone(none, knee, none, none).Squeeze(l); got != l {
			t.Errorf("a knee of %v squeezed %v to %v", knee, l, got)
		}
	}
}

// a grey at or under the knee is as it is. a colour past white is not
// clipped there: its luminance is as it is and it is drawn toward white
// along its hue, as it is above the knee. two of red has a luminance of
// 2 * 0.2126 = 0.4252, so a knee of exactly that has the light on it.
// the red part is then one, and the green and blue, which were equal
// and zero, share the rest: y/(2 - y), y the luminance.
func TestLightOnTheKneeIsAsItIs(t *testing.T) {
	grey := maths.Grey(0.4) // picked: under the knee of a half
	if got := NewTone(none, half, none, none).Squeeze(grey); got != grey {
		t.Errorf("under the knee a grey became %v", got)
	}
	red := maths.RGB(2, 0, 0) // picked: past white, on its own knee
	onKnee := NewTone(none, red.Luminance(), none, none)
	rest := red.Luminance() / (red.R - red.Luminance())
	want := maths.RGB(1, rest, rest)
	if got := onKnee.Squeeze(red); !lightsAgree(got, want) {
		t.Errorf("red on the knee became %v, want %v", got, want)
	}
}

// above the knee light rolls off by the logarithm:
//
//	knee + a*ln(1 + (y - knee)/a)
//
// with a the scale at which top lands on white, a*ln(1 + (top - knee)/a) =
// 1 - knee, found here by Newton's method, apart from the tone's own. a
// coloured light keeps the ratios of its parts.
func TestTheShoulderRollsOffByTheLogarithm(t *testing.T) {
	for _, tone := range shoulderTones {
		top := tone.topped()
		scale := newtonScale(tone.knee, top)
		if got := tone.Scale(); math.Abs(got-scale) > scaleShare*scale {
			t.Errorf("knee %v, top %v: the scale is %v, Newton's %v",
				tone.knee, top, got, scale)
		}
		for _, y := range []float64{tone.knee + aboveKnee, 1, top / 2} {
			want := tone.knee + scale*math.Log1p((y-tone.knee)/scale)
			if got := shoulderOf(tone, y); math.Abs(got-want) > whiteShare {
				t.Errorf("knee %v, top %v: %v lands on %v, want %v",
					tone.knee, top, y, got, want)
			}
		}
		pale := maths.RGB(0.9, 0.8, 0.7) // picked: a pale warm colour
		warm := pale.Scale((tone.knee + aboveKnee*(1-tone.knee)) /
			pale.Luminance())
		got := tone.Squeeze(warm)
		if !lightsAgree(got.Scale(warm.R/got.R), warm) {
			t.Errorf("knee %v: a warm light %v turned to %v", tone.knee,
				warm, got)
		}
	}
}

// a top of one or less, nought among them, is read as a hundred times
// white: such a tone's shoulder is the default top's, to the bit.
func TestATopOfOneOrLessIsAHundred(t *testing.T) {
	hundred := NewTone(none, half, topDefault, none)
	for _, top := range []float64{none, half, whole} {
		tone := NewTone(none, half, top, none)
		if tone.Scale() != hundred.Scale() {
			t.Errorf("a top of %v solves %v, a hundred %v", top,
				tone.Scale(), hundred.Scale())
		}
	}
}

// the anti-test: a top just past one is a top, not the default, and its
// shoulder is not the default's.
func TestATopPastOneIsItsOwn(t *testing.T) {
	hundred := NewTone(none, half, topDefault, none)
	if near := NewTone(none, half, nearOne, none); near.Scale() ==
		hundred.Scale() {
		t.Errorf("a top of %v solves the default's scale %v", nearOne,
			near.Scale())
	}
}

// nearOne is a top just past one.
const nearOne = 1.5 // picked: past one, far under the default

// the shoulder leaves the knee at the slope it meets it with, one: the
// difference quotients just below and just above it agree, so a smooth
// gradient that crosses the knee draws no contour there.
func TestTheShouldersSlopeRunsOnAcrossTheKnee(t *testing.T) {
	for _, tone := range shoulderTones {
		below, above := sides(func(y float64) float64 {
			return shoulderOf(tone, y)
		}, tone.knee)
		if math.Abs(above-below) > slopeShare {
			t.Errorf("knee %v, top %v: slope %.9f below and %.9f above",
				tone.knee, tone.top, below, above)
		}
	}
}

// the anti-test: the shoulder as it was turns a corner at the knee, its
// slope falling from one to under a third, and the check sees it.
func TestTheOldShoulderTurnsACornerAtTheKnee(t *testing.T) {
	for _, tone := range shoulderTones {
		top := tone.topped()
		below, above := sides(func(y float64) float64 {
			return oldShoulder(tone.knee, top, y)
		}, tone.knee)
		if !(math.Abs(above-below) > slopeShare) {
			t.Errorf("knee %v, top %v: the old slope %.9f below and %.9f "+
				"above agree", tone.knee, top, below, above)
		}
	}
}

// the shoulder never falls: the luminance out rises with the luminance in,
// from the knee to past the top, for a grey and for a colour so saturated
// it bleaches.
func TestTheShoulderRises(t *testing.T) {
	red := maths.RGB(1, 0, 0) // picked: red alone
	for _, tone := range shoulderTones {
		for name, curve := range map[string]func(float64) float64{
			"grey": func(y float64) float64 { return shoulderOf(tone, y) },
			"red": func(y float64) float64 {
				share := y / red.Luminance()
				return tone.Squeeze(red.Scale(share)).Luminance()
			},
		} {
			if ok, at := rises(curve, tone.knee, pastTop*tone.topped()); !ok {
				t.Errorf("knee %v, top %v: the %s shoulder falls at %v",
					tone.knee, tone.top, name, at)
			}
		}
	}
}

// the anti-test: the check of the rise sees a curve that falls.
func TestAFallingCurveIsSeenToFall(t *testing.T) {
	if ok, _ := rises(func(y float64) float64 { return -y }, 0, 1); ok {
		t.Error("a falling curve was seen to rise")
	}
}

// the shoulder reaches white at the top and not before: the top lands on
// one within a few roundings, and a light a part in a million under it is
// short of one.
func TestTheShoulderReachesWhiteAtTheTop(t *testing.T) {
	for _, tone := range shoulderTones {
		top := tone.topped()
		if got := shoulderOf(tone, top); math.Abs(got-1) > whiteShare {
			t.Errorf("knee %v: the top %v lands on %.15f", tone.knee, top, got)
		}
		if got := shoulderOf(tone, top*belowTop); !(got < 1) {
			t.Errorf("knee %v: under the top %v is already %v", tone.knee,
				top, got)
		}
	}
}

// the anti-test: a shoulder of slope one at the knee whose scale is not
// solved but taken as the room to white is white long before the top, and
// the check sees it.
func TestAnUnsolvedScaleIsWhiteBeforeTheTop(t *testing.T) {
	for _, tone := range shoulderTones {
		unsolved := tone
		unsolved.scale = 1 - tone.knee
		level := tone.topped() * belowTop
		got := unsolved.Squeeze(maths.Grey(level)).R
		if got < 1 {
			t.Errorf("knee %v: an unsolved scale is %v under the top",
				tone.knee, got)
		}
	}
}

// below the knee nothing changes, bit for bit: a light of any hue whose
// luminance is at or under the knee and whose parts are all at or under
// white comes back as it went in, and its bytes are Encode's of its
// parts, as before the shoulder was changed. a light with a part past
// white is drawn toward white, which the tests below the table hold.
func TestBelowTheKneeNothingChanges(t *testing.T) {
	for _, tone := range shoulderTones {
		for _, light := range belowKnee {
			for _, share := range kneeShares {
				l := light.Scale(share * tone.knee / light.Luminance())
				if max(l.R, l.G, l.B) > whole {
					continue
				}
				if got := tone.Squeeze(l); got != l {
					t.Errorf("knee %v: %v became %v", tone.knee, l, got)
				}
				want := color.RGBA{Encode(l.R), Encode(l.G), Encode(l.B),
					byteMost}
				if got := tone.pixel(l); got != want {
					t.Errorf("knee %v: %v printed %v, want %v", tone.knee, l,
						got, want)
				}
			}
		}
	}
}

// the anti-test: above the knee the shoulder is not the old one, so the
// check below the knee is not passed by a curve that changed nowhere.
func TestAboveTheKneeTheShoulderIsNew(t *testing.T) {
	for _, tone := range shoulderTones {
		y := tone.knee + aboveKnee*(1-tone.knee)
		old := oldShoulder(tone.knee, tone.topped(), y)
		if got := shoulderOf(tone, y); !(math.Abs(got-old) > slopeShare) {
			t.Errorf("knee %v: at %v the shoulder %v is the old %v",
				tone.knee, y, got, old)
		}
	}
}

// continuityShare is how far apart the prints of a light a part in a
// million either side of the knee may be: a few parts in a hundred
// thousand, which the slope of the shoulder there, one, and the bleach
// together move a part by.
const continuityShare = 1e-5 // picked: see above

// kneeSide is how far either side of the knee the continuity is checked,
// as a share of the knee.
const kneeSide = 1e-6 // picked: a part in a million

// saturatedLights are colours whose parts go past white at a luminance
// near a knee, each scaled in the tests to sit on either side of it.
var saturatedLights = []maths.Light{
	maths.RGB(1.3, 0.58, 0.13), // picked: a saturated orange
	maths.RGB(4, 0.5, 0),       // picked: a deep red
	maths.RGB(0.05, 0.1, 2),    // picked: a deep blue
}

// the print is continuous at the knee: a saturated colour a part in a
// million under it and the same colour a part in a million over it print
// to lights that agree, whichever part is past white.
func TestThePrintIsContinuousAtTheKnee(t *testing.T) {
	for _, tone := range shoulderTones {
		for _, light := range saturatedLights {
			unit := tone.knee / light.Luminance()
			under := light.Scale(unit * (1 - kneeSide))
			over := light.Scale(unit * (1 + kneeSide))
			below, above := tone.Squeeze(under), tone.Squeeze(over)
			if math.Abs(below.R-above.R) > continuityShare ||
				math.Abs(below.G-above.G) > continuityShare ||
				math.Abs(below.B-above.B) > continuityShare {
				t.Errorf("knee %v: %v prints %v under it and %v over it",
					tone.knee, light, below, above)
			}
		}
	}
}

// parting is how a part stands to another: above, level or below, as
// one, nought and minus one.
func parting(part, other float64) int {
	switch {
	case part > other:
		return whole
	case part < other:
		return -whole
	}
	return none
}

// a colour under the knee with a part past white comes back with its
// brightest part exactly white, its luminance as it was, and its hue: the
// parts in the same order, and the grey of its luminance between them.
func TestAColourPastWhiteUnderTheKneeBleaches(t *testing.T) {
	for _, tone := range shoulderTones {
		for _, light := range saturatedLights {
			under := light.Scale(tone.knee * (1 - kneeSide) /
				light.Luminance())
			if max(under.R, under.G, under.B) <= whole {
				continue
			}
			got := tone.Squeeze(under)
			grey := under.Luminance()
			brightest := max(got.R, got.G, got.B)
			if math.Abs(brightest-whole) > lightShare {
				t.Errorf("knee %v: %v came out %v, brightest %v", tone.knee,
					under, got, brightest)
			}
			if math.Abs(got.Luminance()-grey) > lightShare {
				t.Errorf("knee %v: %v lost its luminance, %v", tone.knee,
					under, got)
			}
			if !hueKept(got, under) {
				t.Errorf("knee %v: %v turned to %v", tone.knee, under, got)
			}
		}
	}
}

// hueKept is whether the drawn light has the parts of the light in the
// same order, with the grey of the light's luminance between them.
func hueKept(drawn, light maths.Light) bool {
	grey := light.Luminance()
	return parting(drawn.R, drawn.G) == parting(light.R, light.G) &&
		parting(drawn.G, drawn.B) == parting(light.G, light.B) &&
		parting(drawn.R, drawn.B) == parting(light.R, light.B) &&
		min(drawn.R, drawn.G, drawn.B) <= grey &&
		grey <= max(drawn.R, drawn.G, drawn.B)
}

// light past the top, and endless light, are white and not more. endless light
// is drawn white by rule: zero times infinity is no number, so it is not left
// to the arithmetic.
func TestLightPastTheTopIsWhite(t *testing.T) {
	tone := NewTone(none, half, pair, none)
	for _, l := range []maths.Light{
		maths.Grey(4),                    // picked: twice the top
		maths.Grey(1e6),                  // picked: far past it
		maths.RGB(math.Inf(1), 0.3, 0.3), // picked: endless red
		maths.Grey(math.Inf(1)),          // picked: endless grey
	} {
		if got := tone.Squeeze(l); !lightsAgree(got, maths.Grey(1)) {
			t.Errorf("%v squeezed to %v, want white", l, got)
		}
	}
}

// a colour too saturated to fit is drawn toward white along its hue, until its
// brightest part is exactly white and the luminance is what the roll-off gave
// it. four times red through a knee of a half: the luminance is 4 * 0.2126 =
// 0.8504, and the roll-off puts that at
//
//	s = 0.5 + a*ln(1 + 0.3504/a)
//
// a being the scale for the default top. the red part is then 1 and the green
// and blue, which were equal and zero, are equal and share the rest: 0.2126 +
// 0.7874*g = s.
func TestASaturatedColourBleachesAlongItsHue(t *testing.T) {
	a := newtonScale(half, topDefault)
	s := 0.5 + a*math.Log1p(0.3504/a)     // defined: worked above
	g := (s - 0.2126) / (0.7152 + 0.0722) // defined: BT.709's weights
	got := NewTone(none, half, none, none).Squeeze(maths.RGB(4, 0, 0))
	if !lightsAgree(got, maths.RGB(1, g, g)) {
		t.Errorf("four times red is %v, want %v", got, maths.RGB(1, g, g))
	}
	if !(g > 0 && g < s) {
		t.Fatalf("the test's own green, %v, is not under the luminance %v", g, s)
	}
}
