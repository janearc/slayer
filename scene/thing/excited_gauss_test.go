package thing

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The csv tables the gaussian tests read, each row named by what it tries.
const (
	excitedGaussLineTable       = "testdata/excited_gauss-line.csv"
	excitedGaussWidthTable      = "testdata/excited_gauss-width.csv"
	excitedGaussRadarTable      = "testdata/excited_gauss-radar.csv"
	excitedGaussExcitationTable = "testdata/excited_gauss-excitation.csv"
)

// excitedGaussWithin is whether got is no further than tolerance from want.
// A got that is not a number is not within.
func excitedGaussWithin(got, want, tolerance float64) bool {
	return math.Abs(got-want) <= tolerance
}

// excitedGaussGas is a gas of one line, by its strength and width.
func excitedGaussGas(line, strength, width float64) Excited {
	return Excited{Lines: map[float64]float64{line: strength}, Width: width}
}

// excitedGaussOnly is the excitations of a gas that gives one line one.
func excitedGaussOnly(line, excitation float64) map[float64]float64 {
	return map[float64]float64{line: excitation}
}

// a line is a gaussian, strength * exp(-d^2) with d = (band - line) /
// width, the same either side. worked by hand for a line of strength 2
// at 500 nm, width 2, from exp(-0.25) = 0.77880078307, exp(-1) =
// 0.36787944117, exp(-2.25) = 0.10539922456 and exp(-4) = 0.01831563889:
//
//	501 nm  d 0.5   2 exp(-0.25) = 1.55760156614
//	502 nm  d 1     2 exp(-1)    = 0.73575888234
//	503 nm  d 1.5   2 exp(-2.25) = 0.21079844912
//	504 nm  d 2     2 exp(-4)    = 0.03663127778
func TestALineIsAGaussian(t *testing.T) {
	gas := excitedGaussGas(picked(t, "excited_gauss-line"),
		picked(t, "excited_gauss-strength"), picked(t, "excited_gauss-width"))
	tolerance := picked(t, "excited_gauss-tolerance")
	for _, row := range readSheet(t, excitedGaussLineTable) {
		nanometres, want := cell(t, row, "nm"), cell(t, row, "want")
		got := gas.Emits(maths.NewBand(nanometres))
		if !excitedGaussWithin(got, want, tolerance) {
			t.Errorf("at %v nm the line gives %v, want %v",
				nanometres, got, want)
		}
	}
}

// the lines sum: lines of 2 at 500 and 1 at 504, each 2 wide, at 502
// are both a width away, d = 1 and d = -1, so (2 + 1) exp(-1) =
// 1.10363832351.
func TestLinesSum(t *testing.T) {
	unit := picked(t, "excited_gauss-unit-strength")
	width := picked(t, "excited_gauss-width")
	line := picked(t, "excited_gauss-line")
	farther := picked(t, "excited_gauss-farther-line")
	gas := Excited{Lines: map[float64]float64{
		line:    picked(t, "excited_gauss-strength"),
		farther: unit}, Width: width}
	band := maths.NewBand(picked(t, "excited_gauss-sum-band"))
	want := picked(t, "excited_gauss-sum-want")
	tolerance := picked(t, "excited_gauss-tolerance")
	got := gas.Emits(band)
	if !excitedGaussWithin(got, want, tolerance) {
		t.Errorf("between two lines the gas gives %v, want %v", got, want)
	}
	// a line at zero counts once, and no line is added that is not there
	atNought := excitedGaussGas(nought, unit, width)
	if got := atNought.Emits(maths.NewBand(nought)); got != unit {
		t.Errorf("a line at nought gives %v at nought, want %v", got, unit)
	}
}

// a width of zero or below is one nanometre. at 0.5 nm off the line, d = 0.5
// and exp(-0.25) = 0.77880078307; at 1 nm, exp(-1). a width of half a nanometre
// is just that, and at 0.5 nm off, d = 1.
func TestAWidthOfNothingIsOne(t *testing.T) {
	line := picked(t, "excited_gauss-line")
	unit := picked(t, "excited_gauss-unit-strength")
	tolerance := picked(t, "excited_gauss-tolerance")
	rows := readSheet(t, excitedGaussWidthTable)
	widths := []float64{picked(t, "excited_gauss-width-nothing"),
		picked(t, "excited_gauss-width-below")}
	for _, width := range widths {
		gas := excitedGaussGas(line, unit, width)
		for _, row := range rows {
			nanometres, want := cell(t, row, "nm"), cell(t, row, "want")
			got := gas.Emits(maths.NewBand(nanometres))
			if !excitedGaussWithin(got, want, tolerance) {
				t.Errorf("a width of %v at %v nm gives %v, want %v",
					width, nanometres, got, want)
			}
		}
	}
	half := excitedGaussGas(line, unit, picked(t, "excited_gauss-half-width"))
	offset := maths.NewBand(picked(t, "excited_gauss-half-offset"))
	got := half.Emits(offset)
	want := picked(t, "excited_gauss-half-want")
	if !excitedGaussWithin(got, want, tolerance) {
		t.Errorf("a width of a half at 0.5 nm gives %v, want exp(-1)", got)
	}
}

// excited gas returns no radar at any band, whether on a line or not.
func TestExcitedGasReturnsNoRadar(t *testing.T) {
	unit := picked(t, "excited_gauss-unit-strength")
	gas := Excited{Lines: map[float64]float64{
		picked(t, "excited_gauss-line"):     unit,
		picked(t, "excited_gauss-far-line"): unit},
		Width:  picked(t, "excited_gauss-width"),
		Passes: picked(t, "excited_gauss-passes")}
	for _, row := range readSheet(t, excitedGaussRadarTable) {
		nanometres := cell(t, row, "nm")
		got := gas.Scatters(maths.NewBand(nanometres))
		if got != nought {
			t.Errorf("at %v nm excited gas scatters %v", nanometres, got)
		}
	}
}

// the excitation is the nearest line's, or one where that line has none or one
// that is not greater than zero. the nearest of lines at 500 and 502 to a band
// at 1 nm is 500, not some line at zero.
func TestExcitationIsTheNearestLinesOrOne(t *testing.T) {
	unit := picked(t, "excited_gauss-unit-strength")
	line := picked(t, "excited_gauss-line")
	nearest := picked(t, "excited_gauss-nearest-excitation")
	gas := Excited{Lines: map[float64]float64{
		line: unit, picked(t, "excited_gauss-next-line"): unit},
		Excitations: excitedGaussOnly(line, nearest)}
	low := maths.NewBand(picked(t, "excited_gauss-low-band"))
	if got := gas.Excitation(low); got != nearest {
		t.Errorf("at 1 nm the excitation is %v, want the nearest line's 2", got)
	}
	onLine := maths.NewBand(line)
	for _, row := range readSheet(t, excitedGaussExcitationTable) {
		set, want := cell(t, row, "set"), cell(t, row, "want")
		gas.Excitations = excitedGaussOnly(line, set)
		if got := gas.Excitation(onLine); got != want {
			t.Errorf("a line with an excitation of %v gives %v, want %v",
				set, got, want)
		}
	}
	gas.Excitations = excitedGaussOnly(picked(t, "excited_gauss-next-line"),
		picked(t, "excited_gauss-other-excitation"))
	got := gas.Excitation(onLine)
	want := picked(t, "excited_gauss-default-excitation")
	if got != want {
		t.Errorf("a line with none gives %v, want %v", got, want)
	}
}

// a band as far from two lines as from each other takes the shorter's
// excitation, even when only the longer has one: 501 is a nanometre from
// 500 and from 502, and 503 from 502 and from 504.
func TestATieTakesTheShorterLine(t *testing.T) {
	unit := picked(t, "excited_gauss-unit-strength")
	next := picked(t, "excited_gauss-next-line")
	farther := picked(t, "excited_gauss-farther-line")
	second := picked(t, "excited_gauss-tie-second-excitation")
	gas := Excited{Lines: map[float64]float64{
		picked(t, "excited_gauss-line"): unit, next: unit, farther: unit},
		Excitations: map[float64]float64{next: second,
			farther: picked(t, "excited_gauss-tie-third-excitation")}}
	lowerAt := picked(t, "excited_gauss-tie-lower-band")
	upperAt := picked(t, "excited_gauss-tie-upper-band")
	lower := maths.NewBand(lowerAt)
	upper := maths.NewBand(upperAt)
	unlisted := picked(t, "excited_gauss-default-excitation")
	if got := gas.Excitation(lower); got != unlisted {
		t.Errorf("at 501 the excitation is %v, want 500's one, not 502's 3",
			got)
	}
	if got := gas.Excitation(upper); got != second {
		t.Errorf("at 503 the excitation is %v, want 502's 3, not 504's 4", got)
	}
}
