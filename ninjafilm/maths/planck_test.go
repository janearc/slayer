package maths_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/ninjafilm/maths"
	"pgregory.net/rapid"
)

// what Planck's law is worked from here, each defined
const (
	planck    = 6.62607015e-34 // defined: the Planck constant in joule seconds, exact since the SI of 2019
	speed     = 299792458.0    // defined: the speed of light in metres a second, exact since 1983
	boltzmann = 1.380649e-23   // defined: the Boltzmann constant in joules a kelvin, exact since the SI of 2019
	nanometre = 1e-9           // defined: the nano prefix
	pair      = 2              // defined: a count of two
	fifth     = 5              // defined: Planck's law has the wavelength to the fifth power
	fourth    = 4              // defined: the Stefan and Boltzmann law has the temperature to the fourth power
	fifteen   = 15             // defined: the Stefan and Boltzmann constant is two pi to the fifth, k to the fourth, over fifteen c squared h cubed
	cubed     = 3              // defined: see fifteen
	half      = 0.5            // defined: the middle of a band is half way between its edges
)

// what the Planck tests choose, each picked in picked-numbers.yml
var (
	heatLeast      = datum("maths-heat-least")
	heatMost       = datum("maths-heat-most")
	waveLeast      = datum("maths-wavelength-least")
	waveMost       = datum("maths-wavelength-most")
	bandRatio      = datum("maths-band-ratio")
	narrowRatio    = datum("maths-narrow-ratio")
	additiveError  = datum("maths-additive-error")
	narrowError    = datum("maths-narrow-error")
	scaleLeast     = datum("maths-scale-least")
	scaleMost      = datum("maths-scale-most")
	hotterMost     = datum("maths-hotter-most")
	totalShortest  = datum("maths-total-shortest")
	totalLongest   = datum("maths-total-longest")
	totalError     = datum("maths-total-error")
	totalHeatLeast = datum("maths-total-heat-least")
	totalHeatMost  = datum("maths-total-heat-most")
	exponentLeast  = datum("maths-x-least")
	exponentMost   = datum("maths-x-most")
)

// heated is a temperature, or the test stops.
func heated(t stopper, kelvin float64) maths.Temperature {
	t.Helper()
	made, err := maths.Kelvin(kelvin)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// stopper is a test that can be stopped: a testing.T, or a rapid.T.
type stopper interface {
	Helper()
	Fatal(args ...any)
}

// within is whether two numbers are within a share of the larger.
func within(one, other, share float64) bool {
	return math.Abs(one-other) <= share*math.Max(math.Abs(one),
		math.Abs(other))
}

// bandAt is a drawn stretch of wavelengths in nanometres for a black body
// at a temperature. It starts at a drawn value of Planck's exponent. That
// is hc over wavelength k T. It ends a ratio further on. Further into
// the short tail the power is too small to matter. There a band holds
// too many e-folds for the rule that integrates it.
func bandAt(t *rapid.T, kelvin, ratio float64) (short, long float64) {
	exponent := rapid.Float64Range(exponentLeast, exponentMost).Draw(t, "x")
	short = planck * speed / (boltzmann * kelvin * exponent) / nanometre
	return short, short * ratio
}

// anti-tests: a black body at absolute zero gives nothing. Nor does a stretch
// that runs backwards or has no width. Nor does one that starts at zero or
// before.
func TestNothingComesFromWhereNothingIsAsked(t *testing.T) {
	cold := heated(t, none)
	if got := cold.Exitance(waveLeast, waveMost); got != none {
		t.Errorf("a black body at 0 K gives %v W", got)
	}
	warm := heated(t, heatMost)
	silent(t, warm, waveMost, waveLeast)
	silent(t, warm, waveLeast, waveLeast)
	silent(t, warm, none, waveLeast)
	silent(t, warm, -waveLeast, waveLeast)
	silent(t, warm, math.NaN(), waveLeast)
	silent(t, warm, waveLeast, math.NaN())
}

// silent stops the test unless a black body gives nothing between two
// wavelengths in nanometres.
func silent(t *testing.T, heat maths.Temperature, short, long float64) {
	t.Helper()
	if got := heat.Exitance(short, long); got != none {
		t.Errorf("%v to %v nm gives %v W", short, long, got)
	}
}

// Black bodies give light between any two wavelengths. It is some and
// finite. There is more in the whole of a stretch than in any piece of
// it.
func TestABlackBodyGivesLightBetweenTwoWavelengths(t *testing.T) {
	rapid.Check(t, blackBodyGivesLight)
}

func blackBodyGivesLight(t *rapid.T) {
	kelvin := rapid.Float64Range(heatLeast, heatMost).Draw(t, "heat")
	heat := heated(t, kelvin)
	short, long := bandAt(t, kelvin, bandRatio)
	whole := heat.Exitance(short, long)
	middle := short * (unity + (bandRatio-unity)*half)
	piece := heat.Exitance(short, middle)
	if whole < none || math.IsInf(whole, either) || math.IsNaN(whole) ||
		piece > whole {
		t.Fatalf("%v K gives %v W in %v to %v nm and %v W in its first "+
			"half", kelvin, whole, short, long, piece)
	}
}

// The power in a band is the power in its halves added. The rule that
// integrates it is the same on both, to a millionth in a band of
// libtheme's width.
func TestABandIsItsHalvesAdded(t *testing.T) {
	rapid.Check(t, bandIsItsHalves)
}

func bandIsItsHalves(t *rapid.T) {
	kelvin := rapid.Float64Range(heatLeast, heatMost).Draw(t, "heat")
	heat := heated(t, kelvin)
	short, long := bandAt(t, kelvin, bandRatio)
	middle := math.Sqrt(short * long)
	whole := heat.Exitance(short, long)
	halves := heat.Exitance(short, middle) + heat.Exitance(middle, long)
	if !within(whole, halves, additiveError) {
		t.Fatalf("%v to %v nm gives %v W, its halves %v W", short, long,
			whole, halves)
	}
}

// planckAt is Planck's law for the power a black body gives off from a
// square metre at one wavelength in metres, worked here from the SI's
// constants and not asked of the package.
func planckAt(wavelength, kelvin float64) float64 {
	exponent := planck * speed / (wavelength * boltzmann * kelvin)
	return pair * math.Pi * planck * speed * speed /
		(math.Pow(wavelength, fifth) * math.Expm1(exponent))
}

// In a band narrow enough to be one wavelength the power is Planck's law
// there times the width of the band.
func TestANarrowBandIsPlancksLawTimesItsWidth(t *testing.T) {
	rapid.Check(t, narrowBandFollowsPlanck)
}

func narrowBandFollowsPlanck(t *rapid.T) {
	kelvin := rapid.Float64Range(heatLeast, heatMost).Draw(t, "heat")
	short, long := bandAt(t, kelvin, narrowRatio)
	middle := (short + long) * half * nanometre
	width := (long - short) * nanometre
	want := planckAt(middle, kelvin) * width
	got := heated(t, kelvin).Exitance(short, long)
	if want > none && !within(got, want, narrowError) {
		t.Fatalf("%v K gives %v W in %v to %v nm, Planck's law says %v",
			kelvin, got, short, long, want)
	}
}

// Hotter gives more, in any band.
func TestHotterGivesMoreLight(t *testing.T) {
	rapid.Check(t, hotterGivesMore)
}

func hotterGivesMore(t *rapid.T) {
	kelvin := rapid.Float64Range(heatLeast, heatMost).Draw(t, "heat")
	more := kelvin + rapid.Float64Range(unity, hotterMost).Draw(t, "more")
	short, long := bandAt(t, more, bandRatio)
	cooler, hotter := heated(t, kelvin), heated(t, more)
	if cooler.Exitance(short, long) > hotter.Exitance(short, long) {
		t.Fatalf("%v K gives more than %v K in %v to %v nm", kelvin, more,
			short, long)
	}
}

// Planck's curve is one curve, stretched. Scale the temperature and
// every wavelength the other way by one factor, and the power is the
// factor to the fourth times what it was.
func TestPlancksCurveScalesWithTemperatureAndWavelength(t *testing.T) {
	rapid.Check(t, planckScales)
}

func planckScales(t *rapid.T) {
	kelvin := rapid.Float64Range(heatLeast, heatMost).Draw(t, "heat")
	factor := rapid.Float64Range(scaleLeast, scaleMost).Draw(t, "factor")
	short, long := bandAt(t, kelvin, bandRatio)
	base := heated(t, kelvin).Exitance(short, long)
	scaled := heated(t, kelvin*factor).Exitance(short/factor, long/factor)
	if !within(scaled, base*math.Pow(factor, fourth), additiveError) {
		t.Fatalf("%v K gives %v W, and scaled by %v gives %v W", kelvin,
			base, factor, scaled)
	}
}

// stefanBoltzmann is the power a black body gives off from a square
// metre for each kelvin to the fourth. It is two pi to the fifth times k
// to the fourth over fifteen c squared h cubed. It is worked from the
// SI's constants.
func stefanBoltzmann() float64 {
	top := pair * math.Pow(math.Pi, fifth) * math.Pow(boltzmann, fourth)
	bottom := fifteen * speed * speed * math.Pow(planck, cubed)
	return top / bottom
}

// All the power a black body gives off is what Stefan and Boltzmann
// say. It is summed over bands from the shortest gamma ray to the
// longest radio wave.
func TestABlackBodyGivesWhatStefanAndBoltzmannSay(t *testing.T) {
	rapid.Check(t, blackBodyGivesStefansPower)
}

func blackBodyGivesStefansPower(t *rapid.T) {
	kelvin := rapid.Float64Range(totalHeatLeast, totalHeatMost).Draw(t,
		"heat")
	heat := heated(t, kelvin)
	var total float64
	for short := totalShortest; short < totalLongest; short *= bandRatio {
		total += heat.Exitance(short, short*bandRatio)
	}
	want := stefanBoltzmann() * math.Pow(kelvin, fourth)
	if !within(total, want, totalError) {
		t.Fatalf("%v K gives %.6g W a square metre, Stefan and "+
			"Boltzmann say %.6g", kelvin, total, want)
	}
}
