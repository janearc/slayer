package airs

import (
	"math"
	"testing"
)

// The two standard profiles of the TOMS guide (NTRS 19940019882),
// Appendix D, p 83, Mid Latitude (45 degrees), in Dobson units a layer,
// layers 0 and 1 together first and layer 10 last. The shipped layers
// are their means.
var (
	profile275 = []float64{28.0, 15.0, 29.0, 58.0, 63.7, 40.6, 24.5, 11.1, 3.7, 1.4} // read: the guide's 275 Dobson unit column
	profile325 = []float64{30.0, 26.0, 45.0, 74.7, 66.9, 41.7, 24.5, 11.1, 3.7, 1.4} // read: the guide's 325 Dobson unit column
)

// ozoneColumn is the ozone in the shells of a data, in Dobson units,
// added up over the shells.
func ozoneColumn(t *testing.T, from earthData) float64 {
	t.Helper()
	sampled, err := standardAt(from.edges())
	if err != nil {
		t.Fatal(err)
	}
	total := 0.0
	for shell := 0; shell < sampled.shells(); shell++ {
		total += from.Ozone.amount(sampled.pressure[shell], sampled.pressure[shell+1])
	}
	return total
}

// The layers are the means of the guide's 275 and 325 profiles, and the
// guide's two profiles are themselves 275 and 325 Dobson units.
func TestOzoneLayersAreTheMeansOfTheGuidesProfiles(t *testing.T) {
	sums := [2]float64{}
	for index, layer := range earthRead.Ozone.Layers {
		sums[0] += profile275[index]
		sums[1] += profile325[index]
		mean := (profile275[index] + profile325[index]) / 2 // defined: halfway between the two profiles
		if math.Abs(layer.Amount-mean) > 1e-9 {             // picked: rounding only
			t.Errorf("%s holds %g, the mean of the guide's is %g", layer.Name,
				layer.Amount, mean)
		}
	}
	if len(earthRead.Ozone.Layers) != len(profile275) {
		t.Fatalf("%d layers, the guide has %d", len(earthRead.Ozone.Layers), len(profile275))
	}
	if math.Abs(sums[0]-275) > 1e-9 || math.Abs(sums[1]-325) > 1e-9 { // defined: the guide's totals; picked: rounding only
		t.Errorf("the guide's profiles make %g and %g", sums[0], sums[1])
	}
}

// The ozone in the shells, in Dobson units, is the source's column to a
// percent: nothing is lost between the layers and the shells.
func TestOzoneColumnOfTheShellsIsTheSources(t *testing.T) {
	want := (275.0 + 325.0) / 2                                       // defined: the mean of the guide's two profiles' totals
	if got := ozoneColumn(t, earthRead); relative(got, want) > 0.01 { // picked: the brief's 1 percent
		t.Errorf("ozone column %g Dobson units, the source's %g", got, want)
	}
	if got := ozoneColumn(t, earthRead); relative(got, earthRead.Ozone.Total.Value) > 0.01 { // picked: as above
		t.Errorf("ozone column %g, the data's total %g", got, earthRead.Ozone.Total.Value)
	}
}

// A column with a layer left out is short of the source's by far more
// than a percent, so the test above would notice a lost layer.
func TestOzoneColumnWithALayerLeftOutIsNotTheSources(t *testing.T) {
	changed := copiedEarth(t)
	changed.Ozone.Layers[4].Amount = 0                            // picked: the fifth layer, 66 Dobson units
	if got := ozoneColumn(t, changed); relative(got, 300) < 0.1 { // picked: a tenth of the column
		t.Errorf("ozone column %g with a layer gone", got)
	}
}

// The ozone sits in the stratosphere: its mean height is between 18 and
// 30 km; and there is none above the table's top, the middle of layer
// 10's pressure, about 55 km.
func TestOzoneSitsAtTheHeightsOfTheStratosphere(t *testing.T) {
	sampled, err := standardAt(earthRead.edges())
	if err != nil {
		t.Fatal(err)
	}
	moment, total := 0.0, 0.0
	for shell := 0; shell < sampled.shells(); shell++ {
		held := earthRead.Ozone.amount(sampled.pressure[shell], sampled.pressure[shell+1])
		moment += held * (sampled.edges[shell] + sampled.edges[shell+1]) / 2
		total += held
	}
	if mean := moment / total; mean < 18000 || mean > 30000 { // picked: the ozone layer's heights, 18 to 30 km
		t.Errorf("the mean height of the ozone is %g m", mean)
	}
	column := earthColumnAt(t, earthRead, 600) // picked: the Chappuis band's peak
	if top := column.Layers[len(column.Layers)-1]; top.GasAbsorption != 0 {
		t.Errorf("absorption %g in the highest shell", top.GasAbsorption)
	}
	if low := column.Layers[0]; low.GasAbsorption <= 0 {
		t.Errorf("no absorption in the lowest shell: %g", low.GasAbsorption)
	}
}

// The ozone's absorption depth at 600 nm is its cross section there times
// its column, 300 Dobson units of 2.687e20 molecules a square metre: about
// 0.04, as the Chappuis band is known to take.
func TestOzoneAbsorptionDepthIsCrossSectionTimesColumn(t *testing.T) {
	const loschmidtDobson = 2.687e20                          // defined: molecules in a square metre of a Dobson unit, 2.687e20, Loschmidt's number a hundredth of a millimetre thick
	want := 5.035e-21 * 1e-4 * 300 * loschmidtDobson          // read: the cross section at 600 nm, 5.035e-21 cm2, the table; defined: square metres in a square centimetre; picked: 300 Dobson units
	got := earthColumnAt(t, earthRead, 600).AbsorptionDepth() // picked: the Chappuis band's peak
	if miss := relative(got, want); miss > 0.01 {             // picked: a percent, the constants' digits
		t.Errorf("ozone depth %g, working %g", got, want)
	}
	if got < 0.03 || got > 0.05 { // picked: about 0.04, the Chappuis band at 300 Dobson units
		t.Errorf("ozone depth %g is not about 0.04", got)
	}
}

// Ozone takes light in the Chappuis band, 500 to 700 nm, and little at
// the minimum near 380 nm and in the blue: the twilight zenith is blue
// because the red and green are the ozone's.
func TestOzoneAbsorbsInTheChappuisBandAndNotInTheBlue(t *testing.T) {
	red := earthColumnAt(t, earthRead, 600).AbsorptionDepth()  // picked: the peak
	blue := earthColumnAt(t, earthRead, 450).AbsorptionDepth() // picked: the blue
	dip := earthColumnAt(t, earthRead, 380).AbsorptionDepth()  // picked: the minimum between Huggins and Chappuis
	if red < 20*blue {                                         // picked: the cross sections are 30 times apart
		t.Errorf("depth %g at 600 nm, %g at 450", red, blue)
	}
	if dip > blue/50 { // picked: the cross sections are over 100 times apart
		t.Errorf("depth %g at 380 nm, %g at 450", dip, blue)
	}
	if blue < dip {
		t.Errorf("the blue's depth %g is under the minimum's %g", blue, dip)
	}
}

// The table's integral over 410 to 690 nm is the paper's, to two
// percent: Serdyuchenko and others 2014, Table 3, 6.15e-19 square
// centimetre nanometres at 221 K. A table scaled by five percent is not.
func TestOzoneCrossSectionsIntegrateToTheBandsFigure(t *testing.T) {
	integral := func(scale float64) float64 {
		points := earthRead.Ozone.CrossSection.Points
		sum := 0.0
		for index := 1; index < len(points); index++ {
			low, high := points[index-1], points[index]
			if low[0] >= 410 && high[0] <= 690 { // defined: the Chappuis band's limits in the paper
				sum += scale * (low[1] + high[1]) / 2 * (high[0] - low[0])
			}
		}
		return sum
	}
	const paper = 6.15e-19                                 // read: Serdyuchenko and others 2014, Table 3, "This work", Chappuis, 221 K, x 1e-19 cm2 molecule-1 nm
	if miss := relative(integral(1), paper); miss > 0.02 { // picked: two percent, the table's spacing
		t.Errorf("band integral %g, the paper's %g", integral(1), paper)
	}
	if relative(integral(1.05), paper) < 0.02 { // picked: scaled by 5 percent
		t.Errorf("a table 5 percent high still holds to the paper's")
	}
}

// Between the table's points the cross section is straight in its
// logarithm: at 305 nm it is the geometric mean of the points at 300 and
// 310, which is under the arithmetic mean.
func TestOzoneCrossSectionIsLogarithmicBetweenPoints(t *testing.T) {
	got, err := earthRead.Ozone.cross(305) // picked: between the first two points
	if err != nil {
		t.Fatal(err)
	}
	low, high := earthRead.Ozone.CrossSection.Points[0][1], earthRead.Ozone.CrossSection.Points[1][1]
	want := math.Sqrt(low*high) * 1e-4 // defined: the geometric mean; and square metres in a square centimetre
	if relative(got, want) > 1e-9 {    // picked: rounding only
		t.Errorf("cross section %g, geometric mean %g", got, want)
	}
	if arithmetic := (low + high) / 2 * 1e-4; relative(got, arithmetic) < 0.1 { // picked: a tenth, far over rounding
		t.Errorf("cross section %g is the arithmetic mean %g", got, arithmetic)
	}
}

// The ozone's cross section is refused outside its table's wavelengths
// and given at its ends.
func TestOzoneCrossSectionIsRefusedOutsideItsTable(t *testing.T) {
	for _, nanometres := range []float64{299.9, 1000.1, math.NaN()} { // picked: just outside either end, and no number
		if _, err := earthRead.Ozone.cross(nanometres); err == nil {
			t.Errorf("%g nm was taken", nanometres)
		}
	}
	for _, nanometres := range []float64{300, 1000} { // defined: the table's ends
		if got, err := earthRead.Ozone.cross(nanometres); err != nil || !(got > 0) {
			t.Errorf("%g nm: %g, %v", nanometres, got, err)
		}
	}
}

// The source gives each layer's central pressure and no more; the layers
// are taken as a factor of two in pressure, and the central pressures are
// their middles, to two percent.
func TestOzoneLayersAreAFactorOfTwoAboutTheGuidesCentres(t *testing.T) {
	centres := []float64{190, 95, 48, 24, 12, 6, 3, 1.5, 0.75} // read: the guide's Appendix C, p 81, Umkehr layers 2 to 10, millibars
	layers := earthRead.Ozone.Layers[1:]
	for index, centre := range centres {
		layer := layers[index]
		if relative((layer.Bottom+layer.Top)/2, centre) > 0.02 { // picked: two percent; the layers' middles are within about one
			t.Errorf("%s: middle %g hPa, the guide's centre %g", layer.Name,
				(layer.Bottom+layer.Top)/2, centre)
		}
		if layer.Bottom/layer.Top != 2 { // defined: a factor of two in pressure
			t.Errorf("%s spans %g in pressure", layer.Name, layer.Bottom/layer.Top)
		}
	}
	shifted := (layers[3].Bottom*3 + layers[3].Top*3) / 2 // picked: the same layer three times as high in pressure
	if relative(shifted, centres[3]) < 0.02 {             // picked: as above
		t.Error("a layer three times higher in pressure still holds to the centre")
	}
}
