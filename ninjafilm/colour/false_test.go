package colour_test

import (
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/colour"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/frame"
)

// the tables the false colour tests are run over
const (
	gainsFile       = "testdata/gains.csv"
	standsWhiteFile = "testdata/stands-white.csv"
)

// what the false colour tests choose, each picked in picked-numbers.yml
var (
	standsSamples = data.Whole("colour-stands-samples")
	longBand      = maths.Band{Nanometres: data.Picked("colour-long-band")}
	shortBand     = maths.Band{Nanometres: data.Picked("colour-short-band")}
	halpha        = maths.Band{Nanometres: data.Picked("colour-halpha")}
)

// visible is a layer of visible light from a frame, at a gain.
func visible(grid *frame.Frame, gain float64) colour.Layer {
	return colour.NewLayer(maths.Band{}, grid, gain, maths.Light{})
}

// Visible layers add as their own colour. That is all three parts.
// Half and a quarter and an eighth of the light come out as the bytes
// the curve gives them.
func TestAVisibleLayerIsItsOwnColour(t *testing.T) {
	seen := frameOf(maths.RGB(half, half*half, half*half*half))
	got := colour.False([]colour.Layer{visible(seen, none)}).RGBAAt(none,
		none)
	want := color.RGBA{colour.Encode(half), colour.Encode(half * half),
		colour.Encode(half * half * half), byteMost}
	if got != want {
		t.Errorf("a half, a quarter and an eighth came out %v, want %v",
			got, want)
	}
}

// Gains are how strongly a layer shows. They are multiples. Visible
// layers scale all of their colour. The layer of a band scales only the
// red, which is all it reads. The green and blue of its pixel do not
// count.
func TestAGainScalesALayer(t *testing.T) {
	seen := frameOf(maths.RGB(whole, half, half*half))
	got := colour.False([]colour.Layer{visible(seen, half)}).RGBAAt(none,
		none)
	want := color.RGBA{colour.Encode(half), colour.Encode(half * half),
		colour.Encode(half * half * half), byteMost}
	if got != want {
		t.Errorf("a visible layer at half gain came out %v, want %v", got,
			want)
	}
	stand := colour.Stands(longBand)
	for _, row := range rows(t, gainsFile) {
		red := amount(t, row, "red")
		full := frameOf(maths.RGB(red, whole, whole))
		layer := colour.NewLayer(longBand, full, amount(t, row, "gain"),
			maths.Light{})
		got := colour.False([]colour.Layer{layer}).RGBAAt(none, none)
		light := stand.Scale(amount(t, row, "scale"))
		want := color.RGBA{colour.Encode(light.R), colour.Encode(light.G),
			colour.Encode(light.B), byteMost}
		if got != want {
			t.Errorf("%s: came out %v, want %v", text(t, row, "label"), got,
				want)
		}
	}
}

// Layers add before the tone. Two eighths of light at twice the exposure
// are a half. Bright layers through a knee are rolled off and not
// clipped.
func TestAToneIsAppliedToTheSumOfTheLayers(t *testing.T) {
	eighth := frameOf(maths.Grey(half * half * half))
	two := []colour.Layer{visible(eighth, none), visible(eighth, none)}
	got := colour.NewTone(pair, none, none, none).False(two).RGBAAt(none,
		none)
	if want := colour.Encode(half); got.R != want {
		t.Errorf("two eighths at twice the exposure came out %v, want %d",
			got, want)
	}
	bright := frameOf(maths.Grey(brightLight))
	knee := colour.NewTone(none, kneeDrawn, none, none)
	rolled := knee.Squeeze(maths.Grey(brightLight)).R
	got = knee.False([]colour.Layer{visible(bright, none)}).RGBAAt(none, none)
	if got.R != colour.Encode(rolled) || got.R == byteMost {
		t.Errorf("bright light through the knee is %v, want %d", got,
			colour.Encode(rolled))
	}
}

// Tinted layers stand in their tint and not the colour of libtheme for
// their range.
func TestATintedLayerStandsInItsTint(t *testing.T) {
	seen := frameOf(maths.RGB(whole, none, none))
	green := maths.RGB(none, whole, none)
	got := colour.False([]colour.Layer{colour.NewLayer(halpha, seen, none,
		green)}).RGBAAt(none, none)
	if want := (color.RGBA{none, byteMost, none, byteMost}); got != want {
		t.Errorf("hydrogen alpha tinted green came out %v, want %v", got,
			want)
	}
}

// Bands stand for a visible colour as bright as white at unit strength.
// Different ends of a range stand for different hues. False colour
// pictures add their layers. The long end of gamma at half strength over
// a dark visible layer is half that colour.
func TestFalseColourAddsItsLayers(t *testing.T) {
	short, long := colour.Stands(shortBand), colour.Stands(longBand)
	if !(short.B > short.R && short.B > short.G) {
		t.Errorf("the short end of gamma stands for %v, not violet", short)
	}
	if !(long.R > long.G && long.R > long.B) {
		t.Errorf("the long end of gamma stands for %v, not red", long)
	}
	dim := frame.New(pair, whole)
	dim.Set(none, none, greyPixel(dimLight))
	band := frame.New(pair, whole)
	band.Set(none, none, greyPixel(half))
	img := colour.False([]colour.Layer{visible(dim, none),
		colour.NewLayer(longBand, band, none, maths.Light{})})
	want := maths.Grey(dimLight).Add(long.Scale(half))
	got := img.RGBAAt(none, none)
	if got.R != colour.Encode(want.R) || got.G != colour.Encode(want.G) ||
		got.B != colour.Encode(want.B) {
		t.Errorf("the composite is %v, want %v", got, want)
	}
	if empty := img.RGBAAt(whole, none); empty != (color.RGBA{A: byteMost}) {
		t.Errorf("an empty pixel is %v", empty)
	}
}

// anti-tests: layers of nothing are a picture of nothing. It has no size.
// It is not a panic on the first layer that is not there.
func TestNoLayersIsNoPicture(t *testing.T) {
	tone := colour.NewTone(pair, none, none, none)
	for name, img := range map[string]*image.RGBA{
		"false":      colour.False(nil),
		"tone false": tone.False([]colour.Layer{}),
	} {
		if img == nil || img.Bounds() != image.Rect(none, none, none, none) ||
			len(img.Pix) != none {
			t.Errorf("%s of no layers is %v", name, img)
		}
	}
}

// falseOfWidthsApart is a false colour picture of two layers a width
// apart, for a call that panics.
func falseOfWidthsApart() {
	square, wide := frame.New(pair, pair), frame.New(pair+whole, pair)
	colour.False([]colour.Layer{visible(square, none), visible(wide, none)})
}

// falseOfHeightsApart is a false colour picture of two layers a height
// apart, for a call that panics.
func falseOfHeightsApart() {
	square, tall := frame.New(pair, pair), frame.New(pair, pair+whole)
	colour.False([]colour.Layer{visible(square, none), visible(tall, none)})
}

// anti-tests: layers are one size, in width and in height. Either alone
// is enough to refuse one, by name.
func TestALayerOfAnotherWidthOrHeightIsRefused(t *testing.T) {
	if said := refusalOf(falseOfWidthsApart); !strings.Contains(said,
		"layer 1 is 3 by 2") {
		t.Errorf("a layer a width apart: %q", said)
	}
	if said := refusalOf(falseOfHeightsApart); !strings.Contains(said,
		"layer 1 is 2 by 3") {
		t.Errorf("a layer a height apart: %q", said)
	}
}

// Every pixel is its own place in a picture of several rows. Of a three
// by two, the pixel at the end of the second row is a half and the one
// in the middle of the first a quarter. The rest are dark.
func TestTheLayersPixelsKeepTheirPlaces(t *testing.T) {
	seen := frame.New(pair+whole, pair)
	seen.Set(pair, whole, greyPixel(half))
	seen.Set(whole, none, greyPixel(half*half))
	img := colour.False([]colour.Layer{visible(seen, none)})
	if img.Bounds() != image.Rect(none, none, pair+whole, pair) {
		t.Fatalf("a layer made a picture %v", img.Bounds())
	}
	dark := color.RGBA{A: byteMost}
	light := colour.Encode(half)
	quarter := colour.Encode(half * half)
	for y := range pair {
		for x := range pair + whole {
			want := dark
			switch {
			case x == pair && y == whole:
				want = color.RGBA{light, light, light, byteMost}
			case x == whole && y == none:
				want = color.RGBA{quarter, quarter, quarter, byteMost}
			}
			if got := img.RGBAAt(x, y); got != want {
				t.Errorf("(%d, %d) is %v, want %v", x, y, got, want)
			}
		}
	}
}

// Bands stand for a colour that no lamp has to be less than dark for. It is as
// bright as white by BT.709 luminance at any wavelength of any range. The
// violet end of gamma needs a negative green from the lamps in the arithmetic
// of libtheme. Stands gives it zero instead.
func TestAStandingColourIsNeverBelowNoughtAndAsBrightAsWhite(t *testing.T) {
	for _, profile := range []radiation.Range{radiation.Gamma(),
		radiation.XRay(), radiation.ExtremeUltraviolet(),
		radiation.Ultraviolet(), radiation.Infrared(), radiation.Radio(),
		radiation.Microwave(), radiation.Fermi()} {
		low, high := profile.Edges()
		for step := range standsSamples {
			share := float64(step) / float64(standsSamples)
			band := maths.Band{Nanometres: low * math.Pow(high/low, share)}
			light := colour.Stands(band)
			if light.R < none || light.G < none || light.B < none {
				t.Errorf("%v at %g nm stands for %v, less than dark",
					profile, band.Nanometres, light)
			}
			if math.Abs(light.Luminance()-whole) > within {
				t.Errorf("%v at %g nm stands at luminance %v", profile,
					band.Nanometres, light.Luminance())
			}
		}
	}
	gamma, _ := radiation.Gamma().Edges()
	violet := colour.Stands(maths.Band{Nanometres: gamma})
	if violet.G != none || violet.R <= none || violet.B <= violet.R {
		t.Errorf("the violet end of gamma stands for %v", violet)
	}
}

// Ranges hold their lowest wavelength and not their highest.
// Wavelengths in no range are white.
//
// The lowest wavelength of every range lands on the first row of the
// observer. So the lowest of gamma and of x-ray are one violet. X-ray
// starts where gamma ends. So the end of gamma is not gamma. The end of
// ultraviolet is visible light and white. So is anything between
// ultraviolet and infrared. The lowest of fermi is fermi alone. Fermi is
// an instrument and not a range.
func TestARangeHoldsItsLowEndAndNotItsHighEnd(t *testing.T) {
	gamma, _ := radiation.Gamma().Edges()
	lowest, _ := radiation.XRay().Edges()
	_, ultraviolet := radiation.Ultraviolet().Edges()
	violet := colour.Stands(maths.Band{Nanometres: gamma})
	if violet == maths.Grey(whole) {
		t.Fatal("the low end of gamma is white")
	}
	xray := maths.Band{Nanometres: lowest}
	if got := colour.Stands(xray); got != violet {
		t.Errorf("x-ray's lowest wavelength stands for %v, gamma's for %v",
			got, violet)
	}
	ends := maths.Band{Nanometres: ultraviolet}
	if got := colour.Stands(ends); got != maths.Grey(whole) {
		t.Errorf("the end of ultraviolet stands for %v, want white", got)
	}
	for _, row := range rows(t, standsWhiteFile) {
		band := maths.Band{Nanometres: amount(t, row, "nanometres")}
		if got := colour.Stands(band); got != maths.Grey(whole) {
			t.Errorf("%s: %g nm stands for %v, want white",
				text(t, row, "label"), band.Nanometres, got)
		}
	}
}

// Stands scales a colour by one over its luminance. It would not survive
// a colour with none. Every whole nanometre of the rows of the observer
// has some light in at least one lamp. Those rows are all that the
// colours of a range can land on.
func TestEveryRowTheRangesLandOnHasLight(t *testing.T) {
	low, high := swatch.Observed()
	for row := low; row <= high; row++ {
		red, green, blue := srgb.Light(swatch.Monochrome(row))
		light := maths.RGB(max(red, none), max(green, none), max(blue, none))
		if !(light.Luminance() > none) {
			t.Errorf("row %d has no light: %v %v %v", row, red, green, blue)
		}
	}
}

// anti-tests: light without bound through a knee is white. It is not the black
// that zero times infinity made of it. Parts that are no number are black by
// rule.
func TestEndlessLightIsWhiteAndNoNumberIsBlack(t *testing.T) {
	knee := colour.NewTone(none, kneeDrawn, none, none)
	inf := maths.RGB(math.Inf(whole), half, half)
	endless := frameOf(inf)
	if got := knee.Develop(endless).RGBAAt(none, none); got !=
		(color.RGBA{byteMost, byteMost, byteMost, byteMost}) {
		t.Errorf("endless red through the knee is %v", got)
	}
	nan := maths.RGB(math.NaN(), half, half)
	lost := frameOf(nan)
	if got := colour.Develop(lost, whole).RGBAAt(none, none); got.R != none {
		t.Errorf("a part that is no number is %v", got)
	}
}
