package planet

import (
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
)

// methane's rayleigh cross-section from its index is what he and the
// others worked from it, and within their measurement's spread of what
// they measured: at 532 nm, 11.34 and 11.3 times 1e-27 square
// centimetres; at 660 nm, 4.693 and 4.68 (acp-21-14927, table 2).
func TestMethaneScattersAsMeasured(t *testing.T) {
	section := Methane(nil).section
	for _, row := range []struct{ nm, worked, measured float64 }{
		{532, 11.34e-31, 11.3e-31}, {660, 4.693e-31, 4.68e-31},
	} {
		got := section(row.nm)
		if !near(got, row.worked, 5e-3) || !near(got, row.measured, 2e-2) {
			t.Errorf("%g nm: %g, worked %g, measured %g", row.nm, got,
				row.worked, row.measured)
		}
	}
}

// the noble gases' indices are the database's formula at its own
// numbers: neon at 500 nm is 0.00128145 over 180.661 and 0.0220486
// over 372.84, which is 6.623e-5.
func TestTheNobleIndicesAreTheFormula(t *testing.T) {
	want := 0.00128145/(184.661-4) + 0.0220486/(376.840-4)
	if got := neonIndex(500); !near(got, want, 1e-12) ||
		!near(got, 6.623e-5, 1e-3) {
		t.Errorf("neon at 500 nm: %g, want %g", got, want)
	}
	for name, index := range map[string]func(float64) float64{
		"krypton": kryptonIndex, "xenon": xenonIndex} {
		if !(index(450) > index(650)) {
			t.Errorf("%s bends blue less than red", name)
		}
	}
}

// a heavier noble gas scatters more, as its index squared over neon's:
// xenon about a hundred times neon at 550 nm, and every one more in the
// blue than the red, near the wavelength to the fourth.
func TestHeavierGasesScatterMore(t *testing.T) {
	neon, krypton, xenon := Neon(), Krypton(), Xenon()
	ratio := xenon.section(550) / neon.section(550)
	want := math.Pow(xenonIndex(550)/neonIndex(550), 2)
	if !near(ratio, want, 1e-3) {
		t.Errorf("xenon over neon %g, want %g", ratio, want)
	}
	if !(krypton.section(550) > neon.section(550)) {
		t.Error("krypton scatters less than neon")
	}
	for _, gas := range []Gas{neon, krypton, xenon, Methane(nil)} {
		blue := gas.section(450) / gas.section(650)
		if !(blue > 4 && blue < 5) {
			t.Errorf("%s blue over red %g", gas.name, blue)
		}
	}
}

// a methane table's rows are read, each band takes the mean of its
// rows by reciprocal kilometre-amagats as square metres a molecule, and
// where the table has no rows methane absorbs nothing.
func TestMethaneAbsorbsByItsTable(t *testing.T) {
	table := "Vacuum Air Methane albedo\n(nm) (nm) (1/km-am)\n" +
		"600.0 599.8 .2000 .1\n600.4 600.2 .4000 .1\n" +
		"900.0 899.8 18.0 .1\n"
	absorbing, err := MethaneAbsorbing(strings.NewReader(table))
	if err != nil {
		t.Fatal(err)
	}
	if got := absorbing(600.2); !near(got, 0.3/kilometreAmagat, 1e-12) {
		t.Errorf("600 nm: %g", got)
	}
	if got := absorbing(400); got != 0 {
		t.Errorf("400 nm, no rows: %g", got)
	}
	if !near(kilometreAmagat, 2.6868e28, 1e-4) {
		t.Errorf("a kilometre-amagat of %g", kilometreAmagat)
	}
	for _, wrong := range []string{"", "a b c d\n",
		"600.0 599.8 -1 .1\n"} {
		if _, err := MethaneAbsorbing(strings.NewReader(wrong)); err == nil {
			t.Errorf("%q taken", wrong)
		}
	}
}

// on the pinned table, methane's 619 nm band absorbs far more than the
// clear window at 640 nm, and the 890 nm band far more again.
func TestTheMethaneTableHasItsBands(t *testing.T) {
	shelf := os.Getenv("SLAYER_SHELF")
	if shelf == "" {
		t.Skip("no shelf named in SLAYER_SHELF")
	}
	file, err := os.Open(filepath.Join(shelf,
		"karkoschka-1994-methane.txt"))
	if err != nil {
		t.Skip(err)
	}
	defer file.Close()
	absorbing, err := MethaneAbsorbing(file)
	if err != nil {
		t.Fatal(err)
	}
	band, window, deep := absorbing(619), absorbing(640), absorbing(889)
	if !(band > 3*window) || !(deep > 10*band) {
		t.Errorf("619 nm %g, 640 nm %g, 889 nm %g", band, window, deep)
	}
}

// the cross-section is he and the others' equation 1, bit for bit: for an index
// of 1.0003 at a number of 2.5e25 a cubic metre and a king factor of 1.05, at
// 500 nm.
func TestTheCrossSectionIsEquationOne(t *testing.T) {
	section := scatters(func(float64) float64 { return 3e-4 }, 2.5e25,
		1.05)
	lorentz := (1.0003*1.0003 - 1) / (1.0003*1.0003 + 2)
	want := 24 * math.Pi * math.Pi * math.Pi / (2.5e25 * 2.5e25 *
		math.Pow(500e-9, 4)) * lorentz * lorentz * 1.05
	if got := section(500); !near(got, want, 1e-12) {
		t.Errorf("%g, want %g", got, want)
	}
}

// the standard number is loschmidt's, 2.686780e25 a cubic metre at zero celsius
// and 101,325 pascals.
func TestTheStandardNumberIsLoschmidts(t *testing.T) {
	if !near(standard, 2.6867801e25, 1e-7) {
		t.Errorf("%g", standard)
	}
}

// krypton's and xenon's indices are the database's formulas at their own
// numbers, at 500 nm, where one over the wavelength squared is four.
func TestKryptonAndXenonAreTheirFormulas(t *testing.T) {
	krypton := 0.00253637/(65.4742-4) + 0.00273649/(73.698-4) +
		0.0620802/(181.08-4)
	xenon := 0.00322869/(46.301-4) + 0.00355393/(59.578-4) +
		0.0606764/(112.74-4)
	if got := kryptonIndex(500); !near(got, krypton, 1e-12) {
		t.Errorf("krypton %g, want %g", got, krypton)
	}
	if got := xenonIndex(500); !near(got, xenon, 1e-12) {
		t.Errorf("xenon %g, want %g", got, xenon)
	}
	if got := Krypton().section(500); !near(got,
		scatters(kryptonIndex, standard, 1)(500), 1e-12) {
		t.Errorf("krypton's section %g", got)
	}
}

// the molar masses are iupac's standard atomic weights, in kilograms a
// mole.
func TestTheMolarMasses(t *testing.T) {
	for _, row := range []struct {
		gas  Gas
		want float64
	}{{Neon(), 0.0201797}, {Krypton(), 0.083798}, {Xenon(), 0.131293},
		{Methane(nil), 0.016043}} {
		if row.gas.molar != row.want {
			t.Errorf("%s: %g", row.gas.name, row.gas.molar)
		}
	}
}

// a band reaches a forty-eighth of an octave either side of its
// middle: a row at the low edge is in it, one at the high edge is not,
// and the band is found by the vacuum wavelength, not the air one.
func TestABandsEdgesAndItsColumn(t *testing.T) {
	middle := 600.0
	low, high := middle/math.Pow(2, 1.0/48), middle*math.Pow(2, 1.0/48)
	table := tableRow(low, 1, 1) + tableRow(high, 1, 9) +
		tableRow(high-0.01, high+5, 3)
	absorbing, err := MethaneAbsorbing(strings.NewReader(table))
	if err != nil {
		t.Fatal(err)
	}
	if got := absorbing(middle) * kilometreAmagat; !near(got, 2, 1e-12) {
		t.Errorf("the band's mean %g, want 2", got)
	}
	_, err = MethaneAbsorbing(strings.NewReader("612.4 612.2 -3 .1\n"))
	if err == nil || !strings.Contains(err.Error(), "612.4") ||
		!strings.Contains(err.Error(), "-3") {
		t.Errorf("a bad row said as %v", err)
	}
	broken := io.MultiReader(strings.NewReader("600.0 599.8 .2 .1\n"),
		iotest.ErrReader(io.ErrUnexpectedEOF))
	if _, err := MethaneAbsorbing(broken); err == nil {
		t.Error("a table that failed partway was taken")
	}
}

// tableRow is a table row: a vacuum wavelength, an air one and a coefficient,
// each written so it reads back bit for bit.
func tableRow(vacuum, air, coefficient float64) string {
	return fmt.Sprintf("%v %v %v .1\n", vacuum, air, coefficient)
}
