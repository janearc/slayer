package cities

import (
	"math"
	"strings"
	"testing"
)

// The room a frontage test's runs take, and how the street is laid. The
// numbers are the tests' own: a run of forty metres and ten deep, a gap of
// eight between runs, fronts seven metres off a line of two hundred.
const (
	frontRunLength  = 40.0
	frontRunDepth   = 10.0
	frontGap        = 8.0
	frontOffset     = 7.0
	frontLineLength = 200.0
	frontSpacing    = 9.0
	frontClose      = 1e-6
	frontTurnAround = 180.0
	// frontBendTangent is the tangent of half the right-angle turn of the
	// bent line, which is one, and southward is the bearing of south.
	frontBendTangent = 1.0
	southward        = 180.0
)

// aStraightLine is a line of a channel that runs north from the origin.
func aStraightLine() [][2]float64 {
	return [][2]float64{{0, 0}, {0, frontLineLength}}
}

// aBentLine is a line that runs north, then turns right to run east: a
// bend whose inside is on the right of the line, and whose outside is on
// the left.
func aBentLine() [][2]float64 {
	return [][2]float64{{0, 0}, {0, frontLineLength},
		{frontLineLength, frontLineLength}}
}

// terraced is an item that is a terrace of every run the same size.
func terraced() Item {
	return Item{Kind: "terrace-test", Run: func(int) (Run, error) {
		return Run{Length: frontRunLength, Depth: frontRunDepth}, nil
	}}
}

// treed is an item laid at a spacing and left to face as it is stood.
func treed() Item {
	return Item{Kind: "tree-test", Spaced: true, Spacing: frontSpacing}
}

// laidAlong is the sites a frontage lays on a line, or the test's end.
func laidAlong(t *testing.T, line [][2]float64, right bool, item Item,
	most int) Laid {
	t.Helper()
	along := FrontageAlong(line, frontOffset, right, frontGap)
	return sitesOf(t, frontage{}, OnFrontage(along), item, 1, most)
}

// Terraces on the left of a line that runs north stand with their fronts
// to the east, which is to the water, their back corners behind the line
// of fronts, and run north end to end with a gap between.
func TestTerracesOnTheLeftFaceTheWaterAndRunAlongTheLine(t *testing.T) {
	laid := laidAlong(t, aStraightLine(), false, terraced(), 0)
	if len(laid.Sites) != 4 {
		t.Fatalf("two hundred metres took %d runs of forty and a gap of "+
			"eight", len(laid.Sites))
	}
	for index, site := range laid.Sites {
		wantNorth := frontGap/2 + float64(index)*(frontRunLength+frontGap)
		if !site.Faced || math.Abs(site.Heading-eastward) > frontClose {
			t.Errorf("run %d looks along %v, and the water is east", index,
				site.Heading)
		}
		if math.Abs(site.East-(-frontOffset-frontRunDepth)) > frontClose ||
			math.Abs(site.North-wantNorth) > frontClose {
			t.Errorf("run %d starts at %v e, %v n, not %v e, %v n", index,
				site.East, site.North, -frontOffset-frontRunDepth, wantNorth)
		}
	}
}

// Terraces on the right of the same line stand east of it, face west, and
// run south from the north end, since a run extends to the left of
// someone looking along its front.
func TestTerracesOnTheRightFaceWestAndRunSouth(t *testing.T) {
	laid := laidAlong(t, aStraightLine(), true, terraced(), 0)
	if len(laid.Sites) != 4 {
		t.Fatalf("%d runs", len(laid.Sites))
	}
	first := laid.Sites[0]
	if math.Abs(first.Heading-westward) > frontClose ||
		math.Abs(first.East-(frontOffset+frontRunDepth)) > frontClose ||
		math.Abs(first.North-(frontLineLength-frontGap/2)) > frontClose {
		t.Errorf("the first run is %+v", first)
	}
	for index := 1; index < len(laid.Sites); index++ {
		step := laid.Sites[index-1].North - laid.Sites[index].North
		if math.Abs(step-(frontRunLength+frontGap)) > frontClose {
			t.Errorf("run %d is %v south of the one before", index, step)
		}
	}
}

// Most stops the laying, and the runs it lays are the first.
func TestFrontageStopsAtMost(t *testing.T) {
	laid := laidAlong(t, aStraightLine(), false, terraced(), 2)
	if len(laid.Sites) != 2 {
		t.Fatalf("%d runs laid, 2 asked for", len(laid.Sites))
	}
}

// Items at a spacing stand at the middle of their stretches, and are not
// faced unless their data turns them.
func TestItemsAtASpacingStandAtTheMiddleOfTheirStretches(t *testing.T) {
	laid := laidAlong(t, aStraightLine(), false, treed(), 0)
	least := frontGap / 2
	want := int((frontLineLength - 2*least) / frontSpacing)
	if len(laid.Sites) != want {
		t.Fatalf("%d trees, want %d", len(laid.Sites), want)
	}
	for index, site := range laid.Sites {
		middle := least + (float64(index)+0.5)*frontSpacing
		if site.Faced || math.Abs(site.North-middle) > frontClose ||
			math.Abs(site.East-(-frontOffset)) > frontClose {
			t.Errorf("tree %d is %+v, want %v n on the line of fronts",
				index, site, middle)
		}
	}
}

// An item turned by its data looks that far from the line's direction:
// half a turn from north is south, so a boat's bow is north.
func TestAnItemTurnedByItsDataLooksThatFarFromTheLine(t *testing.T) {
	item := treed()
	item.Turned, item.Turn = true, frontTurnAround
	laid := laidAlong(t, aStraightLine(), false, item, 1)
	if site := laid.Sites[0]; !site.Faced ||
		math.Abs(site.Heading-frontTurnAround) > frontClose {
		t.Fatalf("the item looks along %+v", site)
	}
}

// At a bend the runs stop short of the corner on the inside by the depth
// of a run times the tangent of half the turn, so that two rows do not
// overlap there, and by half a gap on the outside.
func TestRunsStopShortOfAnInsideCorner(t *testing.T) {
	bentLine := aBentLine()
	terrace := terraced()
	inside := firstFacing(t, laidAlong(t, bentLine, true, terrace, 0),
		westward)
	wantNorth := frontLineLength - frontOffset - frontGap/2 -
		frontRunDepth*frontBendTangent
	if math.Abs(inside.North-wantNorth) > frontClose ||
		math.Abs(inside.East-(frontOffset+frontRunDepth)) > frontClose {
		t.Errorf("the first run past the inside corner starts at %v e, %v n, "+
			"not %v e, %v n", inside.East, inside.North,
			frontOffset+frontRunDepth, wantNorth)
	}
}

// On the outside of a bend a run stops short of the corner by half a gap
// and no more: the turn is away from the houses and takes none of their
// depth.
func TestRunsStopShortOfAnOutsideCornerByHalfAGap(t *testing.T) {
	outsideLine := aBentLine()
	outsideTerrace := terraced()
	outside := firstFacing(t, laidAlong(t, outsideLine, false, outsideTerrace, 0),
		southward)
	wantEast := -frontOffset + frontGap/2
	wantNorth := frontLineLength + frontOffset + frontRunDepth
	if math.Abs(outside.East-wantEast) > frontClose ||
		math.Abs(outside.North-wantNorth) > frontClose {
		t.Errorf("the first run past the outside corner starts at %v e, %v n, "+
			"not %v e, %v n", outside.East, outside.North, wantEast,
			wantNorth)
	}
}

// firstFacing is the first site laid whose front looks along a bearing, or
// the test's end.
func firstFacing(t *testing.T, laid Laid, bearing float64) Site {
	t.Helper()
	for _, site := range laid.Sites {
		if math.Abs(site.Heading-bearing) < frontClose {
			return site
		}
	}
	t.Fatalf("no run looks along %v, in %+v", bearing, laid.Sites)
	return Site{}
}

// No two runs of a bend lay over each other: a run's four corners are not
// inside any run laid before it.
func TestRunsOfABendNeverOverlap(t *testing.T) {
	for _, right := range []bool{false, true} {
		laid := laidAlong(t, aBentLine(), right, terraced(), 0)
		for first := range laid.Sites {
			for _, second := range laid.Sites[first+1:] {
				if runsOverlap(laid.Sites[first], second) {
					t.Errorf("right %v: %+v and %+v overlap", right,
						laid.Sites[first], second)
				}
			}
		}
	}
}

// runsOverlap is whether the rectangles of two runs of the test's size cross.
// A run's rectangle has its back corner at the site, runs the length to the
// left of its front and the depth toward its front.
func runsOverlap(first, second Site) bool {
	return crossing(runCorners(first), second) ||
		crossing(runCorners(second), first)
}

// runCorners are the four corners of the rectangle of a run of the test's
// size.
func runCorners(site Site) []point {
	front, along := toward(site.Heading)
	lengthwise := leftOf(point{front, along})
	across := point{front, along}
	origin := point{site.East, site.North}
	return []point{origin,
		origin.plus(lengthwise, frontRunLength),
		origin.plus(across, frontRunDepth),
		origin.plus(lengthwise, frontRunLength).plus(across, frontRunDepth)}
}

// crossing is whether any corner of one rectangle is strictly inside the
// other run's rectangle, by a margin.
func crossing(corner []point, other Site) bool {
	front, along := toward(other.Heading)
	across := point{front, along}
	lengthwise := leftOf(across)
	origin := point{other.East, other.North}
	for _, each := range corner {
		offset := each.minus(origin)
		down, right := offset.dot(lengthwise), offset.dot(across)
		if down > frontClose && down < frontRunLength-frontClose &&
			right > frontClose && right < frontRunDepth-frontClose {
			return true
		}
	}
	return false
}

// Anti-test: a frontage with no line to stand along is refused.
func TestFrontageWithNoLineIsRefused(t *testing.T) {
	_, err := frontage{}.Sites(Ground{}, terraced(), 1, 0)
	if err == nil || !strings.Contains(err.Error(), "line of a channel") {
		t.Fatalf("a frontage with no line: %v", err)
	}
}

// Anti-test: an item that gives neither a run nor a spacing is refused,
// naming its kind.
func TestFrontageOfAnItemWithNoRoomIsRefused(t *testing.T) {
	along := FrontageAlong(aStraightLine(), frontOffset, false, frontGap)
	_, err := frontage{}.Sites(OnFrontage(along), Item{Kind: "stone"}, 1, 0)
	if err == nil || !strings.Contains(err.Error(), "stone") {
		t.Fatalf("an item with no room: %v", err)
	}
}

// Anti-test: a line of one point, a line that stays at a point, a line that
// turns back on itself and a gap below nought are each refused.
func TestFrontageOfABadLineIsRefused(t *testing.T) {
	for name, row := range map[string]struct {
		line [][2]float64
		gap  float64
		says string
	}{
		"one point": {[][2]float64{{0, 0}}, frontGap, "two points"},
		"repeated point": {[][2]float64{{0, 0}, {0, 0}, {0, 10}}, frontGap,
			"stays at point"},
		"hairpin": {[][2]float64{{0, 0}, {0, 100}, {0, 0}}, frontGap,
			"turns back"},
		"negative gap": {aStraightLine(), -1, "gap"},
	} {
		along := FrontageAlong(row.line, frontOffset, false, row.gap)
		_, err := frontage{}.Sites(OnFrontage(along), terraced(), 1, 0)
		if err == nil || !strings.Contains(err.Error(), row.says) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// Anti-test: an offset of nought or below, which carries the line of fronts
// to the middle of the channel or past it and turns the sides about, is
// refused.
func TestFrontageWithNoOffsetToASideIsRefused(t *testing.T) {
	for _, offset := range []float64{0, -frontOffset} {
		along := FrontageAlong(aStraightLine(), offset, false, frontGap)
		_, err := frontage{}.Sites(OnFrontage(along), terraced(), 1, 0)
		if err == nil || !strings.Contains(err.Error(), "above nought") {
			t.Errorf("offset %v: %v", offset, err)
		}
	}
}

// Anti-test: a run longer than every reach is not laid, and a run whose
// size cannot be drawn ends the laying with the reason.
func TestFrontageLaysNoRunThatDoesNotFit(t *testing.T) {
	long := terraced()
	long.Run = func(int) (Run, error) {
		return Run{Length: 2 * frontLineLength, Depth: frontRunDepth}, nil
	}
	if laid := laidAlong(t, aStraightLine(), false, long, 0); len(laid.Sites) != 0 {
		t.Errorf("a run of %v m was laid on a line of %v", 2*frontLineLength,
			frontLineLength)
	}
	empty := terraced()
	empty.Run = func(int) (Run, error) { return Run{}, nil }
	along := FrontageAlong(aStraightLine(), frontOffset, false, frontGap)
	if _, err := (frontage{}).Sites(OnFrontage(along), empty, 1, 0); err == nil {
		t.Error("a run with no length was laid")
	}
}

// The habit is named, and its name is one a stand may give.
func TestFrontageIsAHabitByName(t *testing.T) {
	habit, err := HabitNamed(HabitFrontage)
	if err != nil {
		t.Fatal(err)
	}
	if _, is := habit.(frontage); !is {
		t.Fatalf("the habit named %s is %T", HabitFrontage, habit)
	}
}

// The data gives trees and boats the room a frontage lays them in.
func TestTheDataGivesTreesAndBoatsTheirSpacing(t *testing.T) {
	for _, kind := range []string{"alder", "poplar", "sampan-at-anchor",
		"fishing-boat-at-anchor", "ferry-at-anchor"} {
		item, err := ItemOf(kind)
		if err != nil || !item.Spaced || !(item.Spacing > 0) {
			t.Errorf("%s: %+v, %v", kind, item, err)
		}
	}
	boat, _ := ItemOf("sampan-at-anchor")
	tree, _ := ItemOf("alder")
	if !boat.Turned || tree.Turned {
		t.Errorf("a boat is turned to the line and a tree is not: %v, %v",
			boat.Turned, tree.Turned)
	}
}
