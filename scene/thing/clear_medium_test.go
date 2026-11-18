package thing

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestClearMediumCopiesFinitePropertiesAndConvertsWorldUnits(t *testing.T) {
	medium, err := NewClearMedium("ocean", maths.RGB(0.2, 0.1, 0.05), 2)
	if err != nil {
		t.Fatal(err)
	}
	if !medium.Defined() || medium.Name() != "ocean" ||
		medium.Extinction() != maths.RGB(0.2, 0.1, 0.05) ||
		medium.MetresPerUnit() != 2 {
		t.Fatalf("clear medium did not preserve its checked values: %+v", medium)
	}
}

func TestClearMediumRejectsInvalidIdentityCoefficientsAndUnits(t *testing.T) {
	for _, row := range []struct {
		test       string
		name       string
		extinction maths.Light
		unit       float64
	}{
		{"blank name", "", maths.Grey(0.1), 1},
		{"untrimmed name", " water", maths.Grey(0.1), 1},
		{"negative extinction", "water", maths.RGB(-0.1, 0.1, 0.1), 1},
		{"nan extinction", "water", maths.RGB(math.NaN(), 0.1, 0.1), 1},
		{"infinite extinction", "water", maths.RGB(math.Inf(1), 0.1, 0.1), 1},
		{"zero unit", "water", maths.Grey(0.1), 0},
		{"negative unit", "water", maths.Grey(0.1), -1},
		{"infinite unit", "water", maths.Grey(0.1), math.Inf(1)},
	} {
		t.Run(row.test, func(t *testing.T) {
			if _, err := NewClearMedium(row.name, row.extinction, row.unit); err == nil {
				t.Fatal("invalid clear medium was accepted")
			}
		})
	}
}

func TestParticipatingClearMediumKeepsAbsorptionScatteringAndBound(t *testing.T) {
	medium, err := NewClearMediumWithScattering("synthetic", maths.RGB(0.1, 0.2, 0.3),
		maths.RGB(0.4, 0.5, 0.6), 0, 2, 8, 12)
	if err != nil {
		t.Fatal(err)
	}
	if got := medium.Absorption(); got != maths.RGB(0.1, 0.2, 0.3) {
		t.Errorf("absorption = %v", got)
	}
	if got := medium.Scattering(); got != maths.RGB(0.4, 0.5, 0.6) {
		t.Errorf("scattering = %v", got)
	}
	absorption, scattering := medium.Absorption(), medium.Scattering()
	wantExtinction := maths.RGB(absorption.R+scattering.R,
		absorption.G+scattering.G, absorption.B+scattering.B)
	if got := medium.Extinction(); got != wantExtinction {
		t.Errorf("extinction = %v", got)
	}
	if got := medium.Phase(); got != 0 {
		t.Errorf("phase = %g, want existing gas zero convention (Rayleigh)", got)
	}
	if got := medium.InscatterSteps(); got != 8 {
		t.Errorf("steps = %d", got)
	}
	if got := medium.MaxInscatterDistanceMetres(); got != 12 {
		t.Errorf("maximum path = %g", got)
	}
	if !medium.HasScattering() {
		t.Fatal("positive scattering was not enabled")
	}
}

func TestDiffuseSkyInscatterIsSeparateImmutableOptIn(t *testing.T) {
	base, err := NewClearMediumWithScattering("sky-water", maths.Grey(.1),
		maths.Grey(.02), .2, 1, 8, 12)
	if err != nil {
		t.Fatal(err)
	}
	if base.DiffuseSkyInscatterSamples() != 0 {
		t.Fatal("default clear medium unexpectedly samples the diffuse sky")
	}
	configured, err := base.WithDiffuseSkyInscatter(32)
	if err != nil {
		t.Fatal(err)
	}
	if configured == base || configured.DiffuseSkyInscatterSamples() != 32 ||
		base.DiffuseSkyInscatterSamples() != 0 {
		t.Fatalf("sky opt-in did not return an independent configured value: base=%d configured=%d",
			base.DiffuseSkyInscatterSamples(),
			configured.DiffuseSkyInscatterSamples())
	}
	for _, samples := range []int{-1, 0, maxClearSkyInscatterSamples + 1} {
		if _, err := base.WithDiffuseSkyInscatter(samples); err == nil {
			t.Errorf("accepted invalid diffuse-sky sample count %d", samples)
		}
	}
	legacy, err := NewClearMedium("extinction-only", maths.Grey(.1), 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.WithDiffuseSkyInscatter(8); err == nil {
		t.Fatal("accepted sky source integration without participating scattering")
	}
}

func TestParticipatingClearMediumRefusesInvalidCoefficientsPhaseAndWork(t *testing.T) {
	for name, values := range map[string]struct {
		absorb, scatter maths.Light
		phase, units    float64
		steps           int
		limit           float64
	}{
		"negative scattering":     {maths.Grey(0), maths.RGB(-1, 0, 0), 0.5, 1, 4, 10},
		"overflow extinction":     {maths.Grey(math.MaxFloat64), maths.Grey(math.MaxFloat64), 0.5, 1, 4, 10},
		"phase outside lobe":      {maths.Grey(0), maths.Grey(1), 1, 1, 4, 10},
		"nan phase":               {maths.Grey(0), maths.Grey(1), math.NaN(), 1, 4, 10},
		"zero steps":              {maths.Grey(0), maths.Grey(1), 0.5, 1, 0, 10},
		"too many steps":          {maths.Grey(0), maths.Grey(1), 0.5, 1, maxClearInscatterSteps + 1, 10},
		"zero distance limit":     {maths.Grey(0), maths.Grey(1), 0.5, 1, 4, 0},
		"infinite distance limit": {maths.Grey(0), maths.Grey(1), 0.5, 1, 4, math.Inf(1)},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := NewClearMediumWithScattering(name, values.absorb,
				values.scatter, values.phase, values.units, values.steps,
				values.limit)
			if err == nil {
				t.Fatal("invalid participating medium was accepted")
			}
		})
	}
}

func TestLegacyClearMediumRemainsPureExtinctionAndNoScatter(t *testing.T) {
	medium, err := NewClearMedium("legacy", maths.RGB(0.1, 0.2, 0.3), 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := medium.Absorption(); got != maths.RGB(0.1, 0.2, 0.3) {
		t.Errorf("legacy absorption = %v", got)
	}
	if got := medium.Scattering(); got != (maths.Light{}) {
		t.Errorf("legacy scattering = %v", got)
	}
	if medium.HasScattering() || medium.InscatterSteps() != 0 ||
		medium.MaxInscatterDistanceMetres() != 0 {
		t.Error("legacy constructor unexpectedly enabled source integration")
	}
}
