package colour_test

import (
	"image/color"
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/colour"
	"github.com/janearc/slayer/ninjafilm/frame"
	"pgregory.net/rapid"
)

// anti-tests: visible layers are not read by their red alone. Layers of
// only green light have green in the picture.
func TestAVisibleLayerIsNotReadByItsRedAlone(t *testing.T) {
	seen := frameOf(maths.RGB(none, half, none))
	got := colour.False([]colour.Layer{visible(seen, none)}).RGBAAt(none,
		none)
	if got.G == none || got.R != none {
		t.Errorf("a layer of green came out %v", got)
	}
}

// anti-tests: the layer of a band does not read green or blue. Pixels
// with only those on a band's layer are black.
func TestTheLayerOfABandDoesNotReadGreenOrBlue(t *testing.T) {
	seen := frameOf(maths.RGB(none, whole, whole))
	layer := colour.NewLayer(longBand, seen, none, maths.Light{})
	got := colour.False([]colour.Layer{layer}).RGBAAt(none, none)
	if got != (color.RGBA{A: byteMost}) {
		t.Errorf("green and blue on a band's layer came out %v", got)
	}
}

// anti-tests: the tone is not applied to each layer. Two eighths at twice
// the exposure are a half, not two bytes of one eighth.
func TestTheToneIsNotAppliedToEachLayerBeforeTheyAdd(t *testing.T) {
	eighth := frameOf(maths.Grey(half * half * half))
	two := []colour.Layer{visible(eighth, none), visible(eighth, none)}
	got := colour.False(two).RGBAAt(none, none).R
	alone := colour.False(two[:whole]).RGBAAt(none, none).R
	if got == alone+alone || got != colour.Encode(half*half) {
		t.Errorf("two eighths came out %d, one came out %d", got, alone)
	}
}

// anti-tests: tints are not the colour of the range. Tinted layers come
// out other than the same layers untinted.
func TestATintedLayerIsNotTheColourOfItsRange(t *testing.T) {
	seen := frameOf(maths.RGB(whole, none, none))
	plain := colour.NewLayer(halpha, seen, none, maths.Light{})
	green := colour.NewLayer(halpha, seen, none, maths.RGB(none, whole, none))
	first := colour.False([]colour.Layer{plain}).RGBAAt(none, none)
	second := colour.False([]colour.Layer{green}).RGBAAt(none, none)
	if first == second {
		t.Errorf("a tint changed nothing: %v", first)
	}
}

// anti-tests: a layer left out changes the picture. The composite with
// the band's layer is not the composite without it.
func TestALayerLeftOutChangesThePicture(t *testing.T) {
	dim := frameOf(maths.Grey(dimLight))
	bright := frameOf(maths.Grey(half))
	with := colour.False([]colour.Layer{visible(dim, none),
		colour.NewLayer(longBand, bright, none, maths.Light{})})
	without := colour.False([]colour.Layer{visible(dim, none)})
	if with.RGBAAt(none, none) == without.RGBAAt(none, none) {
		t.Errorf("a band's layer changed nothing")
	}
}

// anti-tests: a pixel is not copied to another place. The picture of a
// pixel in one place differs from the picture of it in another.
func TestAPixelInOnePlaceIsNotAPixelInAnother(t *testing.T) {
	rapid.Check(t, pixelsKeepTheirPlace)
}

func pixelsKeepTheirPlace(t *rapid.T) {
	across := rapid.IntRange(pair, pictureMost).Draw(t, "across")
	first := frame.New(across, whole)
	second := frame.New(across, whole)
	first.Set(none, none, greyPixel(half))
	second.Set(across-whole, none, greyPixel(half))
	one := colour.False([]colour.Layer{visible(first, none)})
	other := colour.False([]colour.Layer{visible(second, none)})
	if string(one.Pix) == string(other.Pix) {
		t.Fatalf("a pixel at either end of %d came out one picture", across)
	}
}

// anti-tests: bands in other ranges do not stand for one colour. The
// short end of gamma and a radio band differ.
func TestBandsInOtherRangesDoNotStandForOneColour(t *testing.T) {
	gammaShort, _ := radiation.Gamma().Edges()
	radioShort, _ := radiation.Radio().Edges()
	gamma := maths.Band{Nanometres: gammaShort}
	radio := maths.Band{Nanometres: radioShort}
	if colour.Stands(gamma) == colour.Stands(radio) {
		t.Errorf("gamma and radio stand for %v", colour.Stands(gamma))
	}
}

// anti-tests: a wavelength just under the top of a range is not white
// when the range is not visible. Ultraviolet is not white inside it.
func TestAWavelengthInsideARangeIsNotWhite(t *testing.T) {
	_, ultraviolet := radiation.Ultraviolet().Edges()
	inside := maths.Band{Nanometres: ultraviolet - within}
	if colour.Stands(inside) == maths.Grey(whole) {
		t.Errorf("a wavelength inside ultraviolet stands for white")
	}
}

// anti-tests: the middle of no range is white by accident. Every named
// range but fermi has a colour of its own at its middle.
func TestTheMiddleOfEveryRangeHasAColourOfItsOwn(t *testing.T) {
	for _, span := range []radiation.Range{radiation.Gamma(),
		radiation.XRay(), radiation.ExtremeUltraviolet(),
		radiation.Ultraviolet(), radiation.Infrared(), radiation.Radio(),
		radiation.Microwave()} {
		low, high := span.Edges()
		band := maths.Band{Nanometres: (low + high) * half}
		if colour.Stands(band) == maths.Grey(whole) {
			t.Errorf("the middle of %v stands for white", span)
		}
	}
}
