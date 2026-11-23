package thing

import (
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
)

// The csv tables the radiant tests read: the bands they try, by name, and
// the strengths of the lines of the gas, by the name of the band each is at.
const (
	radiantBandTable = "testdata/radiant-bands.csv"
	radiantLineTable = "testdata/radiant-lines.csv"
)

// radiantBand is the band a row of the band table is at, by its name.
func radiantBand(t *testing.T, name string) maths.Band {
	t.Helper()
	row := readSheet(t, radiantBandTable).row(t, radiantBandTable, name)
	return maths.NewBand(cell(t, row, "nm"))
}

// radiantGas is a gas of the lines in the line table, each at its band,
// with the width and passes the glow test gives it.
func radiantGas(t *testing.T) Excited {
	t.Helper()
	lines := map[float64]float64{}
	for _, line := range readSheet(t, radiantLineTable) {
		band := radiantBand(t, line["name"])
		lines[band.Nanometres] = cell(t, line, "strength")
	}
	return Excited{Lines: lines, Width: picked(t, "radiant-gas-width"),
		Passes: picked(t, "radiant-gas-passes")}
}

// A band falls in the range libtheme holds it in: a ten-thousandth of a
// nanometre is gamma, a nanometre x-ray, a metre radio; the zero band
// and green are visible; a spectrum answers by the range and is opaque
// and dark where it declares none.
func TestSpectrumByRange(t *testing.T) {
	for _, each := range []struct {
		band string
		want radiation.Range
	}{{"gamma", radiation.Gamma()}, {"xray", radiation.XRay()},
		{"extreme-ultraviolet", radiation.ExtremeUltraviolet()},
		{"infrared", radiation.Infrared()}, {"radio", radiation.Radio()}} {
		band := radiantBand(t, each.band)
		if got := Range(band); got != each.want {
			t.Errorf("%v nm is %v, want %v", band.Nanometres, got, each.want)
		}
	}
	if Range(radiantBand(t, "zero")) != Visible() ||
		Range(radiantBand(t, "green")) != Visible() ||
		Range(radiantBand(t, "gamma")) == Visible() {
		t.Error("visible is wrong")
	}
	spectrum := Spectrum{
		Passes: map[radiation.Range]float64{
			radiation.Gamma(): picked(t, "radiant-gamma-passes")},
		Gives: map[radiation.Range]float64{
			radiation.XRay(): picked(t, "radiant-xray-gives")}}
	gamma, xray := radiantBand(t, "inside-gamma"), radiantBand(t, "inside-xray")
	if spectrum.Through(gamma) != picked(t, "radiant-gamma-passes") ||
		spectrum.Through(xray) != nought {
		t.Errorf("through: gamma %v, x-ray %v",
			spectrum.Through(gamma), spectrum.Through(xray))
	}
	if spectrum.Emits(xray) != picked(t, "radiant-xray-gives") ||
		spectrum.Emits(gamma) != nought {
		t.Errorf("emits: x-ray %v, gamma %v",
			spectrum.Emits(xray), spectrum.Emits(gamma))
	}
}

// Excited gas gives off its lines at their wavelengths and little
// between, passes what it passes, returns no radar, and says it is lit.
func TestExcitedGasGlowsInLines(t *testing.T) {
	gas := radiantGas(t)
	hydrogen := gas.Emits(radiantBand(t, "h-alpha"))
	if hydrogen < picked(t, "radiant-glow-floor") {
		t.Errorf("h-alpha is %v", hydrogen)
	}
	oxygen := gas.Emits(radiantBand(t, "o-iii"))
	if oxygen < picked(t, "radiant-oxygen-floor") ||
		oxygen > picked(t, "radiant-oxygen-ceiling") {
		t.Errorf("o iii is %v", oxygen)
	}
	between := gas.Emits(radiantBand(t, "between-lines"))
	if between > picked(t, "radiant-between-ceiling") {
		t.Errorf("between the lines is %v", between)
	}
	if gas.Through(radiantBand(t, "xray")) != picked(t, "radiant-gas-passes") ||
		gas.Scatters(maths.NewBand(nought)) != nought {
		t.Error("the rest of the spectrum is wrong")
	}
	given := picked(t, "radiant-oxygen-excitation")
	gas.Excitations = map[float64]float64{
		radiantBand(t, "o-iii").Nanometres: given}
	beside, far := radiantBand(t, "beside-o-iii"), radiantBand(t, "h-alpha")
	if gas.Excitation(beside) != given ||
		gas.Excitation(far) != picked(t, "radiant-default-excitation") {
		t.Error("the excitations are not the nearest line's")
	}
}

// A visible band's range is Visible, which libtheme names as a range
// and not as a profile; the zero band, every band at once, is Visible
// too; and no band is ever fermi's, an instrument's and not a range.
func TestTheVisibleIsARange(t *testing.T) {
	if got := Range(radiantBand(t, "h-alpha")); got != Visible() {
		t.Errorf("h alpha is in %v", got)
	}
	if got := Range(radiantBand(t, "zero")); got != Visible() {
		t.Errorf("the zero band is in %v", got)
	}
	if got := Range(radiantBand(t, "xray")); got != radiation.XRay() {
		t.Errorf("a nanometre is in %v", got)
	}
	first := picked(t, "radiant-sweep-first")
	limit := picked(t, "radiant-sweep-limit")
	step := picked(t, "radiant-sweep-step")
	for nanometres := first; nanometres < limit; nanometres *= step {
		if Range(maths.NewBand(nanometres)) == radiation.Fermi() {
			t.Errorf("%g nm is fermi's", nanometres)
		}
	}
}

// Where libtheme's ranges overlap, a thing may declare either and the
// narrowest it declares answers: 110 nm is both extreme ultraviolet and
// ultraviolet; a thing that declares the ultraviolet alone passes it by
// that, one that declares both passes it by the extreme.
func TestOverlappingRangesTakeTheNarrowestDeclared(t *testing.T) {
	band := radiantBand(t, "overlap")
	ultraviolet := picked(t, "radiant-ultraviolet-pass")
	extreme := picked(t, "radiant-extreme-pass")
	wide := Spectrum{Passes: map[radiation.Range]float64{
		radiation.Ultraviolet(): ultraviolet}}
	both := Spectrum{Passes: map[radiation.Range]float64{
		radiation.Ultraviolet():        ultraviolet,
		radiation.ExtremeUltraviolet(): extreme}}
	if wide.Through(band) != ultraviolet || both.Through(band) != extreme {
		t.Errorf("110 nm passes %v wide and %v with both",
			wide.Through(band), both.Through(band))
	}
	microwave := radiantBand(t, "microwave")
	radio := picked(t, "radiant-radio-pass")
	declaresRadio := Spectrum{Passes: map[radiation.Range]float64{
		radiation.Radio(): radio}}
	if declaresRadio.Through(microwave) != radio {
		t.Errorf("a microwave band through a thing that declares radio: %v",
			declaresRadio.Through(microwave))
	}
}
