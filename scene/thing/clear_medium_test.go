package thing

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// The csv files the clear medium tests read.
const (
	clearMediumLights        = "testdata/clear_medium-lights.csv"
	clearMediumInvalid       = "testdata/clear_medium-invalid.csv"
	clearMediumParticipating = "testdata/clear_medium-participating.csv"
	clearMediumRefusals      = "testdata/clear_medium-refusals.csv"
)

// clearMediumRow is one row of a clear medium table.
type clearMediumRow map[string]string

// clearMediumLight is the colour in a row's three columns, named by the
// prefix and Red, Green and Blue.
func clearMediumLight(t *testing.T, row map[string]string,
	prefix string) maths.Light {
	t.Helper()
	return maths.RGB(cell(t, row, prefix+"Red"),
		cell(t, row, prefix+"Green"), cell(t, row, prefix+"Blue"))
}

// clearMediumSteps is a row's in-scatter steps: the most the medium allows
// and one more where the row says it is over the limit.
func clearMediumSteps(t *testing.T, row map[string]string) int {
	t.Helper()
	if cellCount(t, row, "overLimit") != nought {
		return maxClearInscatterSteps + unity
	}
	return cellCount(t, row, "steps")
}

// clearMediumBuild is the participating medium a row describes, with the
// error the constructor gave.
func clearMediumBuild(t *testing.T,
	row map[string]string) (*ClearMedium, error) {
	t.Helper()
	return NewClearMediumWithScattering(row["name"],
		clearMediumLight(t, row, "absorption"),
		clearMediumLight(t, row, "scattering"), cell(t, row, "phase"),
		cell(t, row, "unit"), clearMediumSteps(t, row),
		cell(t, row, "limit"))
}

// clearMediumNamed is the row called name in a clear medium table.
func clearMediumNamed(t *testing.T, file, name string) map[string]string {
	t.Helper()
	rows := readSheet(t, file)
	return rows.row(t, file, name)
}

// refusesIdentity is a subtest: the row's name, extinction or unit is
// refused by NewClearMedium.
func (row clearMediumRow) refusesIdentity(t *testing.T) {
	extinction := clearMediumLight(t, row, "extinction")
	_, err := NewClearMedium(row["identity"], extinction,
		cell(t, row, "unit"))
	if err == nil {
		t.Fatal("invalid clear medium was accepted")
	}
}

// refusesWork is a subtest: the row's coefficients, phase, steps or
// distance limit are refused by NewClearMediumWithScattering.
func (row clearMediumRow) refusesWork(t *testing.T) {
	if _, err := clearMediumBuild(t, row); err == nil {
		t.Fatal("invalid participating medium was accepted")
	}
}

// a clear medium keeps the extinction, the unit and the name it was
// given, and says it is defined.
func TestClearMediumCopiesFinitePropertiesAndConvertsWorldUnits(t *testing.T) {
	row := clearMediumNamed(t, clearMediumLights, "ocean")
	extinction := clearMediumLight(t, row, "extinction")
	unit := cell(t, row, "unit")
	medium, err := NewClearMedium("ocean", extinction, unit)
	if err != nil {
		t.Fatal(err)
	}
	if !medium.Defined() || medium.Name() != "ocean" ||
		medium.Extinction() != extinction ||
		medium.MetresPerUnit() != unit {
		t.Fatalf("clear medium did not preserve its checked values: %+v",
			medium)
	}
}

// the anti-test: a blank or untrimmed name, a negative, not-a-number or
// infinite extinction, and a zero, negative or infinite unit are refused.
func TestClearMediumRejectsInvalidIdentityCoefficientsAndUnits(t *testing.T) {
	for _, each := range readSheet(t, clearMediumInvalid) {
		t.Run(each["name"], clearMediumRow(each).refusesIdentity)
	}
}

// a participating clear medium keeps its absorption, scattering, phase,
// steps and distance bound, and its extinction is their sum.
func TestParticipatingClearMediumKeepsAbsorptionScatteringAndBound(t *testing.T) {
	row := clearMediumNamed(t, clearMediumParticipating, "synthetic")
	medium, err := clearMediumBuild(t, row)
	if err != nil {
		t.Fatal(err)
	}
	wantAbsorption := clearMediumLight(t, row, "absorption")
	if got := medium.Absorption(); got != wantAbsorption {
		t.Errorf("absorption = %v", got)
	}
	wantScattering := clearMediumLight(t, row, "scattering")
	if got := medium.Scattering(); got != wantScattering {
		t.Errorf("scattering = %v", got)
	}
	clearMediumCheckBound(t, medium, row)
}

// clearMediumCheckBound checks a participating medium's extinction, phase,
// steps and distance bound against its row.
func clearMediumCheckBound(t *testing.T, medium *ClearMedium,
	row map[string]string) {
	t.Helper()
	absorption, scattering := medium.Absorption(), medium.Scattering()
	wantExtinction := maths.RGB(absorption.R+scattering.R,
		absorption.G+scattering.G, absorption.B+scattering.B)
	if got := medium.Extinction(); got != wantExtinction {
		t.Errorf("extinction = %v", got)
	}
	if got := medium.Phase(); got != nought {
		t.Errorf("phase = %g, want existing gas zero convention "+
			"(Rayleigh)", got)
	}
	wantSteps, wantLimit := cellCount(t, row, "steps"), cell(t, row, "limit")
	if got := medium.InscatterSteps(); got != wantSteps {
		t.Errorf("steps = %d", got)
	}
	if got := medium.MaxInscatterDistanceMetres(); got != wantLimit {
		t.Errorf("maximum path = %g", got)
	}
	if !medium.HasScattering() {
		t.Fatal("positive scattering was not enabled")
	}
}

// the sky opt-in is off by default, returns an independent configured
// value, refuses a bad count and refuses a medium that does not scatter.
func TestDiffuseSkyInscatterIsSeparateImmutableOptIn(t *testing.T) {
	row := clearMediumNamed(t, clearMediumParticipating, "sky-water")
	base, err := clearMediumBuild(t, row)
	if err != nil {
		t.Fatal(err)
	}
	if base.DiffuseSkyInscatterSamples() != nought {
		t.Fatal("default clear medium unexpectedly samples the diffuse sky")
	}
	samples := pickedCount(t, "clear_medium-sky-samples")
	configured, err := base.WithDiffuseSkyInscatter(samples)
	if err != nil {
		t.Fatal(err)
	}
	if configured == base ||
		configured.DiffuseSkyInscatterSamples() != samples ||
		base.DiffuseSkyInscatterSamples() != nought {
		t.Fatalf("sky opt-in did not return an independent configured "+
			"value: base=%d configured=%d",
			base.DiffuseSkyInscatterSamples(),
			configured.DiffuseSkyInscatterSamples())
	}
	clearMediumCheckSkyCounts(t, base)
	clearMediumCheckSkyNeedsScattering(t)
}

// clearMediumCheckSkyCounts checks a medium refuses a negative count, a
// zero count and a count above the most it allows.
func clearMediumCheckSkyCounts(t *testing.T, base *ClearMedium) {
	t.Helper()
	negative := pickedCount(t, "clear_medium-sky-negative-samples")
	for _, samples := range []int{negative, nought,
		maxClearSkyInscatterSamples + unity} {
		if _, err := base.WithDiffuseSkyInscatter(samples); err == nil {
			t.Errorf("accepted invalid diffuse-sky sample count %d", samples)
		}
	}
}

// clearMediumCheckSkyNeedsScattering checks a medium of extinction alone
// refuses the sky source integration.
func clearMediumCheckSkyNeedsScattering(t *testing.T) {
	t.Helper()
	row := clearMediumNamed(t, clearMediumLights, "grey")
	extinctionOnly, err := NewClearMedium("extinction-only",
		clearMediumLight(t, row, "extinction"), cell(t, row, "unit"))
	if err != nil {
		t.Fatal(err)
	}
	samples := pickedCount(t, "clear_medium-sky-unscattered-samples")
	_, err = extinctionOnly.WithDiffuseSkyInscatter(samples)
	if err == nil {
		t.Fatal("accepted sky source integration without " +
			"participating scattering")
	}
}

// the anti-test: negative scattering, an overflowing extinction, a phase
// outside the lobe or not a number, no steps or too many, and a zero or
// infinite distance limit are each refused.
func TestParticipatingClearMediumRefusesInvalidCoefficientsPhaseAndWork(t *testing.T) {
	for _, each := range readSheet(t, clearMediumRefusals) {
		t.Run(each["name"], clearMediumRow(each).refusesWork)
	}
}

// A clear medium made from a total extinction is pure extinction: its
// absorption is that total, and it scatters nothing and integrates no
// source.
func TestAnExtinctionOnlyClearMediumIsPureExtinctionAndNoScatter(t *testing.T) {
	row := clearMediumNamed(t, clearMediumLights, "spread")
	extinction := clearMediumLight(t, row, "extinction")
	medium, err := NewClearMedium("extinction-only", extinction,
		cell(t, row, "unit"))
	if err != nil {
		t.Fatal(err)
	}
	if got := medium.Absorption(); got != extinction {
		t.Errorf("extinction-only absorption = %v", got)
	}
	if got := medium.Scattering(); got != maths.Grey(nought) {
		t.Errorf("extinction-only scattering = %v", got)
	}
	if medium.HasScattering() || medium.InscatterSteps() != nought ||
		medium.MaxInscatterDistanceMetres() != nought {
		t.Error("an extinction-only medium enabled source integration")
	}
}
