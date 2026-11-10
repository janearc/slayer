package flame

import (
	"math"
	"slices"
	"strings"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/radiance"
)

// LineSpec names one narrow spectral source. Radiance is integrated over the
// line in W/m^2/sr at Reference, not a concentration or handbook intensity.
// Wavelength retains its source convention; RGB integration uses that
// coordinate directly and does not perform an air/vacuum conversion.
type LineSpec struct {
	Species, Source, Convention string
	Wavelength                  measure.Length
	Radiance                    float64
}

// CompositionSpec supplies a prescribed continuum plus narrow emission lines.
// Continuum scales the thermal source function; it is not a soot mass fraction.
// Line strength cools by a Planck ratio at the supplied wavelength. This is an
// authored thermal response, not an excitation, population or reaction model.
type CompositionSpec struct {
	Reference measure.Temperature
	Continuum float64
	Lines     []LineSpec
}

type composition struct {
	spec      CompositionSpec
	reference []float64
}

// Composition names the immutable profile returned by NewComposition, so
// callers can retain profiles in named fields and collections.
type Composition = composition

// NewComposition validates and copies source data. A nil composition on Fire
// preserves its original thermal-soot source function.
func NewComposition(spec CompositionSpec) (*composition, error) {
	if !sane(spec.Reference.InKelvin()) ||
		!(spec.Continuum >= 0 && spec.Continuum <= 1) ||
		len(spec.Lines) > 32 || (spec.Continuum == 0 && len(spec.Lines) == 0) {
		return nil, errFlame
	}
	profile := &composition{spec: spec}
	profile.spec.Lines = slices.Clone(spec.Lines)
	for _, line := range spec.Lines {
		wavelength := line.Wavelength.InMetres()
		if !sane(wavelength) || !(line.Radiance >= 0) || math.IsInf(line.Radiance, 0) ||
			strings.TrimSpace(line.Species) == "" || strings.TrimSpace(line.Source) == "" ||
			(line.Convention != "air" && line.Convention != "vacuum") {
			return nil, errFlame
		}
		reference := radiance.Planck(spec.Reference, line.Wavelength).InWattsPerNanometre()
		if !sane(reference) {
			return nil, errFlame
		}
		profile.reference = append(profile.reference, reference)
	}
	return profile, nil
}

// Spectrum returns the source radiance, with line powers at or below their
// prescribed reference values. It does not infer abundance from a species name.
func (profile *composition) Spectrum(heat measure.Temperature) bands.Light {
	if !sane(heat.InKelvin()) {
		return bands.Light{}
	}
	heat = measure.Kelvin(math.Min(heat.InKelvin(), profile.spec.Reference.InKelvin()))
	lines := make(map[float64]float64, len(profile.spec.Lines))
	for i, line := range profile.spec.Lines {
		share := radiance.Planck(heat, line.Wavelength).InWattsPerNanometre() / profile.reference[i]
		lines[line.Wavelength.InMetres()/nanometre] += line.Radiance * share
	}
	light := bands.Lines(lines)
	if profile.spec.Continuum > 0 {
		light = bands.Add(light, Glowing(heat).Scale(profile.spec.Continuum))
	}
	return light
}

// Reference is the temperature at which the authored line powers apply.
func (profile *composition) Reference() measure.Temperature { return profile.spec.Reference }

// Lines returns copied provenance and source inputs.
func (profile *composition) Lines() []LineSpec { return slices.Clone(profile.spec.Lines) }
