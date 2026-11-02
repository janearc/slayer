package record_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/crystal"
	"github.com/janearc/slayer/ninjafilm/record"
	"pgregory.net/rapid"
)

// anti-tests: a record does not keep a window it was not given. Records
// of two windows that differ are told apart by them.
func TestARecordDoesNotKeepAWindowItWasNotGiven(t *testing.T) {
	rapid.Check(t, recordKeepsOnlyItsOwnWindow)
}

func recordKeepsOnlyItsOwnWindow(t *rapid.T) {
	_, window, _, tone := drawn(t)
	other := window.Scale(pair)
	image := drawnLamps().Draw(t, "image")
	first := recordOf(t, window, image, acrossLeast, tone)
	second := recordOf(t, other, image, acrossLeast, tone)
	if first.Window() == second.Window() {
		t.Fatalf("windows %v and %v were kept as one", window, other)
	}
}

// anti-tests: a window or an image that adds to zero or less is not a record,
// however the other is. The lamps are drawn dark or less than zero.
func TestADarkWindowOrImageMakesNoRecord(t *testing.T) {
	rapid.Check(t, darkWindowOrImageMakesNone)
}

func darkWindowOrImageMakesNone(t *rapid.T) {
	dark := maths.Grey(-rapid.Float64Range(none, oddMost).Draw(t, "dark"))
	lit := drawnLamps().Draw(t, "lit")
	grain, err := crystal.New(acrossLeast)
	if err != nil {
		t.Fatal(err)
	}
	tone := toneOf(t, none, unity, unity)
	var zero record.Record
	if made, err := record.New(dark, lit, grain, tone); err == nil ||
		made != zero {
		t.Fatalf("a dark window %v made a record", dark)
	}
	if made, err := record.New(lit, dark, grain, tone); err == nil ||
		made != zero {
		t.Fatalf("a dark image %v made a record", dark)
	}
}

// anti-tests: the middle grey does not land at either end of the curve.
// It is neither the most density nor the least.
func TestTheMiddleGreyDoesNotLandAtAnEndOfTheCurve(t *testing.T) {
	rapid.Check(t, middleGreyIsNotAnEnd)
}

func middleGreyIsNotAnEnd(t *rapid.T) {
	made, _, _, tone := drawn(t)
	shade := maths.Grey(grey())
	got := made.Developed(shade, area(t), none)
	most, least := tone.Density(none), tone.Density(math.Inf(upward))
	if within(got, most, toleranceLoose) ||
		within(got, least, toleranceLoose) {
		t.Fatalf("the middle grey develops to %v, an end", got)
	}
}

// anti-tests: no record develops past the ends of its curve, whatever
// the light. That includes light that is lost.
func TestNoRecordDevelopsPastTheEndsOfItsCurve(t *testing.T) {
	rapid.Check(t, recordStaysInsideItsCurve)
}

func recordStaysInsideItsCurve(t *rapid.T) {
	made, _, _, tone := drawn(t)
	light := lampsUpTo(t, "light", brightMost)
	got := made.Developed(light, area(t), none)
	least, most := tone.Density(math.Inf(upward)), tone.Density(none)
	if got < least-toleranceLoose || got > most+toleranceLoose ||
		math.IsNaN(got) {
		t.Fatalf("%v develops to %v outside %v to %v", light, got, least, most)
	}
}

// anti-tests: light in the window does expose the record. Bright light
// in every lamp leaves it less dense than a record nothing exposes.
func TestLightInTheWindowDoesExposeTheRecord(t *testing.T) {
	rapid.Check(t, windowLightExposes)
}

func windowLightExposes(t *rapid.T) {
	made, _, _, tone := drawn(t)
	bright := maths.Grey(rapid.Float64Range(whiteMost, brightMost).Draw(t,
		"bright"))
	got := made.Developed(bright, area(t), none)
	if got >= tone.Density(none)-toleranceLoose {
		t.Fatalf("bright light %v left the record at %v", bright, got)
	}
}

// anti-tests: records of other windows do not expose alike to coloured
// light. Red windows and blue ones meet red light differently.
func TestRecordsOfOtherWindowsDoNotExposeAlikeToColour(t *testing.T) {
	rapid.Check(t, coloursTellWindowsApart)
}

func coloursTellWindowsApart(t *rapid.T) {
	tone := toneOf(t, whiteLeast, whiteCurveMost, whiteSlope)
	image := drawnLamps().Draw(t, "image")
	red := recordOf(t, maths.RGB(unity, none, none), image, whiteCrystal,
		tone)
	blue := recordOf(t, maths.RGB(none, none, unity), image, whiteCrystal,
		tone)
	light := maths.RGB(rapid.Float64Range(whiteMost, brightMost).Draw(t,
		"red"), none, none)
	over := area(t)
	if within(red.Developed(light, over, none), blue.Developed(light, over,
		none), toleranceLoose) {
		t.Fatalf("red light developed a red and a blue record alike")
	}
}

// anti-tests: a bigger draw of chance moves a record further. Two draws
// move it by more than one does.
func TestABiggerDrawOfChanceMovesARecordFurther(t *testing.T) {
	rapid.Check(t, biggerChanceMovesFurther)
}

func biggerChanceMovesFurther(t *rapid.T) {
	made, _, _, _ := drawn(t)
	light := lampsUpTo(t, "light", brightMost)
	over := area(t)
	smooth := made.Developed(light, over, none)
	once := made.Developed(light, over, unity) - smooth
	twice := made.Developed(light, over, pair) - smooth
	if once > none && !(twice > once) {
		t.Fatalf("a draw of two moved %v, a draw of one moved %v", twice,
			once)
	}
}

// anti-tests: darkness is as dense as it gets. Records that nothing
// exposes are never less dense than ones some light exposes.
func TestNoLightIsNeverLessDenseThanSomeLight(t *testing.T) {
	rapid.Check(t, darkIsAsDenseAsItGets)
}

func darkIsAsDenseAsItGets(t *rapid.T) {
	made, _, _, _ := drawn(t)
	light := lampsUpTo(t, "light", dimMost)
	over := area(t)
	if made.Developed(maths.Light{}, over, none) < made.Developed(light, over,
		none)-toleranceLoose {
		t.Fatalf("light %v developed denser than none", light)
	}
}
