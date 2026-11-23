package thing

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The tables the audit tests read, each row named in its name column.
const (
	auditLinesFile       = "testdata/audit-lines.csv"
	auditExcitationsFile = "testdata/audit-excitations.csv"
	auditCosinesFile     = "testdata/audit-cosines.csv"
)

// auditPick is a number the audit tests chose, by its name in numbers.yaml
// after the stem audit.
func auditPick(t *testing.T, name string) float64 {
	t.Helper()
	return picked(t, "audit-"+name)
}

// auditLines is the lines of the excited gas, each wavelength with its
// strength.
func auditLines(t *testing.T) map[float64]float64 {
	t.Helper()
	lines := map[float64]float64{}
	for _, row := range readSheet(t, auditLinesFile) {
		lines[cell(t, row, "wavelength")] = cell(t, row, "strength")
	}
	return lines
}

// auditExcitations is the excitation each of the two lines wants, by its
// wavelength.
func auditExcitations(t *testing.T) map[float64]float64 {
	t.Helper()
	excitations := map[float64]float64{}
	for _, row := range readSheet(t, auditExcitationsFile) {
		excitations[cell(t, row, "wavelength")] = cell(t, row, "excitation")
	}
	return excitations
}

// auditTieLines is a line of full strength at each wavelength that wants
// an excitation: two lines, so that a band midway between them is a tie.
func auditTieLines(t *testing.T) map[float64]float64 {
	t.Helper()
	lines := map[float64]float64{}
	for wavelength := range auditExcitations(t) {
		lines[wavelength] = unity
	}
	return lines
}

// excited gas gives the same light every time it is asked: three lines
// that overlap at a band, asked two hundred times, give one answer, and
// a band as near two lines takes the shorter's excitation.
func TestExcitedGasIsTheSameEveryTime(t *testing.T) {
	gas := Excited{Lines: auditLines(t), Width: auditPick(t, "width"),
		Excitations: auditExcitations(t)}
	repeats := int(auditPick(t, "repeats"))
	band := maths.NewBand(auditPick(t, "band"))
	first := gas.Emits(band)
	for range repeats {
		got := gas.Emits(band)
		if math.Float64bits(got) != math.Float64bits(first) {
			t.Fatalf("the gas gave %v and then %v", first, got)
		}
	}
	tie := Excited{Lines: auditTieLines(t), Excitations: auditExcitations(t)}
	midway := maths.NewBand(auditPick(t, "midway"))
	shorter := readSheet(t, auditExcitationsFile).row(t,
		auditExcitationsFile, "shorter")
	want := cell(t, shorter, "excitation")
	for range repeats {
		if got := tie.Excitation(midway); got != want {
			t.Fatalf("midway between 500 and 502 the excitation is %v, "+
				"not 500's 2", got)
		}
	}
}

// auditExpectRefusal asks for the period of an orbit and holds it to be
// refused with a message that names the 292 years a Duration holds.
func auditExpectRefusal(t *testing.T, mass, radius float64) {
	t.Helper()
	defer auditRefusedWithYears(t, radius)
	Period(mass, radius)
}

// auditRefusedWithYears is deferred: it reads the refusal and holds it to
// name 292.
func auditRefusedWithYears(t *testing.T, radius float64) {
	t.Helper()
	got, _ := recover().(string)
	if !strings.Contains(got, "292") {
		t.Errorf("an orbit of %v m: %q", radius, got)
	}
}

// an orbit longer than a Duration holds, or of no number, is refused by
// name: worked by hand, Sedna at 506 au round the sun is some 11,400
// years, and would have been 292.
func TestAnOrbitPastADurationIsRefused(t *testing.T) {
	sun := auditPick(t, "sun-mass")
	astronomicalUnit := auditPick(t, "astronomical-unit")
	sedna := auditPick(t, "sedna-aphelion-au") * astronomicalUnit
	for _, radius := range []float64{sedna, math.Inf(unity)} {
		auditExpectRefusal(t, sun, radius)
	}
}

// a surface mirrors no more than lands on it: a Gloss of two is a Gloss
// of one, face on and grazing.
func TestNoSurfaceMirrorsMoreThanLands(t *testing.T) {
	index := auditPick(t, "index")
	over := &Material{Gloss: auditPick(t, "over-gloss"), Index: index}
	whole := &Material{Gloss: unity, Index: index}
	for _, row := range readSheet(t, auditCosinesFile) {
		cosine := cell(t, row, "cosine")
		got, want := over.Mirrors(cosine), whole.Mirrors(cosine)
		if got != want || got > unity {
			t.Errorf("a gloss of two at %v mirrors %v, want %v",
				cosine, got, want)
		}
	}
}
