package set

import (
	"github.com/janearc/slayer/scene/thing"
	yaml "gopkg.in/yaml.v3"
	"strings"
)

// PartSpin selects one native material group for explicit local-axis motion.
type PartSpin struct {
	material    string
	pivot, axis [3]float64
	rpm, phase  float64
}

// PartSpin returns the optional, copied native part-motion description.
func (made Thing) PartSpin() (PartSpin, bool) {
	if made.partSpin == nil {
		return PartSpin{}, false
	}
	return *made.partSpin, true
}

// Material is the exact native material name within this placed object only.
func (spin PartSpin) Material() string { return spin.material }

// PivotMetres is the native factory-local pivot, not a map/world coordinate.
func (spin PartSpin) PivotMetres() [3]float64 { return spin.pivot }

// Axis is the nonzero native local rotation axis.
func (spin PartSpin) Axis() [3]float64 { return spin.axis }

// RPM is the signed number of revolutions per minute.
func (spin PartSpin) RPM() float64 { return spin.rpm }

// PhaseRadians is the initial angular offset, zero when omitted.
func (spin PartSpin) PhaseRadians() float64 { return spin.phase }

// partSpin returns a step that reads a spin block into target.
func (reading reader) partSpin(target **PartSpin) step {
	return func(_, block *yaml.Node) error {
		made := &PartSpin{}
		err := reading.keys(block, map[string]step{
			"material":      reading.text(&made.material),
			"pivot-metres":  reading.three(&made.pivot, -mostSpaceCoordinate, mostSpaceCoordinate),
			"axis":          reading.three(&made.axis, -mostSpaceCoordinate, mostSpaceCoordinate),
			"rpm":           reading.number(&made.rpm, -thing.MaximumAxisSpinRPM, thing.MaximumAxisSpinRPM),
			"phase-radians": reading.number(&made.phase, -mostSpaceCoordinate, mostSpaceCoordinate),
		}, nil)
		if err == nil {
			err = reading.all(block, "part-spin", "material", "pivot-metres", "axis", "rpm")
		}
		if err == nil && strings.TrimSpace(made.material) == "" {
			err = reading.refuse(block, "part-spin needs a nonempty native material name")
		}
		if err == nil && made.axis == ([3]float64{}) {
			err = reading.refuse(block, "part-spin needs a nonzero native axis")
		}
		if err == nil {
			*target = made
		}
		return err
	}
}
