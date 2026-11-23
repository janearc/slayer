package weathering

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// the numbers the shelter tests lay a world out by, each a choice of the
// tests
const (
	// shelterBlock is how wide the cells are, in metres.
	shelterBlock = 2
	// kilometres is the side of the huge ground, in metres: the size at
	// which cells in proportion to area was eighteen minutes and 11 GB.
	kilometres = 20000.0
	// smallSide is the side of each small plate, in metres.
	smallSide = 10.0
	// smallPlates is how many small plates stand beside the ground.
	smallPlates = 4
	// plateGap is the gap between one small plate and the next, in metres.
	plateGap = 20.0
	// plateHeight is how high the small plates stand, in metres.
	plateHeight = 3.0
	// moderateSide is the side of the ground the anti-test and the
	// equality test fill with cells, in metres: wide enough to be large
	// at the picked threshold, small enough to keep as cells without cost.
	moderateSide = 240.0
	// everything is a threshold past any footprint, so nothing is large.
	everything = math.MaxInt
	// walkStep is the step of the grid of points a test asks about.
	walkStep = 3.7
	// walkCount is how many steps the grid has along each side.
	walkCount = 20
)

// facesOfPlate is how many triangles a plate is.
const facesOfPlate = 2

// entriesOf is how many places in cells the shelter keeps a triangle in,
// counted over every cell, once the cells are worked.
func entriesOf(shelter *Shelter) int {
	shelter.ready()
	return len(shelter.entries)
}

// plateAt adds a flat plate between two corners to a shelter, as two
// triangles.
func plateAt(shelter *Shelter, low, high maths.Vec, name string) {
	across := maths.V(low.X, low.Y, high.Z)
	along := maths.V(high.X, low.Y, low.Z)
	shelter.Add(low, along, high, name)
	shelter.Add(low, high, across, name)
}

// plateEntries is the most entries a plate of a side may cost in cells:
// the cells along a side, and one more for where it lies across an edge,
// squared, for each of its triangles.
func plateEntries(side float64) int {
	across := int(side/shelterBlock) + 2
	return facesOfPlate * across * across
}

// worldWith is a shelter holding a ground of a side and a row of small
// plates over it, which keeps triangles in cells until its threshold of
// footprint is changed to threshold.
func worldWith(ground float64, threshold int) *Shelter {
	shelter := NewShelter(shelterBlock)
	shelter.largeAbove = threshold
	plateAt(shelter, maths.V(0, 0, 0), maths.V(ground, 0, ground), "ground")
	for index := range smallPlates {
		start := float64(index) * plateGap
		plateAt(shelter, maths.V(start, plateHeight, 0),
			maths.V(start+smallSide, plateHeight, smallSide), "roof")
	}
	return shelter
}

// TestALargeFaceCostsNoCells says that a ground of kilometres, among
// small plates, is kept on the list and not in cells, so that the cells
// used are the small plates' and no more, and that it still answers.
func TestALargeFaceCostsNoCells(t *testing.T) {
	shelter := worldWith(kilometres, largeFootprint)
	if len(shelter.large) != facesOfPlate {
		t.Errorf("%d large triangles, want the ground's %d",
			len(shelter.large), facesOfPlate)
	}
	bound := smallPlates * plateEntries(smallSide)
	if used := entriesOf(shelter); used > bound {
		t.Errorf("%d entries in cells, want at most %d", used, bound)
	}
	if far, found := shelter.Below(maths.V(5, 1, 5), 10); !found ||
		math.Abs(far-1) > 1e-9 {
		t.Errorf("%v %v looking down at the ground, want one metre", far,
			found)
	}
	if rise, name, found := shelter.Above(maths.V(5, 1, 5), 10); !found ||
		name != "roof" || math.Abs(rise-(plateHeight-1)) > 1e-9 {
		t.Errorf("%v %q %v looking up, want two metres to a roof", rise,
			name, found)
	}
}

// TestWithoutALargeListTheBoundIsBroken is the anti-test of the one
// above: with no triangle allowed to be large, a ground of a few hundred
// metres alone passes the bound the huge ground is held to, and the large
// list is empty. A bound that nothing could break would say nothing.
func TestWithoutALargeListTheBoundIsBroken(t *testing.T) {
	shelter := worldWith(moderateSide, everything)
	if len(shelter.large) != 0 {
		t.Errorf("%d large triangles with none allowed", len(shelter.large))
	}
	bound := smallPlates * plateEntries(smallSide)
	if used := entriesOf(shelter); used <= bound {
		t.Errorf("%d entries in cells, want more than %d", used, bound)
	}
}

// TestLargeAndCellsGiveTheSameAnswers says that the same triangles,
// kept once all on the large list and once all in cells, are found the
// same way by every look up and down: same distance, same name.
func TestLargeAndCellsGiveTheSameAnswers(t *testing.T) {
	cells := worldWith(moderateSide, everything)
	large := worldWith(moderateSide, 0)
	if len(large.large) == 0 || entriesOf(large) != 0 {
		t.Fatalf("%d large, %d in cells, want all large", len(large.large),
			entriesOf(large))
	}
	if len(cells.large) != 0 || entriesOf(cells) == 0 {
		t.Fatalf("%d large, %d in cells, want all in cells",
			len(cells.large), entriesOf(cells))
	}
	for _, height := range []float64{-1, 0.5, 2, 5} {
		for _, reach := range []float64{0.4, 3, 100} {
			for one := range walkCount {
				for two := range walkCount {
					from := maths.V(float64(one)*walkStep-5, height,
						float64(two)*walkStep-5)
					sameAnswers(t, cells, large, from, reach)
				}
			}
		}
	}
}

// sameAnswers fails the test if two shelters answer a look up and a look
// down from a point differently.
func sameAnswers(t *testing.T, want, got *Shelter, from maths.Vec,
	reach float64) {
	t.Helper()
	wantRise, wantName, wantFound := want.Above(from, reach)
	gotRise, gotName, gotFound := got.Above(from, reach)
	if wantRise != gotRise || wantName != gotName || wantFound != gotFound {
		t.Errorf("up from %v within %v: %v %q %v, want %v %q %v", from,
			reach, gotRise, gotName, gotFound, wantRise, wantName,
			wantFound)
	}
	wantDrop, wantBelow := want.Below(from, reach)
	gotDrop, gotBelow := got.Below(from, reach)
	if wantDrop != gotDrop || wantBelow != gotBelow {
		t.Errorf("down from %v within %v: %v %v, want %v %v", from, reach,
			gotDrop, gotBelow, wantDrop, wantBelow)
	}
}

// TestATieGoesToTheLaterWhereverEachIsKept says that of two triangles
// the same in space, the one added later is the one found, whether the
// earlier is on the list and the later in cells, or the other way, as it
// was when all were kept in cells in the order they came.
func TestATieGoesToTheLaterWhereverEachIsKept(t *testing.T) {
	corners := [3]maths.Vec{maths.V(0, plateHeight, 0),
		maths.V(smallSide, plateHeight, 0),
		maths.V(0, plateHeight, smallSide)}
	from := maths.V(1, 0, 1)
	for _, early := range []int{0, everything} {
		shelter := NewShelter(shelterBlock)
		shelter.largeAbove = early
		shelter.Add(corners[0], corners[1], corners[2], "earlier")
		shelter.largeAbove = everything - early
		shelter.Add(corners[0], corners[1], corners[2], "later")
		if _, name, found := shelter.Above(from, plateHeight*2); !found ||
			name != "later" {
			t.Errorf("found %q %v with the earlier at %d, want the later",
				name, found, early)
		}
		if drop, found := shelter.Below(maths.V(1, 2*plateHeight, 1),
			plateHeight*2); !found || math.Abs(drop-plateHeight) > 1e-9 {
			t.Errorf("%v %v looking down, want %v", drop, found,
				plateHeight)
		}
	}
}
