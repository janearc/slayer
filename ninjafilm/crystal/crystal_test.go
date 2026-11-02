package crystal_test

import (
	"math"
	"os"
	"strconv"
	"testing"

	"github.com/janearc/slayer/ninjafilm/crystal"
	"github.com/janearc/slayer/ninjafilm/data"
	"pgregory.net/rapid"
)

// what these tests count in, each defined
const (
	none         = 0   // defined: no amount at all
	pair         = 2   // defined: a count of two
	quarter      = 4   // defined: the area of a disc is pi times its width squared over four
	sixteenth    = 16  // defined: a disc half a micrometre wide shadows pi over sixteen square micrometres
	halfWide     = 0.5 // defined: half of a micrometre
	unity        = 1   // defined: one
	upward       = 1   // defined: math.Inf's code for infinity above all numbers
	downward     = -1  // defined: math.Inf's code for infinity below all numbers
	either       = 0   // defined: math.IsInf's code for infinity of either sign
	decimal      = 10  // defined: the base ten numbers are written in
	wordBits     = 64  // defined: a uint64 is 64 bits wide
	fractionBits = 53  // defined: the hash's fraction is 53 bits, so its smallest step is 2 to the minus 53
	rootTwo      = 2   // defined: Box and Muller's radius is the root of two times a logarithm
)

// the tables the tests are run over
const (
	countedFile = "testdata/counted.csv"
	pinnedFile  = "testdata/chance-pinned.csv"
)

// what these tests choose, each picked in picked-numbers.yml
var (
	widthLeast     = data.Picked("crystal-width-least")
	widthMost      = data.Picked("crystal-width-most")
	densityLeast   = data.Picked("crystal-density-least")
	densityMost    = data.Picked("crystal-density-most")
	areaLeast      = data.Picked("crystal-area-least")
	areaMost       = data.Picked("crystal-area-most")
	countLeast     = data.Picked("crystal-count-least")
	countMost      = data.Picked("crystal-count-most")
	chanceMost     = data.Picked("crystal-chance-most")
	moreMost       = data.Picked("crystal-more-most")
	stepLeast      = int64(data.Whole("crystal-step-least"))
	stepMost       = int64(data.Whole("crystal-step-most"))
	fieldWidth     = data.Whole("crystal-field-width")
	fieldSites     = data.Whole("crystal-field-sites")
	meanError      = data.Picked("crystal-mean-error")
	spreadError    = data.Picked("crystal-spread-error")
	shareError     = data.Picked("crystal-share-error")
	expectedMost   = data.Picked("crystal-count-most")
	toleranceTight = data.Picked("tolerance-tight")
)

// within is whether two numbers are as near as a tolerance of the
// larger one, which is how a count of crystals is compared.
func within(one, other, tolerance float64) bool {
	return math.Abs(one-other) <= tolerance*math.Max(math.Abs(one),
		math.Abs(other))
}

// grain is a crystal a film could have, and its width.
func grain(t *rapid.T) (crystal.Crystal, float64) {
	across := rapid.Float64Range(widthLeast, widthMost).Draw(t, "width")
	made, err := crystal.New(across)
	if err != nil {
		t.Fatalf("a crystal %v wide was refused: %v", across, err)
	}
	return made, across
}

// density is a density a film could reach.
func density(t *rapid.T) float64 {
	return rapid.Float64Range(densityLeast, densityMost).Draw(t, "density")
}

// area is the area of a site on a film, in square micrometres.
func area(t *rapid.T) float64 {
	return rapid.Float64Range(areaLeast, areaMost).Draw(t, "area")
}

func TestACrystalHasAWidth(t *testing.T) {
	made, err := crystal.New(halfWide)
	if err != nil {
		t.Fatalf("a crystal half a micrometre wide was refused: %v", err)
	}
	// the shadow of a disc half a micrometre across is pi over sixteen
	// square micrometres, and a density of ln 10 takes ln 10 of the
	// area's worth of them
	want := math.Ln10 * math.Ln10 * sixteenth / math.Pi
	if got := made.Expected(math.Ln10, unity); !within(got, want,
		toleranceTight) {
		t.Errorf("a crystal half a micrometre wide gives %v, want %v",
			got, want)
	}
}

// anti-tests: no width, a negative width and a width that is no number
// make no crystal. What comes back is the zero crystal.
func TestACrystalRefusesWhatItCannotBe(t *testing.T) {
	var zero crystal.Crystal
	for _, bad := range []float64{none, math.Copysign(none, downward),
		-unity, -math.MaxFloat64, math.NaN(), math.Inf(upward),
		math.Inf(downward)} {
		got, err := crystal.New(bad)
		if err == nil || got != zero {
			t.Errorf("a crystal %v wide was not refused: %v, %v", bad,
				got, err)
		}
	}
}

// Crystals are made exactly when the width is greater than zero and finite. The
// widths drawn here are often the odd ones.
func TestACrystalIsMadeExactlyWhenItHasAWidth(t *testing.T) {
	rapid.Check(t, crystalIsMadeWhenItHasAWidth)
}

func crystalIsMadeWhenItHasAWidth(t *rapid.T) {
	across := rapid.Float64().Draw(t, "across")
	want := across > none && !math.IsInf(across, either)
	if _, err := crystal.New(across); (err == nil) != want {
		t.Fatalf("a crystal %v wide: made is %v, want %v", across,
			err == nil, want)
	}
}

// Nutting's law: a density of one is crystals whose shadows add up to
// the natural logarithm of ten times the area. Each shadow is a disc.
func TestADensityIsACountOfCrystalsWhoseShadowsFillItsShare(
	t *testing.T) {
	rapid.Check(t, shadowsFillTheShare)
}

func shadowsFillTheShare(t *rapid.T) {
	made, across := grain(t)
	wanted, over := density(t), area(t)
	shadow := math.Pi * across * across / quarter
	count := made.Expected(wanted, over)
	covered := count * shadow / over
	if !within(covered, wanted*math.Ln10, toleranceTight) {
		t.Fatalf("%v crystals cover %v of the area, want %v", count,
			covered, wanted*math.Ln10)
	}
}

// Twice the density or twice the area takes twice the crystals. Twice
// the width takes a quarter of them, for each shadow is four times the
// size.
func TestCrystalsCountInProportion(t *testing.T) {
	rapid.Check(t, crystalsCountInProportion)
}

func crystalsCountInProportion(t *rapid.T) {
	made, across := grain(t)
	wanted, over := density(t), area(t)
	count := made.Expected(wanted, over)
	if got := made.Expected(pair*wanted, over); !within(got, pair*count,
		toleranceTight) {
		t.Fatalf("twice the density is %v crystals, want %v", got,
			pair*count)
	}
	if got := made.Expected(wanted, pair*over); !within(got, pair*count,
		toleranceTight) {
		t.Fatalf("twice the area is %v crystals, want %v", got,
			pair*count)
	}
	broad, err := crystal.New(pair * across)
	if err != nil {
		t.Fatal(err)
	}
	if got := broad.Expected(wanted, over); !within(got, count/quarter,
		toleranceTight) {
		t.Fatalf("twice the width is %v crystals, want %v", got,
			count/quarter)
	}
}

// Counting a density in crystals and back again gives the density.
func TestADensityCountedAndUncountedIsTheSame(t *testing.T) {
	rapid.Check(t, densityCountsBack)
}

func densityCountsBack(t *rapid.T) {
	made, _ := grain(t)
	wanted, over := density(t), area(t)
	back := made.Density(made.Expected(wanted, over), over)
	if !within(back, wanted, toleranceTight) {
		t.Fatalf("a density of %v counted and uncounted is %v", wanted,
			back)
	}
}

// The same crystals over twice the area are half the density, and
// twice the crystals are twice it.
func TestTheSameCrystalsOverMoreAreaAreLessDense(t *testing.T) {
	rapid.Check(t, moreAreaIsLessDense)
}

func moreAreaIsLessDense(t *rapid.T) {
	made, _ := grain(t)
	count := rapid.Float64Range(countLeast, countMost).Draw(t, "count")
	over := area(t)
	whole := made.Density(count, over)
	if got := made.Density(count, pair*over); !within(got, whole/pair,
		toleranceTight) {
		t.Fatalf("over twice the area it is %v, want %v", got, whole/pair)
	}
	if got := made.Density(pair*count, over); !within(got, pair*whole,
		toleranceTight) {
		t.Fatalf("twice the crystals are %v, want %v", got, pair*whole)
	}
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

// floats are the numbers in the named columns of a row, by column.
func floats(t *testing.T, row data.Row, columns ...string) map[string]float64 {
	t.Helper()
	found := map[string]float64{}
	for _, column := range columns {
		number, err := row.Float(column)
		if err != nil {
			t.Fatal(err)
		}
		found[column] = number
	}
	return found
}

func TestACountWandersByItsSquareRoot(t *testing.T) {
	for _, row := range table(t, countedFile) {
		shown := floats(t, row, "expected", "chance", "want")
		got := crystal.Counted(shown["expected"], shown["chance"])
		if got != shown["want"] {
			t.Errorf("%v expected, chance %v: found %v, want %v",
				shown["expected"], shown["chance"], got, shown["want"])
		}
	}
}

// What is found is what was expected moved by the square root of it
// times the draw, and never less than none. No chance finds what was
// expected, and more chance never finds fewer.
func TestCountedIsExpectedMovedByChanceAndNeverBelowNone(t *testing.T) {
	rapid.Check(t, countedFollowsChance)
}

func countedFollowsChance(t *rapid.T) {
	expected := rapid.Float64Range(none, expectedMost).Draw(t, "expected")
	low := rapid.Float64Range(-chanceMost, chanceMost).Draw(t, "low")
	high := low + rapid.Float64Range(none, moreMost).Draw(t, "more")
	found := crystal.Counted(expected, low)
	if want := math.Max(none, expected+math.Sqrt(expected)*low); found !=
		want {
		t.Fatalf("%v expected, chance %v: found %v, want %v", expected,
			low, found, want)
	}
	if found < none {
		t.Fatalf("%v crystals were found", found)
	}
	if crystal.Counted(expected, high) < found {
		t.Fatalf("chance %v found fewer than chance %v", high, low)
	}
	if crystal.Counted(expected, none) != expected {
		t.Fatalf("no chance did not find the %v expected", expected)
	}
	if crystal.Counted(none, low) != none {
		t.Fatalf("none expected, and some were found")
	}
}

// What a seed has always drawn. These are not right because they are
// these numbers. They are pinned so that no change to the hash a site
// takes or to the bell made of two draws can pass unseen.
func TestChanceHasAlwaysDrawnWhatItDraws(t *testing.T) {
	for _, row := range table(t, pinnedFile) {
		text, err := row.Text("seed")
		if err != nil {
			t.Fatal(err)
		}
		seed, err := strconv.ParseUint(text, decimal, wordBits)
		if err != nil {
			t.Fatal(err)
		}
		site := floats(t, row, "x", "y", "frame", "want")
		got := crystal.Chance(seed, int64(site["x"]), int64(site["y"]),
			int64(site["frame"]))
		if want := site["want"]; got != want {
			t.Errorf("%d at %v drew %v, it always drew %v", seed, site, got,
				want)
		}
	}
}

// Chance is always a number, and never further from zero than the hash's finest
// step allows: the root of two times the logarithm of 2 to the 53.
func TestChanceIsAlwaysANumberOfBoundedSize(t *testing.T) {
	rapid.Check(t, chanceIsBounded)
}

func chanceIsBounded(t *rapid.T) {
	bound := math.Sqrt(rootTwo * fractionBits * math.Ln2)
	seed := rapid.Uint64().Draw(t, "seed")
	x, y := rapid.Int64().Draw(t, "x"), rapid.Int64().Draw(t, "y")
	frame := rapid.Int64().Draw(t, "frame")
	got := crystal.Chance(seed, x, y, frame)
	if math.IsNaN(got) || math.Abs(got) > bound {
		t.Fatalf("chance drew %v, which is outside %v", got, bound)
	}
}

// One site draws the same forever. Moving any part of it draws another.
func TestChanceIsTheSameForEverAndDiffersByPlace(t *testing.T) {
	rapid.Check(t, chanceDiffersByPlace)
}

func chanceDiffersByPlace(t *rapid.T) {
	seed := rapid.Uint64().Draw(t, "seed")
	x, y := rapid.Int64().Draw(t, "x"), rapid.Int64().Draw(t, "y")
	frame := rapid.Int64().Draw(t, "frame")
	by := rapid.Int64Range(stepLeast, stepMost).Draw(t, "by")
	here := crystal.Chance(seed, x, y, frame)
	if again := crystal.Chance(seed, x, y, frame); again != here {
		t.Fatalf("one site drew %v and then %v", here, again)
	}
	moved := map[string]float64{
		"seed":  crystal.Chance(seed+uint64(by), x, y, frame),
		"x":     crystal.Chance(seed, x+by, y, frame),
		"y":     crystal.Chance(seed, x, y+by, frame),
		"frame": crystal.Chance(seed, x, y, frame+by),
	}
	for part, other := range moved {
		if other == here {
			t.Fatalf("moving the %s left the draw at %v", part, here)
		}
	}
}

// Over a field of sites the draws are the bell curve. Their mean is zero and
// their spread is one. About 68 in 100 fall within one of zero. The bounds are
// six standard errors of what forty thousand draws can show, so only a wrong
// curve fails them.
func TestChanceIsTheBellCurve(t *testing.T) {
	rapid.Check(t, chanceIsABell)
}

func chanceIsABell(t *rapid.T) {
	seed := rapid.Uint64().Draw(t, "seed")
	frame := rapid.Int64().Draw(t, "frame")
	var total, squares, inside float64
	for index := range fieldSites {
		column, row := index%fieldWidth, index/fieldWidth
		draw := crystal.Chance(seed, int64(column), int64(row), frame)
		total += draw
		squares += draw * draw
		if math.Abs(draw) <= unity {
			inside++
		}
	}
	many := float64(fieldSites)
	mean := total / many
	spread := math.Sqrt(squares/many - mean*mean)
	if math.Abs(mean) > meanError || math.Abs(spread-unity) > spreadError {
		t.Fatalf("chance has mean %v and spread %v, want 0 and 1", mean,
			spread)
	}
	within := math.Erf(unity / math.Sqrt2)
	if share := inside / many; math.Abs(share-within) > shareError {
		t.Fatalf("%v of draws fall within one, want %v", share, within)
	}
}
