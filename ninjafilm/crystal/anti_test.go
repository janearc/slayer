package crystal_test

import (
	"math"
	"strconv"
	"testing"

	"github.com/janearc/slayer/ninjafilm/crystal"
	"pgregory.net/rapid"
)

// anti-tests: widths of zero or less make no crystal. Nor do lost ones. The
// widths are drawn, so the table is not the only refusal.
func TestNoWidthOfNoughtOrLostMakesACrystal(t *testing.T) {
	rapid.Check(t, noWidthMakesACrystal)
}

func noWidthMakesACrystal(t *rapid.T) {
	odd := rapid.SampledFrom([]float64{math.NaN(), math.Inf(upward),
		math.Inf(downward)})
	below := rapid.Float64Range(-widthMost, none)
	across := rapid.OneOf(odd, below).Draw(t, "width")
	var zero crystal.Crystal
	if made, err := crystal.New(across); err == nil || made != zero {
		t.Fatalf("a width of %v made a crystal", across)
	}
}

// anti-tests: the area crystals shadow does not depend on how wide they
// are. Wide crystals are fewer and cover the same share of the area.
func TestWiderCrystalsDoNotCoverMoreOfTheArea(t *testing.T) {
	rapid.Check(t, widthDoesNotChangeTheCover)
}

func widthDoesNotChangeTheCover(t *rapid.T) {
	narrow, across := grain(t)
	broad, err := crystal.New(pair * across)
	if err != nil {
		t.Fatal(err)
	}
	wanted, over := density(t), area(t)
	first := narrow.Expected(wanted, over) * across * across
	second := broad.Expected(wanted, over) * pair * across * pair * across
	if !within(first, second, toleranceTight) {
		t.Fatalf("crystals covered %v and then %v of the same share", first,
			second)
	}
}

// anti-tests: a width is squared and not cubed. Three times the width
// takes a ninth of the crystals, and not a third.
func TestAWidthIsSquaredAndNotCubed(t *testing.T) {
	rapid.Check(t, widthIsSquared)
}

func widthIsSquared(t *rapid.T) {
	made, across := grain(t)
	broad, err := crystal.New((pair + unity) * across)
	if err != nil {
		t.Fatal(err)
	}
	wanted, over := density(t), area(t)
	count := made.Expected(wanted, over)
	if within(broad.Expected(wanted, over), count/(pair+unity),
		toleranceTight) {
		t.Fatalf("three times the width took a third of %v", count)
	}
}

// anti-tests: a count of crystals is not the density it was over another
// area. Counting at one area and uncounting at twice it gives half.
func TestADensityUncountedOverAnotherAreaIsNotTheSame(t *testing.T) {
	rapid.Check(t, uncountedOverAnotherArea)
}

func uncountedOverAnotherArea(t *rapid.T) {
	made, _ := grain(t)
	wanted, over := density(t), area(t)
	back := made.Density(made.Expected(wanted, over), pair*over)
	if within(back, wanted, toleranceTight) {
		t.Fatalf("a density of %v was the same over twice the area", wanted)
	}
}

// anti-tests: more crystals over one area are never less dense. They are
// denser by as many as were added.
func TestMoreCrystalsOverOneAreaAreNeverLessDense(t *testing.T) {
	rapid.Check(t, moreCrystalsAreDenser)
}

func moreCrystalsAreDenser(t *rapid.T) {
	made, _ := grain(t)
	count := rapid.Float64Range(countLeast, countMost).Draw(t, "count")
	over := area(t)
	if made.Density(pair*count, over) <= made.Density(count, over) {
		t.Fatalf("twice %v crystals were no denser over %v", count, over)
	}
}

// anti-tests: counts never go below none however far chance pulls them
// down. Chance far past what any site draws still finds none.
func TestNoChanceFindsFewerThanNoneCrystals(t *testing.T) {
	rapid.Check(t, chanceFindsNoneAtWorst)
}

func chanceFindsNoneAtWorst(t *rapid.T) {
	expected := rapid.Float64Range(none, expectedMost).Draw(t, "expected")
	pull := -rapid.Float64Range(chanceMost, chanceMost*chanceMost).Draw(t,
		"pull")
	if found := crystal.Counted(expected, pull); found < none ||
		math.IsNaN(found) {
		t.Fatalf("%v expected and chance %v found %v", expected, pull, found)
	}
}

// anti-tests: a chance that is not zero does not find what was expected. Only
// no chance does, or no crystals expected.
func TestAChanceThatIsNotNoughtMovesTheCount(t *testing.T) {
	rapid.Check(t, chanceMovesTheCount)
}

func chanceMovesTheCount(t *rapid.T) {
	expected := rapid.Float64Range(unity, expectedMost).Draw(t, "expected")
	chance := rapid.Float64Range(unity, chanceMost).Draw(t, "chance")
	if crystal.Counted(expected, chance) == expected {
		t.Fatalf("chance %v found what was expected, %v", chance, expected)
	}
}

// anti-tests: no neighbour of a pinned site draws the number pinned for
// it. The table is of sites, not of numbers any site may draw.
func TestNoNeighbourOfAPinnedSiteDrawsAsItDoes(t *testing.T) {
	for _, row := range table(t, pinnedFile) {
		text, _ := row.Text("seed")
		seed, err := strconv.ParseUint(text, decimal, wordBits)
		if err != nil {
			t.Fatal(err)
		}
		site := floats(t, row, "x", "y", "frame", "want")
		got := crystal.Chance(seed, int64(site["x"])+int64(unity),
			int64(site["y"]), int64(site["frame"]))
		if got == site["want"] {
			t.Errorf("%d at %v and its neighbour drew %v", seed, site, got)
		}
	}
}

// anti-tests: chance is not one number. Across sites it takes many values.
func TestChanceIsNotOneNumberForEverySite(t *testing.T) {
	rapid.Check(t, chanceTakesManyValues)
}

func chanceTakesManyValues(t *rapid.T) {
	seed := rapid.Uint64().Draw(t, "seed")
	frame := rapid.Int64().Draw(t, "frame")
	seen := map[float64]bool{}
	for index := range fieldWidth {
		seen[crystal.Chance(seed, int64(index), int64(none), frame)] = true
	}
	if len(seen) < fieldWidth/pair {
		t.Fatalf("%d sites drew only %d numbers", fieldWidth, len(seen))
	}
}

// anti-tests: another seed does not draw the same at a site. Two seeds
// that differ draw two numbers.
func TestAnotherSeedDoesNotDrawTheSame(t *testing.T) {
	rapid.Check(t, anotherSeedDrawsAnother)
}

func anotherSeedDrawsAnother(t *rapid.T) {
	seed := rapid.Uint64().Draw(t, "seed")
	x, y := rapid.Int64().Draw(t, "x"), rapid.Int64().Draw(t, "y")
	frame := rapid.Int64().Draw(t, "frame")
	by := rapid.Int64Range(stepLeast, stepMost).Draw(t, "by")
	if crystal.Chance(seed, x, y, frame) == crystal.Chance(seed+uint64(by),
		x, y, frame) {
		t.Fatalf("seeds %d and %d drew alike at %d,%d", seed, seed+uint64(by),
			x, y)
	}
}

// anti-tests: the draws are not squeezed inside one spread. Over a field of
// sites some fall further than two from zero, as a bell curve's do.
func TestSomeDrawsFallBeyondTwoSpreads(t *testing.T) {
	rapid.Check(t, someDrawsFallFarOut)
}

func someDrawsFallFarOut(t *rapid.T) {
	seed := rapid.Uint64().Draw(t, "seed")
	frame := rapid.Int64().Draw(t, "frame")
	var far int
	for index := range fieldSites {
		column, row := index%fieldWidth, index/fieldWidth
		draw := crystal.Chance(seed, int64(column), int64(row), frame)
		if math.Abs(draw) > pair {
			far++
		}
	}
	if far == none {
		t.Fatalf("no draw of %d fell past two spreads", fieldSites)
	}
}
