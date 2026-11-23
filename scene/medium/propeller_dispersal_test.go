package medium

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// propellerDispersalInvalidFile is the csv file of the dispersal controls
// that are refused.
const propellerDispersalInvalidFile = "testdata/propeller-dispersal-invalid.csv"

// propellerDispersalSpread gives a configuration the downstream spread the
// tests chose: where it starts, and how fast it widens each metre.
func propellerDispersalSpread(t *testing.T, config *PropellerCavitationConfig) {
	t.Helper()
	config.SpreadStartM = picked(t, "propeller-dispersal-spread-start")
	config.SpreadPerM = picked(t, "propeller-dispersal-spread-per")
}

// propellerDispersalWandered is the standard configuration with the wander
// and the spread the tests chose.
func propellerDispersalWandered(t *testing.T) PropellerCavitationConfig {
	t.Helper()
	config := testPropellerCavitation(t).Parameters()
	config.WanderM = picked(t, "propeller-dispersal-wander")
	config.WanderScaleM = picked(t, "propeller-dispersal-wander-scale")
	propellerDispersalSpread(t, &config)
	return config
}

// propellerDispersalSymmetry holds that wandering leaves the rotor's origin
// where it was, and breaks the symmetry between blades downstream.
func propellerDispersalSymmetry(
	t *testing.T, base, wandered PropellerCavitation,
) {
	t.Helper()
	for blade := range base.blades {
		moved := wandered.filamentCentre(nought, blade).Sub(
			base.filamentCentre(nought, blade))
		if moved.Len() != nought {
			t.Fatal("rotor origin wandered")
		}
	}
	station := picked(t, "propeller-dispersal-station")
	first := wandered.filamentCentre(station, nought).Sub(
		base.filamentCentre(station, nought))
	second := wandered.filamentCentre(station, unity).Sub(
		base.filamentCentre(station, unity))
	least := picked(t, "propeller-dispersal-displacement-least")
	if first.Len() < least || first.Sub(second).Len() < least {
		t.Fatal("blade displacement remains symmetric")
	}
}

// propellerDispersalConvection holds that the meander is carried down the
// axis with the slipstream.
func propellerDispersalConvection(t *testing.T, wandered PropellerCavitation) {
	t.Helper()
	later := wandered.Tick(time.Second).(PropellerCavitation)
	// q=s-U*t and rotor retarded phase are identical along this characteristic.
	blade := pickedCount(t, "propeller-dispersal-convected-blade")
	station := picked(t, "propeller-dispersal-station")
	laterStation := picked(t, "propeller-dispersal-later-station")
	here := wandered.filamentCentre(station, blade).Sub(wandered.origin).
		Sub(wandered.axis.Scale(station))
	there := later.filamentCentre(laterStation, blade).Sub(wandered.origin).
		Sub(wandered.axis.Scale(laterStation))
	tolerance := picked(t, "propeller-dispersal-convection-tolerance")
	if here.Sub(there).Len() > tolerance {
		t.Fatal("meander is not convected with slipstream")
	}
}

// propellerDispersalEscapes is whether a point is outside a box by more than
// the slack on any axis.
func propellerDispersalEscapes(
	point maths.Vec, box maths.Box, slack float64,
) bool {
	return point.X < box.Min.X-slack || point.Y < box.Min.Y-slack ||
		point.Z < box.Min.Z-slack || point.X > box.Max.X+slack ||
		point.Y > box.Max.Y+slack || point.Z > box.Max.Z+slack
}

// propellerDispersalSupport holds that the dispersed support of every
// filament, along its length, stays inside the field's bounds.
func propellerDispersalSupport(t *testing.T, field PropellerCavitation) {
	t.Helper()
	steps := pickedCount(t, "propeller-dispersal-support-steps")
	widths := picked(t, "propeller-dispersal-support-widths")
	slack := picked(t, "propeller-dispersal-bound-slack")
	box := field.Bounds()
	directions := []maths.Vec{field.radial, field.cross,
		field.radial.Scale(-unity), field.cross.Scale(-unity)}
	for place := nought; place <= steps; place++ {
		station := field.activeLength * float64(place) / float64(steps)
		for blade := range field.blades {
			centre := field.filamentCentre(station, blade)
			grown := math.Max(nought, station-field.spreadStart)
			width := field.core + field.spread*grown
			for _, direction := range directions {
				point := centre.Add(direction.Scale(widths * width))
				if propellerDispersalEscapes(point, box, slack) {
					t.Fatal("dispersed support escapes bound")
				}
			}
		}
	}
}

// TestPropellerDispersalPreservesRotorAndBreaksBladeSymmetry holds that
// wander and spread leave the rotor alone, break the symmetry of the blades,
// travel with the slipstream and stay within the bounds.
func TestPropellerDispersalPreservesRotorAndBreaksBladeSymmetry(t *testing.T) {
	base := testPropellerCavitation(t)
	config := propellerDispersalWandered(t)
	wandered, err := NewPropellerCavitation(config)
	if err != nil {
		t.Fatal(err)
	}
	propellerDispersalSymmetry(t, base, wandered)
	propellerDispersalConvection(t, wandered)
	config.Axis = maths.V(picked(t, "propeller-dispersal-axis-x"),
		picked(t, "propeller-dispersal-axis-y"),
		picked(t, "propeller-dispersal-axis-z"))
	tilted, err := NewPropellerCavitation(config)
	if err != nil {
		t.Fatal(err)
	}
	propellerDispersalSupport(t, tilted)
}

// propellerDispersalApply changes the one dispersal control the invalid file
// names, to a value that is not valid for it.
func propellerDispersalApply(
	t *testing.T, config *PropellerCavitationConfig, field string,
	value float64,
) {
	t.Helper()
	switch field {
	case "wander":
		config.WanderM = value
	case "spread_per":
		config.SpreadPerM = value
	case "wander_scale":
		config.WanderScaleM = value
	case "spread_start":
		config.SpreadStartM = value
	default:
		t.Fatalf("the invalid file names a control, %q, the test does not know",
			field)
	}
}

// TestPropellerDispersalControlsRefuseInvalid holds that each dispersal
// control in the invalid file is refused.
func TestPropellerDispersalControlsRefuseInvalid(t *testing.T) {
	for _, bad := range readSheet(t, propellerDispersalInvalidFile) {
		config := testPropellerCavitation(t).Parameters()
		propellerDispersalApply(t, &config, bad["field"], cell(t, bad, "value"))
		if _, err := NewPropellerCavitation(config); err == nil {
			t.Fatal("invalid dispersion accepted")
		}
	}
}

// propellerDispersalIntegral is the sum of a field's density over a square
// of its cross section a station downstream, taken in small cells.
func propellerDispersalIntegral(
	t *testing.T, field PropellerCavitation,
) float64 {
	t.Helper()
	half := picked(t, "propeller-dispersal-quadrature-half")
	step := picked(t, "propeller-dispersal-quadrature-step")
	downstream := picked(t, "propeller-dispersal-section-station")
	station := field.origin.Add(field.axis.Scale(downstream))
	var sum float64
	for across := -half; across <= half; across += step {
		for up := -half; up <= half; up += step {
			point := station.Add(field.radial.Scale(across)).
				Add(field.cross.Scale(up))
			sum += field.Density(point) * step * step
		}
	}
	return sum
}

// TestPropellerDispersalDilutesWithoutCreatingCrossSectionDensity holds that
// spreading leaves the density of a cross section as it was, thins the
// peak, and widens the support.
func TestPropellerDispersalDilutesWithoutCreatingCrossSectionDensity(
	t *testing.T,
) {
	config := testPropellerCavitation(t).Parameters()
	config.Blades = pickedCount(t, "propeller-dispersal-single-blade")
	narrow, _ := NewPropellerCavitation(config)
	propellerDispersalSpread(t, &config)
	widened, _ := NewPropellerCavitation(config)
	// Independent area quadrature of the complete compact Gaussian support.
	// Both fields have identical pressure/fade at this cross section.
	narrowSum := propellerDispersalIntegral(t, narrow)
	widenedSum := propellerDispersalIntegral(t, widened)
	tolerance := picked(t, "propeller-dispersal-dilution-tolerance")
	if narrowSum <= nought ||
		math.Abs(narrowSum-widenedSum)/narrowSum > tolerance {
		t.Fatalf("spreading creates density: %g vs %g", narrowSum, widenedSum)
	}
	section := picked(t, "propeller-dispersal-section-station")
	centre := narrow.filamentCentre(section, nought)
	if widened.Density(centre) >= narrow.Density(centre) {
		t.Fatal("spreading does not dilute peak")
	}
	reach := picked(t, "propeller-dispersal-outside-offset")
	outside := centre.Add(narrow.radial.Scale(reach))
	if narrow.Density(outside) != nought || widened.Density(outside) <= nought {
		t.Fatal("cloud support did not widen")
	}
}

// TestPropellerDispersalOmittedControlsRemainExact holds that a wander scale
// and a spread start with no wander or spread to use them leave the density
// exactly as it was.
func TestPropellerDispersalOmittedControlsRemainExact(t *testing.T) {
	plain := testPropellerCavitation(t)
	config := plain.Parameters()
	config.WanderScaleM = picked(t, "propeller-dispersal-omitted-wander-scale")
	config.SpreadStartM = picked(t, "propeller-dispersal-omitted-spread-start")
	omitted, err := NewPropellerCavitation(config)
	if err != nil {
		t.Fatal(err)
	}
	ring := picked(t, "propeller-dispersal-ring-radius")
	period := pickedCount(t, "propeller-dispersal-rise-period")
	divisor := picked(t, "propeller-dispersal-rise-divisor")
	for place := range pickedCount(t, "propeller-dispersal-samples") {
		angle := float64(place)
		rise := float64(place%period) / divisor
		offset := maths.V(math.Sin(angle)*ring, math.Cos(angle)*ring, rise)
		point := plain.origin.Add(offset)
		if plain.Density(point) != omitted.Density(point) {
			t.Fatal("disabled controls altered density")
		}
	}
}
