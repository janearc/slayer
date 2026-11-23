package cities

import (
	"fmt"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The street the tests hang cables over: two blocks 12 m apart, along z
// for 60 m, 26 m high.
const (
	streetGap    = 12.0
	streetLength = 60.0
	streetHeight = 26.0
	blockDepth   = 14.0
)

// The seeds the tests that look at many draw from, and how many.
const (
	lashingsSeeds = 6
	loneSeed      = 3
)

// closeEnough is how near two lengths must be to be the same, in metres.
const closeEnough = 1e-6

// floorClear is the lowest a cable may hang, from the culture: 4.6 m.
const floorClear = 4.6

// lamplessShare is the share of lanterns that may be lights, at most: the
// culture makes one in six, and the test allows a third.
const lamplessShare = 3

// facingBlocks are two blocks that face each other across a street along
// z, the street's width apart.
func facingBlocks(gap float64) []Volume {
	return []Volume{
		{Kind: "block", Plot: 0, Low: [3]float64{-gap/2 - blockDepth, 0, 0},
			High: [3]float64{-gap / 2, streetHeight, streetLength}},
		{Kind: "block", Plot: 1, Low: [3]float64{gap / 2, 0, 0},
			High: [3]float64{gap/2 + blockDepth, streetHeight, streetLength}}}
}

// turnedBlocks are the same two blocks turned so that the street runs
// along x and the cables cross along z.
func turnedBlocks(gap float64) []Volume {
	blocks := facingBlocks(gap)
	for index := range blocks {
		low, high := blocks[index].Low, blocks[index].High
		blocks[index].Low = [3]float64{low[2], low[1], low[0]}
		blocks[index].High = [3]float64{high[2], high[1], high[0]}
	}
	return blocks
}

// hungOver is the lashings over a group at a density, or the test's end.
func hungOver(t *testing.T, group []Volume, density float64,
	seed Seed) *Lashings {
	t.Helper()
	lashings, err := NewLashings(LashingsAsk{Density: density}, group, seed)
	if err != nil {
		t.Fatal(err)
	}
	return lashings
}

// solidsNamed are the solids of a design that have a name.
func solidsNamed(design Design, name string) []solid {
	var found []solid
	for _, placed := range design.Elements {
		if solid, is := placed.Element.(solid); is && solid.Name == name {
			found = append(found, solid)
		}
	}
	return found
}

func TestAHangingCablePassesThroughBothEnds(t *testing.T) {
	from, to := maths.V(-6, 12, 5), maths.V(6, 13, 5)
	hung, err := hang(from, to, 0.6)
	if err != nil {
		t.Fatal(err)
	}
	ends := map[float64]maths.Vec{0: from, 1: to}
	for share, point := range ends {
		if got := hung.at(share); got.Sub(point).Len() > closeEnough {
			t.Errorf("at %v the cable is at %v, not %v", share, got, point)
		}
	}
}

func TestAHangingCableSagsByWhatItWasGiven(t *testing.T) {
	from, to := maths.V(-6, 12, 5), maths.V(6, 12, 5)
	const sag = 0.6
	hung, err := hang(from, to, sag)
	if err != nil {
		t.Fatal(err)
	}
	if low := hung.lowest(); math.Abs(low-(from.Y-sag)) > closeEnough {
		t.Errorf("the cable is lowest at %v, not %v", low, from.Y-sag)
	}
	for _, point := range hung.points(16) {
		if point.Y > from.Y+closeEnough {
			t.Errorf("the cable rises to %v over its ends at %v", point.Y,
				from.Y)
		}
	}
}

func TestACableBetweenUnequalHeightsStillSagsUnderTheLine(t *testing.T) {
	from, to := maths.V(-6, 10, 0), maths.V(6, 12, 0)
	hung, err := hang(from, to, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	middle := hung.at(0.5)
	if line := (from.Y + to.Y) / 2; !(middle.Y < line) {
		t.Errorf("the middle is at %v, not under the line at %v", middle.Y,
			line)
	}
}

// Anti: points one above the other, and a sag that is not above nought,
// are refused.
func TestACableRefusesWhatCannotHang(t *testing.T) {
	if _, err := hang(maths.V(0, 10, 0), maths.V(0, 12, 0), 0.5); err == nil {
		t.Error("points one above the other were taken")
	}
	for _, sag := range []float64{0, -1, math.NaN()} {
		if _, err := hang(maths.V(-6, 10, 0), maths.V(6, 10, 0),
			sag); err == nil {
			t.Errorf("a sag of %v was taken", sag)
		}
	}
}

func TestNoCableHangsUnderTheFloor(t *testing.T) {
	for seed := range Seed(lashingsSeeds) {
		lashings := hungOver(t, facingBlocks(streetGap), 1, seed)
		cables := solidsNamed(lashings.Design(), "lashing cable")
		if len(cables) == 0 {
			t.Fatalf("seed %d: no cables to look at", seed)
		}
		for _, cable := range cables {
			for _, point := range cable.Form.(lashingCable).Points {
				if point.Y < floorClear-closeEnough {
					t.Errorf("seed %d: a cable hangs at %v, under %v", seed,
						point.Y, floorClear)
				}
			}
		}
	}
}

func TestDensityNoughtHangsNothingAndOneHangsMore(t *testing.T) {
	none := hungOver(t, facingBlocks(streetGap), 0, loneSeed)
	few := hungOver(t, facingBlocks(streetGap), 0.3, loneSeed)
	many := hungOver(t, facingBlocks(streetGap), 1, loneSeed)
	if none.Cables != 0 {
		t.Errorf("density nought hangs %d cables", none.Cables)
	}
	if !(few.Cables > none.Cables && many.Cables > few.Cables*2) {
		t.Errorf("cables at nought, 0.3 and one are %d, %d and %d",
			none.Cables, few.Cables, many.Cables)
	}
	if !(many.Lanterns > few.Lanterns) {
		t.Errorf("lanterns at 0.3 and one are %d and %d", few.Lanterns,
			many.Lanterns)
	}
}

// Anti: a density that is not from nought to one is refused.
func TestADensityOutOfRangeIsRefused(t *testing.T) {
	for _, density := range []float64{-0.1, 1.1, math.NaN(), math.Inf(1)} {
		_, err := NewLashings(LashingsAsk{Density: density},
			facingBlocks(streetGap), loneSeed)
		if err == nil {
			t.Errorf("a density of %v was taken", density)
		}
	}
}

func TestTheSameSeedHangsTheSameStreet(t *testing.T) {
	one := hungOver(t, facingBlocks(streetGap), 0.7, loneSeed)
	other := hungOver(t, facingBlocks(streetGap), 0.7, loneSeed)
	if one.Cables != other.Cables || one.Lanterns != other.Lanterns ||
		len(one.Design().Elements) != len(other.Design().Elements) ||
		len(one.Slots) != len(other.Slots) {
		t.Errorf("one seed hung %d cables and %d lanterns, then %d and %d",
			one.Cables, one.Lanterns, other.Cables, other.Lanterns)
	}
}

// Anti: a volume standing in the air of the street refuses the pair of
// walls, and walls too near or too far apart are no street.
func TestAStreetIsRefusedWhatBlocksOrCannotBeCrossed(t *testing.T) {
	blocked := append(facingBlocks(streetGap), Volume{Kind: "block", Plot: 2,
		Low:  [3]float64{-2, 0, streetLength / 3},
		High: [3]float64{2, streetHeight, streetLength / 2}})
	if got := hungOver(t, blocked, 1, loneSeed); len(got.Streets) != 0 {
		t.Errorf("a street was found with a block standing in it: %v",
			got.Streets)
	}
	for _, gap := range []float64{1, 200} {
		if got := hungOver(t, facingBlocks(gap), 1, loneSeed); len(
			got.Streets) != 0 {
			t.Errorf("a street was found %v m wide", gap)
		}
	}
}

func TestStreetsAreFoundOnBothAxes(t *testing.T) {
	across := map[int]*Lashings{
		crossingX: hungOver(t, facingBlocks(streetGap), 1, loneSeed),
		crossingZ: hungOver(t, turnedBlocks(streetGap), 1, loneSeed)}
	for axis, lashings := range across {
		if len(lashings.Streets) == 0 {
			t.Fatalf("no street across axis %d", axis)
		}
		for _, street := range lashings.Streets {
			if street.Across != axis || math.Abs(street.width()-
				streetGap) > closeEnough {
				t.Errorf("across %d, a street across %d and %v wide", axis,
					street.Across, street.width())
			}
		}
		if lashings.Cables == 0 {
			t.Errorf("across axis %d no cable was hung", axis)
		}
	}
}

func TestEveryCableEndsOnTheTwoWalls(t *testing.T) {
	lashings := hungOver(t, turnedBlocks(streetGap), 1, loneSeed)
	for _, cable := range solidsNamed(lashings.Design(), "lashing cable") {
		points := cable.Form.(lashingCable).Points
		first, last := points[0].Z, points[len(points)-1].Z
		if math.Abs(first+streetGap/2) > 0.2 ||
			math.Abs(last-streetGap/2) > 0.2 {
			t.Errorf("a cable runs from z %v to z %v", first, last)
		}
	}
}

func TestEveryBannerIsASlotAndEveryLanternAPart(t *testing.T) {
	lashings := hungOver(t, facingBlocks(streetGap), 1, loneSeed)
	design := lashings.Design()
	if len(lashings.Slots) == 0 {
		t.Fatal("no banner at full density")
	}
	if banners := solidsNamed(design, "banner"); len(banners) != len(
		lashings.Slots) {
		t.Errorf("%d banners for %d slots", len(banners), len(lashings.Slots))
	}
	if lanterns := solidsNamed(design, "lantern"); len(lanterns) !=
		lashings.Lanterns {
		t.Errorf("%d lanterns drawn, %d counted", len(lanterns),
			lashings.Lanterns)
	}
}

func TestOnlySomeLanternsAreLights(t *testing.T) {
	lashings := hungOver(t, facingBlocks(streetGap), 1, loneSeed)
	lamps, lanterns := 0, solidsNamed(lashings.Design(), "lantern")
	for _, lantern := range lanterns {
		if lantern.Lamp != nil {
			lamps++
		}
	}
	if lamps == 0 || lamps*lamplessShare > len(lanterns) {
		t.Errorf("%d of %d lanterns are lights", lamps, len(lanterns))
	}
}

func TestASignKindLeavesTheSlotsAndDrawsNoPanels(t *testing.T) {
	plain := hungOver(t, facingBlocks(streetGap), 1, loneSeed)
	signed, err := NewLashings(LashingsAsk{Density: 1, Sign: "any"},
		facingBlocks(streetGap), loneSeed)
	if err != nil {
		t.Fatal(err)
	}
	design := plain.Design()
	if len(solidsNamed(design, "banner")) == 0 {
		t.Fatal("the plain street has no banner to compare")
	}
	if got := solidsNamed(signed.Design(), "banner"); len(got) != 0 {
		t.Errorf("a signed street draws %d panels", len(got))
	}
	if len(signed.Slots) != len(plain.Slots) {
		t.Errorf("%d slots signed, %d plain", len(signed.Slots),
			len(plain.Slots))
	}
}

// edgeHalfStep is half the last place edgeKey prints, in metres: a
// coordinate nearer nought than this prints as nought.
const edgeHalfStep = 5e-6

// printed is a coordinate as edgeKey prints it: one that would print as
// nought, a small negative or a negative nought as a pole of a lantern
// has, is nought, not "-0.00000".
func printed(value float64) float64 {
	if math.Abs(value) < edgeHalfStep {
		return 0
	}
	return value
}

// edgeKey is a directed edge of a mesh, to a precision that makes two
// corners that are one the same.
func edgeKey(from, to maths.Vec) string {
	return fmt.Sprintf("%.5f,%.5f,%.5f>%.5f,%.5f,%.5f", printed(from.X),
		printed(from.Y), printed(from.Z), printed(to.X), printed(to.Y),
		printed(to.Z))
}

// closedMesh is whether every directed edge of a mesh has its opposite in
// the mesh: no hole, and every face wound the same way round.
func closedMesh(mesh *Mesh) bool {
	edges := map[string]int{}
	for _, face := range mesh.Triangles {
		for index := range face {
			edges[edgeKey(face[index], face[(index+1)%len(face)])]++
		}
	}
	for _, face := range mesh.Triangles {
		for index := range face {
			if edges[edgeKey(face[(index+1)%len(face)], face[index])] !=
				edges[edgeKey(face[index], face[(index+1)%len(face)])] {
				return false
			}
		}
	}
	return len(edges) > 0
}

func TestALanternIsClosedAndWoundOutward(t *testing.T) {
	mesh, err := lashingLantern{Radius: 0.3, Stretch: 1.25}.Mesh(Detail("quick"))
	if err != nil {
		t.Fatal(err)
	}
	if !closedMesh(mesh) {
		t.Error("the lantern has a hole in it, or a face turned in")
	}
	for _, face := range mesh.Triangles {
		normal := face[1].Sub(face[0]).Cross(face[2].Sub(face[0]))
		if normal.Dot(middle(face[:])) <= 0 {
			t.Errorf("a face of the lantern at %v turns in", middle(face[:]))
		}
	}
}

func TestACableIsClosedAndWoundOutward(t *testing.T) {
	line := []maths.Vec{maths.V(-2, 0, 0), maths.V(0, 0, 0), maths.V(2, 0, 0)}
	mesh, err := lashingCable{Points: line, Radius: 0.05}.Mesh(Detail("quick"))
	if err != nil {
		t.Fatal(err)
	}
	if !closedMesh(mesh) {
		t.Error("the cable has a hole in it, or a face turned in")
	}
	for _, face := range mesh.Triangles {
		normal := face[1].Sub(face[0]).Cross(face[2].Sub(face[0]))
		at := middle(face[:])
		if math.Abs(normal.Unit().X) > 1-closeEnough {
			if normal.X*at.X <= 0 {
				t.Errorf("an end of the cable at %v turns in", at)
			}
		} else if normal.Dot(maths.V(0, at.Y, at.Z)) <= 0 {
			t.Errorf("a side of the cable at %v turns in", at)
		}
	}
}

// Anti: a cable of one point or no radius, a cable that doubles back on
// a point, and a lantern with no size, are refused.
func TestACableOrLanternThatCannotBeMadeIsRefused(t *testing.T) {
	line := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0)}
	for name, cable := range map[string]lashingCable{
		"one point": {Points: line[:1], Radius: 0.05},
		"no radius": {Points: line},
		"doubled":   {Points: []maths.Vec{line[0], line[0]}, Radius: 0.05}} {
		if _, err := cable.Mesh(Detail("quick")); err == nil {
			t.Errorf("a cable of %s was made", name)
		}
	}
	for _, lantern := range []lashingLantern{{0, 1}, {1, 0}, {-1, 1}} {
		if _, err := lantern.Mesh(Detail("quick")); err == nil {
			t.Errorf("the lantern %v was made", lantern)
		}
	}
}
