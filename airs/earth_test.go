package airs

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/celestial/atmosphere"
	"github.com/janearc/celestial/measure"
)

// earthWavelengths are the wavelengths the Earth tests look at, in
// nanometres: the ozone table's two ends and the visible between them.
var earthWavelengths = []float64{300, 350, 400, 450, 500, 550, 600, 650, 700, 750, 800, 1000} // picked: the table's range, in steps that reach the Chappuis band

// copiedEarth is Earth's data as shipped, with every slice in it copied,
// so that a test may change them.
func copiedEarth(t *testing.T) earthData {
	t.Helper()
	if earthError != nil {
		t.Fatal(earthError)
	}
	changed := earthRead
	changed.Shells.Runs = slices.Clone(earthRead.Shells.Runs)
	changed.Ozone.Layers = slices.Clone(earthRead.Ozone.Layers)
	changed.Ozone.CrossSection.Points = slices.Clone(earthRead.Ozone.CrossSection.Points)
	changed.Aerosol.Albedo.Points = slices.Clone(earthRead.Aerosol.Albedo.Points)
	changed.Aerosol.Asymmetry.Points = slices.Clone(earthRead.Aerosol.Asymmetry.Points)
	changed.Ground.Points = slices.Clone(earthRead.Ground.Points)
	return changed
}

// earthColumnAt is Earth's column at a wavelength of the data given, the
// test stopped where it is refused.
func earthColumnAt(t *testing.T, from earthData, nanometres float64) Column {
	t.Helper()
	column, err := from.column(nanometres)
	if err != nil {
		t.Fatalf("%g nm: %v", nanometres, err)
	}
	return column
}

// celestialDepth is celestial's own Rayleigh optical depth, straight up
// from the ground, at a wavelength in nanometres: Bodhaine and others'
// equation 25, from the pressure at the ground and a latitude.
func celestialDepth(t *testing.T, nanometres float64) float64 {
	t.Helper()
	air, err := atmosphere.DryAir(earthRead.Gas.CarbonDioxide.Value)
	if err != nil {
		t.Fatal(err)
	}
	depth, err := air.DepthAbove(measure.Metres(nanometres*1e-9), // defined: metres in a nanometre
		atmosphere.Pascals(earthRead.Planet.SurfacePressure.Value),
		measure.Degrees(45), measure.Metres(0)) // picked: mid-latitude, at sea level
	if err != nil {
		t.Fatal(err)
	}
	return *depth
}

// The shells run without a gap from the ground to the top of the
// standard atmosphere, 86 km, at every wavelength, the thinnest at the
// ground and the thickest aloft.
func TestEarthShellsRunFromTheGroundToTheTopWithoutGaps(t *testing.T) {
	for _, nanometres := range earthWavelengths {
		column := earthColumnAt(t, earthRead, nanometres)
		if found := faults(column, 86000); len(found) > 0 { // defined: the standard atmosphere's top, 86 km
			t.Errorf("%g nm: %v", nanometres, found)
		}
	}
	column := earthColumnAt(t, earthRead, 550) // picked: the middle of the visible
	last := column.Layers[len(column.Layers)-1]
	if column.Layers[0].Top != 500 || last.Top-last.Bottom != 6000 { // defined: the first and last runs' thicknesses
		t.Errorf("the lowest shell ends at %g, the highest is %g thick",
			column.Layers[0].Top, last.Top-last.Bottom)
	}
}

// A stack of Earth's shells with one taken out, or one set over another,
// is found, as Venus's is.
func TestEarthStackWithAGapOrAnOverlapIsFound(t *testing.T) {
	column := earthColumnAt(t, earthRead, 550)                                                     // picked: the middle of the visible
	gap := Column{Layers: append(append([]Layer{}, column.Layers[:20]...), column.Layers[21:]...)} // picked: the twenty-first shell taken out
	if len(faults(gap, 86000)) == 0 {                                                              // defined: the standard atmosphere's top, 86 km
		t.Error("a gap was not found")
	}
	overlap := Column{Layers: append([]Layer{column.Layers[0]}, column.Layers...)}
	if len(faults(overlap, 86000)) == 0 { // defined: as above
		t.Error("an overlap was not found")
	}
}

// Two codes, one answer: the Rayleigh optical depth of the whole column
// is celestial's own, from the pressure at the ground, to a percent, at
// every wavelength.
func TestEarthRayleighDepthIsCelestials(t *testing.T) {
	for _, nanometres := range earthWavelengths {
		got := earthColumnAt(t, earthRead, nanometres).gasDepth()
		want := celestialDepth(t, nanometres)
		if miss := relative(got, want); miss > 0.01 { // picked: the brief's 1 percent
			t.Errorf("%g nm: column depth %g, celestial's %g", nanometres, got, want)
		}
	}
}

// A third working: Levy and others 2007's Table 2 (NTRS 20070016619,
// p 26) gives the sea-level Rayleigh optical depth for three MODIS
// channels, which the column holds to four percent.
func TestEarthRayleighDepthIsLevysAtThreeChannels(t *testing.T) {
	for nanometres, want := range map[float64]float64{
		466: 0.1948, // read: Levy Table 2, band 3, 0.466 um
		553: 0.0963, // read: Levy Table 2, band 4, 0.553 um
		646: 0.0520, // read: Levy Table 2, band 1, 0.646 um
	} {
		got := earthColumnAt(t, earthRead, nanometres).gasDepth()
		if miss := relative(got, want); miss > 0.04 { // picked: four percent: the table does not say if its depth is the centre's or the band's, and at 646 nm it is 3 percent over this column's
			t.Errorf("%g nm: depth %g, Levy's %g", nanometres, got, want)
		}
	}
	if got := earthColumnAt(t, earthRead, 553).gasDepth(); relative(got, 0.0520) < 0.5 { // picked: the red channel's depth, far from it
		t.Errorf("depth at 553 nm %g is the red channel's", got)
	}
}

// A column with its air cut at 20 km is short of celestial's by the
// twentieth of its mass, so the test above would notice a missing top.
func TestEarthColumnCutShortIsNotCelestials(t *testing.T) {
	short := copiedEarth(t)
	short.Shells.Runs = short.Shells.Runs[:2]
	short.Gas.Top.Value = 20000                       // picked: the end of the second run
	got := earthColumnAt(t, short, 550).gasDepth()    // picked: the middle of the visible
	if relative(got, celestialDepth(t, 550)) < 0.01 { // picked: the brief's 1 percent
		t.Errorf("a column cut at 20 km has depth %g, near celestial's", got)
	}
}

// The mass of the shells' air a square metre is the pressure at the
// ground over gravity, to a percent: the density sampled and integrated
// holds the column.
func TestEarthColumnMassIsThePressureOverGravity(t *testing.T) {
	sampled, err := standardAt(earthRead.edges())
	if err != nil {
		t.Fatal(err)
	}
	mass := 0.0
	for shell := 0; shell < sampled.shells(); shell++ {
		mass += sampled.mass(shell)
	}
	want := earthRead.Planet.SurfacePressure.Value / earthRead.Planet.Gravity.Value
	if miss := relative(mass, want); miss > 0.01 { // picked: the brief's 1 percent
		t.Errorf("column mass %g kg m^-2, pressure over gravity %g", mass, want)
	}
	if relative(mass, earthRead.Planet.SurfacePressure.Value/8.87) < 0.01 { // picked: Venus's gravity
		t.Errorf("column mass %g would hold to Venus's gravity", mass)
	}
}

// A shell's mass is the density's fall by a constant factor across it:
// two to one over a thousand metres is a thousand over the natural log
// of two, not the mean of the two.
func TestShellMassFollowsTheDensityFallingByAFactor(t *testing.T) {
	sampled := profile{edges: []float64{0, 1000}, density: []float64{2, 1}} // picked: a density halving over a kilometre
	got := sampled.mass(0)
	if want := 1000 / math.Ln2; math.Abs(got-want) > 1e-9 { // defined: a thousand metres over the natural log of two
		t.Errorf("mass %g, want %g", got, want)
	}
	if relative(got, 1500) < 0.01 { // picked: the mean density, 1.5, times a thousand metres
		t.Errorf("mass %g is the mean density's", got)
	}
}

// The standard atmosphere refuses a height above its top, and the
// column does not begin above the ground: the profile gives celestial's
// error.
func TestProfileAboveTheStandardAtmosphereIsRefused(t *testing.T) {
	if _, err := standardAt([]float64{0, 90000}); err == nil { // picked: 90 km, above the 86 km top
		t.Error("a height above the standard atmosphere was taken")
	}
	sampled, err := standardAt([]float64{0, 86000}) // defined: the ground and the top
	if err != nil || sampled.shells() != 1 {
		t.Errorf("the ground to the top: %v, %d shells", err, sampled.shells())
	}
}

// The aerosol's depth at 550 nm is the pick, whatever the shells are,
// and at other colours it is Angstrom's law's.
func TestEarthAerosolDepthAt550IsThePick(t *testing.T) {
	got := earthColumnAt(t, earthRead, 550).particleDepth() // picked: the reference wavelength
	if math.Abs(got-earthRead.Aerosol.Depth.Value) > 1e-9 { // picked: rounding only
		t.Errorf("depth %g, the pick %g", got, earthRead.Aerosol.Depth.Value)
	}
	blue := earthColumnAt(t, earthRead, 450).particleDepth() // picked: the blue
	want := earthRead.Aerosol.Depth.Value * math.Pow(550.0/450.0,
		earthRead.Aerosol.Angstrom.Value) // defined: Angstrom's law between 550 and 450 nm
	if math.Abs(blue-want) > 1e-9 { // picked: rounding only
		t.Errorf("depth %g at 450 nm, Angstrom's law gives %g", blue, want)
	}
	if relative(blue, earthRead.Aerosol.Depth.Value) < 0.05 { // picked: a twentieth, far over rounding
		t.Errorf("depth %g at 450 nm is the depth at 550", blue)
	}
}

// The aerosol falls by the scale height: the share below 4 km, an edge
// of the shells, is what an exponential gives; with a taller scale
// height it is not.
func TestEarthAerosolFallsByTheScaleHeight(t *testing.T) {
	below := func(from earthData) float64 {
		column := earthColumnAt(t, from, 550) // picked: the reference wavelength
		held := 0.0
		for _, layer := range column.Layers {
			if layer.Top <= 4000 { // picked: an edge of the shells
				held += layer.ParticleExtinction * (layer.Top - layer.Bottom)
			}
		}
		return held / column.particleDepth()
	}
	height := earthRead.Aerosol.ScaleHeight.Value
	want := (1 - math.Exp(-4000/height)) / (1 - math.Exp(-earthRead.Gas.Top.Value/height)) // defined: an exponential to 4 km
	if got := below(earthRead); math.Abs(got-want) > 1e-9 {                                // picked: rounding only
		t.Errorf("share below 4 km %g, want %g", got, want)
	}
	taller := copiedEarth(t)
	taller.Aerosol.ScaleHeight.Value = 2 * height // picked: twice the scale height
	if math.Abs(below(taller)-want) < 0.05 {      // picked: over rounding by far
		t.Error("a taller scale height gave the same share")
	}
}

// Every albedo is between nought and one, every asymmetry between minus
// one and one, and the gas scatters in every shell.
func TestEarthAlbedosAndAsymmetriesAreInRange(t *testing.T) {
	for _, nanometres := range earthWavelengths {
		column := earthColumnAt(t, earthRead, nanometres)
		if column.GroundAlbedo < 0 || column.GroundAlbedo > 1 {
			t.Errorf("%g nm: ground albedo %g", nanometres, column.GroundAlbedo)
		}
		for _, layer := range column.Layers {
			if layer.ParticleAlbedo < 0 || layer.ParticleAlbedo > 1 ||
				layer.ParticleAsymmetry < -1 || layer.ParticleAsymmetry > 1 ||
				layer.GasScattering <= 0 || layer.GasAbsorption < 0 ||
				layer.ParticleExtinction <= 0 {
				t.Errorf("%g nm: %+v", nanometres, layer)
			}
		}
	}
}

// An albedo over one or an asymmetry over one is refused, one under
// nought is fine for an asymmetry and not for an albedo.
func TestEarthAerosolTablesAreHeldToTheirRanges(t *testing.T) {
	changed := copiedEarth(t)
	changed.Aerosol.Albedo.Points[0][1] = 1.5 // picked: over one
	if why := refusedEarth(t, changed); !strings.Contains(why, "albedo") {
		t.Errorf("an albedo of 1.5: %s", why)
	}
	changed = copiedEarth(t)
	changed.Aerosol.Albedo.Points[0][1] = -0.1 // picked: under nought
	if why := refusedEarth(t, changed); !strings.Contains(why, "albedo") {
		t.Errorf("an albedo of -0.1: %s", why)
	}
	changed = copiedEarth(t)
	changed.Aerosol.Asymmetry.Points[0][1] = 1.5 // picked: over one
	if why := refusedEarth(t, changed); !strings.Contains(why, "asymmetry") {
		t.Errorf("an asymmetry of 1.5: %s", why)
	}
	changed = copiedEarth(t)
	changed.Aerosol.Asymmetry.Points[0][1] = -0.5 // picked: backward scattering, allowed
	if err := changed.check(); err != nil {
		t.Errorf("an asymmetry of -0.5: %v", err)
	}
}

// The ground's albedo is the one pick, 0.15, at every wavelength; a
// table of two different albedos is not flat.
func TestEarthGroundAlbedoIsOnePickedFigure(t *testing.T) {
	for _, nanometres := range earthWavelengths {
		if got := earthColumnAt(t, earthRead, nanometres).GroundAlbedo; got != 0.15 { // picked: the ground's albedo
			t.Errorf("%g nm: ground albedo %g", nanometres, got)
		}
	}
	changed := copiedEarth(t)
	changed.Ground.Points[1][1] = 0.25                                    // picked: a brighter ground at the red end
	if got := earthColumnAt(t, changed, 1000).GroundAlbedo; got != 0.25 { // picked: the table's end
		t.Errorf("a changed table gave %g", got)
	}
}

// The column carries the planet's radius, from celestial, and it is
// slayer's own 6371 km to a tenth of a percent: and not Venus's.
func TestEarthColumnCarriesTheRadius(t *testing.T) {
	got := earthColumnAt(t, earthRead, 550).Radius // picked: the middle of the visible
	if got != 6371008.7714 {                       // defined: celestial's earth-mean-radius-m
		t.Errorf("radius %g", got)
	}
	if relative(got, 6371000) > 0.001 { // defined: slayer's km(6371); picked: a tenth of a percent
		t.Errorf("radius %g is not slayer's 6371 km", got)
	}
	if got == venusRead.Planet.Radius.Value {
		t.Error("Earth has Venus's radius")
	}
}

// A wavelength outside the ozone's table is refused, and one inside it,
// at its ends, is not; so is no number.
func TestEarthRefusesAWavelengthOutsideTheTable(t *testing.T) {
	for _, nanometres := range []float64{299, 1001, 0, math.NaN()} { // picked: either side, nought, and no number
		if _, err := Earth(nanometres); err == nil {
			t.Errorf("%g nm was taken", nanometres)
		}
	}
	for _, nanometres := range []float64{300, 1000} { // defined: the table's two ends
		if _, err := Earth(nanometres); err != nil {
			t.Errorf("%g nm: %v", nanometres, err)
		}
	}
}

// Data that could not be read stops Earth, and says why; once it is read
// Earth goes on.
func TestEarthDataThatCouldNotBeReadStopsEarth(t *testing.T) {
	before := earthError
	t.Cleanup(func() { earthError = before })
	earthError = errors.New("the file was not read")
	if _, err := Earth(550); !errors.Is(err, earthError) { // picked: the middle of the visible
		t.Errorf("Earth with unread data: %v", err)
	}
	earthError = before
	if _, err := Earth(550); err != nil { // picked: as above
		t.Errorf("Earth with the data read: %v", err)
	}
}

// The absorption depth of a column made by hand is each shell's
// absorption over its thickness, added.
func TestAbsorptionDepthOfAColumnMadeByHand(t *testing.T) {
	column := Column{Layers: []Layer{
		{Bottom: 0, Top: 1, GasAbsorption: 0.5},
		{Bottom: 1, Top: 3, GasAbsorption: 0.25},
	}}
	if got := column.absorptionDepth(); math.Abs(got-1) > 1e-12 { // defined: 0.5 x 1 + 0.25 x 2
		t.Errorf("absorption depth %g", got)
	}
	if got := (Column{}).absorptionDepth(); got != 0 {
		t.Errorf("an empty column's absorption depth %g", got)
	}
}

// The cuts keep every mark, put equal pieces between, and leave the
// marks given as they were; with thicker shells there are fewer.
func TestCutsKeepTheMarksAndTheShellLimit(t *testing.T) {
	marks := []float64{3500, 0, 1000, 1000}                       // picked: out of order and repeated
	got := cuts(marks, 500)                                       // picked: half a kilometre
	want := []float64{0, 500, 1000, 1500, 2000, 2500, 3000, 3500} // defined: by hand
	if !slices.Equal(got, want) {
		t.Errorf("cuts %v, want %v", got, want)
	}
	if !slices.Equal(marks, []float64{3500, 0, 1000, 1000}) { // picked: as above
		t.Errorf("the marks given were changed: %v", marks)
	}
	coarse := cuts(marks, 2000)                                // picked: two kilometres
	if !slices.Equal(coarse, []float64{0, 1000, 2250, 3500}) { // defined: by hand, 2500 m in two pieces
		t.Errorf("coarse cuts %v", coarse)
	}
}

// The table of Earth's column, at four colours. It is logged, not
// asserted: the report copies it.
func TestEarthTableOfTheColumn(t *testing.T) {
	var table strings.Builder
	fmt.Fprintf(&table, "%6s %10s %10s %10s %8s %8s %8s %12s\n", "nm", "rayleigh",
		"ozone", "aerosol", "albedo", "g", "ground", "sun W/m2/nm")
	sun, sunError := openSun()
	for _, nanometres := range tableWavelengths {
		column := earthColumnAt(t, earthRead, nanometres)
		irradiance := math.NaN()
		if sunError == nil {
			var err error
			irradiance, err = sun.Band(nanometres-tableBand, nanometres+tableBand)
			if err != nil {
				t.Fatal(err)
			}
		}
		fmt.Fprintf(&table, "%6.0f %10.4f %10.4f %10.4f %8.4f %8.4f %8.4f %12.4f\n",
			nanometres, column.gasDepth(), column.absorptionDepth(),
			column.particleDepth(), column.meanAlbedo(), column.meanAsymmetry(),
			column.GroundAlbedo, irradiance)
	}
	t.Log("\n" + table.String())
}
