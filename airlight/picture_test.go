package airlight

import (
	"math"
	"testing"

	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/moonlight"
)

// What the colour of the wavelengths' light is held to.
const (
	// equalEnergy is the chromaticity, x and y alike, of light of the
	// same power at every wavelength: the white the observer's own
	// three rows make.
	equalEnergy = 1.0 / 3 // defined: x, y and z share alike
	// whiteShare is how far from it an even spectrum may come out: a
	// tenth of the 0.01 in y that lines at the wavelengths missed by,
	// measured in FIX-REPORT.md.
	whiteShare = 1e-3 // picked: see above
	// purpleShare is how far below it lines at the wavelengths must
	// come out in y for the anti-test: half that measured 0.01.
	purpleShare = 5e-3 // picked: see above
)

// evenFields is fields at the plan's wavelengths and band, with a sun of
// one watt a square metre a nanometre at every one of them.
func evenFields() *Fields {
	waves := wavelengths()
	sun := make([]float64, len(waves))
	for wave := range sun {
		sun[wave] = one
	}
	return &Fields{wavelengths: waves, band: everyNanometres, sun: sun}
}

// placeOf is a picture's light's chromaticity, x and y.
func placeOf(light maths.Light) (float64, float64) {
	place := srgb.FromLight(light.R, light.G, light.B).XY()
	return place.X, place.Y
}

// light of the same power at each of the plan's wavelengths, each
// standing for its band, is the observer's own white in the picture,
// and holds the power of every band.
func TestAnEvenSpectrumAtTheWavelengthsIsWhite(t *testing.T) {
	fields := evenFields()
	light := fields.seen(func(int) float64 { return one })
	x, y := placeOf(light.Picture(one))
	if math.Abs(x-equalEnergy) > whiteShare ||
		math.Abs(y-equalEnergy) > whiteShare {
		t.Errorf("an even spectrum is x %.4f y %.4f, want %.4f both", x, y,
			equalEnergy)
	}
	want := float64(len(fields.wavelengths)) * everyNanometres
	if got := light.Total(); math.Abs(got-want) > handShare*want {
		t.Errorf("an even spectrum holds %g, want %g", got, want)
	}
}

// the anti-test: the same light given as one line at each wavelength,
// as the fields were seen before, comes out purple, below white in y,
// so the check above sees how the bands are handed over.
func TestLinesAtTheWavelengthsAreNotWhite(t *testing.T) {
	fields := evenFields()
	lines := make(moonlight.Spectrum, len(fields.wavelengths))
	for _, nanometres := range fields.wavelengths {
		lines[nanometres] = everyNanometres
	}
	if _, y := placeOf(lines.Picture(one)); !(y < equalEnergy-purpleShare) {
		t.Errorf("lines at the wavelengths are y %.4f, not below %.4f", y,
			equalEnergy-purpleShare)
	}
}
