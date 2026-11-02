package lights_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/lights"
	"github.com/janearc/slayer/ninjafilm/maths"
	"pgregory.net/rapid"
)

// what these tests count in, each defined
const (
	none      = 0              // defined: no power at all
	pair      = 2              // defined: a count of two
	cubed     = 3              // defined: the Stefan and Boltzmann constant has h cubed
	fourth    = 4              // defined: the Stefan and Boltzmann law has the temperature to the fourth power
	fifth     = 5              // defined: the Stefan and Boltzmann constant has pi to the fifth
	fifteen   = 15             // defined: the Stefan and Boltzmann constant is two pi to the fifth, k to the fourth, over fifteen c squared h cubed
	upward    = 1              // defined: math.Inf's code for infinity above all numbers
	planck    = 6.62607015e-34 // defined: the Planck constant in joule seconds, exact since the SI of 2019
	speed     = 299792458.0    // defined: the speed of light in metres a second, exact since 1983
	boltzmann = 1.380649e-23   // defined: the Boltzmann constant in joules a kelvin, exact since the SI of 2019
)

// what these tests choose, each picked in picked-numbers.yml
var (
	bulb           = data.Picked("lights-bulb")
	sixty          = data.Picked("lights-watts")
	eyeShortest    = data.Picked("lights-eye-shortest")
	eyeLongest     = data.Picked("lights-eye-longest")
	eyeShare       = data.Picked("lights-eye-share")
	wattsMost      = data.Picked("lights-watts-most")
	heatLeast      = data.Picked("lights-heat-least")
	heatMost       = data.Picked("lights-heat-most")
	stefanError    = data.Picked("lights-stefan-error")
	wattsError     = data.Picked("lights-watts-error")
	linesLeast     = data.Whole("lights-lines-least")
	linesMost      = data.Whole("lights-lines-most")
	lineLeast      = data.Picked("lights-line-least")
	lineMost       = data.Picked("lights-line-most")
	linePowerLeast = data.Picked("lights-line-power-least")
	linePowerMost  = data.Picked("lights-line-power-most")
)

// heated is a temperature the test needs, or the test stops.
func heated(t stopper, kelvin float64) maths.Temperature {
	t.Helper()
	heat, err := maths.Kelvin(kelvin)
	if err != nil {
		t.Fatal(err)
	}
	return heat
}

// stopper is a test that can be stopped: a testing.T, or a rapid.T.
type stopper interface {
	Helper()
	Fatal(args ...any)
}

// powered is a power the test needs, or the test stops.
func powered(t stopper, watts float64) maths.Power {
	t.Helper()
	power, err := maths.Watts(watts)
	if err != nil {
		t.Fatal(err)
	}
	return power
}

// within is the power in a light between two wavelengths, by its bands.
func within(light bands.Light, shortest, longest float64) float64 {
	first, last := light.Span()
	var total float64
	for band := first; band <= last; band++ {
		short, long := band.Edges()
		if short >= shortest && long <= longest {
			total += light.Power(band)
		}
	}
	return total
}

// everything is the power in a light in every band.
func everything(light bands.Light) float64 {
	return within(light, none, math.Inf(upward))
}

// stefanBoltzmann is the power a black body gives off from a square
// metre for each kelvin to the fourth. It is two pi to the fifth times k
// to the fourth over fifteen c squared h cubed, worked from the SI's
// constants.
func stefanBoltzmann() float64 {
	top := pair * math.Pow(math.Pi, fifth) * math.Pow(boltzmann, fourth)
	bottom := fifteen * speed * speed * math.Pow(planck, cubed)
	return top / bottom
}

// Black bodies hold in their spectrum all the power Stefan and
// Boltzmann say. The bands are the ones libtheme names, from gamma rays
// to radio.
func TestABlackBodyGivesWhatStefanAndBoltzmannSay(t *testing.T) {
	rapid.Check(t, planckGivesStefansPower)
}

func planckGivesStefansPower(t *rapid.T) {
	kelvin := rapid.Float64Range(heatLeast, heatMost).Draw(t, "heat")
	spectrum := lights.Planck(heated(t, kelvin))
	given := everything(spectrum)
	want := stefanBoltzmann() * math.Pow(kelvin, fourth)
	if off := math.Abs(given-want) / want; off > stefanError {
		t.Fatalf("%v K gives %.6g W a square metre, and Stefan and "+
			"Boltzmann say %.6g: off by %.2g", kelvin, given, want, off)
	}
}

func TestMostOfABulbsLightIsHeat(t *testing.T) {
	spectrum := lights.Planck(heated(t, bulb))
	seen := within(spectrum, eyeShortest, eyeLongest) / everything(spectrum)
	if seen > eyeShare {
		t.Errorf("a %v K bulb gives %.0f%% of its power as light an eye "+
			"can see; tungsten gives about a tenth", bulb, seen)
	}
}

func TestAPointGivesTheWattsItIsGiven(t *testing.T) {
	spectrum := lights.Planck(heated(t, bulb))
	lamp, err := lights.Point(spectrum, powered(t, sixty))
	if err != nil {
		t.Fatal(err)
	}
	given := everything(lamp.Light())
	if math.Abs(given-sixty) > sixty*wattsError {
		t.Errorf("a %v W point gives %v W", sixty, given)
	}
}

// drawnSpectrum is a spectrum of some lines of power at wavelengths.
func drawnSpectrum(t *rapid.T) bands.Light {
	many := rapid.IntRange(linesLeast, linesMost).Draw(t, "lines")
	lines := map[float64]float64{}
	where := rapid.Float64Range(lineLeast, lineMost)
	power := rapid.Float64Range(linePowerLeast, linePowerMost)
	for range many {
		lines[where.Draw(t, "where")] = power.Draw(t, "power")
	}
	return bands.Lines(lines)
}

// Points give the watts they are given however the spectrum they are
// given lies. The light they give is the light of the spectrum scaled.
func TestAPointGivesTheWattsOfAnySpectrum(t *testing.T) {
	rapid.Check(t, pointGivesItsWatts)
}

func pointGivesItsWatts(t *rapid.T) {
	spectrum := drawnSpectrum(t)
	watts := rapid.Float64Range(none, wattsMost).Draw(t, "watts")
	lamp, err := lights.Point(spectrum, powered(t, watts))
	if err != nil {
		t.Fatal(err)
	}
	given := everything(lamp.Light())
	if math.Abs(given-watts) > watts*wattsError {
		t.Fatalf("a %v W point gives %v W", watts, given)
	}
}

// anti-tests: a spectrum with no power in any band cannot be scaled to
// watts, so a point of it is not made. What comes back is no lamp.
func TestAPointRefusesADarkSpectrum(t *testing.T) {
	var dark bands.Light
	var noLamp lights.Emitter
	got, err := lights.Point(dark, powered(t, sixty))
	if err == nil || !reflect.DeepEqual(got, noLamp) {
		t.Errorf("a point with no light in any band was made: %v, %v",
			got, err)
	}
}

// Every band of a black body's spectrum holds the power of that band
// and no other, and the spectrum starts and ends where there is power.
// The bands with none are left out.
func TestEachBandOfABlackBodyHoldsItsOwnPower(t *testing.T) {
	rapid.Check(t, bandsHoldTheirOwnPower)
}

func bandsHoldTheirOwnPower(t *rapid.T) {
	kelvin := rapid.Float64Range(heatLeast, heatMost).Draw(t, "heat")
	heat := heated(t, kelvin)
	spectrum := lights.Planck(heat)
	first, last := spectrum.Span()
	shortest, _ := radiation.Gamma().Edges()
	_, longest := radiation.Radio().Edges()
	gamma, radio := bands.Of(shortest), bands.Of(longest)
	if first < gamma || last != radio {
		t.Fatalf("%v K: the spectrum runs from band %d to band %d, "+
			"within %d to %d", kelvin, first, last, gamma, radio)
	}
	if spectrum.Power(first) <= none || spectrum.Power(last) <= none {
		t.Fatalf("%v K: the spectrum starts at %v W and ends at %v W",
			kelvin, spectrum.Power(first), spectrum.Power(last))
	}
	band := bands.Band(rapid.IntRange(int(first), int(last)).Draw(t, "band"))
	short, long := band.Edges()
	if got, want := spectrum.Power(band), heat.Exitance(short, long); got !=
		want {
		t.Fatalf("%v K: band %d holds %v W, its edges say %v W", kelvin,
			band, got, want)
	}
}
