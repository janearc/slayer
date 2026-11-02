package record_test

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/crystal"
	"github.com/janearc/slayer/ninjafilm/curve"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/record"
	"pgregory.net/rapid"
)

// what these tests count in, each defined
const (
	none     = 0   // defined: no amount at all
	unity    = 1   // defined: one, the whole of a lamp
	pair     = 2   // defined: a count of two
	cubed    = 3   // defined: lightness is the cube root of luminance
	quarter  = 4   // defined: the area of a disc is pi times its width squared over four
	halfway  = 0.5 // defined: the lightness of the middle grey, half of libtheme's scale
	upward   = 1   // defined: math.Inf's code for infinity above all numbers
	downward = -1  // defined: math.Inf's code for infinity below all numbers
	either   = 0   // defined: math.IsInf's code for infinity of either sign
)

// the table of records that are refused
const refusedFile = "testdata/refused.csv"

// what these tests choose, each picked in picked-numbers.yml
var (
	acrossLeast    = data.Picked("record-across-least")
	acrossMost     = data.Picked("record-across-most")
	floorLeast     = data.Picked("record-floor-least")
	floorMost      = data.Picked("record-floor-most")
	roomLeast      = data.Picked("record-room-least")
	roomMost       = data.Picked("record-room-most")
	slopeLeast     = data.Picked("record-slope-least")
	slopeMost      = data.Picked("record-slope-most")
	areaLeast      = data.Picked("record-area-least")
	areaMost       = data.Picked("record-area-most")
	lampLeast      = data.Picked("record-lamp-least")
	brightMost     = data.Picked("record-bright-most")
	dimMost        = data.Picked("record-dim-most")
	outsideMost    = data.Picked("record-outside-most")
	whiteMost      = data.Picked("record-white-most")
	scaleLeast     = data.Picked("record-scale-least")
	scaleMost      = data.Picked("record-scale-most")
	chanceMost     = data.Picked("record-chance-most")
	oddMost        = data.Picked("record-odd-most")
	whiteLeast     = data.Picked("record-white-curve-least")
	whiteCurveMost = data.Picked("record-white-curve-most")
	whiteSlope     = data.Picked("record-white-curve-slope")
	whiteCrystal   = data.Picked("record-white-crystal")
	toleranceLoose = data.Picked("tolerance-loose")
	toleranceFloat = data.Picked("tolerance-single")
)

// stopper is a test that can be stopped: a testing.T, or a rapid.T.
type stopper interface {
	Helper()
	Fatal(args ...any)
}

// within is whether two numbers are within a tolerance of each other.
func within(one, other, tolerance float64) bool {
	return math.Abs(one-other) <= tolerance
}

// grey is the luminance of the grey halfway up libtheme's lightness,
// which a stock is rated so that it lands on the middle of its curve.
func grey() float64 {
	_, luminance, _ := ok.Grey(halfway).Swatch().XYZ()
	return luminance
}

// toneOf is a curve, or the test stops.
func toneOf(t stopper, least, most, slope float64) curve.Curve {
	t.Helper()
	made, err := curve.New(least, most, slope)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// recordOf is a record, or the test stops.
func recordOf(t stopper, window, image maths.Light, across float64,
	tone curve.Curve) record.Record {
	t.Helper()
	grain, err := crystal.New(across)
	if err != nil {
		t.Fatal(err)
	}
	made, err := record.New(window, image, grain, tone)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// shines is whether a light has any light in it.
func shines(light maths.Light) bool {
	return light.R+light.G+light.B > none
}

// drawnLamps is a light of some lamps lit by various amounts and some
// not lit, at least one lit.
func drawnLamps() *rapid.Generator[maths.Light] {
	return rapid.Custom(someLamps).Filter(shines)
}

// someLamps is a light with each lamp lit or not.
func someLamps(t *rapid.T) maths.Light {
	darkLamp := float64(none)
	lamp := rapid.OneOf(rapid.Just(darkLamp),
		rapid.Float64Range(lampLeast, unity))
	return maths.RGB(lamp.Draw(t, "red"), lamp.Draw(t, "green"),
		lamp.Draw(t, "blue"))
}

// lampsUpTo is a light with each lamp drawn up to a ceiling.
func lampsUpTo(t *rapid.T, label string, ceiling float64) maths.Light {
	lamp := rapid.Float64Range(none, ceiling)
	return maths.RGB(lamp.Draw(t, label), lamp.Draw(t, label),
		lamp.Draw(t, label))
}

// drawn is a record a film could have, with its parts.
func drawn(t *rapid.T) (made record.Record, window maths.Light,
	across float64, tone curve.Curve) {
	window = drawnLamps().Draw(t, "window")
	image := drawnLamps().Draw(t, "image")
	across = rapid.Float64Range(acrossLeast, acrossMost).Draw(t, "across")
	least := rapid.Float64Range(floorLeast, floorMost).Draw(t, "least")
	room := rapid.Float64Range(roomLeast, roomMost).Draw(t, "room")
	slope := rapid.Float64Range(slopeLeast, slopeMost).Draw(t, "slope")
	tone = toneOf(t, least, least+room, slope)
	return recordOf(t, window, image, across, tone), window, across, tone
}

// area is a site on the film in square micrometres, large enough to
// hold many crystals.
func area(t *rapid.T) float64 {
	return rapid.Float64Range(areaLeast, areaMost).Draw(t, "area")
}

// seenBy is how strongly a light exposes a window against the middle
// grey. It is worked here from the window's lamps.
func seenBy(light, window maths.Light) float64 {
	taken := light.R*window.R + light.G*window.G + light.B*window.B
	return taken / (window.R + window.G + window.B) / grey()
}

func TestARecordSeesSomeOfTheLight(t *testing.T) {
	rapid.Check(t, recordKeepsWhatItIsGiven)
}

func recordKeepsWhatItIsGiven(t *rapid.T) {
	made, window, _, _ := drawn(t)
	image := drawnLamps().Draw(t, "image")
	again := recordOf(t, window, image, acrossLeast, toneOf(t, none, unity,
		unity))
	if made.Window() != window || again.Window() != window ||
		again.Image() != image {
		t.Fatalf("a record of %v and %v has %v and %v", window, image,
			again.Window(), again.Image())
	}
}

// anti-tests: a window no lamp shines in and an image that blocks no
// lamp make no record. Each is refused for its own reason.
func TestARecordRefusesWhatItCannotBe(t *testing.T) {
	grain, err := crystal.New(acrossLeast)
	if err != nil {
		t.Fatal(err)
	}
	tone := toneOf(t, none, unity, unity)
	text, err := os.ReadFile(refusedFile)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := data.Parse(string(text))
	if err != nil {
		t.Fatal(err)
	}
	var zero record.Record
	for _, row := range rows {
		window, image := lightsOf(t, row, "window"), lightsOf(t, row, "image")
		says, _ := row.Text("says")
		got, err := record.New(window, image, grain, tone)
		if err == nil || got != zero || !strings.Contains(err.Error(),
			says) {
			t.Errorf("%v and %v was not refused for %q: %v, %v", window,
				image, says, got, err)
		}
	}
}

// lightsOf is the light named by the three columns that start with a
// word.
func lightsOf(t *testing.T, row data.Row, word string) maths.Light {
	t.Helper()
	return maths.RGB(lampOf(t, row, word+"_red"),
		lampOf(t, row, word+"_green"), lampOf(t, row, word+"_blue"))
}

// lampOf is the number in a column of a row, or the test stops.
func lampOf(t *testing.T, row data.Row, column string) float64 {
	t.Helper()
	amount, err := row.Float(column)
	if err != nil {
		t.Fatal(err)
	}
	return amount
}

// Whether a record is made follows from its lamps alone: the window and the
// image must each add up to greater than zero and be a number.
func TestARecordIsMadeExactlyWhenItSeesAndBlocksSomeLight(t *testing.T) {
	rapid.Check(t, recordIsMadeWhenItCan)
}

// oddLamp is a lamp of an ordinary size or an odd one.
func oddLamp() *rapid.Generator[float64] {
	odd := rapid.SampledFrom([]float64{math.NaN(), math.Inf(upward),
		math.Inf(downward), none})
	return rapid.OneOf(rapid.Float64Range(-oddMost, oddMost), odd)
}

// good is whether a light adds up to greater than zero and to a number.
func good(light maths.Light) bool {
	total := light.R + light.G + light.B
	return total > none && !math.IsInf(total, either)
}

func recordIsMadeWhenItCan(t *rapid.T) {
	lamp := oddLamp()
	window := maths.RGB(lamp.Draw(t, "r"), lamp.Draw(t, "g"),
		lamp.Draw(t, "b"))
	image := maths.RGB(lamp.Draw(t, "ir"), lamp.Draw(t, "ig"),
		lamp.Draw(t, "ib"))
	grain, err := crystal.New(acrossLeast)
	if err != nil {
		t.Fatal(err)
	}
	_, err = record.New(window, image, grain, toneOf(t, none, unity, unity))
	if want := good(window) && good(image); (err == nil) != want {
		t.Fatalf("%v and %v: made is %v, want %v", window, image,
			err == nil, want)
	}
}

// The middle grey is halfway up libtheme's lightness. That is an eighth
// of the light a white gives back. It lands on the middle of the curve.
func TestTheMiddleGreyLandsOnTheMiddleOfTheCurve(t *testing.T) {
	if got, want := grey(), math.Pow(halfway, cubed); !within(got, want,
		toleranceFloat) {
		t.Fatalf("the middle grey gives back %v of the light, want %v",
			got, want)
	}
	rapid.Check(t, middleGreyLandsInTheMiddle)
}

func middleGreyLandsInTheMiddle(t *rapid.T) {
	made, _, _, tone := drawn(t)
	want := tone.Density(unity)
	shade := maths.Grey(grey())
	if got := made.Developed(shade, area(t), none); !within(got, want,
		toleranceLoose) {
		t.Fatalf("the middle grey develops to %v, not %v", got, want)
	}
}

// Exposure is the light in the window against the middle grey. With no
// chance a record develops to its curve at that exposure, and counting
// crystals out and back loses nothing.
func TestARecordDevelopsToItsCurveAtTheLightInItsWindow(t *testing.T) {
	rapid.Check(t, recordDevelopsToItsCurve)
}

func recordDevelopsToItsCurve(t *rapid.T) {
	made, window, _, tone := drawn(t)
	light := lampsUpTo(t, "light", brightMost)
	want := tone.Density(seenBy(light, window))
	got := made.Developed(light, area(t), none)
	if !within(got, want, toleranceLoose) {
		t.Fatalf("%v on a window of %v develops to %v, its curve says %v",
			light, window, got, want)
	}
}

// Light in lamps outside the window does not expose the record, and a
// record that nothing exposes is as dense as it gets.
func TestLightOutsideTheWindowLeavesTheRecordUnexposed(t *testing.T) {
	rapid.Check(t, outsideLightExposesNothing)
}

func outsideLightExposesNothing(t *rapid.T) {
	made, window, _, tone := drawn(t)
	outside := lampsUpTo(t, "outside", outsideMost)
	if window.R > none {
		outside.R = none
	}
	if window.G > none {
		outside.G = none
	}
	if window.B > none {
		outside.B = none
	}
	got, most := made.Developed(outside, area(t), none), tone.Density(none)
	if !within(got, most, toleranceLoose) {
		t.Fatalf("%v outside %v develops to %v, not %v", outside, window,
			got, most)
	}
}

// Only which lamps are in the window matters. White light exposes every
// record alike, whatever its window.
func TestAWhiteLightExposesEveryRecordAlike(t *testing.T) {
	rapid.Check(t, whiteExposesAlike)
}

func whiteExposesAlike(t *rapid.T) {
	tone := toneOf(t, whiteLeast, whiteCurveMost, whiteSlope)
	image := drawnLamps().Draw(t, "image")
	one := recordOf(t, drawnLamps().Draw(t, "window"), image, whiteCrystal,
		tone)
	other := recordOf(t, drawnLamps().Draw(t, "other"), image,
		whiteCrystal, tone)
	white := maths.Grey(rapid.Float64Range(none, whiteMost).Draw(t, "white"))
	over := area(t)
	first, second := one.Developed(white, over, none),
		other.Developed(white, over, none)
	if !within(first, second, toleranceLoose) {
		t.Fatalf("white %v develops one record to %v and another to %v",
			white, first, second)
	}
	scale := rapid.Float64Range(scaleLeast, scaleMost).Draw(t, "scale")
	window := drawnLamps().Draw(t, "scaled")
	light := lampsUpTo(t, "light", dimMost)
	plain := recordOf(t, window, image, whiteCrystal, tone)
	wider := recordOf(t, window.Scale(scale), image, whiteCrystal, tone)
	first, second = plain.Developed(light, over, none),
		wider.Developed(light, over, none)
	if !within(first, second, toleranceLoose) {
		t.Fatalf("a window %v times as strong changes %v to %v", scale,
			first, second)
	}
}

// With no chance in it a record develops to its curve. One draw of
// chance moves it by its granularity. That is Siedentopf's: the square
// root of the logarithm of e times a crystal's shadow times the density
// over the area.
func TestOneDrawOfChanceMovesARecordByItsGranularity(t *testing.T) {
	rapid.Check(t, chanceMovesByGranularity)
}

func chanceMovesByGranularity(t *rapid.T) {
	made, _, across, _ := drawn(t)
	light := lampsUpTo(t, "light", brightMost)
	over := area(t)
	smooth := made.Developed(light, over, none)
	shadow := math.Pi * across * across / quarter
	grainy := math.Sqrt(math.Log10E * shadow * smooth / over)
	up := made.Developed(light, over, unity) - smooth
	down := smooth - made.Developed(light, over, -unity)
	if !within(up, grainy, toleranceLoose) ||
		!within(down, grainy, toleranceLoose) {
		t.Fatalf("a draw of one moves %v up and %v down, granularity "+
			"says %v", up, down, grainy)
	}
}

// More chance never makes a record less dense, and more light never
// makes it denser.
func TestMoreChanceIsDenserAndMoreLightIsNot(t *testing.T) {
	rapid.Check(t, chanceAndLightMoveDensityOneWay)
}

func chanceAndLightMoveDensityOneWay(t *rapid.T) {
	made, _, _, _ := drawn(t)
	light := lampsUpTo(t, "light", dimMost)
	over := area(t)
	low := rapid.Float64Range(-chanceMost, chanceMost).Draw(t, "low")
	high := low + rapid.Float64Range(none, chanceMost).Draw(t, "more")
	if made.Developed(light, over, high) < made.Developed(light, over,
		low) {
		t.Fatalf("chance %v develops less than chance %v", high, low)
	}
	twice := made.Developed(light.Scale(pair), over, none)
	if twice > made.Developed(light, over, none)+toleranceLoose {
		t.Fatalf("twice the light %v develops denser", light)
	}
}
