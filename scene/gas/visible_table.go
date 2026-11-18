package gas

import (
	"fmt"
	"math"
	"sort"

	"github.com/janearc/slayer/maths"
)

// VisibleSample is caller-supplied visible-light optics at one wavelength.
// Absorb and Scatter are extinction coefficients in inverse metres at unit
// density. The values are inputs, not a water preset or a claim about any
// particular water composition.
type VisibleSample struct {
	Nanometres float64
	Absorb     float64
	Scatter    float64
}

// VisibleTable is an owned, validated sampled visible spectrum. Between two
// supplied wavelengths the absorption and scattering coefficients are
// interpolated linearly in wavelength. The table does not extrapolate.
//
// The scene tracer's RGB eye channels sample this spectrum at 610, 550 and
// 465 nm for red, green and blue respectively. This is an RGB approximation,
// not a full spectral integration.
//
// Phase uses the existing Gas convention: zero selects Rayleigh scattering,
// not the isotropic Henyey–Greenstein lobe at g=0. This adapter therefore does
// not express isotropic Henyey–Greenstein scattering.
type VisibleTable struct {
	samples []VisibleSample
	phase   float64
}

// NewVisibleTable checks and copies wavelength samples and a scattering phase
// asymmetry. At least two samples are needed to define an interpolation
// interval. Wavelengths must be finite, positive and strictly increasing;
// coefficients must be finite and nonnegative; phase must be strictly between
// -1 and 1. Phase zero uses the existing Rayleigh phase function, not the
// isotropic Henyey–Greenstein lobe at g=0.
func NewVisibleTable(samples []VisibleSample, phase float64) (VisibleTable, error) {
	if len(samples) < 2 {
		return VisibleTable{}, fmt.Errorf("gas: visible table needs at least two samples")
	}
	if !finite(phase) || math.Abs(phase) >= 1 {
		return VisibleTable{}, fmt.Errorf("gas: visible phase must be finite and strictly between -1 and 1")
	}
	owned := append([]VisibleSample(nil), samples...)
	for index, sample := range owned {
		if !finite(sample.Nanometres) || sample.Nanometres <= 0 {
			return VisibleTable{}, fmt.Errorf("gas: visible sample %d has invalid wavelength %g nm", index, sample.Nanometres)
		}
		if !finite(sample.Absorb) || sample.Absorb < 0 {
			return VisibleTable{}, fmt.Errorf("gas: visible sample %d has invalid absorption %g per metre", index, sample.Absorb)
		}
		if !finite(sample.Scatter) || sample.Scatter < 0 {
			return VisibleTable{}, fmt.Errorf("gas: visible sample %d has invalid scattering %g per metre", index, sample.Scatter)
		}
		if !finite(sample.Absorb + sample.Scatter) {
			return VisibleTable{}, fmt.Errorf("gas: visible sample %d has unrepresentable total extinction", index)
		}
		if index > 0 && sample.Nanometres <= owned[index-1].Nanometres {
			return VisibleTable{}, fmt.Errorf("gas: visible sample wavelengths must be strictly increasing")
		}
	}
	return VisibleTable{samples: owned, phase: phase}, nil
}

// Samples returns a copy of the table's wavelength samples.
func (table VisibleTable) Samples() []VisibleSample {
	return append([]VisibleSample(nil), table.samples...)
}

// At returns linearly interpolated absorption and scattering coefficients at
// wavelength nm. It refuses nonfinite or out-of-table wavelengths instead of
// extrapolating.
func (table VisibleTable) At(nm float64) (Optics, error) {
	if !finite(nm) || nm <= 0 {
		return Optics{}, fmt.Errorf("gas: invalid visible wavelength %g nm", nm)
	}
	if len(table.samples) < 2 {
		return Optics{}, fmt.Errorf("gas: empty visible table")
	}
	if nm < table.samples[0].Nanometres || nm > table.samples[len(table.samples)-1].Nanometres {
		return Optics{}, fmt.Errorf("gas: wavelength %g nm is outside visible table [%g, %g] nm",
			nm, table.samples[0].Nanometres, table.samples[len(table.samples)-1].Nanometres)
	}
	index := sort.Search(len(table.samples), func(index int) bool {
		return table.samples[index].Nanometres >= nm
	})
	if table.samples[index].Nanometres == nm {
		sample := table.samples[index]
		optics := Optics{Absorb: sample.Absorb, Scatter: sample.Scatter}
		if !finite(optics.Absorb + optics.Scatter) {
			return Optics{}, fmt.Errorf("gas: unrepresentable total extinction at %g nm", nm)
		}
		return optics, nil
	}
	lo, hi := table.samples[index-1], table.samples[index]
	fraction := (nm - lo.Nanometres) / (hi.Nanometres - lo.Nanometres)
	optics := Optics{
		Absorb:  interpolate(lo.Absorb, hi.Absorb, fraction),
		Scatter: interpolate(lo.Scatter, hi.Scatter, fraction),
	}
	if !finite(optics.Absorb + optics.Scatter) {
		return Optics{}, fmt.Errorf("gas: unrepresentable total extinction at %g nm", nm)
	}
	return optics, nil
}

// Transmit is the homogeneous Beer-Lambert transmittance over distance metres.
// Absorption and scattering both remove light from the unscattered beam. A
// positive infinite path has zero transmittance for any nonzero extinction and
// unit transmittance for zero extinction. Negative and NaN distances are
// refused. This is path attenuation only; it does not integrate in-scattered
// light.
func (table VisibleTable) Transmit(nm, distance float64) (float64, error) {
	optics, err := table.At(nm)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(distance) || distance < 0 || math.IsInf(distance, -1) {
		return 0, fmt.Errorf("gas: invalid path distance %g m", distance)
	}
	if math.IsInf(distance, 1) {
		if optics.Absorb == 0 && optics.Scatter == 0 {
			return 1, nil
		}
		return 0, nil
	}
	return table.Gas("").Transmit(maths.Band{Nanometres: nm}, distance), nil
}

// SingleScatter integrates one homogeneous slab's in-scattered source toward
// the observer. The directional incident light is assumed constant and
// unattenuated throughout the slab (or already evaluated at each point with
// the same value); extinction attenuates only the outgoing path to the eye.
// Coefficients and phase are constant, density is one, and there is no multiple
// scattering. This is a closed-form source term, not a scene transport path.
func (table VisibleTable) SingleScatter(nm, distance float64,
	incident maths.Light, cosine float64) (maths.Light, error) {
	optics, err := table.At(nm)
	if err != nil {
		return maths.Light{}, err
	}
	if math.IsNaN(distance) || distance < 0 || math.IsInf(distance, -1) {
		return maths.Light{}, fmt.Errorf("gas: invalid path distance %g m", distance)
	}
	if !finite(cosine) || cosine < -1 || cosine > 1 {
		return maths.Light{}, fmt.Errorf("gas: invalid scattering cosine %g", cosine)
	}
	if !finite(incident.R) || !finite(incident.G) || !finite(incident.B) ||
		incident.R < 0 || incident.G < 0 || incident.B < 0 {
		return maths.Light{}, fmt.Errorf("gas: incident light must be finite and nonnegative")
	}
	if distance == 0 || optics.Scatter == 0 {
		return maths.Light{}, nil
	}
	extinction := optics.Absorb + optics.Scatter
	if extinction == 0 {
		return maths.Light{}, nil
	}
	depth := extinction * distance
	length := 1 / extinction
	if !math.IsInf(distance, 1) {
		length = -math.Expm1(-depth) / extinction
	}
	gas := table.Gas("")
	source := gas.Inscatter(maths.Band{Nanometres: nm}, incident, cosine)
	if source.R == 0 && source.G == 0 && source.B == 0 {
		return maths.Light{}, nil
	}
	result := source.Scale(length)
	if !finite(result.R) || !finite(result.G) || !finite(result.B) {
		return maths.Light{}, fmt.Errorf("gas: homogeneous single-scatter result overflows")
	}
	return result, nil
}

// Gas adapts the validated table to the existing gas optics API. Samples
// outside the table's interval are opaque, matching Gas's existing behavior
// for undeclared bands. Use At or Transmit when the caller needs an explicit
// error for an out-of-range wavelength or invalid distance.
func (table VisibleTable) Gas(name string) Gas {
	samples := table.Samples()
	return Gas{Name: name, Phase: table.phase, Visible: func(nm float64) (absorb, scatter float64) {
		owned := VisibleTable{samples: samples, phase: table.phase}
		optics, err := owned.At(nm)
		if err != nil {
			return math.Inf(1), 0
		}
		return optics.Absorb, optics.Scatter
	}}
}

// interpolate blends a and b by fraction, zero giving a and one giving b.
func interpolate(a, b, fraction float64) float64 {
	if fraction == 0 {
		return a
	}
	if fraction == 1 {
		return b
	}
	return a*(1-fraction) + b*fraction
}

// finite reports whether value is a real number, not NaN or infinite.
func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
