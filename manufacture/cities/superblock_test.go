package cities

import (
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// leastLit and mostLit bound the share of windows alight that a drawing
// of the superblock may show: the culture's chance of 0.4, with room for
// the few hundred windows a small parcel has.
const (
	leastLit = 0.25
	mostLit  = 0.55
)

// areaSlack is how far the plots and alleys may be from the parcel's
// area, in square metres.
const areaSlack = 1e-3

// The parcel the tests grow on: the size and ceiling of the picture the
// superblock is for, 210 by 120 m and fourteen storeys.
const (
	parcelWidth   = 210.0
	parcelDepth   = 120.0
	parcelCeiling = 14
)

// superblockSeeds is how many seeds the tests that look at many grow.
const superblockSeeds = 5

// smallWidth, smallDepth and smallCeiling are a parcel small enough to
// draw whole in a test: 60 by 40 m and eight storeys.
const (
	smallWidth   = 60.0
	smallDepth   = 40.0
	smallCeiling = 8
)

// raster is the step, in metres, of the grid the test looks at the alleys
// through, which is not the way the code looks at them.
const raster = 0.5

// growSuperblock is a superblock of a size and seed, or the test's end.
func growSuperblock(t *testing.T, ask SuperblockAsk, seed Seed) Superblock {
	t.Helper()
	block, err := NewSuperblock(ask, seed)
	if err != nil {
		t.Fatal(err)
	}
	return block
}

// The people the tests plan for: nine thousand on the picture's parcel,
// which its slabs hold in thirteen storeys, and eight hundred on the
// small one, in six.
const (
	parcelPeople = 9000
	smallPeople  = 800
)

// crushing is a hundred: a crowding past the crush limit at any floor a
// person, and as many times the people as no ceiling holds.
const crushing = 100

// grownOver is the accretion of a block grown over the whole way.
var grownOver = Accretion{Value: 1}

// kowloon is the ask of the picture: the whole parcel grown over, no
// berths.
func kowloon() SuperblockAsk {
	return SuperblockAsk{Width: parcelWidth, Depth: parcelDepth,
		Ceiling: parcelCeiling, Occupants: parcelPeople, Crowding: 1,
		Accretion: grownOver}
}

// small is the ask of a parcel a test can draw whole, grown over.
func small(berths bool) SuperblockAsk {
	return SuperblockAsk{Width: smallWidth, Depth: smallDepth,
		Ceiling: smallCeiling, Berths: berths, Occupants: smallPeople,
		Crowding: 1, Accretion: grownOver}
}

// at is an ask at another accretion.
func (ask SuperblockAsk) at(grown float64) SuperblockAsk {
	ask.Accretion = Accretion{Value: grown}
	return ask
}

// usedAs are a superblock's plots that are built as one thing.
func usedAs(block Superblock, use string) []plot {
	var plots []plot
	for _, plot := range block.Plots {
		if plot.Use == use {
			plots = append(plots, plot)
		}
	}
	return plots
}

// cut are the alleys of the court's cuts: all but the four of the lane
// round it.
func cut(block Superblock) []rect {
	return block.Alleys[sidesAround:]
}

// drawnSuperblock is a superblock drawn at the hull level.
func drawnSuperblock(t *testing.T, block Superblock) []Rendered {
	t.Helper()
	drawn, err := block.Design().Rendered("hull")
	if err != nil {
		t.Fatal(err)
	}
	return drawn
}

func TestNoTwoVolumesOfASuperblockMeet(t *testing.T) {
	for seed := range Seed(superblockSeeds) {
		volumes := growSuperblock(t, kowloon(), seed).Volumes()
		for first := range volumes {
			for second := first + 1; second < len(volumes); second++ {
				if volumes[first].meets(volumes[second]) {
					t.Fatalf("seed %d: %+v meets %+v", seed, volumes[first],
						volumes[second])
				}
			}
		}
	}
}

func TestNothingStandsOverTheCeiling(t *testing.T) {
	block := growSuperblock(t, small(true), 3)
	blocks, homes := drawerOf(t, superblockCulture, 1),
		drawerOf(t, condominiumCulture, 1)
	ceiling := block.Deck + float64(smallCeiling+1)*max(
		blocks.most("storey-height"), homes.most("storey-height"))
	limit := ceiling + blocks.fixed("clutter-allowance")
	if block.Ceiling > ceiling || block.Allowance != limit-ceiling {
		t.Errorf("ceiling %v and allowance %v, want %v and %v",
			block.Ceiling, block.Allowance, ceiling, limit-ceiling)
	}
	for _, plot := range block.Plots {
		over := plot.Storeys > smallCeiling+1 ||
			(plot.Use != useCore && plot.Use != useTopping &&
				plot.Storeys > smallCeiling)
		if over || plot.Height > ceiling {
			t.Errorf("a %s of %d storeys, %v m, over the ceiling",
				plot.Use, plot.Storeys, plot.Height)
		}
	}
	for _, one := range drawnSuperblock(t, block) {
		for _, triangle := range one.Mesh.Triangles {
			for _, corner := range triangle {
				if top := one.Place.Out(corner).Y; top > limit {
					t.Fatalf("%s reaches %v m, over %v", one.Mesh.Name, top,
						limit)
				}
			}
		}
	}
}

func TestMostBuildingsStandNearTheCeiling(t *testing.T) {
	block := growSuperblock(t, kowloon(), 1)
	court := usedAs(block, useInfill)
	near := 0
	for _, plot := range court {
		if plot.Storeys >= parcelCeiling-1 {
			near++
		}
	}
	if share := float64(near) / float64(len(court)); share < 0.5 {
		t.Errorf("%d of %d buildings of the court near the ceiling, not "+
			"most", near, len(court))
	}
}

// cellsOf are the cells of the test's grid whose middles lie in each
// alley, as the alley's index, or minus one where there is no alley.
func cellsOf(parcel rect, alleys []rect) ([][]int, int, int) {
	wide := parcel.width() / raster
	across := int(math.Ceil(wide))
	deep := parcel.depth() / raster
	down := int(math.Ceil(deep))
	cells := make([][]int, across)
	for column := range cells {
		cells[column] = make([]int, down)
		for row := range cells[column] {
			cells[column][row] = -1
			x := parcel.X0 + (float64(column)+0.5)*raster
			z := parcel.Z0 + (float64(row)+0.5)*raster
			for index, alley := range alleys {
				if x > alley.X0 && x < alley.X1 && z > alley.Z0 &&
					z < alley.Z1 {
					cells[column][row] = index
				}
			}
		}
	}
	return cells, across, down
}

// unreachedBy is which alleys no cell of is reached by spreading from
// the cells on the edge of the parcel, through cells of alleys.
func unreachedBy(parcel rect, alleys []rect) []bool {
	cells, across, down := cellsOf(parcel, alleys)
	seen := map[[2]int]bool{}
	var queue [][2]int
	for column := range across {
		for row := range down {
			edge := column == 0 || row == 0 || column == across-1 ||
				row == down-1
			if edge && cells[column][row] >= 0 {
				seen[[2]int{column, row}] = true
				queue = append(queue, [2]int{column, row})
			}
		}
	}
	for ; len(queue) > 0; queue = queue[1:] {
		for _, step := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			next := [2]int{queue[0][0] + step[0], queue[0][1] + step[1]}
			if next[0] < 0 || next[1] < 0 || next[0] >= across ||
				next[1] >= down || seen[next] || cells[next[0]][next[1]] < 0 {
				continue
			}
			seen[next] = true
			queue = append(queue, next)
		}
	}
	unreached := make([]bool, len(alleys))
	for index := range unreached {
		unreached[index] = true
	}
	for cell := range seen {
		unreached[cells[cell[0]][cell[1]]] = false
	}
	return unreached
}

func TestAlleysAreAMinorityAndEveryOneThatIsSealedIsReported(t *testing.T) {
	most := drawerOf(t, superblockCulture, 1).fixed("alley-share-most")
	for seed := range Seed(superblockSeeds) {
		block := growSuperblock(t, kowloon(), seed)
		var area float64
		for _, alley := range cut(block) {
			area += alley.area()
		}
		inner := block.Court.inset(drawerOf(t, superblockCulture,
			1).fixed("lane-width"))
		if share := area / inner.area(); share > most {
			t.Errorf("seed %d: alleys are %v of the court, over %v", seed,
				share, most)
		}
		unreached := unreachedBy(block.Court, block.Alleys)
		for index, alley := range block.Alleys {
			if unreached[index] && !slices.Contains(block.Sealed, alley) {
				t.Errorf("seed %d: an alley %+v reaches nothing on the "+
					"grid and is not reported", seed, alley)
			}
		}
	}
}

// strewn is which alleys, by index, are sealed by the test's own count:
// a doorway is a stretch of shared edge as long as the culture's opening,
// found by the lengths where the two rectangles' sides overlap, and the
// edge is the parcel's.
func strewn(t *testing.T, parcel rect, alleys []rect) []rect {
	t.Helper()
	opening := drawerOf(t, superblockCulture, 1).fixed("alley-opening")
	reached := make([]bool, len(alleys))
	for again := true; again; {
		again = false
		for index, alley := range alleys {
			if reached[index] {
				continue
			}
			door := alley.X0 <= parcel.X0 || alley.X1 >= parcel.X1 ||
				alley.Z0 <= parcel.Z0 || alley.Z1 >= parcel.Z1
			for other, beyond := range alleys {
				door = door || (reached[other] && opens(alley, beyond,
					opening))
			}
			reached[index], again = door, again || door
		}
	}
	var shut []rect
	for index, alley := range alleys {
		if !reached[index] {
			shut = append(shut, alley)
		}
	}
	return shut
}

// opens is whether two rectangles that stand side by side share a stretch
// of their edge longer than a doorway.
func opens(first, second rect, doorway float64) bool {
	meetX := math.Abs(first.X1-second.X0) < meeting ||
		math.Abs(second.X1-first.X0) < meeting
	meetZ := math.Abs(first.Z1-second.Z0) < meeting ||
		math.Abs(second.Z1-first.Z0) < meeting
	shareX := math.Min(first.X1, second.X1) - math.Max(first.X0, second.X0)
	shareZ := math.Min(first.Z1, second.Z1) - math.Max(first.Z0, second.Z0)
	return (meetX && shareZ > doorway) || (meetZ && shareX > doorway)
}

func TestSealedAlleysAreThoseThatReachNoEdgeByADoorway(t *testing.T) {
	for seed := range Seed(superblockSeeds) {
		block := growSuperblock(t, kowloon(), seed)
		if got, want := block.Sealed, strewn(t, block.Court,
			block.Alleys); !slices.Equal(got, want) {
			t.Errorf("seed %d: %d alleys reported sealed, and the count "+
				"by doorways is %d", seed, len(got), len(want))
		}
	}
}

func TestAlleysAreNarrow(t *testing.T) {
	widest := drawerOf(t, superblockCulture, 1).most("alley-width")
	for seed := range Seed(superblockSeeds) {
		block := growSuperblock(t, kowloon(), seed)
		for _, alley := range cut(block) {
			if thin := math.Min(alley.width(), alley.depth()); thin > widest+
				meeting {
				t.Errorf("seed %d: an alley %v m wide, over %v", seed, thin,
					widest)
			}
		}
	}
}

func TestPlotsYardsAndAlleysTileTheCourtAndPlotsAreIrregular(
	t *testing.T) {
	least := drawerOf(t, superblockCulture, 1).fixed("plot-least")
	block := growSuperblock(t, kowloon(), 2)
	var area float64
	for _, yard := range block.Yards {
		area += yard.area()
	}
	widths := map[int]bool{}
	for _, plot := range usedAs(block, useInfill) {
		area += plot.Lot.area()
		widths[int(plot.Lot.width())] = true
		if plot.Lot.width() < least-meeting || plot.Lot.depth() < least-meeting {
			t.Errorf("a plot %v by %v m, under the least %v",
				plot.Lot.width(), plot.Lot.depth(), least)
		}
	}
	for _, alley := range block.Alleys {
		area += alley.area()
	}
	if math.Abs(area-block.Court.area()) > areaSlack {
		t.Errorf("plots, yards and alleys cover %v of %v square metres",
			area, block.Court.area())
	}
	if len(widths) < 5 {
		t.Errorf("plots of only %d different widths: not irregular",
			len(widths))
	}
}

func TestTheSameSeedGrowsTheSameSuperblock(t *testing.T) {
	first := growSuperblock(t, small(true), 9)
	second := growSuperblock(t, small(true), 9)
	if !reflect.DeepEqual(first.Volumes(), second.Volumes()) ||
		!reflect.DeepEqual(first.Plots, second.Plots) ||
		!reflect.DeepEqual(first.Alleys, second.Alleys) ||
		!reflect.DeepEqual(first.Berths, second.Berths) ||
		!reflect.DeepEqual(first.parts, second.parts) {
		t.Error("one seed grew two different superblocks")
	}
}

func TestDifferentSeedsGrowDifferentSuperblocks(t *testing.T) {
	first := growSuperblock(t, small(false), 1)
	second := growSuperblock(t, small(false), 2)
	if reflect.DeepEqual(first.Plots, second.Plots) {
		t.Error("two seeds grew the same plots")
	}
}

func TestBuildingsAreBuiltInAnOrderTheSeedGives(t *testing.T) {
	block := growSuperblock(t, kowloon(), 4)
	seen := make([]bool, len(block.Plots))
	inOrder := true
	for index, plot := range block.Plots {
		if plot.Order < 0 || plot.Order >= len(seen) || seen[plot.Order] {
			t.Fatalf("plot %d has order %d, a place taken or none", index,
				plot.Order)
		}
		seen[plot.Order] = true
		inOrder = inOrder && plot.Order == index
	}
	if inOrder {
		t.Error("the buildings were built in the order the plots were cut")
	}
}

// kindsOf counts the volumes of each kind.
func kindsOf(volumes []Volume) map[string]int {
	counts := map[string]int{}
	for _, volume := range volumes {
		counts[volume.Kind]++
	}
	return counts
}

func TestAccretionHangsWalkwaysCagesBoxesJetties(t *testing.T) {
	counts := map[string]int{}
	for seed := range Seed(superblockSeeds) {
		block := growSuperblock(t, kowloon(), seed)
		for kind, count := range kindsOf(
			block.Fixtures) {
			counts[kind] += count
		}
	}
	for _, kind := range []string{"bridge", "jetty", "cage", "box", "sign",
		"clutter", "cable", "pipe", "rail", "line", "deck", "canopy",
		"ramp"} {
		if counts[kind] == 0 {
			t.Errorf("no %s in %d superblocks: %v", kind, superblockSeeds,
				counts)
		}
	}
}

func TestBridgesCrossAlleys(t *testing.T) {
	block := growSuperblock(t, kowloon(), 1)
	for _, volume := range block.Fixtures {
		if volume.Kind != "bridge" {
			continue
		}
		across := false
		for _, alley := range block.Alleys {
			across = across || (volume.Low[0] >= alley.X0-meeting &&
				volume.High[0] <= alley.X1+meeting &&
				volume.Low[2] >= alley.Z0-meeting &&
				volume.High[2] <= alley.Z1+meeting)
		}
		if !across {
			t.Errorf("a bridge %+v is not over an alley", volume)
		}
		if volume.Low[1] <= block.Deck {
			t.Errorf("a bridge on the podium: %+v", volume)
		}
	}
}

func TestSomeWindowsAreLitAndSignsGlow(t *testing.T) {
	block := growSuperblock(t, small(false), 5)
	drawn := drawnSuperblock(t, block)
	lit, dark, signs := 0, 0, 0
	for _, one := range drawn {
		switch {
		case strings.HasPrefix(one.Material.Name, "cities lit window"):
			lit++
		case one.Material.Name == "cities glazing":
			dark++
		case strings.HasPrefix(one.Material.Name, "cities sign"):
			signs++
			if one.Material.Glow == (maths.Light{}) {
				t.Error("a sign that does not glow")
			}
		}
	}
	share := float64(lit) / float64(lit+dark)
	if lit == 0 || dark == 0 || share < leastLit || share > mostLit {
		t.Errorf("%d lit windows and %d dark: a share of %v", lit, dark,
			share)
	}
	if signs == 0 {
		t.Error("no sign among the parts")
	}
}

func TestWindowsAreManyAndSmall(t *testing.T) {
	block := growSuperblock(t, small(false), 5)
	drawn := drawnSuperblock(t, block)
	windows := 0
	for _, one := range drawn {
		if strings.HasPrefix(one.Material.Name, "cities lit window") ||
			one.Material.Name == "cities glazing" {
			windows++
		}
	}
	if windows < 500 {
		t.Errorf("%d windows on a parcel of %v square metres", windows,
			smallWidth*smallDepth)
	}
}

func TestRoofsCarryClutterStrappedWithBands(t *testing.T) {
	block := growSuperblock(t, small(false), 5)
	drawn := drawnSuperblock(t, block)
	bands, lashing := 0, ""
	for _, one := range drawn {
		if one.Mesh.Name == "band" {
			bands++
			lashing = one.Material.Name
		}
	}
	if bands == 0 || lashing != "cities lashing" {
		t.Errorf("%d bands of %q, want some of the scavenged lashing", bands,
			lashing)
	}
}

func TestRoofBerthsAreKeptOnlyWhereAPartIsAsked(t *testing.T) {
	asked := kowloon()
	without := growSuperblock(t, asked, 6)
	asked.Berths = true
	with := growSuperblock(t, asked, 6)
	if len(without.Berths) != 0 {
		t.Errorf("%d berths with no parts asked", len(without.Berths))
	}
	if len(with.Berths) == 0 {
		t.Fatal("no berths where parts are asked")
	}
	for _, berth := range with.Berths {
		for _, volume := range with.Volumes() {
			if berth.meets(volume) {
				t.Errorf("a berth %+v meets %+v", berth, volume)
			}
		}
	}
}

func TestAParcelTooSmallOrACeilingOfNoStoreysIsRefused(t *testing.T) {
	flawed := map[string]func(*SuperblockAsk){
		"narrow":     func(ask *SuperblockAsk) { ask.Width = 5 },
		"shallow":    func(ask *SuperblockAsk) { ask.Depth = 5 },
		"no parcel":  func(ask *SuperblockAsk) { ask.Width, ask.Depth = 0, 0 },
		"no ceiling": func(ask *SuperblockAsk) { ask.Ceiling = 0 },
		"negative":   func(ask *SuperblockAsk) { ask.Ceiling = -2 },
		"no one":     func(ask *SuperblockAsk) { ask.Occupants = 0 },
		"no crowd":   func(ask *SuperblockAsk) { ask.Crowding = 0 },
		"crushed":    func(ask *SuperblockAsk) { ask.Crowding = crushing },
		"too many": func(ask *SuperblockAsk) {
			ask.Occupants = crushing * parcelPeople
		},
		"overgrown": func(ask *SuperblockAsk) { ask.Accretion.Value = 1 + 1 },
		"ungrown":   func(ask *SuperblockAsk) { ask.Accretion.Value = -1 },
	}
	for name, flaw := range flawed {
		ask := kowloon()
		flaw(&ask)
		if _, err := NewSuperblock(ask, 1); err == nil {
			t.Errorf("%s: a superblock of %+v was grown", name, ask)
		}
	}
}

func TestAnOverlapOfTwoBuildingsIsFound(t *testing.T) {
	first := Volume{Low: [3]float64{0, 0, 0}, High: [3]float64{4, 4, 4}}
	inside := Volume{Low: [3]float64{3, 1, 3}, High: [3]float64{6, 2, 6}}
	touching := Volume{Low: [3]float64{4, 0, 0}, High: [3]float64{8, 4, 4}}
	above := Volume{Low: [3]float64{0, 4, 0}, High: [3]float64{4, 8, 4}}
	if !first.meets(inside) {
		t.Error("two volumes that share space were said not to meet")
	}
	if first.meets(touching) || first.meets(above) {
		t.Error("volumes that only touch were said to meet")
	}
}

func TestACornerThatOnlyMeetsIsNotAnEdgeAlongWhichAlleysJoin(t *testing.T) {
	first := rect{0, 0, 2, 10}
	if first.joins(rect{2, 10, 12, 12}, 1) {
		t.Error("two rectangles that meet at a corner were said to join")
	}
	if first.joins(rect{2, 9.5, 12, 12}, 1) {
		t.Error("two rectangles that share half a metre were said to join")
	}
	if !first.joins(rect{2, 4, 12, 6}, 1) {
		t.Error("two rectangles that share an edge were said not to join")
	}
}

func TestAnAlleyThatReachesNothingIsReportedSealed(t *testing.T) {
	parcel := rect{-10, -10, 10, 10}
	open := rect{-10, -1, 0, 1}
	joined := rect{-2, 1, 1, 6}
	shut := rect{4, 4, 6, 6}
	closed := sealed([]rect{open, joined, shut}, parcel, 1)
	if !reflect.DeepEqual(closed, []rect{shut}) {
		t.Errorf("sealed %v, want only the one that does not join the "+
			"edge through the others: %v", closed, []rect{shut})
	}
}
