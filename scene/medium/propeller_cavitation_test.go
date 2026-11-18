package medium

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

func testPropellerCavitation(t *testing.T) PropellerCavitation {
	t.Helper()
	field, err := NewPropellerCavitation(PropellerCavitationConfig{
		Blades: 5,
		Origin: maths.V(10, -20, 30), Axis: maths.V(0, 0, 4),
		TipRadius: 2.1, RPM: 300, SlipstreamMPS: 20, LengthM: 40,
		CoreRadiusM: 0.2, DensityPerM: 0.8, PressureDeficitPa: 12000,
		RecoveryPaPerM: 600, VisualOnset: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return field
}

func TestPropellerCavitationHasFivePhaseLockedFilaments(t *testing.T) {
	field := testPropellerCavitation(t)
	first := field.Tick(0)
	fivefold := field.Tick(time.Duration(float64(time.Second) / (5 * 5)))
	point := field.origin.Add(field.radial.Scale(field.radius)).Add(field.axis.Scale(8))
	if got, want := first.Density(point), fivefold.Density(point); got < want*0.999 || got > want*1.001 {
		t.Fatalf("one fifth blade-passage period changed the fivefold field: %g then %g", got, want)
	}
	later := field.Tick(time.Second / 20).Density(point)
	if later == first.Density(point) {
		t.Fatal("the phase-locked field did not move with time")
	}
}

func TestPropellerCavitationAdvectsAndDecaysWithPressureRecovery(t *testing.T) {
	field := testPropellerCavitation(t)
	// At 300 rpm and 20 m/s, 8 m downstream samples the same rotor phase.
	// At half a metre the emission delay is 0.025 s, so the 5 Hz shaft
	// has turned one eighth of a revolution: use that analytic tip position.
	offset := field.radial.Scale(field.radius / math.Sqrt2).
		Add(field.cross.Scale(-field.radius / math.Sqrt2))
	nearAt := field.origin.Add(field.axis.Scale(0.5)).Add(offset)
	farAt := field.origin.Add(field.axis.Scale(8.5)).Add(offset)
	near := field.Density(nearAt)
	far := field.Density(farAt)
	if near <= 0 || far <= 0 || far >= near {
		t.Fatalf("the vapor trail should advect, then lose density downstream: near %g far %g", near, far)
	}
	bounds := field.Bounds()
	if bounds.Max.Z-bounds.Min.Z != field.activeLength {
		t.Fatalf("pressure recovery left axial extent %g, want %g: %+v",
			bounds.Max.Z-bounds.Min.Z, field.activeLength, bounds)
	}
	if got := field.Density(field.origin.Add(field.axis.Scale(field.activeLength + 0.01))); got != 0 {
		t.Fatalf("field persisted past pressure recovery at %g m", got)
	}
}

func TestPropellerCavitationPressureRecoveryShortensTheField(t *testing.T) {
	config := PropellerCavitationConfig{Blades: 5, Origin: maths.V(0, 0, 0), Axis: maths.V(0, 0, 1),
		TipRadius: 2, RPM: 300, SlipstreamMPS: 20, LengthM: 40,
		CoreRadiusM: 0.2, DensityPerM: 1, PressureDeficitPa: 6000,
		RecoveryPaPerM: 600, VisualOnset: true}
	field, err := NewPropellerCavitation(config)
	if err != nil {
		t.Fatal(err)
	}
	if field.activeLength != 10 {
		t.Fatalf("authored deficit/recovery ratio gives %g m, want 10 m", field.activeLength)
	}
}

func TestPropellerCavitationFilamentPhaseTravelsWithSlipstream(t *testing.T) {
	field := testPropellerCavitation(t)
	before := field.filamentCentre(10, 2)
	seconds := 0.25
	later := field.Tick(time.Duration(seconds * float64(time.Second)))
	after := later.(PropellerCavitation).filamentCentre(15, 2)
	want := before.Add(field.axis.Scale(field.slip * seconds))
	if !after.Near(want, 1e-12) {
		t.Fatalf("advected filament centre is %v, want %v", after, want)
	}
}

func TestPropellerCavitationIsOffWithoutExplicitVisualOnset(t *testing.T) {
	config := PropellerCavitationConfig{Blades: 5, Origin: maths.V(0, 0, 0), Axis: maths.V(0, 0, 1),
		TipRadius: 2, RPM: 300, SlipstreamMPS: 20, LengthM: 20,
		CoreRadiusM: 0.2, DensityPerM: 1, PressureDeficitPa: 1000,
		RecoveryPaPerM: 100, VisualOnset: false}
	field, err := NewPropellerCavitation(config)
	if err != nil {
		t.Fatal(err)
	}
	if got := field.Density(maths.V(2, 0, 5)); got != 0 {
		t.Fatalf("unrequested visual onset produced density %g", got)
	}
}

func TestPropellerCavitationDensityEndsAtItsDeclaredRadialSupport(t *testing.T) {
	field := testPropellerCavitation(t)
	// Eight metres is two full turns of the authored helix. At this phase
	// one filament lies on radial, so this point is just beyond its bound.
	point := field.origin.Add(field.axis.Scale(8)).
		Add(field.radial.Scale(field.radius + 3*field.core + 0.001))
	if got := field.Density(point); got != 0 {
		t.Fatalf("density outside declared compact support = %g, want zero", got)
	}
}

func TestPropellerCavitationUsesAuthoredBladeCount(t *testing.T) {
	config := PropellerCavitationConfig{Blades: 2,
		Origin: maths.V(0, 0, 0), Axis: maths.V(0, 0, 1),
		TipRadius: 2, RPM: 300, SlipstreamMPS: 20, LengthM: 40,
		CoreRadiusM: .2, DensityPerM: 1, PressureDeficitPa: 12000,
		RecoveryPaPerM: 600, VisualOnset: true}
	field, err := NewPropellerCavitation(config)
	if err != nil {
		t.Fatal(err)
	}
	// Two full turns downstream, a two-blade propeller has opposed tips.
	point := field.origin.Add(field.axis.Scale(8)).
		Sub(field.radial.Scale(field.radius))
	if got := field.Density(point); got <= 0 {
		t.Fatalf("the authored second opposed tip has no density: %g", got)
	}
	for _, blades := range []int{0, 17} {
		config.Blades = blades
		if _, err := NewPropellerCavitation(config); err == nil {
			t.Fatalf("accepted unsupported blade count %d", blades)
		}
	}
}

func TestPropellerCavitationBreaksIntoAdvectedFragments(t *testing.T) {
	config := PropellerCavitationConfig{Blades: 5,
		Origin: maths.V(0, 0, 0), Axis: maths.V(0, 0, 1),
		TipRadius: 2, RPM: 300, SlipstreamMPS: 20, LengthM: 40,
		CoreRadiusM: .2, DensityPerM: 1, PressureDeficitPa: 12000,
		RecoveryPaPerM: 600, VisualOnset: true,
		FragmentScaleM: .5, FragmentContrast: 1, FragmentSeed: 705}
	broken, err := NewPropellerCavitation(config)
	if err != nil {
		t.Fatal(err)
	}
	config.FragmentContrast = 0
	continuous, err := NewPropellerCavitation(config)
	if err != nil {
		t.Fatal(err)
	}
	var gaps, kept int
	for sample := range 80 {
		distance := 10 + float64(sample)/10
		point := continuous.filamentCentre(distance, 0)
		got, base := broken.Density(point), continuous.Density(point)
		if got < 0 || got > base {
			t.Fatalf("fragmentation created density: broken=%g base=%g", got, base)
		}
		if got == 0 {
			gaps++
		} else {
			kept++
		}
	}
	if gaps == 0 || kept == 0 {
		t.Fatalf("fragment mask did not create bounded gaps and retained cavities: gaps=%d kept=%d", gaps, kept)
	}
	// The same downstream packet moves5m in0.25s. Both points are beyond
	// the breakup ramp; divide out the independent pressure-recovery fade.
	distance, seconds := 11.0, .25
	point := broken.filamentCentre(distance, 0)
	later := broken.Tick(time.Duration(seconds * float64(time.Second)))
	moved := point.Add(broken.axis.Scale(broken.slip * seconds))
	first := broken.Density(point) / (1 - distance/broken.activeLength)
	second := later.Density(moved) / (1 - (distance+broken.slip*seconds)/broken.activeLength)
	if math.Abs(first-second) > 1e-10 {
		t.Fatalf("fragment noise failed slipstream advection: %g then %g", first, second)
	}
}

func TestPropellerCavitationRejectsUnboundedOrInvalidInputs(t *testing.T) {
	base := PropellerCavitationConfig{Blades: 5, Origin: maths.V(0, 0, 0), Axis: maths.V(0, 0, 1),
		TipRadius: 2, RPM: 300, SlipstreamMPS: 20, LengthM: 20,
		CoreRadiusM: 0.2, DensityPerM: 1, PressureDeficitPa: 1000,
		RecoveryPaPerM: 100, VisualOnset: true}
	for name, change := range map[string]func(*PropellerCavitationConfig){
		"zero axis":           func(c *PropellerCavitationConfig) { c.Axis = maths.V(0, 0, 0) },
		"too long":            func(c *PropellerCavitationConfig) { c.LengthM = 501 },
		"no pressure deficit": func(c *PropellerCavitationConfig) { c.PressureDeficitPa = 0 },
		"no recovery":         func(c *PropellerCavitationConfig) { c.RecoveryPaPerM = 0 },
		"zero rotation":       func(c *PropellerCavitationConfig) { c.RPM = 0 },
		"fragment contrast":   func(c *PropellerCavitationConfig) { c.FragmentContrast = 2 },
		"fragment scale":      func(c *PropellerCavitationConfig) { c.FragmentContrast = 1; c.FragmentScaleM = 0 },
		"tiny fragment scale": func(c *PropellerCavitationConfig) { c.FragmentContrast = 1; c.FragmentScaleM = 1e-300 },
	} {
		t.Run(name, func(t *testing.T) {
			config := base
			change(&config)
			if _, err := NewPropellerCavitation(config); err == nil {
				t.Fatal("invalid cavitation field was accepted")
			}
		})
	}
}
