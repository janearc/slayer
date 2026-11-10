package flame

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/observer"
	"math"
	"testing"
)

func TestBodyUsesItsCompositionAndReusesOnlyTheMatchingPalette(t *testing.T) {
	spec := lineSpec()
	spec.Continuum = .4
	profile, err := NewComposition(spec)
	if err != nil {
		t.Fatal(err)
	}
	fire := campfire
	fire.Composition = profile
	fire.Hottest = profile.Reference()
	palettes := map[paletteKey]glowing{}
	first, err := bodyWithPalette(fire, 1, palettes)
	if err != nil {
		t.Fatal(err)
	}
	second, err := bodyWithPalette(fire, 1, palettes)
	if err != nil {
		t.Fatal(err)
	}
	if len(palettes) != 1 || &first.glows.glows[0] != &second.glows.glows[0] {
		t.Fatal("identical fire patches rebuilt their expensive spectrum table")
	}
	_, err = bodyWithPalette(fire, 2, palettes)
	if err != nil {
		t.Fatal(err)
	}
	if len(palettes) != 2 {
		t.Fatal("different exposures shared a palette")
	}
	// Endpoints are exact table knots; interior comparison keeps the existing
	// interpolation tolerance rather than asserting a second copy of its index.
	for _, heat := range []measure.Temperature{fire.Edge, fire.Hottest} {
		spectrum := profile.Spectrum(heat)
		r, g, b := srgb.Light(observer.Swatch(spectrum))
		want := maths.RGB(math.Max(0, r), math.Max(0, g), math.Max(0, b))
		got := first.glows.at(heat)
		delta := math.Abs(got.R-want.R) + math.Abs(got.G-want.G) + math.Abs(got.B-want.B)
		if !(delta <= 1e-12*(want.R+want.G+want.B)) {
			t.Fatalf("body ignored its composition: %v want %v", got, want)
		}
	}
}

// Samples is the CPU-to-GPU transfer contract: its 256 entries span the
// fire's edge and peak temperatures uniformly. Read the spectrum directly,
// independently of the CPU LUT interpolator that the GPU does not receive.
func TestExportedCompositionSamplesFollowTheirTemperatureCoordinates(t *testing.T) {
	spec := lineSpec()
	spec.Continuum = .4
	profile, err := NewComposition(spec)
	if err != nil {
		t.Fatal(err)
	}
	fire := campfire
	fire.Composition = profile
	fire.Hottest = profile.Reference()
	body, err := NewBody(fire, 1)
	if err != nil {
		t.Fatal(err)
	}
	description, _, _, samples := body.Samples()
	if len(samples) != 256 {
		t.Fatal("GPU palette no longer has 256 samples")
	}
	cold, hot := description.Edge.InKelvin(), description.Hottest.InKelvin()
	for _, index := range []int{0, 127, 254, 255} {
		fraction := float64(index) / float64(len(samples)-1)
		heat := measure.Kelvin(cold*(1-fraction) + hot*fraction)
		r, g, b := srgb.Light(observer.Swatch(profile.Spectrum(heat)))
		want := maths.RGB(math.Max(0, r), math.Max(0, g), math.Max(0, b))
		got := samples[index]
		delta := math.Abs(got.R-want.R) + math.Abs(got.G-want.G) + math.Abs(got.B-want.B)
		if !(delta <= 1e-11*(want.R+want.G+want.B)) {
			t.Fatalf("GPU sample %d: %v want direct spectrum %v", index, got, want)
		}
	}
}
