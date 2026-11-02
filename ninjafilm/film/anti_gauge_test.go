package film_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/film"
	"pgregory.net/rapid"
)

// what the lens anti-tests choose, each picked in picked-numbers.yml
var (
	wideLensFocal = data.Picked("film-wide-lens")
	longLensFocal = data.Picked("film-long-lens")
)

// anti-tests: a gauge with no name is not made, whatever its size. What
// comes back is the zero gauge.
func TestAGaugeWithNoNameIsNotMade(t *testing.T) {
	rapid.Check(t, nameLessGaugeIsRefused)
}

func nameLessGaugeIsRefused(t *rapid.T) {
	width := rapid.Float64Range(sizeLeast, sizeMost).Draw(t, "width")
	height := rapid.Float64Range(sizeLeast, sizeMost).Draw(t, "height")
	var zero film.Gauge
	if got, err := film.NewGauge("", width, height, none); err == nil ||
		got != zero {
		t.Fatalf("a gauge of no name, %v by %v, was made", width, height)
	}
}

// anti-tests: the aspect of a frame is not its height over its width.
// Only a square is its own reciprocal.
func TestTheAspectOfAFrameIsNotItsHeightOverItsWidth(t *testing.T) {
	rapid.Check(t, aspectIsNotTurnedOver)
}

func aspectIsNotTurnedOver(t *rapid.T) {
	width := rapid.Float64Range(sizeLeast, sizeMost).Draw(t, "width")
	height := width + rapid.Float64Range(whole, sizeMost).Draw(t, "more")
	made, err := film.NewGauge("drawn", width, height, none)
	if err != nil {
		t.Fatal(err)
	}
	if got := made.Aspect(); got == height/width || !(got < whole) {
		t.Fatalf("%v by %v has the aspect %v", width, height, got)
	}
}

// anti-tests: a table does not give a gauge by another name. Two rows
// with names that differ give gauges that differ.
func TestATableDoesNotGiveAGaugeByAnotherName(t *testing.T) {
	shipped := film.Shipped()
	names := gaugeNames()
	for index, name := range names[:len(names)-whole] {
		first, err := shipped.Named(name)
		if err != nil {
			t.Fatal(err)
		}
		second, err := shipped.Named(names[index+whole])
		if err != nil {
			t.Fatal(err)
		}
		if first == second || first.Name() == second.Name() {
			t.Errorf("%s and %s are one gauge", name, names[index+whole])
		}
	}
}

// anti-tests: no shipped gauge has a size of zero or less. None has a lost
// size. None has a grain less than zero.
func TestNoShippedGaugeHasNoSize(t *testing.T) {
	for _, name := range gaugeNames() {
		each := gauged(t, name)
		for _, side := range []float64{each.Width(), each.Height()} {
			if !(side > none) || math.IsInf(side, either) {
				t.Errorf("%s has a side of %v", name, side)
			}
		}
		if each.Grain() < none || math.IsNaN(each.Grain()) {
			t.Errorf("%s has a grain of %v", name, each.Grain())
		}
	}
}

// anti-tests: a gauge the code leans on is the gauge of its name and not
// another. The one Must gives is the one Named gives.
func TestMustDoesNotGiveAnotherGaugeThanNamed(t *testing.T) {
	shipped := film.Shipped()
	for _, name := range gaugeNames() {
		found, err := shipped.Named(name)
		if err != nil {
			t.Fatal(err)
		}
		if shipped.Must(name) != found {
			t.Errorf("Must gave another gauge than Named for %s", name)
		}
	}
}

// anti-tests: a lens of 24 does not see what a 200 does. The wide one
// sees more.
func TestAWideLensDoesNotSeeWhatALongOneDoes(t *testing.T) {
	frame := gauged(t, standardName)
	wide := film.NewLens(wideLensFocal, frame, none)
	long := film.NewLens(longLensFocal, frame, none)
	if !(wide.Wide() > long.Wide()) {
		t.Errorf("a 24 sees %g and a 200 sees %g", wide.Wide(), long.Wide())
	}
}

// anti-tests: the telescope is not an ordinary lens. It sees a sliver of
// what the zero lens does.
func TestTheTelescopeIsNotAnOrdinaryLens(t *testing.T) {
	var zero film.Lens
	if !(film.Hubble().Wide() < zero.Wide()/pair) {
		t.Errorf("the telescope sees %g, the zero lens %g", film.Hubble().Wide(),
			zero.Wide())
	}
}

// anti-tests: a wider field is not a longer focal length. Focal gets
// shorter as the field gets wider.
func TestAWiderFieldIsNotALongerFocalLength(t *testing.T) {
	rapid.Check(t, widerFieldIsShorter)
}

func widerFieldIsShorter(t *rapid.T) {
	frame := gauged(t, rapid.SampledFrom(gaugeNames()).Draw(t, "gauge"))
	narrow := rapid.Float64Range(wideLeast, wideMost/pair).Draw(t, "narrow")
	wide := narrow + rapid.Float64Range(wideLeast, wideMost/pair).Draw(t,
		"more")
	if !(film.Focal(wide, frame) < film.Focal(narrow, frame)) {
		t.Fatalf("a field of %v is not a shorter lens than %v", wide, narrow)
	}
}

// anti-tests: lenses of two fields do not make one ray. Rays off the
// middle of the picture look another way through a wider lens.
func TestLensesOfTwoFieldsDoNotMakeOneRay(t *testing.T) {
	rapid.Check(t, widerLensLooksOtherwise)
}

func widerLensLooksOtherwise(t *rapid.T) {
	frame := gauged(t, standardName)
	short := rapid.Float64Range(focalLeast, focalMost).Draw(t, "short")
	long := short + rapid.Float64Range(focalLeast, focalMost).Draw(t, "more")
	across, down := picture(t)
	if across < pair {
		t.Skip("a picture with no side")
	}
	band := maths.Band{}
	first := film.NewLens(short, frame, none).Ray(none, down/pair, across,
		down, band)
	second := film.NewLens(long, frame, none).Ray(none, down/pair, across,
		down, band)
	if first.Dir.Near(second.Dir, tight) {
		t.Fatalf("lenses of %v and %v looked alike at the edge", short, long)
	}
}

// anti-tests: the zero lens is not a lens of any other focal length. Half
// millimetre lenses see more than it.
func TestTheZeroLensIsNotALensOfAnotherFocalLength(t *testing.T) {
	var zero film.Lens
	other := film.NewLens(half, gauged(t, standardName), none)
	if zero.Wide() == other.Wide() || !(other.Wide() > zero.Wide()) {
		t.Errorf("the zero lens sees %g and a half millimetre lens %g",
			zero.Wide(), other.Wide())
	}
}

// anti-tests: a higher swath is not a wider lens. The same strip seen
// from twice as high sees a narrower field.
func TestAHigherSwathIsNotAWiderLens(t *testing.T) {
	rapid.Check(t, higherSwathIsNarrower)
}

func higherSwathIsNarrower(t *rapid.T) {
	above := rapid.Float64Range(wideLeast, eyeMost).Draw(t, "above")
	across := rapid.Float64Range(wideLeast, eyeMost).Draw(t, "across")
	if !(film.Swath(across, pair*above).Wide() < film.Swath(across,
		above).Wide()) {
		t.Fatalf("a strip of %v from %v and from twice it looked alike",
			across, above)
	}
}

// anti-tests: swaths of the same strip from other heights are not one
// lens. Forty from forty and forty from four hundred differ.
func TestSwathsFromOtherHeightsAreNotOneLens(t *testing.T) {
	rows := table(t, knownSwaths)
	first, second := rows[none], rows[whole]
	one := film.Swath(number(t, first, "across"), number(t, first, "above"))
	other := film.Swath(number(t, second, "across"), number(t, second,
		"above"))
	if one == other || one.Wide() == other.Wide() {
		t.Errorf("two swaths made one lens")
	}
}
