package airs

import (
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/iotest"

	"github.com/janearc/slayer/sources"
)

// sunShelfFile is the sun's spectrum's name on the shelf: the shelf names
// a file by its key and the ending of the address it came from, which
// for the TSIS-1 spectrum is .csv.
const sunShelfFile = SunSource + ".csv" // defined: see above

// sharedSun reads the spectrum from the shelf once, for the tests that
// want the real one.
var sharedSun = sync.OnceValues(func() (Sun, error) {
	shelf := os.Getenv(sources.Placed)
	file, err := os.Open(filepath.Join(shelf, sunShelfFile))
	if err != nil {
		return Sun{}, err
	}
	defer file.Close()
	return ReadSun(file)
})

// openSun is the real spectrum, or why it could not be had: the tables
// that print it print nothing for it where there is no shelf.
func openSun() (Sun, error) {
	return sharedSun()
}

// realSun is the real spectrum, read from the shelf, or the test skipped
// where no shelf is named, as the tree's other tests that read the shelf
// are. The tests that need only a spectrum's shape use the fixture.
func realSun(t *testing.T) Sun {
	t.Helper()
	if os.Getenv(sources.Placed) == "" {
		t.Skip("no shelf named in SLAYER_SHELF")
	}
	sun, err := sharedSun()
	if err != nil {
		t.Fatal(err)
	}
	return sun
}

// fixtureSpectrum is four samples at 0.025 nm, in the file's own form.
const fixtureSpectrum = "wavelength (nm),irradiance (W/m^2/nm),uncertainty (W/m^2/nm),bandwidth (nm)\n" +
	"400.000,1.0,0.01,0.1\n400.025,2.0,0.01,0.1\n400.050,3.0,0.01,0.1\n400.075,4.0,0.01,0.1\n"

// fixtureSun is the fixture, read.
func fixtureSun(t *testing.T) Sun {
	t.Helper()
	sun, err := ReadSun(strings.NewReader(fixtureSpectrum))
	if err != nil {
		t.Fatal(err)
	}
	return sun
}

// A band's irradiance is the mean of the samples in it, ends included.
func TestBandIsTheMeanOfTheSamplesInIt(t *testing.T) {
	sun := fixtureSun(t)
	for name, each := range map[string]struct{ low, high, want float64 }{
		"three":  {400.0, 400.05, 2.0},
		"two":    {400.01, 400.06, 2.5},
		"one":    {400.025, 400.025, 2.0},
		"all":    {400.0, 400.075, 2.5},
		"narrow": {400.07, 400.075, 4.0},
	} {
		got, err := sun.Band(each.low, each.high)
		if err != nil || math.Abs(got-each.want) > 1e-12 { // picked: rounding only
			t.Errorf("%s: %g, %v, want %g", name, got, err, each.want)
		}
	}
}

// A band that has no samples, or runs backwards, or runs off the data, or
// is not a number, is refused.
func TestBandRefusals(t *testing.T) {
	sun := fixtureSun(t)
	for name, each := range map[string][2]float64{
		"backwards": {400.05, 400.0},
		"between":   {400.026, 400.049},
		"below":     {300, 400.01},
		"above":     {400.01, 500},
		"nan":       {math.NaN(), 400.05},
	} {
		if got, err := sun.Band(each[0], each[1]); err == nil {
			t.Errorf("%s: %g was taken", name, got)
		}
	}
}

// The sun at Venus is the sun at an astronomical unit over the square of
// Venus's distance in them.
func TestSunAtVenusIsScaledByTheDistanceSquared(t *testing.T) {
	sun := fixtureSun(t)
	near, err := sun.Band(400, 400.05)
	if err != nil {
		t.Fatal(err)
	}
	got, err := sun.AtVenus(400, 400.05)
	if err != nil {
		t.Fatal(err)
	}
	distance := venusRead.Planet.Distance.Value
	if math.Abs(got-near/(distance*distance)) > 1e-12 { // picked: rounding only
		t.Errorf("at Venus %g, at one au %g, distance %g", got, near, distance)
	}
	if got < 1.5*near { // picked: Venus is nearer the sun by a quarter, so it is brighter by well over this
		t.Errorf("the sun is no brighter at Venus: %g against %g", got, near)
	}
	if _, err := sun.AtVenus(400.05, 400); err == nil {
		t.Error("a backwards band was taken at Venus")
	}
}

// A spectrum that is not in the file's form is refused, line by line.
func TestReadSunRefusals(t *testing.T) {
	header := strings.SplitN(fixtureSpectrum, "\n", 2)[0] + "\n"
	for name, text := range map[string]string{
		"empty":        "",
		"header":       "wavelength,irradiance\n400,1\n401,2\n",
		"one sample":   header + "400.000,1.0,0.01,0.1\n",
		"no column":    header + "400.000\n400.025,2.0\n",
		"not a number": header + "400.000,1.0\n400.025,two\n",
		"wavelength":   header + "400.000,1.0\nfour hundred,2.0\n",
		"order":        header + "400.025,1.0\n400.000,2.0\n",
		"negative":     header + "400.000,1.0\n400.025,-2.0\n",
	} {
		if _, err := ReadSun(strings.NewReader(text)); err == nil {
			t.Errorf("%s was taken", name)
		}
	}
	if _, err := ReadSun(strings.NewReader(fixtureSpectrum)); err != nil {
		t.Errorf("the fixture: %v", err)
	}
}

// A reader that fails part way through stops the reading, and the error
// is the reader's own; the same text, read whole, is taken.
func TestAReaderThatFailsStopsTheReading(t *testing.T) {
	broken := errors.New("the disc stopped")
	reader := io.MultiReader(strings.NewReader(fixtureSpectrum),
		iotest.ErrReader(broken))
	if _, err := ReadSun(reader); !errors.Is(err, broken) {
		t.Errorf("a failing reader: %v", err)
	}
	if _, err := ReadSun(strings.NewReader(fixtureSpectrum)); err != nil {
		t.Errorf("the same text read whole: %v", err)
	}
}

// Over the whole file, 202 to 2730 nm at one astronomical unit, the sun
// is within 5 percent of the total solar irradiance, 1361 W/m^2 (Kopp
// and Lean 2011, as session/afloat.go has it). The file stops short of
// the far infrared, so it is a little under.
func TestSpectrumTotalsTheSolarIrradiance(t *testing.T) {
	const total = 1361.0 // measured: the total solar irradiance, Kopp and Lean 2011, W/m^2
	sun := realSun(t)
	low, high := sun.wavelengths[0], sun.wavelengths[len(sun.wavelengths)-1]
	mean, err := sun.Band(low, high)
	if err != nil {
		t.Fatal(err)
	}
	if got := mean * (high - low); relative(got, total) > 0.05 { // defined: the brief's 5 percent
		t.Errorf("the spectrum from %g to %g nm totals %g W/m^2", low, high, got)
	}
}

// The visible alone is well under the total: the test above counts the
// whole spectrum, and would notice if only part of it were read.
func TestTheVisibleAloneIsNotTheTotal(t *testing.T) {
	const total = 1361.0 // measured: as above
	sun := realSun(t)
	mean, err := sun.Band(380, 780) // defined: the visible, to the eye's limits
	if err != nil {
		t.Fatal(err)
	}
	if got := mean * (780 - 380); relative(got, total) < 0.05 {
		t.Errorf("the visible totals %g W/m^2, as the whole does", got)
	}
}

// The sun at 550 nm is the sample the file has there, scaled: a check
// of a value read from the file, 1.967 W/m^2/nm near 550 nm.
func TestSunAt550IsWhatTheFileHas(t *testing.T) {
	sun := realSun(t)
	got, err := sun.Band(549.9, 550.1) // picked: a fifth of a nanometre, a few samples
	if err != nil {
		t.Fatal(err)
	}
	if relative(got, 1.967) > 0.05 { // read: the facts file's sample, 550.000 nm 1.9670, to within the lines at its sides
		t.Errorf("sun at 550 nm is %g W/m^2/nm", got)
	}
}
