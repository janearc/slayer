package airs

import (
	"math"
	"testing"
)

// relative is how far a is from b, as a share of b.
func relative(got, want float64) float64 {
	return math.Abs(got-want) / math.Abs(want)
}

// columnAt is Venus's column at a wavelength of the data given, the test
// stopped where it is refused.
func columnAt(t *testing.T, from data, nanometres float64) Column {
	t.Helper()
	column, err := from.column(nanometres)
	if err != nil {
		t.Fatalf("%g nm: %v", nanometres, err)
	}
	return column
}

// Carbon dioxide's refractive index at 550 nm, worked by hand from
// Bideau-Mehu's coefficients, term by term: about 4.502e-4 over one.
// This checks the code's arithmetic against the working; the working is
// not a source.
func TestCarbonDioxideIndexAt550IsWhatHandWorkingGives(t *testing.T) {
	const worked = 4.502e-4 // picked: a hand working of the formula at 0.55 um, 2026-11-15
	index, err := venusRead.Gas.Refraction.index(550)
	if err != nil {
		t.Fatal(err)
	}
	if miss := relative(index-1, worked); miss > 1e-3 { // picked: the hand working's own digits
		t.Errorf("n - 1 = %g, hand working %g", index-1, worked)
	}
}

// The index is not the same at every wavelength: it is larger in the
// blue, which is why blue scatters more.
func TestCarbonDioxideIndexRisesTowardTheBlue(t *testing.T) {
	blue, err := venusRead.Gas.Refraction.index(450)
	if err != nil {
		t.Fatal(err)
	}
	green, err := venusRead.Gas.Refraction.index(550)
	if err != nil {
		t.Fatal(err)
	}
	if blue-1 < 1.01*(green-1) { // picked: a percent, which dispersion exceeds
		t.Errorf("n - 1 is %g at 450 nm and %g at 550", blue-1, green-1)
	}
}

// Carbon dioxide's cross section at 550 nm, worked by hand from the
// index, 24 pi^3 over lambda^4 and the molecules at the index's
// conditions squared, and king's 1.15: about 1.167e-30 square metres.
func TestCarbonDioxideCrossSectionAt550IsWhatHandWorkingGives(t *testing.T) {
	const worked = 1.167e-30 // picked: a hand working, 2026-11-15
	got, err := venusRead.Gas.carbonDioxideCross(550)
	if err != nil {
		t.Fatal(err)
	}
	if miss := relative(got, worked); miss > 0.01 { // picked: a percent, the working's own digits
		t.Errorf("cross section %g, hand working %g", got, worked)
	}
	air, err := venusRead.Gas.standInCross(550)
	if err != nil {
		t.Fatal(err)
	}
	if got < 2*air || got > 3*air { // picked: carbon dioxide's is about 2.6 times air's
		t.Errorf("carbon dioxide %g against air %g", got, air)
	}
}

// The Rayleigh optical depth of the whole column at 550 nm is 16, give
// or take 2: a hand working of 2026-11-15, from Sneep and Ubachs'
// cross section. A check against an independent working, not a source.
func TestRayleighDepthOfTheColumnAt550(t *testing.T) {
	const worked, give = 16.0, 2.0 // picked: a hand working and its give, 2026-11-15
	depth := columnAt(t, venusRead, 550).GasDepth()
	if math.Abs(depth-worked) > give {
		t.Errorf("rayleigh depth %g, working %g +- %g", depth, worked, give)
	}
}

// Nitrogen alone, without carbon dioxide's strength, would give a depth
// far from the working's: the test above would notice.
func TestRayleighDepthOfNitrogenAloneIsFarFromIt(t *testing.T) {
	changed := copied(t)
	changed.Gas.CarbonDioxide.Value, changed.Gas.Nitrogen.Value = 0, 1
	depth := columnAt(t, changed, 550).GasDepth()
	if depth >= 14 { // picked: the working's 16 less its give of 2
		t.Errorf("nitrogen alone gives %g", depth)
	}
}

// The depth falls as about the fourth power of wavelength.
func TestRayleighDepthFallsAsTheFourthPowerOfWavelength(t *testing.T) {
	blue := columnAt(t, venusRead, 450).GasDepth()
	red := columnAt(t, venusRead, 750).GasDepth()
	want := math.Pow(750.0/450.0, 4)                  // defined: rayleigh's law
	if miss := relative(blue/red, want); miss > 0.1 { // picked: dispersion adds a few percent
		t.Errorf("depth %g at 450 nm over %g at 750: %g, not near %g",
			blue, red, blue/red, want)
	}
}

// The particles do not fall so: their ratio is far from the fourth
// power, so the test above is about the gas.
func TestParticleDepthDoesNotFallAsTheFourthPower(t *testing.T) {
	blue := columnAt(t, venusRead, 450).ParticleDepth()
	red := columnAt(t, venusRead, 750).ParticleDepth()
	if blue/red > 2 { // picked: far under the fourth power's 7.7
		t.Errorf("particle depth %g over %g", blue, red)
	}
}

// The column's mass a square metre is the surface pressure over gravity:
// the exponential stand-in for VIRA is held to 92 bar, within 2 percent.
func TestColumnMassIsTheSurfacePressureOverGravity(t *testing.T) {
	const bar = 1e5 // defined: pascals in a bar
	gas := venusRead.Gas
	want := venusRead.Planet.SurfacePressure.Value * bar /
		venusRead.Planet.Gravity.Value
	if miss := relative(gas.columnMass(), want); miss > 0.02 { // picked: the brief's 2 percent
		t.Errorf("column mass %g kg m^-2, 92 bar over g is %g", gas.columnMass(), want)
	}
}

// A scale height of ten kilometres would not hold to it.
func TestAShorterScaleHeightFailsTheSurfacePressure(t *testing.T) {
	const bar = 1e5 // defined: pascals in a bar
	gas := venusRead.Gas
	gas.ScaleHeight.Value = 10000 // picked: ten kilometres
	want := venusRead.Planet.SurfacePressure.Value * bar /
		venusRead.Planet.Gravity.Value
	if relative(gas.columnMass(), want) < 0.02 {
		t.Errorf("column mass %g still holds to %g", gas.columnMass(), want)
	}
}

// Each shell's mean density is the mass it holds over its thickness, so
// that the shells together hold the column's whole mass.
func TestShellsHoldTheWholeColumn(t *testing.T) {
	edges := venusRead.edges()
	held := 0.0
	for index := 1; index < len(edges); index++ {
		held += venusRead.Gas.meanDensity(edges[index-1], edges[index]) *
			(edges[index] - edges[index-1])
	}
	if miss := relative(held, venusRead.Gas.columnMass()); miss > 1e-9 { // picked: rounding only
		t.Errorf("shells hold %g, the column %g", held, venusRead.Gas.columnMass())
	}
	taller := venusRead.Gas
	taller.Top.Value *= 2
	if relative(held, taller.columnMass()) < 1e-6 { // picked: far over rounding
		t.Errorf("a taller column holds the same")
	}
}

// A wavelength the carbon dioxide's index is not known at is refused,
// and one inside it is not.
func TestAWavelengthOutsideTheIndexIsRefused(t *testing.T) {
	for _, nanometres := range []float64{100, 2000, 0, math.NaN()} { // picked: either side, nought, and no number
		if _, err := Venus(nanometres); err == nil {
			t.Errorf("%g nm was taken", nanometres)
		}
	}
	if _, err := Venus(1000); err != nil { // picked: inside, and past the ground's table
		t.Errorf("1000 nm: %v", err)
	}
}
