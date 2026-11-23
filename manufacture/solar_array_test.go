package manufacture

import (
	"math"
	"slices"
	"testing"

	"github.com/janearc/slayer/maths"
)

const solarTolerance = 1e-9

// solarSlots gives each kind of part its own matter, so a count by matter
// is a count by kind.
func solarSlots() solarLooks {
	return solarLooks{cell: leaf, cover: solarCover, bar: spaceMirror,
		substrate: spaceFoil, frame: spaceSecondary, hardware: spaceGold}
}

// smallSolarArray is two hinged panels of four columns by three rows,
// with covers, bars and a yoke, lying along +x with the cells up.
func smallSolarArray() solarArray {
	return solarArray{
		frame: solarFrame(maths.V(0, 0, 0), maths.XAxis, maths.YAxis),
		panels: solarPanels{along: 2, across: 1, width: 1, depth: .6,
			thickness: .02, gap: .05, reach: .1},
		cells: solarCells{columns: 4, rows: 3, width: .2, depth: .15,
			height: .01, pitchX: .24, pitchZ: .19, barShare: .5,
			cover: .004},
		yoke: solarYoke{length: .5, radius: .02, rootOffset: .03,
			tipOffset: .03, across: []float64{0}},
		looks: solarSlots(),
	}
}

// partsBy counts an assembly's parts by their matter.
func partsBy(assembly *instrumentAssembly) map[slot]int {
	counts := map[slot]int{}
	for _, piece := range assembly.object.parts {
		counts[piece.of]++
	}
	return counts
}

// laid is an assembly with the array laid into it.
func laid(array solarArray) *instrumentAssembly {
	assembly := &instrumentAssembly{fine: cut{around: 8}}
	array.lay(assembly)
	return assembly
}

// TestSolarArrayHasEveryPartOfIt checks a count of each part of a small
// array: substrates, frames, cells, covers, bars, hinges and yoke.
func TestSolarArrayHasEveryPartOfIt(t *testing.T) {
	assembly := laid(smallSolarArray())
	if assembly.trouble != nil {
		t.Fatal(assembly.trouble)
	}
	want := map[slot]int{spaceFoil: 2, spaceSecondary: 8, leaf: 24,
		solarCover: 24, spaceMirror: 6, spaceGold: 9}
	if got := partsBy(assembly); !mapsEqual(got, want) {
		t.Fatalf("parts by matter: got %v, want %v", got, want)
	}
}

// mapsEqual says two counts by matter are the same.
func mapsEqual(left, right map[slot]int) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

// TestSolarCellsAreSeparatedByTheGap checks that cells in a row sit a
// pitch apart, so the gap between them is the pitch less the width, and
// that the first sits one margin inside the first hinge line.
func TestSolarCellsAreSeparatedByTheGap(t *testing.T) {
	array := smallSolarArray()
	assembly := laid(array)
	var centres []float64
	for _, piece := range assembly.object.parts {
		if piece.of == leaf && !slices.Contains(centres, piece.space.Origin.X) {
			centres = append(centres, piece.space.Origin.X)
		}
	}
	slices.Sort(centres)
	margin, _ := array.margins()
	first := array.panels.reach + margin + array.cells.width*spaceHalf
	if math.Abs(centres[0]-first) > solarTolerance {
		t.Fatalf("first cell at %v, want %v", centres[0], first)
	}
	gap := centres[1] - centres[0] - array.cells.width
	want := array.cells.pitchX - array.cells.width
	if math.Abs(gap-want) > solarTolerance {
		t.Fatalf("gap %v, want %v", gap, want)
	}
}

// TestSolarCellsSitOnTheFaceTheFrameChooses is the anti-test of the cells
// sitting up: the same array turned to face down has its cells below the
// mid-plane.
func TestSolarCellsSitOnTheFaceTheFrameChooses(t *testing.T) {
	array := smallSolarArray()
	array.frame = solarFrame(maths.V(0, 0, 0), maths.XAxis,
		maths.V(0, -1, 0))
	for _, piece := range laid(array).object.parts {
		if piece.of == leaf && piece.space.Origin.Y >= 0 {
			t.Fatalf("a cell is on the wrong face: %v", piece.space.Origin)
		}
	}
}

// TestSolarPanelsFoldAboutTheirHingeLine checks the second panel turns by
// the fold about the hinge line, and a flat array does not turn.
func TestSolarPanelsFoldAboutTheirHingeLine(t *testing.T) {
	array := smallSolarArray()
	level := array.hinge(1).Orientation.Out(maths.XAxis)
	if math.Abs(level.Y) > solarTolerance {
		t.Fatalf("a flat array turned: %v", level)
	}
	array.panels.fold = .1
	folded := array.hinge(1).Orientation.Out(maths.XAxis)
	if math.Abs(folded.Y-math.Sin(.1)) > solarTolerance {
		t.Fatalf("fold %v, want %v", folded.Y, math.Sin(.1))
	}
	first := array.hinge(0).Orientation.Out(maths.XAxis)
	if math.Abs(first.Y) > solarTolerance {
		t.Fatalf("the first panel folded: %v", first)
	}
}

// TestRolledSolarArrayHasSpreadersAndBoomsNotAFrame checks a rolled
// wing has a bar at each end and four booms, and no side rails.
func TestRolledSolarArrayHasSpreadersAndBoomsNotAFrame(t *testing.T) {
	array := smallSolarArray()
	array.rolled = true
	got := partsBy(laid(array))
	if got[spaceSecondary] != 4 {
		t.Fatalf("%d spreader bars, want 4", got[spaceSecondary])
	}
	if want := 6 + 4*2 + 3; got[spaceGold] != want {
		t.Fatalf("%d hardware parts, want %d", got[spaceGold], want)
	}
}

// TestSolarArrayLeavesOutWhatItIsNotGiven is the anti-test of the parts
// that are optional: no yoke, no cover, no bars and no second string.
func TestSolarArrayLeavesOutWhatItIsNotGiven(t *testing.T) {
	array := smallSolarArray()
	array.yoke.length = 0
	array.cells.cover = 0
	array.cells.barShare = 0
	array.cells.second = spacePanel
	got := partsBy(laid(array))
	if got[solarCover] != 0 || got[spaceMirror] != 0 || got[spacePanel] != 0 {
		t.Fatalf("parts that were not asked for: %v", got)
	}
	if got[spaceGold] != 6 {
		t.Fatalf("%d hardware parts, want the hinges alone", got[spaceGold])
	}
}

// TestSolarArrayDealsRowsToTheSecondString checks the rows from the
// second string on are of its matter, and no others.
func TestSolarArrayDealsRowsToTheSecondString(t *testing.T) {
	array := smallSolarArray()
	array.cells.second = spacePanel
	array.cells.secondFrom = 2
	got := partsBy(laid(array))
	if got[leaf] != 2*4*2 || got[spacePanel] != 2*4*1 {
		t.Fatalf("cells by string: %v", got)
	}
}

// TestSolarArrayRefusesWhatCannotBeBuilt is the anti-test of the layout:
// each of these leaves the first error and no part.
func TestSolarArrayRefusesWhatCannotBeBuilt(t *testing.T) {
	spoilers := map[string]func(*solarArray){
		"no width":   func(array *solarArray) { array.panels.width = 0 },
		"no panels":  func(array *solarArray) { array.panels.along = 0 },
		"no columns": func(array *solarArray) { array.cells.columns = 0 },
		"no height":  func(array *solarArray) { array.cells.height = 0 },
		"too wide":   func(array *solarArray) { array.cells.width = 2 },
		"too deep":   func(array *solarArray) { array.cells.depth = 2 },
	}
	for name, spoil := range spoilers {
		array := smallSolarArray()
		spoil(&array)
		assembly := laid(array)
		made := len(assembly.object.parts)
		if assembly.trouble != errSolarArray || made != 0 {
			t.Errorf("%s: trouble %v with %d parts", name,
				assembly.trouble, made)
		}
	}
}

// TestSolarArrayKeepsAnEarlierError checks an assembly that has already
// failed is left as it is.
func TestSolarArrayKeepsAnEarlierError(t *testing.T) {
	assembly := &instrumentAssembly{trouble: errGood}
	smallSolarArray().lay(assembly)
	if assembly.trouble != errGood || len(assembly.object.parts) != 0 {
		t.Fatalf("trouble %v with %d parts", assembly.trouble,
			len(assembly.object.parts))
	}
}

// TestTenThousandSolarCellsAreStoredOnce lays ten thousand cells and
// checks the group holds them all as placements of one mesh.
func TestTenThousandSolarCellsAreStoredOnce(t *testing.T) {
	const side = 100
	array := smallSolarArray()
	array.panels = solarPanels{along: 1, across: 1, width: 10, depth: 10,
		thickness: .02}
	array.cells = solarCells{columns: side, rows: side, width: .09,
		depth: .09, height: .01, pitchX: .1, pitchZ: .1}
	assembly := laid(array)
	if err := assembly.shareSolar(leaf); err != nil {
		t.Fatal(err)
	}
	if partsBy(assembly)[leaf] != 0 {
		t.Fatal("cells were left as separate parts")
	}
	members := assembly.object.groups[leaf].Members()
	if len(members) != side*side {
		t.Fatalf("%d cells in the group, want %d", len(members), side*side)
	}
	prototypes := map[any]bool{}
	for _, member := range members {
		prototypes[member.Mesh] = true
	}
	if len(prototypes) != 1 {
		t.Fatalf("%d meshes stored for the cells, want 1", len(prototypes))
	}
	if assembly.object.PartCount() < side*side {
		t.Fatal("the shared cells are no longer counted")
	}
}

// TestSharingLeavesOtherMattersAlone is the anti-test of sharing: the
// substrate and the hardware stay parts when only the cells are shared.
func TestSharingLeavesOtherMattersAlone(t *testing.T) {
	assembly := laid(smallSolarArray())
	if err := assembly.shareSolar(leaf, solarCover); err != nil {
		t.Fatal(err)
	}
	got := partsBy(assembly)
	if got[spaceFoil] != 2 || got[spaceGold] != 9 || got[leaf] != 0 {
		t.Fatalf("parts by matter after sharing: %v", got)
	}
}

// solarCraftCells is the cells each craft's arrays hold, by the matter
// that holds them: the published or chosen count in each array, summed.
var solarCraftCells = []struct {
	name  string
	cells map[slot]int
}{
	{"spitzer", map[slot]int{leaf: 2 * 392}},
	{"mariner-10", map[slot]int{leaf: 2 * 31 * 9}},
	{"soho", map[slot]int{leaf: 2 * 2 * 10 * 9}},
	{"roman", map[slot]int{leaf: 6 * 12 * 8, romanSolarString: 6 * 12 * 8}},
	{"webb", map[slot]int{leaf: 5 * webbCellColumns * webbCellRows,
		solarCover: 5 * webbCellColumns * webbCellRows}},
	{"hubble", map[slot]int{leaf: 2 * hubbleCellColumns * hubbleCellRows}},
}

// TestEachCraftHoldsItsCellsOnceEach checks that every craft that has an
// array stores its cells as placements, in as many as it has.
func TestEachCraftHoldsItsCellsOnceEach(t *testing.T) {
	for _, craft := range solarCraftCells {
		row, err := kindNamed(craft.name)
		if err != nil {
			t.Fatal(err)
		}
		made, err := makeOf(row, sow(1))
		if err != nil {
			t.Fatal(err)
		}
		for matter, count := range craft.cells {
			group := made.groups[matter]
			if group == nil || group.MemberCount() != count {
				t.Errorf("%s: matter %d holds %v, want %d cells", craft.name,
					matter, group, count)
			}
		}
	}
}

// TestPrimaHasNoArray is the anti-test of the craft: nothing in the
// array's matters is made for a craft that carries no array. Prima holds
// no group of any matter. The matter of Roman's second string of cells
// has the number of the gold hardware, which Prima builds as parts, so
// only the matters that Prima never builds are held to have no part.
func TestPrimaHasNoArray(t *testing.T) {
	row, err := kindNamed("prima")
	if err != nil {
		t.Fatal(err)
	}
	made, err := makeOf(row, sow(1))
	if err != nil {
		t.Fatal(err)
	}
	for _, matter := range []slot{leaf, solarCover, romanSolarString} {
		if made.groups[matter] != nil {
			t.Errorf("prima holds a group of matter %d", matter)
		}
	}
	for _, piece := range made.parts {
		if piece.of == leaf || piece.of == solarCover {
			t.Errorf("prima has a part of matter %d", piece.of)
		}
	}
}
