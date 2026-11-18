// Package specimen accounts for a porous solid at a stated moisture content.
// It describes a physical specimen, without inferring chemistry or appearance.
package specimen

import (
	"fmt"
	"math"
	"strings"
)

// Solid is the mass of one identified solid phase. Density is intrinsic:
// pores counted separately must not also be included in this density.
// Basis records a measurement citation or an explicit modelling assumption.
// Mass includes chemically bound water where it belongs to the solid phase.
type Solid struct {
	Name           string
	MassKG         float64
	DensityKGPerM3 float64
	Basis          string
}

// Pores describes total pore volume relative to the whole specimen, and the
// fraction of that pore volume occupied by liquid water. Water density must
// be supplied at the intended conditions. Empty pores have zero mass in this
// accounting approximation. Basis records the origin of these inputs.
type Pores struct {
	VolumeFraction      float64
	Saturation          float64
	WaterDensityKGPerM3 float64
	Basis               string
}

// Phase is a solid's mass, intrinsic density and share of bulk specimen
// volume. Its volume fraction is not its mass fraction.
type Phase struct {
	Name           string
	MassKG         float64
	DensityKGPerM3 float64
	VolumeFraction float64
	Basis          string
}

// Specimen is an immutable volume balance of identified solids and pores.
// Only New constructs one; a zero value is not a physical specimen.
type Specimen struct {
	phases       []Phase
	pores        Pores
	volume, mass float64
}

// New builds a specimen from current solid phases, not from a fresh batch
// recipe. It does not predict hydration, shrinkage, carbonation or drying.
// Individual solid volumes are mass/density; bulk volume is their sum divided
// by (1-porosity). All units are SI. Names must be distinct, masses and
// densities positive and finite, and every input must have a stated basis.
// Porosity is in [0,1), saturation in [0,1]. No defaults fill missing data.
func New(solids []Solid, pores Pores) (Specimen, error) {
	if len(solids) == 0 {
		return Specimen{}, fmt.Errorf("specimen: no solids")
	}
	if err := validatePores(pores); err != nil {
		return Specimen{}, err
	}
	made := Specimen{pores: pores}
	seen := make(map[string]bool)
	var volumes []float64
	solidVolume := 0.0
	for index, solid := range solids {
		phase, volume, err := phaseOf(index, solid)
		if err != nil {
			return Specimen{}, err
		}
		if seen[solid.Name] {
			return Specimen{}, fmt.Errorf("specimen: duplicate solid %q", solid.Name)
		}
		seen[solid.Name] = true
		solidVolume += volume
		made.mass += solid.MassKG
		made.phases = append(made.phases, phase)
		volumes = append(volumes, volume)
	}
	made.volume = solidVolume / (1 - pores.VolumeFraction)
	if err := made.validateTotals(); err != nil {
		return Specimen{}, err
	}
	for index := range made.phases {
		made.phases[index].VolumeFraction = volumes[index] / made.volume
	}
	return made, nil
}

// validatePores identifies the specific invalid pore input, with its value.
func validatePores(pores Pores) error {
	if strings.TrimSpace(pores.Basis) == "" {
		return fmt.Errorf("specimen: pores have no basis")
	}
	if !fraction(pores.VolumeFraction) || pores.VolumeFraction == 1 {
		return fmt.Errorf("specimen: invalid pore volume fraction %g; want [0,1)", pores.VolumeFraction)
	}
	if !fraction(pores.Saturation) {
		return fmt.Errorf("specimen: invalid pore saturation %g; want [0,1]", pores.Saturation)
	}
	if !positive(pores.WaterDensityKGPerM3) {
		return fmt.Errorf("specimen: invalid pore water density %g kg/m3; required even without water", pores.WaterDensityKGPerM3)
	}
	return nil
}

// phaseOf validates one solid and returns its record and separate volume.
// The caller fills VolumeFraction only after the bulk volume is known.
func phaseOf(index int, solid Solid) (Phase, float64, error) {
	if strings.TrimSpace(solid.Name) == "" || solid.Name != strings.TrimSpace(solid.Name) {
		return Phase{}, 0, fmt.Errorf("specimen: solid %d has an empty or padded name %q", index, solid.Name)
	}
	if strings.TrimSpace(solid.Basis) == "" {
		return Phase{}, 0, fmt.Errorf("specimen: solid %d %q has no basis", index, solid.Name)
	}
	if !positive(solid.MassKG) {
		return Phase{}, 0, fmt.Errorf("specimen: invalid solid mass %g kg for %q", solid.MassKG, solid.Name)
	}
	if !positive(solid.DensityKGPerM3) {
		return Phase{}, 0, fmt.Errorf("specimen: invalid solid density %g kg/m3 for %q", solid.DensityKGPerM3, solid.Name)
	}
	volume := solid.MassKG / solid.DensityKGPerM3
	if !positive(volume) {
		return Phase{}, 0, fmt.Errorf("specimen: unrepresentable solid volume for %q", solid.Name)
	}
	return Phase{Name: solid.Name, MassKG: solid.MassKG,
		DensityKGPerM3: solid.DensityKGPerM3, Basis: solid.Basis}, volume, nil
}

// validateTotals keeps each arithmetic failure distinct from input validation.
// The wet-density check is needed: rounding subnormal volumes can make the
// computed sum overflow even though the real-valued density is bounded.
func (made Specimen) validateTotals() error {
	if !positive(made.volume) {
		return fmt.Errorf("specimen: unrepresentable bulk volume")
	}
	if !positive(made.mass) {
		return fmt.Errorf("specimen: unrepresentable solid mass sum")
	}
	if !positive(made.DryDensityKGPerM3()) {
		return fmt.Errorf("specimen: unrepresentable dry density")
	}
	if !finite(made.WaterMassKG()) {
		return fmt.Errorf("specimen: unrepresentable water mass")
	}
	if !positive(made.BulkDensityKGPerM3()) {
		return fmt.Errorf("specimen: unrepresentable bulk density")
	}
	return nil
}

// Phases returns independent records in the input order.
func (made Specimen) Phases() []Phase { return append([]Phase(nil), made.phases...) }

// Pores returns the specified pore conditions, including their basis.
func (made Specimen) Pores() Pores { return made.pores }

// VolumeM3 is bulk volume, including filled and empty pores.
func (made Specimen) VolumeM3() float64 { return made.volume }

// SolidMassKG includes solid-bound water, but excludes liquid pore water.
func (made Specimen) SolidMassKG() float64 { return made.mass }

// WaterVolumeFraction is the share of bulk volume occupied by pore water.
func (made Specimen) WaterVolumeFraction() float64 {
	return made.pores.VolumeFraction * made.pores.Saturation
}

// EmptyVolumeFraction is the share of bulk volume occupied by empty pores.
func (made Specimen) EmptyVolumeFraction() float64 {
	return made.pores.VolumeFraction * (1 - made.pores.Saturation)
}

// WaterMassKG is the liquid pore water's mass at the supplied density.
func (made Specimen) WaterMassKG() float64 {
	return made.volume * made.WaterVolumeFraction() * made.pores.WaterDensityKGPerM3
}

// DryDensityKGPerM3 is solid mass per bulk volume at the supplied porosity.
// This excludes free pore water; it does not simulate an oven-drying protocol.
func (made Specimen) DryDensityKGPerM3() float64 { return made.mass / made.volume }

// BulkDensityKGPerM3 includes liquid pore water and neglects empty-pore gas mass.
func (made Specimen) BulkDensityKGPerM3() float64 {
	return made.DryDensityKGPerM3() +
		made.WaterVolumeFraction()*made.pores.WaterDensityKGPerM3
}

// finite excludes NaN and both infinities from physical inputs and results.
func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

// positive requires a strictly positive, finite physical quantity.
func positive(value float64) bool { return value > 0 && finite(value) }

// fraction accepts finite shares between zero and one, including both limits.
func fraction(value float64) bool { return value >= 0 && value <= 1 }
