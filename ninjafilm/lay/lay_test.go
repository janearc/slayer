package lay_test

import (
	"math"
	"os"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/crystal"
	"github.com/janearc/slayer/ninjafilm/curve"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/lay"
	"github.com/janearc/slayer/ninjafilm/record"
	"pgregory.net/rapid"
)

// what these tests count in, each defined
const (
	inch     = 25400.0 // defined: an inch is 25.4 millimetres, or 25400 micrometres, by the international agreement of 1959
	none     = 0       // defined: no amount at all
	whole    = 1       // defined: the whole of the light, and the first of the records
	pair     = 2       // defined: a count of two
	upward   = 1       // defined: math.Inf's code for infinity above all numbers
	downward = -1      // defined: math.Inf's code for infinity below all numbers
)

// the tables the tests are run over
const (
	acrossFile  = "testdata/across.csv"
	refusedFile = "testdata/refused.csv"
)

// what these tests choose, each picked in picked-numbers.yml
var (
	pitchLeast     = data.Picked("lay-pitch-least")
	pitchMost      = data.Picked("lay-pitch-most")
	widerMost      = data.Picked("lay-wider-most")
	elementsLeast  = data.Picked("lay-elements-least")
	elementsMost   = data.Picked("lay-elements-most")
	elementsSample = data.Picked("lay-elements-example")
	passesSample   = data.Picked("lay-passes-example")
	passesLeast    = data.Picked("lay-passes-least")
	allPass        = data.Picked("lay-allpass-elements")
	oddLeast       = data.Picked("lay-odd-least")
	oddMost        = data.Picked("lay-odd-most")
	kindsMost      = data.Whole("lay-kinds-most")
	fieldWidth     = data.Whole("lay-field-width")
	fieldSites     = data.Whole("lay-field-sites")
	freshSites     = data.Whole("lay-fresh-sites")
	otherMost      = data.Whole("lay-other-most")
	examplePass    = data.Picked("lay-example-pass")
	coverError     = data.Picked("lay-cover-error")
	sigmas         = data.Picked("lay-six-sigma")
	grained        = data.Picked("lay-crystal")
	curveMost      = data.Picked("lay-curve-most")
	curveSlope     = data.Picked("lay-curve-slope")
)

// stopper is a test that can be stopped: a testing.T, or a rapid.T.
type stopper interface {
	Helper()
	Fatal(args ...any)
}

// records are records told apart by their windows. The first sees one
// measure of red. The second sees two.
func records(t stopper, many int) []record.Record {
	t.Helper()
	grain, err := crystal.New(grained)
	if err != nil {
		t.Fatal(err)
	}
	tone, err := curve.New(none, curveMost, curveSlope)
	if err != nil {
		t.Fatal(err)
	}
	var made []record.Record
	for index := range many {
		window := maths.RGB(float64(index+whole), none, none)
		each, err := record.New(window, maths.Grey(whole), grain, tone)
		if err != nil {
			t.Fatal(err)
		}
		made = append(made, each)
	}
	return made
}

// laid is a lay of a name, or the test stops.
func laid(t stopper, name string, perInch, passes float64) lay.Lay {
	t.Helper()
	made, err := lay.New(name, perInch, passes)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// stackedLay is the lay of records one under another.
func stackedLay(t stopper) lay.Lay { return laid(t, "stacked", none, none) }

// pitch is how wide a pixel is on the film, in micrometres.
func pitch(t *rapid.T) float64 {
	return rapid.Float64Range(pitchLeast, pitchMost).Draw(t, "pitch")
}

// table is the rows of a table of cases in testdata, or the test stops.
func table(t *testing.T, name string) []data.Row {
	t.Helper()
	text, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := data.Parse(string(text))
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

// Every pixel is one site and every site holds every record, however
// wide the pixel and wherever the site is.
func TestAStackedSiteIsEveryRecordUnderNoFilter(t *testing.T) {
	rapid.Check(t, stackedSiteHoldsEverything)
}

func stackedSiteHoldsEverything(t *rapid.T) {
	stacked := stackedLay(t)
	held := records(t, rapid.IntRange(whole, kindsMost).Draw(t, "kinds"))
	seed := rapid.Uint64().Draw(t, "seed")
	x, y := rapid.Int64().Draw(t, "x"), rapid.Int64().Draw(t, "y")
	frame := rapid.Int64().Draw(t, "frame")
	if got := stacked.Across(pitch(t)); got != whole {
		t.Fatalf("a stacked pixel is %d sites across", got)
	}
	here, filter := stacked.Site(held, seed, x, y, frame)
	if filter != maths.Grey(whole) || len(here) != len(held) {
		t.Fatalf("a stacked site holds %d of %d records under %v",
			len(here), len(held), filter)
	}
	for index := range held {
		if here[index] != held[index] {
			t.Fatalf("record %d of the site is not the stock's", index)
		}
	}
}

// anti-tests: a stacked lay given elements or a dimness is not made.
// Nor is a mosaic that cannot be, or a lay nobody has heard of. What
// comes back is no lay.
func TestALayRefusesWhatItCannotBe(t *testing.T) {
	for _, row := range table(t, refusedFile) {
		name, _ := row.Text("lay")
		perInch, _ := row.Float("per_square_inch")
		passes, _ := row.Float("passes")
		got, err := lay.New(name, perInch, passes)
		if err == nil || got != nil {
			label, _ := row.Text("label")
			t.Errorf("%s was not refused: %v, %v", label, got, err)
		}
	}
}

// odd draws an ordinary number or an odd one.
func odd() *rapid.Generator[float64] {
	strange := rapid.SampledFrom([]float64{math.NaN(), math.Inf(upward),
		math.Inf(downward), none, whole})
	return rapid.OneOf(rapid.Float64Range(oddLeast, oddMost), strange)
}

// Stacked lays are made exactly when they are given nothing.
func TestAStackedLayIsMadeExactlyWhenItIsGivenNothing(t *testing.T) {
	rapid.Check(t, stackedIsMadeFromNothing)
}

func stackedIsMadeFromNothing(t *rapid.T) {
	perInch, passes := odd().Draw(t, "perInch"), odd().Draw(t, "passes")
	want := perInch == none && passes == none
	if _, err := lay.New("stacked", perInch, passes); (err == nil) != want {
		t.Fatalf("a stacked lay of %v and %v: made is %v, want %v",
			perInch, passes, err == nil, want)
	}
}

// 6451600 is 2540 squared, so its elements are ten micrometres wide.
func TestAMosaicsElementsAreSizedByTheirCount(t *testing.T) {
	scatter := laid(t, "mosaic", elementsSample, passesSample)
	for _, row := range table(t, acrossFile) {
		width, err := row.Float("pitch")
		if err != nil {
			t.Fatal(err)
		}
		want, err := row.Whole("want")
		if err != nil {
			t.Fatal(err)
		}
		if got := scatter.Across(width); got != want {
			t.Errorf("a pixel %v wide is %d sites across, want %d",
				width, got, want)
		}
	}
	if _, err := lay.New("mosaic", allPass, whole); err != nil {
		t.Errorf("a mosaic that passes all the light was refused: %v",
			err)
	}
}

// Whether a mosaic is made follows from its numbers. Elements to the square
// inch are greater than zero and finite. The share of light is above zero and
// no more than all. The numbers drawn are often odd.
func TestAMosaicIsMadeExactlyWhenItCanBe(t *testing.T) {
	rapid.Check(t, mosaicIsMadeWhenItCan)
}

func mosaicIsMadeWhenItCan(t *rapid.T) {
	perInch, passes := odd().Draw(t, "perInch"), odd().Draw(t, "passes")
	want := perInch > none && !math.IsInf(perInch, upward) &&
		passes > none && passes <= whole
	if _, err := lay.New("mosaic", perInch, passes); (err == nil) != want {
		t.Fatalf("a mosaic of %v and %v: made is %v, want %v", perInch,
			passes, err == nil, want)
	}
}

// The sites across a pixel are the fewest elements that cover it. They
// are never fewer than one. Pixels finer than an element see one.
func TestAMosaicCoversAPixelWithTheFewestElements(t *testing.T) {
	rapid.Check(t, mosaicCoversWithFewest)
}

func mosaicCoversWithFewest(t *rapid.T) {
	perInch := rapid.Float64Range(elementsLeast, elementsMost).Draw(t,
		"perInch")
	scatter := laid(t, "mosaic", perInch, passesSample)
	element := inch / math.Sqrt(perInch)
	width := pitch(t)
	across := scatter.Across(width)
	if across < whole || float64(across)*element < width*(whole-coverError) {
		t.Fatalf("%d elements of %v do not cover %v", across, element,
			width)
	}
	if across > whole &&
		float64(across-whole)*element >= width*(whole+coverError) {
		t.Fatalf("%d elements of %v already cover %v", across-whole,
			element, width)
	}
	wider := width + rapid.Float64Range(none, widerMost).Draw(t, "more")
	if scatter.Across(wider) < across {
		t.Fatalf("a wider pixel, %v, is fewer sites across than %v",
			wider, width)
	}
}

// Mosaic sites hold one of the stock's records. It is under its own
// window, dimmed so that the windows together pass what the plate
// passes. The same site always holds the same one.
func TestAMosaicSiteHoldsOneRecordUnderItsOwnWindow(t *testing.T) {
	rapid.Check(t, mosaicSiteHoldsOne)
}

func mosaicSiteHoldsOne(t *rapid.T) {
	kinds := rapid.IntRange(whole, kindsMost).Draw(t, "kinds")
	passes := rapid.Float64Range(passesLeast, whole).Draw(t, "passes")
	scatter := laid(t, "mosaic", elementsSample, passes)
	held := records(t, kinds)
	seed := rapid.Uint64().Draw(t, "seed")
	x, y := rapid.Int64().Draw(t, "x"), rapid.Int64().Draw(t, "y")
	frame := rapid.Int64().Draw(t, "frame")
	here, filter := scatter.Site(held, seed, x, y, frame)
	if len(here) != whole {
		t.Fatalf("a mosaic site holds %d records", len(here))
	}
	window := here[none].Window()
	if want := window.Scale(passes * float64(kinds)); filter != want {
		t.Fatalf("a site of %v is under %v, want %v", window, filter, want)
	}
	if window.R < whole || window.R > float64(kinds) ||
		window.R != math.Trunc(window.R) {
		t.Fatalf("the site holds %v, which is no record of the stock's",
			window)
	}
	again, same := scatter.Site(held, seed, x, y, frame)
	if again[none] != here[none] || same != filter {
		t.Fatalf("site %d,%d held one record and then another", x, y)
	}
}

// Over a field of sites each kind of record holds its share: an equal
// part. The bounds are six standard errors of what ten thousand sites
// can show, so that only an uneven pick fails them.
func TestAMosaicHoldsEachKindOfRecordInEqualShares(t *testing.T) {
	rapid.Check(t, mosaicSharesKindsEqually)
}

func mosaicSharesKindsEqually(t *rapid.T) {
	kinds := rapid.IntRange(pair, kindsMost).Draw(t, "kinds")
	scatter := laid(t, "mosaic", elementsSample, examplePass)
	held := records(t, kinds)
	seed := rapid.Uint64().Draw(t, "seed")
	frame := rapid.Int64().Draw(t, "frame")
	picked := make([]float64, kinds)
	for index := range fieldSites {
		column, row := int64(index%fieldWidth), int64(index/fieldWidth)
		here, _ := scatter.Site(held, seed, column, row, frame)
		picked[int(here[none].Window().R)-whole]++
	}
	share := whole / float64(kinds)
	bound := sigmas * math.Sqrt(share*(whole-share)/float64(fieldSites))
	for kind, count := range picked {
		if got := count / float64(fieldSites); math.Abs(got-share) >
			bound {
			t.Fatalf("kind %d holds %v of the sites, want %v", kind, got,
				share)
		}
	}
}

// Mosaics are laid afresh for each frame and each roll. The same frame
// of the same roll is the same at every site. Another matches only as
// often as chance says.
func TestAMosaicIsLaidAfreshForEachFrameAndRoll(t *testing.T) {
	rapid.Check(t, mosaicIsLaidAfresh)
}

func mosaicIsLaidAfresh(t *rapid.T) {
	kinds := rapid.IntRange(pair, kindsMost).Draw(t, "kinds")
	scatter := laid(t, "mosaic", elementsSample, examplePass)
	held := records(t, kinds)
	seed := rapid.Uint64().Draw(t, "seed")
	frame := rapid.Int64().Draw(t, "frame")
	other := rapid.Int64Range(whole, int64(otherMost)).Draw(t, "other")
	chance := whole / float64(kinds)
	bound := sigmas * math.Sqrt(chance*(whole-chance)*float64(freshSites))
	want := chance * float64(freshSites)
	if got := matches(scatter, held, seed, frame, seed, frame); got != float64(freshSites) {
		t.Fatalf("a frame matched itself at %v sites of %d", got,
			freshSites)
	}
	if got := matches(scatter, held, seed, frame, seed, frame+other); math.Abs(got-want) > bound {
		t.Fatalf("another frame matched at %v sites of %d, want about %v",
			got, freshSites, want)
	}
	if got := matches(scatter, held, seed, frame, seed+uint64(other), frame); math.Abs(got-want) >
		bound {
		t.Fatalf("another roll matched at %v sites of %d, want about %v",
			got, freshSites, want)
	}
}

// matches is at how many sites in a row two layings of a mosaic hold
// the same record.
func matches(scatter lay.Lay, held []record.Record, seed uint64,
	frame int64, seeded uint64, framed int64) float64 {
	var alike float64
	for column := range freshSites {
		first, _ := scatter.Site(held, seed, int64(column), none, frame)
		second, _ := scatter.Site(held, seeded, int64(column), none, framed)
		if first[none] == second[none] {
			alike++
		}
	}
	return alike
}
