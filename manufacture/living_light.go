// Living light factories supply an emission spectrum and a flash envelope.
// The caller supplies organism-specific measurements and excitation.
package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/observer"
)

// LivingLightSpec describes an approximate Gaussian emission band and a
// single flash. Wavelength and FWHM are in nanometres, duration in seconds,
// and luminance is a relative renderer value. Source identifies the evidence.
type LivingLightSpec struct {
	PeakNanometres float64
	FWHMNanometres float64
	Luminance      float64
	FlashSeconds   float64
	Source         string
}

// LivingLight is a reusable light-producing organism's emission response.
// Its spectrum and timing are independent of the stimulus and geometry.
type LivingLight struct {
	spec   LivingLightSpec
	colour maths.Light
}

// DinoflagellateLight describes an approximate blue emission band and a
// 100 ms flash for mechanically stimulated L. polyedra. The 50 nm FWHM is
// an illustrative width rather than a measured species-specific curve.
// https://pmc.ncbi.nlm.nih.gov/articles/PMC11177854/
// https://scripps.ucsd.edu/news/red-tide-brings-bioluminescence-san-diego-beaches
func DinoflagellateLight() LivingLightSpec {
	return LivingLightSpec{PeakNanometres: 475, FWHMNanometres: 50,
		Luminance: 1, FlashSeconds: .1,
		Source: "https://pmc.ncbi.nlm.nih.gov/articles/PMC11177854/"}
}

// NewLivingLight makes an emission from an approximate spectral band.
// The conversion to the renderer's linear RGB clips negative out-of-gamut
// components and normalizes to relative luminance. It is not a spectral
// light-transport calculation.
func NewLivingLight(spec LivingLightSpec) (LivingLight, error) {
	if math.IsNaN(spec.PeakNanometres) || math.IsInf(spec.PeakNanometres, 0) ||
		math.IsNaN(spec.FWHMNanometres) || math.IsInf(spec.FWHMNanometres, 0) ||
		math.IsNaN(spec.Luminance) || math.IsInf(spec.Luminance, 0) ||
		math.IsNaN(spec.FlashSeconds) || math.IsInf(spec.FlashSeconds, 0) ||
		spec.PeakNanometres < 380 || spec.PeakNanometres > 780 ||
		spec.FWHMNanometres < 1 || spec.FWHMNanometres > 300 ||
		spec.Luminance < 0 || spec.FlashSeconds <= 0 {
		return LivingLight{}, fmt.Errorf("manufacture: living light needs a visible peak, positive band width and flash duration, and nonnegative luminance")
	}
	lines := make(map[float64]float64, 401)
	for nm := 380; nm <= 780; nm++ {
		lines[float64(nm)] = livingLightSpectrum(spec, float64(nm))
	}
	red, green, blue := srgb.Light(observer.Swatch(bands.Lines(lines)))
	colour := maths.RGB(math.Max(0, red), math.Max(0, green), math.Max(0, blue))
	if colour.Luminance() <= 0 {
		return LivingLight{}, fmt.Errorf("manufacture: living light band has no visible colour")
	}
	return LivingLight{spec: spec, colour: colour.Scale(spec.Luminance / colour.Luminance())}, nil
}

// Colour returns the constant relative emission at the peak of a flash.
func (light LivingLight) Colour() maths.Light { return light.colour }

// livingLightSpectrum gives the uncalibrated relative emission at a
// wavelength in nanometres. The Gaussian peaks at one; it is an approximate
// band shape.
func livingLightSpectrum(spec LivingLightSpec, nanometres float64) float64 {
	if math.IsNaN(nanometres) || math.IsInf(nanometres, 0) || spec.FWHMNanometres <= 0 {
		return 0
	}
	sigma := spec.FWHMNanometres / (2 * math.Sqrt(2*math.Ln2))
	difference := (nanometres - spec.PeakNanometres) / sigma
	return math.Exp(-.5 * difference * difference)
}

// Flash returns a smooth illustrative single-flash envelope after a stimulus.
// Biological flash shapes differ between organisms and conditions.
func (light LivingLight) Flash(secondsSinceStimulus float64) maths.Light {
	if math.IsNaN(secondsSinceStimulus) || secondsSinceStimulus <= 0 ||
		secondsSinceStimulus >= light.spec.FlashSeconds {
		return maths.Light{}
	}
	return light.colour.Scale(math.Sin(math.Pi * secondsSinceStimulus / light.spec.FlashSeconds))
}

// Spec returns the original spectral, timing and source parameters.
func (light LivingLight) Spec() LivingLightSpec { return light.spec }
