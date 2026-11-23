package weathering

import (
	"slices"
	"testing"

	"github.com/janearc/slayer/maths"
)

// TestEachCellKeepsItsTrianglesInTheOrderTheyCame says that, once the
// cells are worked, every cell's run holds the places of its triangles
// rising, as appending each in turn kept them, and that every run lies
// inside the flat list, one after another with no gap.
func TestEachCellKeepsItsTrianglesInTheOrderTheyCame(t *testing.T) {
	shelter := worldWith(moderateSide, everything)
	shelter.ready()
	for cell, slot := range shelter.cells {
		run := shelter.entries[shelter.offsets[slot]:shelter.offsets[slot+1]]
		if len(run) == 0 || !slices.IsSorted(run) {
			t.Errorf("cell %v holds %v, want places rising", cell, run)
		}
	}
	if last := shelter.offsets[len(shelter.offsets)-1]; int(last) !=
		len(shelter.entries) || shelter.offsets[0] != 0 {
		t.Errorf("runs from %d to %d over %d places", shelter.offsets[0],
			last, len(shelter.entries))
	}
}

// TestAPointInNoCellAsksOnlyTheLarge says that a point over no cell's
// triangle is asked only about the large list.
func TestAPointInNoCellAsksOnlyTheLarge(t *testing.T) {
	shelter := worldWith(kilometres, largeFootprint)
	shelter.ready()
	kept := shelter.kept(maths.V(-kilometres, 0, -kilometres))
	if len(kept[0]) != 0 || len(kept[1]) != facesOfPlate {
		t.Errorf("%d in the cell and %d large, want none and %d",
			len(kept[0]), len(kept[1]), facesOfPlate)
	}
}

// TestAddingAfterALookPanics is the anti-test of the cells' being worked
// once: a triangle added after the first look would be on no cell's run
// and never found, so Add refuses it out loud.
func TestAddingAfterALookPanics(t *testing.T) {
	shelter := worldWith(smallSide, everything)
	shelter.Above(maths.V(1, 0, 1), plateHeight)
	defer func() {
		if recover() == nil {
			t.Error("a triangle added after a look was taken quietly")
		}
	}()
	plateAt(shelter, maths.V(0, plateHeight, 0),
		maths.V(smallSide, plateHeight, smallSide), "late")
}
