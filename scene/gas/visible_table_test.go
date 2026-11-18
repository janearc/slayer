package gas

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

func sampleVisible(t *testing.T) VisibleTable {
	t.Helper()
	got, err := NewVisibleTable([]VisibleSample{
		{Nanometres: 465, Absorb: 0.1, Scatter: 0.4},
		{Nanometres: 550, Absorb: 0.2, Scatter: 0.3},
		{Nanometres: 610, Absorb: 0.4, Scatter: 0.1},
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestVisibleTableInterpolatesCallerCoefficientsAndUsesRGBBandBasis(t *testing.T) {
	table := sampleVisible(t)
	for _, row := range []struct {
		band            float64
		absorb, scatter float64
	}{
		{465, 0.1, 0.4},
		{550, 0.2, 0.3},
		{610, 0.4, 0.1},
		{507.5, 0.15, 0.35},
	} {
		got, err := table.At(row.band)
		if err != nil || math.Abs(got.Absorb-row.absorb) > 1e-15 ||
			math.Abs(got.Scatter-row.scatter) > 1e-15 {
			t.Errorf("at %g nm: optics %+v, error %v; want absorb %g scatter %g",
				row.band, got, err, row.absorb, row.scatter)
		}
	}
	gas := table.Gas("authored sample")
	for _, band := range []struct {
		nm   float64
		want float64
	}{{610, math.Exp(-1)}, {550, math.Exp(-1)}, {465, math.Exp(-1)}} {
		if got := gas.Transmit(maths.Band{Nanometres: band.nm}, 2); math.Abs(got-band.want) > 1e-15 {
			t.Errorf("transmission at %g nm = %.17g, want %.17g", band.nm, got, band.want)
		}
	}
}

func TestVisibleTableUsesBeerLambertAndHandlesNumericalEdges(t *testing.T) {
	table, err := NewVisibleTable([]VisibleSample{
		{Nanometres: 465, Absorb: 0, Scatter: 0},
		{Nanometres: 550, Absorb: 0.2, Scatter: 0.3},
		{Nanometres: 610, Absorb: 0.2, Scatter: 0.3},
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		nm, distance, want float64
	}{
		{550, 2, math.Exp(-1)},
		{465, 1e300, 1},
		{610, 0, 1},
		{610, math.MaxFloat64, 0},
		{610, math.Inf(1), 0},
		{465, math.Inf(1), 1},
	} {
		got, err := table.Transmit(row.nm, row.distance)
		if err != nil || math.Abs(got-row.want) > 1e-15 {
			t.Errorf("transmit(%g nm, %g m) = %.17g, error %v; want %.17g",
				row.nm, row.distance, got, err, row.want)
		}
	}
}

func TestVisibleTableDelegatesHomogeneousSingleScatterToGas(t *testing.T) {
	table, err := NewVisibleTable([]VisibleSample{
		{Nanometres: 465, Absorb: 0.1, Scatter: 0.25},
		{Nanometres: 610, Absorb: 0.1, Scatter: 0.25},
	}, 0.4)
	if err != nil {
		t.Fatal(err)
	}
	gas := table.Gas("test")
	light := maths.RGB(2, 3, 5)
	got := gas.Inscatter(maths.Band{Nanometres: 550}, light, 0.2)
	want := light.Scale(0.25 * gas.PhaseAt(0.2))
	if got != want {
		t.Fatalf("single-scatter source %v, want %v", got, want)
	}
}

func TestVisibleTablePhaseZeroUsesRayleighNotIsotropicHenyeyGreenstein(t *testing.T) {
	table, err := NewVisibleTable([]VisibleSample{
		{Nanometres: 465},
		{Nanometres: 610},
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := table.Gas("zero phase").PhaseAt(0)
	wantRayleigh := 3 / (16 * math.Pi)
	isotropicHG := 1 / (4 * math.Pi)
	if got != wantRayleigh || got == isotropicHG {
		t.Fatalf("phase zero gives %g, want Rayleigh %g and not isotropic HG %g",
			got, wantRayleigh, isotropicHG)
	}
}

func TestVisibleTableIntegratesHomogeneousSingleScatterWithExplicitAssumptions(t *testing.T) {
	table, err := NewVisibleTable([]VisibleSample{
		{Nanometres: 465, Absorb: 0.2, Scatter: 0.3},
		{Nanometres: 610, Absorb: 0.2, Scatter: 0.3},
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Incident light is held constant throughout a unit-density two-metre
	// slab; only the outgoing path is attenuated. The integrated source is
	// incident * beta_s * Rayleigh(cosine) * (1-exp(-sigma_t*d))/sigma_t.
	incident := maths.RGB(2, 3, 5)
	phase := 3 / (16 * math.Pi) // Rayleigh at cosine zero
	length := -math.Expm1(-1) / 0.5
	want := incident.Scale(0.3 * phase * length)
	got, err := table.SingleScatter(550, 2, incident, 0)
	if err != nil || math.Abs(got.R-want.R) > 1e-15 ||
		math.Abs(got.G-want.G) > 1e-15 || math.Abs(got.B-want.B) > 1e-15 {
		t.Fatalf("integrated source %v, error %v; want %v", got, err, want)
	}
	for _, cosine := range []float64{-1.01, 1.01, math.NaN()} {
		if _, err := table.SingleScatter(550, 2, incident, cosine); err == nil {
			t.Errorf("accepted phase cosine %g", cosine)
		}
	}
}

func TestVisibleTableSingleScatterKeepsBlackSourceFiniteAtExtremeOpticalDepth(t *testing.T) {
	table, err := NewVisibleTable([]VisibleSample{
		{Nanometres: 465, Scatter: math.SmallestNonzeroFloat64},
		{Nanometres: 610, Scatter: math.SmallestNonzeroFloat64},
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	got, err := table.SingleScatter(550, math.Inf(1), maths.Light{}, 0)
	if err != nil || got != (maths.Light{}) {
		t.Fatalf("black incident beam over extreme path = %v, error %v; want finite zero", got, err)
	}
}

func TestNewVisibleTableRefusesInvalidCoefficientsWavelengthsAndPhase(t *testing.T) {
	base := []VisibleSample{{Nanometres: 465}, {Nanometres: 610}}
	for _, row := range []struct {
		name    string
		samples []VisibleSample
		phase   float64
	}{
		{"too few samples", base[:1], 0},
		{"nonpositive wavelength", []VisibleSample{{Nanometres: 0}, {Nanometres: 610}}, 0},
		{"nonfinite wavelength", []VisibleSample{{Nanometres: 465}, {Nanometres: math.Inf(1)}}, 0},
		{"unordered wavelengths", []VisibleSample{{Nanometres: 610}, {Nanometres: 465}}, 0},
		{"duplicate wavelengths", []VisibleSample{{Nanometres: 465}, {Nanometres: 465}}, 0},
		{"negative absorption", []VisibleSample{{Nanometres: 465, Absorb: -1}, {Nanometres: 610}}, 0},
		{"infinite absorption", []VisibleSample{{Nanometres: 465}, {Nanometres: 610, Absorb: math.Inf(1)}}, 0},
		{"nan scattering", []VisibleSample{{Nanometres: 465}, {Nanometres: 610, Scatter: math.NaN()}}, 0},
		{"negative phase limit", base, -1},
		{"positive phase limit", base, 1},
		{"nonfinite phase", base, math.NaN()},
	} {
		t.Run(row.name, func(t *testing.T) {
			if _, err := NewVisibleTable(row.samples, row.phase); err == nil {
				t.Fatal("invalid visible table was accepted")
			}
		})
	}
}

func TestVisibleTableRefusesOutOfRangeWavelengthAndInvalidDistance(t *testing.T) {
	table := sampleVisible(t)
	for _, nm := range []float64{0, math.NaN(), math.Inf(1), 464.9, 610.1} {
		if _, err := table.At(nm); err == nil {
			t.Errorf("accepted wavelength %g nm", nm)
		}
	}
	for _, distance := range []float64{-1, math.NaN(), math.Inf(-1)} {
		if _, err := table.Transmit(550, distance); err == nil {
			t.Errorf("accepted path distance %g m", distance)
		}
	}
}

func TestVisibleTableRefusesFiniteCoefficientsWhoseExtinctionOverflows(t *testing.T) {
	samples := []VisibleSample{
		{Nanometres: 465, Absorb: math.MaxFloat64, Scatter: math.MaxFloat64},
		{Nanometres: 610, Absorb: math.MaxFloat64, Scatter: math.MaxFloat64},
	}
	if _, err := NewVisibleTable(samples, 0); err == nil {
		t.Fatal("accepted finite coefficients whose total extinction overflows")
	}
}

func TestGasPhaseIsFiniteAtAsymmetriesImmediatelyInsideLimits(t *testing.T) {
	for _, row := range []struct {
		phase, cosine float64
	}{
		{math.Nextafter(1, 0), 1},
		{math.Nextafter(-1, 0), -1},
	} {
		gas := Gas{Phase: row.phase}
		got := gas.PhaseAt(row.cosine)
		if !finite(got) || got <= 0 {
			t.Errorf("phase %g at cosine %g is %g, want finite positive phase",
				row.phase, row.cosine, got)
		}
	}
}

func TestGasPhaseKeepsPreviousArithmeticBitForBitForOrdinaryInputs(t *testing.T) {
	for _, row := range []struct {
		phase, cosine float64
	}{
		{0.1, -0.7},
		{0.4, 0.2},
		{-0.6, 0.8},
		{0.99, -0.25},
		{-0.97, 0.1},
	} {
		want := (1 - row.phase*row.phase) /
			(4 * math.Pi * math.Pow(1+row.phase*row.phase-2*row.phase*row.cosine, 1.5))
		got := (Gas{Phase: row.phase}).PhaseAt(row.cosine)
		if math.Float64bits(got) != math.Float64bits(want) {
			t.Errorf("phase at g=%g cos=%g changed bits: got %.17g, want %.17g",
				row.phase, row.cosine, got, want)
		}
	}
}

func TestVisibleTableOwnsItsSampleData(t *testing.T) {
	samples := []VisibleSample{{Nanometres: 465, Absorb: 0.1}, {Nanometres: 610, Absorb: 0.2}}
	table, err := NewVisibleTable(samples, 0)
	if err != nil {
		t.Fatal(err)
	}
	samples[0].Absorb = 99
	got, err := table.At(465)
	if err != nil || got.Absorb != 0.1 {
		t.Fatalf("caller mutation changed table: %+v, %v", got, err)
	}
	gas := table.Gas("owned")
	gotAbsorb, _ := gas.Visible(465)
	if gotAbsorb != 0.1 {
		t.Fatalf("caller mutation changed gas callback: %g", gotAbsorb)
	}
}
