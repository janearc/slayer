package airs

import (
	"github.com/janearc/slayer/mie"
)

// Units the data's sizes are in.
const (
	// metresPerMicrometre turns a mode's diameter into metres.
	metresPerMicrometre = 1e-6 // defined: the unit
	// radiusOverDiameter turns a diameter into a radius.
	radiusOverDiameter = 0.5 // defined: a radius is half a diameter
)

// optics is what a layer's particles do to light of one wavelength: the
// extinction a metre, the single-scattering albedo, and the asymmetry.
type optics struct {
	extinction float64
	albedo     float64
	asymmetry  float64
}

// mixture is a layer's modes at one wavelength, added up: the number
// density times the mean extinction cross section, the same times the
// albedo (so, the scattering), and that times the asymmetry. The number
// density is Table 1's, in a cubic centimetre, and so the sums are in
// relative units: only their ratios, and their ratio from one
// wavelength to another, are used.
type mixture struct {
	extinction float64
	scattering float64
	product    float64
}

// mix is the layer's modes at a wavelength in nanometres, each a
// log-normal population of spheres of the data's index.
func (read data) mix(layer layerData, nanometres float64) (mixture, error) {
	index := complex(read.Particles.Real.Value, read.Particles.Imaginary.Value)
	wavelength := nanometres * metresPerNanometre
	var total mixture
	for _, mode := range layer.Modes {
		population, err := mie.NewPopulation(mode.Diameter*radiusOverDiameter*
			metresPerMicrometre, mode.Width.Value)
		if err != nil {
			return mixture{}, err
		}
		average, err := population.Average(wavelength, index)
		if err != nil {
			return mixture{}, err
		}
		// Each term is rounded by a conversion, so that no product is
		// fused into its sum and scattering never sums to more than
		// extinction: an albedo of one and a rounding.
		extinction := float64(mode.Number * average.Extinction)
		scattering := float64(extinction * average.Albedo)
		total.extinction += extinction
		total.scattering += scattering
		total.product += float64(scattering * average.Asymmetry)
	}
	return total, nil
}

// particles are a layer's optics at a wavelength in nanometres. The
// layer's optical depth is the data's at the reference wavelength, held
// spread evenly through its height; at another wavelength the
// extinction a metre is that times the mixture's own change, which is
// Mie's. The albedo is the extinction-weighted mean of the modes', and
// the asymmetry the scattering-weighted: the same while no mode absorbs.
func (read data) particles(layer layerData, nanometres float64) (optics, error) {
	here, err := read.mix(layer, nanometres)
	if err != nil {
		return optics{}, err
	}
	reference, err := read.mix(layer, read.Particles.Reference.Value)
	if err != nil {
		return optics{}, err
	}
	thickness := layer.Top - layer.Bottom
	return optics{
		extinction: layer.Depth.Value / thickness * here.extinction /
			reference.extinction,
		albedo:    here.scattering / here.extinction,
		asymmetry: here.product / here.scattering,
	}, nil
}

// absorbed is the optics with an absorption a metre added: the extinction
// grows by it and the albedo falls to what the scattering is of the new
// extinction. An absorber of nought is no change.
func (particles optics) absorbed(extra float64) optics {
	if extra == 0 {
		return particles
	}
	total := particles.extinction + extra
	return optics{extinction: total,
		albedo:    particles.extinction * particles.albedo / total,
		asymmetry: particles.asymmetry}
}
