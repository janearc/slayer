package sensor_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/maths"
	"github.com/janearc/slayer/ninjafilm/sensor"
	"pgregory.net/rapid"
)

// what these tests count in, each defined
const (
	none     = 0  // defined: no amount at all
	whole    = 1  // defined: all of a band's light
	upward   = 1  // defined: math.Inf's code for infinity above all numbers
	downward = -1 // defined: math.Inf's code for infinity below all numbers
	either   = 0  // defined: math.IsInf's code for infinity of either sign
)

// what these tests choose, each picked in picked-numbers.yml
var (
	areaLeast     = data.Picked("sensor-area-least")
	areaMost      = data.Picked("sensor-area-most")
	linesLeast    = data.Whole("sensor-lines-least")
	linesMost     = data.Whole("sensor-lines-most")
	waveLeast     = data.Picked("sensor-wavelength-least")
	waveMost      = data.Picked("sensor-wavelength-most")
	powerLeast    = data.Picked("sensor-power-least")
	powerMost     = data.Picked("sensor-power-most")
	fractionsMost = data.Whole("sensor-fractions-most")
	oddLeast      = data.Picked("sensor-odd-least")
	oddMost       = data.Picked("sensor-odd-most")
	channelsMost  = data.Whole("sensor-channels-most")
	sumError      = data.Picked("sensor-sum-error")
)

// stopper is a test that can be stopped: a testing.T, or a rapid.T.
type stopper interface {
	Helper()
	Fatal(args ...any)
}

// areaOf is an area, or the test stops.
func areaOf(t stopper, squareMetres float64) maths.Area {
	t.Helper()
	made, err := maths.SquareMetres(squareMetres)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// responseOf is a response, or the test stops.
func responseOf(t stopper, fractions map[bands.Band]float64) sensor.Response {
	t.Helper()
	made, err := sensor.Counts(fractions)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// drawnLines are lines of arriving power at drawn wavelengths: the
// light, and the bands the lines fall in.
func drawnLines(t *rapid.T) (bands.Light, []bands.Band) {
	many := rapid.IntRange(linesLeast, linesMost).Draw(t, "lines")
	lines := map[float64]float64{}
	var falling []bands.Band
	where := rapid.Float64Range(waveLeast, waveMost)
	power := rapid.Float64Range(powerLeast, powerMost)
	for range many {
		wavelength := where.Draw(t, "where")
		lines[wavelength] = power.Draw(t, "power")
		falling = append(falling, bands.Of(wavelength))
	}
	return bands.Lines(lines), falling
}

// drawnFractions are fractions between zero and one counted in some of the
// bands the light falls in, and in some bands it does not.
func drawnFractions(t *rapid.T, falling []bands.Band) map[bands.Band]float64 {
	fractions := map[bands.Band]float64{}
	share := rapid.Float64Range(none, whole)
	for _, each := range falling {
		if rapid.Bool().Draw(t, "listed") {
			fractions[each] = share.Draw(t, "fraction")
		}
	}
	elsewhere := rapid.IntRange(none, fractionsMost).Draw(t, "elsewhere")
	for range elsewhere {
		fractions[bands.Band(rapid.Int16().Draw(t, "band"))] = share.Draw(t,
			"fraction")
	}
	fractions[bands.Of(waveLeast)] = share.Draw(t, "fraction")
	return fractions
}

// odd draws an ordinary fraction or an odd one.
func odd() *rapid.Generator[float64] {
	strange := rapid.SampledFrom([]float64{math.NaN(), math.Inf(upward),
		math.Inf(downward), none, whole})
	return rapid.OneOf(rapid.Float64Range(oddLeast, oddMost), strange)
}

// anti-tests: a response of no bands is not made. Nor is one of a fraction less
// than zero or above one or not a number. What comes back is no response.
func TestAResponseRefusesWhatAFractionCannotBe(t *testing.T) {
	blue := bands.Of(waveLeast)
	var noResponse sensor.Response
	cases := []map[bands.Band]float64{nil, {}, {blue: -whole},
		{blue: whole + whole}, {blue: math.NaN()}}
	for _, fractions := range cases {
		got, err := sensor.Counts(fractions)
		if err == nil || !reflect.DeepEqual(got, noResponse) {
			t.Errorf("a response of %v was not refused: %v, %v",
				fractions, got, err)
		}
	}
}

// Whether a response is made follows from its fractions: there is at least one,
// and each is from zero to one.
func TestAResponseIsMadeExactlyWhenEveryFractionIsFromNoughtToOne(
	t *testing.T) {
	rapid.Check(t, responseIsMadeWhenItCan)
}

func responseIsMadeWhenItCan(t *rapid.T) {
	many := rapid.IntRange(none, fractionsMost).Draw(t, "bands")
	fractions := map[bands.Band]float64{}
	want := many > none
	for place := range many {
		fraction := odd().Draw(t, "fraction")
		fractions[bands.Band(place)] = fraction
		want = want && fraction >= none && fraction <= whole
	}
	if _, err := sensor.Counts(fractions); (err == nil) != want {
		t.Fatalf("a response of %v: made is %v, want %v", fractions,
			err == nil, want)
	}
}

// anti-tests: a collector with no channels is not made. Nor is one with
// a channel that has no name. What comes back is no collector.
func TestACollectorRefusesNoChannelsAndNamelessOnes(t *testing.T) {
	area := areaOf(t, areaLeast)
	all := responseOf(t, map[bands.Band]float64{bands.Of(waveLeast): whole})
	var noCollector sensor.Collector
	for _, channels := range []map[string]sensor.Response{nil, {},
		{"": all}} {
		got, err := sensor.Collect(area, channels)
		if err == nil || !reflect.DeepEqual(got, noCollector) {
			t.Errorf("a collector with channels %v was not refused: %v, "+
				"%v", channels, got, err)
		}
	}
}

// drawnCollector is a collector of a drawn area with some channels. Each
// counts drawn fractions of the bands. It also gives the channels it was
// made from.
func drawnCollector(t *rapid.T, falling []bands.Band) (sensor.Collector,
	float64, map[string]map[bands.Band]float64) {
	squareMetres := rapid.Float64Range(areaLeast, areaMost).Draw(t, "area")
	many := rapid.IntRange(whole, channelsMost).Draw(t, "channels")
	channels := map[string]sensor.Response{}
	counted := map[string]map[bands.Band]float64{}
	for _, name := range []string{"red", "green", "blue"}[:many] {
		fractions := drawnFractions(t, falling)
		channels[name] = responseOf(t, fractions)
		counted[name] = fractions
	}
	made, err := sensor.Collect(areaOf(t, squareMetres), channels)
	if err != nil {
		t.Fatal(err)
	}
	return made, squareMetres, counted
}

// Each channel counts over every band the power arriving on each
// square metre times the area times the fraction it counts of the band.
// Bands it does not list it does not count. Channels the collector
// lacks are not in the reading.
func TestGatherCountsEachChannelItsShare(t *testing.T) {
	rapid.Check(t, gatherCountsEachShare)
}

func gatherCountsEachShare(t *rapid.T) {
	arriving, falling := drawnLines(t)
	collector, squareMetres, counted := drawnCollector(t, falling)
	reading, err := collector.Gather(arriving)
	if err != nil {
		t.Fatal(err)
	}
	for name, fractions := range counted {
		var sum float64
		for band, fraction := range fractions {
			sum += arriving.Power(band) * fraction
		}
		want := sum * squareMetres
		got, ok := reading.Channel(name)
		if !ok || math.Abs(got.Watts()-want) > sumError*math.Max(want,
			got.Watts()) {
			t.Fatalf("%s counted %v watts, not %v", name, got.Watts(), want)
		}
	}
	if power, ok := reading.Channel("violet"); ok {
		t.Fatalf("a reading answered for a channel the collector lacks: %v",
			power)
	}
}

// anti-tests: light so strong that a channel's count passes what a power
// holds is refused, and no reading comes of it.
func TestGatherRefusesACountThatPassesWhatAPowerHolds(t *testing.T) {
	area := areaOf(t, areaMost)
	blue := bands.Of(waveLeast)
	channel := responseOf(t, map[bands.Band]float64{blue: whole})
	collector, err := sensor.Collect(area, map[string]sensor.Response{
		"blue": channel})
	if err != nil {
		t.Fatal(err)
	}
	arriving := bands.Lines(map[float64]float64{waveLeast: math.MaxFloat64})
	var noReading sensor.Reading
	got, err := collector.Gather(arriving)
	if err == nil || !reflect.DeepEqual(got, noReading) {
		t.Errorf("a count past what a power holds was read: %v, %v", got,
			err)
	}
}
