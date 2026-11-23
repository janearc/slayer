package materials

import (
	"math"
	"strings"
	"time"
)

// WaterObservation is a sourced observation of a property of water. The
// property determines the unit: practical salinity is dimensionless,
// suspended sediment is milligrams per litre, and turbidity is FNU. Source
// identifies the sample, dataset or an explicitly authored value.
type WaterObservation struct {
	Value     float64
	Source    string
	At        string
	Qualifier string
}

// WaterComposition retains independent observations. The values do not
// determine absorption, scattering, or refractive index without a separately
// supplied, validated optical model.
type WaterComposition struct {
	salinity, sediment, turbidity *WaterObservation
}

// NewWaterComposition copies at least one sourced, nonnegative observation.
func NewWaterComposition(salinity, sediment, turbidity *WaterObservation) (*WaterComposition, error) {
	if salinity == nil && sediment == nil && turbidity == nil {
		return nil, refuse("water composition with no observations")
	}
	copyObservation := func(observation *WaterObservation) (*WaterObservation, error) {
		if observation == nil {
			return nil, nil
		}
		if math.IsNaN(observation.Value) || math.IsInf(observation.Value, 0) ||
			observation.Value < 0 || strings.TrimSpace(observation.Source) == "" {
			return nil, refuse("water observation needs a finite nonnegative value and a source")
		}
		if observation.At != "" {
			if _, err := time.Parse(time.RFC3339, observation.At); err != nil {
				return nil, refuse("water observation time %q is not RFC 3339", observation.At)
			}
		}
		if observation.Qualifier != "" && strings.TrimSpace(observation.Qualifier) == "" {
			return nil, refuse("water observation qualifier is blank")
		}
		copy := *observation
		return &copy, nil
	}
	salt, err := copyObservation(salinity)
	if err != nil {
		return nil, err
	}
	particles, err := copyObservation(sediment)
	if err != nil {
		return nil, err
	}
	cloudiness, err := copyObservation(turbidity)
	if err != nil {
		return nil, err
	}
	return &WaterComposition{salinity: salt, sediment: particles, turbidity: cloudiness}, nil
}

// Salinity reports practical salinity and its provenance when supplied.
func (water *WaterComposition) Salinity() (WaterObservation, bool) {
	if water == nil || water.salinity == nil {
		return WaterObservation{}, false
	}
	return *water.salinity, true
}

// SuspendedSediment reports sediment in milligrams per litre when supplied.
func (water *WaterComposition) SuspendedSediment() (WaterObservation, bool) {
	if water == nil || water.sediment == nil {
		return WaterObservation{}, false
	}
	return *water.sediment, true
}

// Turbidity reports the measured turbidity in FNU when supplied.
func (water *WaterComposition) Turbidity() (WaterObservation, bool) {
	if water == nil || water.turbidity == nil {
		return WaterObservation{}, false
	}
	return *water.turbidity, true
}
