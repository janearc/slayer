// Package fibre evaluates single-fibre scattering for a circular dielectric
// shaft. It supplies per-path angular densities for a transport integrator.
package fibre

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// Optics describes a fibre surrounded by air. Absorption is RGB optical
// depth across one diameter. Longitudinal and Azimuthal are unitless
// roughness controls. Cuticle is the scale tilt in radians. PrimaryWidth
// scales the longitudinal width of the R lobe, with zero read as one.
type Optics struct {
	Index                   float64
	Absorption              maths.Light
	Longitudinal, Azimuthal float64
	Cuticle, PrimaryWidth   float64
}

// Model holds the optical properties and lobe widths of a shaft.
type Model struct {
	index      float64
	absorption maths.Light
	kappa      [4]float64
	tilt       [4]float64
	logistic   float64
}

// New prepares a near-field fibre model. Roughness controls run from .05
// to 1, index from 1.01 to 3, cuticle tilt from 0 to 10 degrees, and primary
// width from .5 to 2. These bounds define the supported numeric domain.
func New(spec Optics) (*Model, error) {
	if !between(spec.Index, 1.01, 3) {
		return nil, fmt.Errorf("fibre: index outside [1.01, 3]")
	}
	if !between(spec.Longitudinal, .05, 1) || !between(spec.Azimuthal, .05, 1) {
		return nil, fmt.Errorf("fibre: roughness outside [.05, 1]")
	}
	if !between(spec.Cuticle, 0, math.Pi/18) {
		return nil, fmt.Errorf("fibre: cuticle tilt outside [0, 10] degrees")
	}
	primary := spec.PrimaryWidth
	if primary == 0 {
		primary = 1
	}
	if !between(primary, .5, 2) {
		return nil, fmt.Errorf("fibre: primary width outside [.5, 2]")
	}
	for _, channel := range [...]float64{spec.Absorption.R, spec.Absorption.G, spec.Absorption.B} {
		if !between(channel, 0, math.MaxFloat64) {
			return nil, fmt.Errorf("fibre: absorption must be finite and nonnegative")
		}
	}

	// Perceptual width mappings in Pharr et al., pbrt 4e, §9.9.
	m, n := spec.Longitudinal, spec.Azimuthal
	width := .726*m + .812*m*m + 3.7*math.Pow(m, 20)
	concentration := 1 / (width * width)
	scale := math.Sqrt(math.Pi/8) * (.265*n + 1.194*n*n + 5.372*math.Pow(n, 22))
	tilt := spec.Cuticle
	return &Model{
		index: spec.Index, absorption: spec.Absorption,
		kappa:    [4]float64{concentration / (primary * primary), 4 * concentration, .25 * concentration, .25 * concentration},
		tilt:     [4]float64{-2 * tilt, tilt, 4 * tilt},
		logistic: scale,
	}, nil
}

// Defined reports whether the model was prepared from optical parameters.
func (model *Model) Defined() bool { return model != nil && model.index != 0 }

// View fixes the outgoing direction toward the camera. Elevation is the
// angle from the normal plane toward the root-to-tip tangent. Offset
// is the sine of the signed angle from the projected outward radial
// direction to the projected view direction, about the tipward tangent.
type View struct {
	Elevation, Offset float64
}

// Light gives the direction from the fibre toward the source. Elevation
// uses the same convention as View and Azimuth is relative to the
// projected view direction, in radians about the tipward tangent.
type Light struct {
	Elevation, Azimuth float64
}

// Paths separates cuticle reflection (R), direct transmission (TT), and
// transmission after an internal reflection (TRT). Higher sums the paths
// with two or more internal reflections.
type Paths struct {
	R, TT, TRT, Higher maths.Light
}

type lobe struct {
	energy       maths.Light
	azimuth      float64
	sine, cosine float64
}

// Event caches the scattering for one outgoing ray and fibre offset.
type Event struct {
	lobes [4]lobe
	model *Model
}

// At prepares the scattering at one near-field fibre hit.
func (model *Model) At(view View) (Event, error) {
	if !model.Defined() {
		return Event{}, fmt.Errorf("fibre: model must be made with New")
	}
	if !between(view.Elevation, -math.Pi/2, math.Pi/2) || !between(view.Offset, -1, 1) {
		return Event{}, fmt.Errorf("fibre: view lies outside fibre coordinates")
	}
	sine, cosine := math.Sincos(view.Elevation)
	h := view.Offset
	gamma := math.Asin(h)
	entryCosine := cosine * complement(h)
	f := maths.Fresnel(entryCosine, model.index)
	eta := model.index
	refractedSine := h * cosine / math.Sqrt(eta*eta-sine*sine)
	refractedGamma := safeAsin(refractedSine)
	// Divide the internal chord by the diameter to express optical depth
	// in the same units as the material parameter.
	chord := complement(refractedSine) / complement(sine/eta)
	energy := pathEnergy(f, chord, model.absorption)

	event := Event{model: model}
	for p := range event.lobes {
		s, c := math.Sincos(view.Elevation + model.tilt[p])
		// The angular distribution depends on the magnitude of the
		// projection after the cuticle tilt.
		phi := float64(p)*(2*refractedGamma+math.Pi) - 2*gamma
		event.lobes[p] = lobe{energy[p], phi, s, math.Abs(c)}
	}
	return event, nil
}

// Energy gives the integrated per-channel fractions for this view. A
// nonabsorbing fibre returns the incident energy across the four paths.
func (event Event) Energy() Paths {
	return Paths{event.lobes[0].energy, event.lobes[1].energy, event.lobes[2].energy, event.lobes[3].energy}
}

// Evaluate gives each path's scattering density per steradian in the
// supplied direction. It includes the angular measure normally supplied
// by the BSDF's incident cosine. A caller integrates it over solid angle.
func (event Event) Evaluate(light Light) (Paths, error) {
	if event.model == nil {
		return Paths{}, fmt.Errorf("fibre: event must be made with At")
	}
	if !between(light.Elevation, -math.Pi/2, math.Pi/2) ||
		!between(light.Azimuth, -math.MaxFloat64, math.MaxFloat64) {
		return Paths{}, fmt.Errorf("fibre: light lies outside fibre coordinates")
	}
	s, c := math.Sincos(light.Elevation)
	azimuth := math.Remainder(light.Azimuth, 2*math.Pi)
	var density [4]maths.Light
	for p, lobe := range event.lobes {
		if lobe.energy == (maths.Light{}) {
			continue
		}
		across := 1 / (2 * math.Pi)
		if p < 3 {
			across = circularLogistic(azimuth-lobe.azimuth, event.model.logistic)
		}
		along := longitudinal(s, c, lobe.sine, lobe.cosine, event.model.kappa[p])
		density[p] = lobe.energy.Scale(along * across)
	}
	return Paths{density[0], density[1], density[2], density[3]}, nil
}

func pathEnergy(reflectance, chord float64, absorption maths.Light) [4]maths.Light {
	f := reflectance
	a := 1 - f
	pass := maths.RGB(math.Exp(-absorption.R*chord), math.Exp(-absorption.G*chord),
		math.Exp(-absorption.B*chord))
	tt := pass.Scale(a * a)
	repeat := pass.Scale(f)
	trt := tt.Times(repeat)
	return [4]maths.Light{
		maths.Grey(f), tt, trt,
		maths.RGB(tail(trt.R, repeat.R), tail(trt.G, repeat.G), tail(trt.B, repeat.B)),
	}
}

// tail sums all the path orders after TRT channel by channel.
func tail(trt, repeat float64) float64 {
	if repeat == 1 {
		return 0 // All light already reflected at entry, so TRT is zero.
	}
	return trt * repeat / (1 - repeat)
}

func between(value, low, high float64) bool { return value >= low && value <= high }

func complement(sine float64) float64 { return math.Sqrt(max(0, 1-sine*sine)) }
