package lay_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/lay"
	"pgregory.net/rapid"
)

// anti-tests: a stacked site is never fewer records than the stock holds.
// It is not picked down to one, and its filter is not dimmed.
func TestAStackedSiteIsNeverPickedDownToOneRecord(t *testing.T) {
	rapid.Check(t, stackedSiteIsNotPicked)
}

func stackedSiteIsNotPicked(t *rapid.T) {
	held := records(t, rapid.IntRange(pair, kindsMost).Draw(t, "kinds"))
	x, y := rapid.Int64().Draw(t, "x"), rapid.Int64().Draw(t, "y")
	here, filter := stackedLay(t).Site(held, rapid.Uint64().Draw(t, "seed"),
		x, y, rapid.Int64().Draw(t, "frame"))
	if len(here) == whole || filter != maths.Grey(whole) {
		t.Fatalf("a stacked site held %d of %d under %v", len(here),
			len(held), filter)
	}
}

// anti-tests: a stacked lay given any element or any dimness is refused,
// and no lay comes back.
func TestAStackedLayGivenSomethingIsRefused(t *testing.T) {
	rapid.Check(t, stackedGivenSomething)
}

func stackedGivenSomething(t *rapid.T) {
	given := rapid.Float64Range(math.SmallestNonzeroFloat64, oddMost)
	var perInch, passes float64
	if rapid.Bool().Draw(t, "swap") {
		passes = given.Draw(t, "passes")
	} else {
		perInch = given.Draw(t, "perInch")
	}
	if made, err := lay.New("stacked", perInch, passes); err == nil ||
		made != nil {
		t.Fatalf("a stacked lay of %v and %v was made", perInch, passes)
	}
}

// anti-tests: the share of light a mosaic passes does not size its
// elements. Two mosaics of one count and two shares are as many sites
// across.
func TestTheShareOfLightDoesNotSizeAMosaicsElements(t *testing.T) {
	rapid.Check(t, passesDoNotSizeElements)
}

func passesDoNotSizeElements(t *rapid.T) {
	perInch := rapid.Float64Range(elementsLeast, elementsMost).Draw(t,
		"perInch")
	dim := passesSample
	first := laid(t, "mosaic", perInch, dim)
	second := laid(t, "mosaic", perInch, whole)
	width := pitch(t)
	if first.Across(width) != second.Across(width) {
		t.Fatalf("a share of %v and of all gave %d and %d sites across",
			dim, first.Across(width), second.Across(width))
	}
}

// anti-tests: mosaics that pass none of the light are refused. So are
// ones that pass more than all of it. So is one with no elements. No lay
// comes back.
func TestAMosaicThatPassesNoneOrMoreThanAllIsRefused(t *testing.T) {
	rapid.Check(t, mosaicOutOfRangeIsRefused)
}

func mosaicOutOfRangeIsRefused(t *rapid.T) {
	perInch := rapid.Float64Range(elementsLeast, elementsMost).Draw(t,
		"perInch")
	over := whole + rapid.Float64Range(passesLeast, oddMost).Draw(t, "over")
	under := -rapid.Float64Range(none, oddMost).Draw(t, "under")
	for _, passes := range []float64{over, under, none} {
		if made, err := lay.New("mosaic", perInch, passes); err == nil ||
			made != nil {
			t.Fatalf("a mosaic passing %v was made", passes)
		}
	}
	if made, err := lay.New("mosaic", under, whole); err == nil ||
		made != nil {
		t.Fatalf("a mosaic of %v elements was made", under)
	}
}

// anti-tests: no pixel is zero sites across. Pixels finer than an
// element see exactly one, and never none.
func TestNoPixelIsFewerThanOneSiteAcross(t *testing.T) {
	rapid.Check(t, noPixelIsFewerThanOne)
}

func noPixelIsFewerThanOne(t *rapid.T) {
	perInch := rapid.Float64Range(elementsLeast, elementsMost).Draw(t,
		"perInch")
	scatter := laid(t, "mosaic", perInch, passesSample)
	element := inch / math.Sqrt(perInch)
	fine := rapid.Float64Range(math.SmallestNonzeroFloat64, element).Draw(t,
		"fine")
	if got := scatter.Across(fine); got != whole {
		t.Fatalf("a pixel %v wide, finer than %v, is %d sites across", fine,
			element, got)
	}
}

// anti-tests: a mosaic site never holds two records or none, and never a
// record that is not the stock's.
func TestAMosaicSiteNeverHoldsTwoRecordsOrNone(t *testing.T) {
	rapid.Check(t, mosaicSiteHoldsExactlyOne)
}

func mosaicSiteHoldsExactlyOne(t *rapid.T) {
	kinds := rapid.IntRange(pair, kindsMost).Draw(t, "kinds")
	held := records(t, kinds)
	scatter := laid(t, "mosaic", elementsSample, examplePass)
	x, y := rapid.Int64().Draw(t, "x"), rapid.Int64().Draw(t, "y")
	here, _ := scatter.Site(held, rapid.Uint64().Draw(t, "seed"), x, y,
		rapid.Int64().Draw(t, "frame"))
	if len(here) != whole {
		t.Fatalf("a mosaic site of %d kinds held %d records", kinds,
			len(here))
	}
	found := false
	for _, each := range held {
		found = found || each == here[none]
	}
	if !found {
		t.Fatalf("the site held a record that is not the stock's")
	}
}

// anti-tests: no kind of record is left out of a field of sites and none
// takes them all, when the stock holds two or more.
func TestNoKindOfRecordIsLeftOutOrTakesEverySite(t *testing.T) {
	rapid.Check(t, noKindIsLeftOutOrTakesAll)
}

func noKindIsLeftOutOrTakesAll(t *rapid.T) {
	kinds := rapid.IntRange(pair, kindsMost).Draw(t, "kinds")
	scatter := laid(t, "mosaic", elementsSample, examplePass)
	held := records(t, kinds)
	seed := rapid.Uint64().Draw(t, "seed")
	frame := rapid.Int64().Draw(t, "frame")
	picked := make([]int, kinds)
	for index := range fieldSites {
		column, row := int64(index%fieldWidth), int64(index/fieldWidth)
		here, _ := scatter.Site(held, seed, column, row, frame)
		picked[int(here[none].Window().R)-whole]++
	}
	for kind, count := range picked {
		if count == none || count == fieldSites {
			t.Fatalf("kind %d holds %d of %d sites", kind, count, fieldSites)
		}
	}
}

// anti-tests: another frame is not the same laying. It does not match at
// every site.
func TestAnotherFrameDoesNotMatchAtEverySite(t *testing.T) {
	rapid.Check(t, anotherFrameDoesNotMatch)
}

func anotherFrameDoesNotMatch(t *rapid.T) {
	kinds := rapid.IntRange(pair, kindsMost).Draw(t, "kinds")
	scatter := laid(t, "mosaic", elementsSample, examplePass)
	held := records(t, kinds)
	seed := rapid.Uint64().Draw(t, "seed")
	frame := rapid.Int64().Draw(t, "frame")
	other := rapid.Int64Range(whole, int64(otherMost)).Draw(t, "other")
	if got := matches(scatter, held, seed, frame, seed, frame+other); got ==
		float64(freshSites) {
		t.Fatalf("frame %d matched frame %d at every site", frame+other,
			frame)
	}
}
