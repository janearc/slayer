package colour_test

import (
	"image"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/colour"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/frame"
	"pgregory.net/rapid"
)

// lightOf is the light a row of a table gives. It reads the columns r
// and g and b with a prefix before each.
func lightOf(t *testing.T, row data.Row, prefix string) maths.Light {
	t.Helper()
	return maths.RGB(amount(t, row, prefix+"r"), amount(t, row, prefix+"g"),
		amount(t, row, prefix+"b"))
}

// near is whether two lights agree to a part in a billion.
func near(got, want maths.Light) bool {
	return math.Abs(got.R-want.R) < within &&
		math.Abs(got.G-want.G) < within && math.Abs(got.B-want.B) < within
}

// Above the knee light rolls off by the logarithm.
//
// Knee plus room times the log of one plus the light above the knee over the
// room. All that is over the log of one plus the top above the knee over the
// room. Room is the distance from the knee to white. Each case is worked from
// that for a grey light. Its parts are all its luminance.
//
// The other cases are a knee outside zero through one, which is no knee. Others
// are light under the knee and light past the top. The last is a colour too
// saturated to fit. It bleaches along its hue.
func TestTheShoulderRollsLightOff(t *testing.T) {
	for _, row := range rows(t, shoulderFile) {
		tone := colour.NewTone(none, amount(t, row, "knee"),
			amount(t, row, "top"), none)
		got := tone.Squeeze(lightOf(t, row, ""))
		if want := lightOf(t, row, "want_"); !near(got, want) {
			t.Errorf("%s: %v lands on %v, want %v", text(t, row, "label"),
				lightOf(t, row, ""), got, want)
		}
	}
}

// anti-tests: endless light through a knee is white, and not the black that
// zero times infinity made of it.
func TestEndlessLightThroughAKneeIsWhite(t *testing.T) {
	tone := colour.NewTone(none, half, pair, none)
	inf := math.Inf(whole)
	for _, endless := range []maths.Light{maths.Grey(inf),
		maths.RGB(inf, half, half)} {
		if got := tone.Squeeze(endless); got != maths.Grey(whole) {
			t.Errorf("%v squeezed to %v, want white", endless, got)
		}
	}
}

// The shoulder keeps its order. Brighter grey light is never darker
// after it, and it is never brighter than white.
func TestTheShoulderKeepsTheOrderOfLight(t *testing.T) {
	rapid.Check(t, shoulderKeepsOrder)
}

func shoulderKeepsOrder(t *rapid.T) {
	knee := rapid.Float64Range(half, whole-within).Draw(t, "knee")
	top := rapid.Float64Range(pair, lightMost).Draw(t, "top")
	tone := colour.NewTone(none, knee, top, none)
	dim := rapid.Float64Range(none, lightMost).Draw(t, "dim")
	bright := rapid.Float64Range(dim, lightMost).Draw(t, "bright")
	low := tone.Squeeze(maths.Grey(dim)).R
	high := tone.Squeeze(maths.Grey(bright)).R
	if low > high+within || high > whole+within {
		t.Fatalf("%v squeezes to %v and %v to %v", dim, low, bright, high)
	}
}

// frameOf is a frame of one pixel of the light given.
func frameOf(light maths.Light) *frame.Frame {
	made := frame.New(whole, whole)
	made.Set(none, none, pixel(light))
	return made
}

// Exposure scales the light before the squeeze. Nought is read as one.
// Quarter light at twice the exposure is half.
func TestExposureScalesTheLight(t *testing.T) {
	rapid.Check(t, exposureScales)
	quarter := frameOf(maths.Grey(half * half))
	if got := colour.Develop(quarter, pair).RGBAAt(none, none).R; got !=
		colour.Encode(half) {
		t.Errorf("a quarter at twice the exposure is %d, want half", got)
	}
}

func exposureScales(t *rapid.T) {
	share := rapid.Float64Range(none, whole)
	exposure := rapid.Float64Range(whole, exposureMost).Draw(t, "exposure")
	drawn := share.Draw(t, "light")
	light := maths.Grey(snapped(drawn))
	got := colour.Develop(frameOf(light), exposure).RGBAAt(none, none)
	want := colour.Encode(light.R * exposure)
	if got.R != want || got.G != want || got.B != want {
		t.Fatalf("%v at %v came out %v, want %d", light, exposure, got, want)
	}
	if plain := colour.Develop(frameOf(light), none).RGBAAt(none,
		none); plain.R != colour.Encode(light.R) {
		t.Fatalf("no exposure came out %v, not %d", plain, colour.Encode(
			light.R))
	}
}

// Tones with no knee are Develop. With a knee, light far above white
// lands under white and keeps its hue. Light under the knee is as it was.
func TestAToneWithNoKneeIsDevelop(t *testing.T) {
	rapid.Check(t, noKneeIsDevelop)
}

func noKneeIsDevelop(t *rapid.T) {
	exposure := rapid.Float64Range(whole, exposureMost).Draw(t, "exposure")
	red := rapid.Float64Range(none, lightMost).Draw(t, "red")
	light := maths.RGB(snapped(red), none, none)
	var plain *image.RGBA = colour.Develop(frameOf(light), exposure)
	tone := colour.NewTone(exposure, none, none, none)
	same := tone.Develop(frameOf(light))
	if plain.RGBAAt(none, none) != same.RGBAAt(none, none) {
		t.Fatalf("a tone with no knee differs from develop for %v", light)
	}
}

// anti-tests: forty times white through a knee is not white. It keeps
// its order of hue. The clamp left it white.
func TestAShoulderKeepsHueWhereTheClampDoesNot(t *testing.T) {
	bright := frameOf(maths.RGB(brightLight, brightLight/pair, brightLight/four))
	clamped := colour.Develop(bright, whole).RGBAAt(none, none)
	rolled := colour.NewTone(none, kneeDrawn, none, none).Develop(bright).RGBAAt(
		none, none)
	if clamped.R != byteMost || clamped.G != byteMost {
		t.Fatalf("the clamp left %v", clamped)
	}
	red, green, blue := colour.Decode(rolled.R), colour.Decode(rolled.G),
		colour.Decode(rolled.B)
	if rolled.G == byteMost || !(red > green && green > blue) {
		t.Errorf("the bright light clipped or turned: %v %v %v", red,
			green, blue)
	}
}
