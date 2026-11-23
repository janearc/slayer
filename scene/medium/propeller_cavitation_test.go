package medium

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// The csv files the propeller cavitation tests read their fields from.
const (
	propellerCavitationConfigsFile = "testdata/propeller-cavitation-configs.csv"
	propellerCavitationCountsFile  = "testdata/" +
		"propeller-cavitation-blade-counts.csv"
	propellerCavitationInvalidFile = "testdata/propeller-cavitation-invalid.csv"
)

// propellerCavitationConfigNamed is the field configuration called name in
// the configs file.
func propellerCavitationConfigNamed(
	t *testing.T, name string,
) PropellerCavitationConfig {
	t.Helper()
	row := readSheet(t, propellerCavitationConfigsFile).row(t,
		propellerCavitationConfigsFile, name)
	return PropellerCavitationConfig{
		Blades: cellCount(t, row, "blades"),
		Origin: maths.V(cell(t, row, "origin_x"), cell(t, row, "origin_y"),
			cell(t, row, "origin_z")),
		Axis: maths.V(cell(t, row, "axis_x"), cell(t, row, "axis_y"),
			cell(t, row, "axis_z")),
		TipRadius:         cell(t, row, "tip_radius"),
		RPM:               cell(t, row, "rpm"),
		SlipstreamMPS:     cell(t, row, "slipstream"),
		LengthM:           cell(t, row, "length"),
		CoreRadiusM:       cell(t, row, "core_radius"),
		DensityPerM:       cell(t, row, "density"),
		PressureDeficitPa: cell(t, row, "deficit"),
		RecoveryPaPerM:    cell(t, row, "recovery"),
		VisualOnset:       row["onset"] == "on",
		FragmentScaleM:    cell(t, row, "fragment_scale"),
		FragmentContrast:  cell(t, row, "fragment_contrast"),
		FragmentSeed:      uint64(cellCount(t, row, "fragment_seed")),
	}
}

// propellerCavitationFieldNamed is the field the configuration called name
// makes.
func propellerCavitationFieldNamed(
	t *testing.T, name string,
) PropellerCavitation {
	t.Helper()
	config := propellerCavitationConfigNamed(t, name)
	field, err := NewPropellerCavitation(config)
	if err != nil {
		t.Fatal(err)
	}
	return field
}

// testPropellerCavitation is the field the tests of the propeller share.
func testPropellerCavitation(t *testing.T) PropellerCavitation {
	t.Helper()
	return propellerCavitationFieldNamed(t, "standard")
}

// TestPropellerCavitationHasFivePhaseLockedFilaments holds that one fifth of
// a blade-passage period leaves the fivefold field as it was, and that other
// times move it.
func TestPropellerCavitationHasFivePhaseLockedFilaments(t *testing.T) {
	field := testPropellerCavitation(t)
	first := field.Tick(nought)
	shaftRate := picked(t, "propeller-cavitation-shaft-rate")
	passageBlades := picked(t, "propeller-cavitation-passage-blades")
	fifth := time.Duration(float64(time.Second) / (shaftRate * passageBlades))
	fivefold := field.Tick(fifth)
	downstream := picked(t, "propeller-cavitation-two-turns")
	tip := field.origin.Add(field.radial.Scale(field.radius))
	point := tip.Add(field.axis.Scale(downstream))
	lower := picked(t, "propeller-cavitation-lower-share")
	upper := picked(t, "propeller-cavitation-upper-share")
	got, want := first.Density(point), fivefold.Density(point)
	if got < want*lower || got > want*upper {
		t.Fatalf("one fifth blade-passage period changed the fivefold field: "+
			"%g then %g", got, want)
	}
	divisor := pickedCount(t, "propeller-cavitation-later-divisor")
	later := field.Tick(time.Second / time.Duration(divisor)).Density(point)
	if later == first.Density(point) {
		t.Fatal("the phase-locked field did not move with time")
	}
}

// TestPropellerCavitationAdvectsAndDecaysWithPressureRecovery holds that a
// vapor trail advects and loses density downstream, and that the field ends
// where the pressure has recovered.
func TestPropellerCavitationAdvectsAndDecaysWithPressureRecovery(t *testing.T) {
	field := testPropellerCavitation(t)
	// At 300 rpm and 20 m/s, 8 m downstream samples the same rotor phase.
	// At half a metre the emission delay is 0.025 s, so the 5 Hz shaft
	// has turned one eighth of a revolution: use that analytic tip position.
	offset := field.radial.Scale(field.radius / math.Sqrt2).
		Add(field.cross.Scale(-field.radius / math.Sqrt2))
	nearAxial := picked(t, "propeller-cavitation-near-station")
	farAxial := picked(t, "propeller-cavitation-far-station")
	nearAt := field.origin.Add(field.axis.Scale(nearAxial)).Add(offset)
	farAt := field.origin.Add(field.axis.Scale(farAxial)).Add(offset)
	near := field.Density(nearAt)
	far := field.Density(farAt)
	if near <= nought || far <= nought || far >= near {
		t.Fatalf("the vapor trail should advect, then lose density "+
			"downstream: near %g far %g", near, far)
	}
	bounds := field.Bounds()
	if bounds.Max.Z-bounds.Min.Z != field.activeLength {
		t.Fatalf("pressure recovery left axial extent %g, want %g: %+v",
			bounds.Max.Z-bounds.Min.Z, field.activeLength, bounds)
	}
	pastEnd := field.activeLength + picked(t, "propeller-cavitation-past-end")
	past := field.origin.Add(field.axis.Scale(pastEnd))
	if got := field.Density(past); got != nought {
		t.Fatalf("field persisted past pressure recovery at %g m", got)
	}
}

// TestPropellerCavitationPressureRecoveryShortensTheField holds that the
// ratio of the deficit to the recovery is the length the field is active.
func TestPropellerCavitationPressureRecoveryShortensTheField(t *testing.T) {
	field := propellerCavitationFieldNamed(t, "recovery")
	want := picked(t, "propeller-cavitation-recovered-length")
	if field.activeLength != want {
		t.Fatalf("authored deficit/recovery ratio gives %g m, want 10 m",
			field.activeLength)
	}
}

// TestPropellerCavitationFilamentPhaseTravelsWithSlipstream holds that a
// filament's centre moves down the axis at the slipstream's speed.
func TestPropellerCavitationFilamentPhaseTravelsWithSlipstream(t *testing.T) {
	field := testPropellerCavitation(t)
	blade := pickedCount(t, "propeller-cavitation-filament-blade")
	from := picked(t, "propeller-cavitation-filament-from")
	to := picked(t, "propeller-cavitation-filament-to")
	before := field.filamentCentre(from, blade)
	seconds := picked(t, "propeller-cavitation-travel-seconds")
	elapsed := time.Duration(seconds * float64(time.Second))
	later := field.Tick(elapsed)
	after := later.(PropellerCavitation).filamentCentre(to, blade)
	want := before.Add(field.axis.Scale(field.slip * seconds))
	tolerance := picked(t, "propeller-cavitation-travel-tolerance")
	if !after.Near(want, tolerance) {
		t.Fatalf("advected filament centre is %v, want %v", after, want)
	}
}

// TestPropellerCavitationIsOffWithoutExplicitVisualOnset holds that a field
// whose visual onset was not asked for has no density.
func TestPropellerCavitationIsOffWithoutExplicitVisualOnset(t *testing.T) {
	field := propellerCavitationFieldNamed(t, "silent")
	probe := maths.V(picked(t, "propeller-cavitation-probe-x"), nought,
		picked(t, "propeller-cavitation-probe-z"))
	if got := field.Density(probe); got != nought {
		t.Fatalf("unrequested visual onset produced density %g", got)
	}
}

// TestPropellerCavitationDensityEndsAtItsDeclaredRadialSupport holds that
// the density is nought just beyond the radial support of a filament.
func TestPropellerCavitationDensityEndsAtItsDeclaredRadialSupport(
	t *testing.T,
) {
	field := testPropellerCavitation(t)
	// Eight metres is two full turns of the authored helix. At this phase
	// one filament lies on radial, so this point is just beyond its bound.
	downstream := picked(t, "propeller-cavitation-two-turns")
	widths := picked(t, "propeller-cavitation-support-widths")
	margin := picked(t, "propeller-cavitation-support-margin")
	point := field.origin.Add(field.axis.Scale(downstream)).
		Add(field.radial.Scale(field.radius + widths*field.core + margin))
	if got := field.Density(point); got != nought {
		t.Fatalf("density outside declared compact support = %g, want zero",
			got)
	}
}

// TestPropellerCavitationUsesAuthoredBladeCount holds that a two-blade
// field has its opposed tip, and that counts of blades it cannot make are
// refused.
func TestPropellerCavitationUsesAuthoredBladeCount(t *testing.T) {
	config := propellerCavitationConfigNamed(t, "two-blade")
	field, err := NewPropellerCavitation(config)
	if err != nil {
		t.Fatal(err)
	}
	// Two full turns downstream, a two-blade propeller has opposed tips.
	downstream := picked(t, "propeller-cavitation-two-turns")
	point := field.origin.Add(field.axis.Scale(downstream)).
		Sub(field.radial.Scale(field.radius))
	if got := field.Density(point); got <= nought {
		t.Fatalf("the authored second opposed tip has no density: %g", got)
	}
	for _, count := range readSheet(t, propellerCavitationCountsFile) {
		config.Blades = cellCount(t, count, "blades")
		if _, err := NewPropellerCavitation(config); err == nil {
			t.Fatalf("accepted unsupported blade count %d", config.Blades)
		}
	}
}

// propellerCavitationFragmentCounts is how many of the samples down the
// first filament a broken field has no density at, and how many it keeps,
// while it never has more than the continuous field.
func propellerCavitationFragmentCounts(
	t *testing.T, broken, continuous PropellerCavitation,
) (gaps, kept int) {
	t.Helper()
	start := picked(t, "propeller-cavitation-fragment-start")
	divisor := picked(t, "propeller-cavitation-fragment-divisor")
	samples := pickedCount(t, "propeller-cavitation-fragment-samples")
	for sample := range samples {
		distance := start + float64(sample)/divisor
		point := continuous.filamentCentre(distance, nought)
		got, base := broken.Density(point), continuous.Density(point)
		if got < nought || got > base {
			t.Fatalf("fragmentation created density: broken=%g base=%g", got,
				base)
		}
		if got == nought {
			gaps++
		} else {
			kept++
		}
	}
	return gaps, kept
}

// propellerCavitationFragmentsAdvect holds that a broken field's fragment
// noise moves down the axis with the slipstream.
func propellerCavitationFragmentsAdvect(
	t *testing.T, broken PropellerCavitation,
) {
	t.Helper()
	// The same downstream packet moves5m in0.25s. Both points are beyond
	// the breakup ramp; divide out the independent pressure-recovery fade.
	distance := picked(t, "propeller-cavitation-fragment-distance")
	seconds := picked(t, "propeller-cavitation-fragment-seconds")
	point := broken.filamentCentre(distance, nought)
	elapsed := time.Duration(seconds * float64(time.Second))
	later := broken.Tick(elapsed)
	moved := point.Add(broken.axis.Scale(broken.slip * seconds))
	fadeFirst := unity - distance/broken.activeLength
	fadeSecond := unity - (distance+broken.slip*seconds)/broken.activeLength
	first := broken.Density(point) / fadeFirst
	second := later.Density(moved) / fadeSecond
	tolerance := picked(t, "propeller-cavitation-fragment-tolerance")
	if math.Abs(first-second) > tolerance {
		t.Fatalf("fragment noise failed slipstream advection: %g then %g",
			first, second)
	}
}

// TestPropellerCavitationBreaksIntoAdvectedFragments holds that fragments
// leave gaps and kept cavities without making density, and that they move
// with the slipstream.
func TestPropellerCavitationBreaksIntoAdvectedFragments(t *testing.T) {
	config := propellerCavitationConfigNamed(t, "fragments")
	broken, err := NewPropellerCavitation(config)
	if err != nil {
		t.Fatal(err)
	}
	config.FragmentContrast = nought
	continuous, err := NewPropellerCavitation(config)
	if err != nil {
		t.Fatal(err)
	}
	gaps, kept := propellerCavitationFragmentCounts(t, broken, continuous)
	if gaps == nought || kept == nought {
		t.Fatalf("fragment mask did not create bounded gaps and retained "+
			"cavities: gaps=%d kept=%d", gaps, kept)
	}
	propellerCavitationFragmentsAdvect(t, broken)
}

// propellerCavitationApply changes the one field of a configuration that the
// invalid file names, to a value that is not valid for it.
func propellerCavitationApply(
	t *testing.T, config *PropellerCavitationConfig, field string,
	value float64,
) {
	t.Helper()
	switch field {
	case "none":
	case "axis":
		config.Axis = maths.V(value, value, value)
	case "length":
		config.LengthM = value
	case "deficit":
		config.PressureDeficitPa = value
	case "recovery":
		config.RecoveryPaPerM = value
	case "rpm":
		config.RPM = value
	case "contrast":
		config.FragmentContrast = value
	case "scale":
		config.FragmentScaleM = value
	default:
		t.Fatalf("the invalid file names a field, %q, the test does not know",
			field)
	}
}

// propellerCavitationInvalid is a row of the invalid file, to be run as a
// test under its name.
type propellerCavitationInvalid struct{ row map[string]string }

// refused holds that the bounded configuration, changed as the row says, is
// refused.
func (invalid propellerCavitationInvalid) refused(t *testing.T) {
	config := propellerCavitationConfigNamed(t, "bounded")
	propellerCavitationApply(t, &config, invalid.row["field"],
		cell(t, invalid.row, "value"))
	propellerCavitationApply(t, &config, invalid.row["other_field"],
		cell(t, invalid.row, "other_value"))
	if _, err := NewPropellerCavitation(config); err == nil {
		t.Fatal("invalid cavitation field was accepted")
	}
}

// TestPropellerCavitationRejectsUnboundedOrInvalidInputs holds that each
// configuration in the invalid file is refused.
func TestPropellerCavitationRejectsUnboundedOrInvalidInputs(t *testing.T) {
	for _, row := range readSheet(t, propellerCavitationInvalidFile) {
		t.Run(row["name"], propellerCavitationInvalid{row}.refused)
	}
}
