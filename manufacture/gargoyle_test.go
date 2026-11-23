package manufacture

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/texture"
)

// gargoyleSeeds is how many seeds the gargoyle tests look at.
const gargoyleSeeds = 6

// gargoyleCornices is the number of gargoyles the cornice's data draws,
// which the row the study was asked for is.
const gargoyleCornices = 5

// gargoyleTol is how close two lengths are, in metres, to be the same.
const gargoyleTol = 1e-9

// farAway is a pixel footprint, in metres, wider than any grain of stone.
const farAway = 100.0

// gargoyleKindNames are the kind of every creature in every stone.
func gargoyleKindNames(t *testing.T) []string {
	t.Helper()
	creatures, err := cities.GargoyleCreatures()
	if err != nil {
		t.Fatal(err)
	}
	stones, err := cities.GargoyleStones()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, creature := range creatures {
		for _, stone := range stones {
			names = append(names, cities.GargoyleKind(creature, stone))
		}
	}
	return names
}

func TestEveryCreatureAndStoneTheDataNamesHasARowHere(t *testing.T) {
	creatures, _ := cities.GargoyleCreatures()
	for _, name := range creatures {
		if _, known := gargoyleCreatures()[name]; !known {
			t.Errorf("the data names the creature %q, and no row carves it",
				name)
		}
	}
	stones, _ := cities.GargoyleStones()
	for _, name := range stones {
		if _, known := gargoyleStones()[name]; !known {
			t.Errorf("the data names the stone %q, and no row cuts it", name)
		}
		if _, err := kindNamed(gargoyleCorniceName(name)); err != nil {
			t.Errorf("no cornice of %s: %v", name, err)
		}
	}
	for _, name := range gargoyleKindNames(t) {
		if _, err := kindNamed(name); err != nil {
			t.Errorf("%s is not a kind: %v", name, err)
		}
	}
}

func TestAGargoyleIsCarvedWhollyFromOneStoneAndNothingOfTheCreatureStays(t *testing.T) {
	for _, name := range gargoyleKindNames(t) {
		made := mustMake(t, name, 1)
		if len(made.native) == 0 {
			t.Fatalf("%s: no creature was carved", name)
		}
		stone := made.native[0].Material
		for _, piece := range made.native {
			if piece.Material.Name == spoutMaterial {
				continue
			}
			if piece.Material != stone {
				t.Errorf("%s: a piece is %q, not the stone %q", name,
					piece.Material.Name, stone.Name)
			}
		}
		if !strings.Contains(name, strings.TrimPrefix(stone.Name, "gargoyle ")) {
			t.Errorf("%s is carved of %q", name, stone.Name)
		}
		for _, look := range made.dressed {
			if _, boarded := look.Texture.(texture.Boarded); boarded {
				continue
			}
			if _, textured := look.Texture.(texture.Dusted); !textured {
				t.Errorf("%s: the corbel's %q has no stone grain", name,
					look.Name)
			}
		}
	}
}

func TestTheStoneFadesToItsAverageWhereAPixelIsWiderThanItsGrain(t *testing.T) {
	for name, stone := range gargoyleStones() {
		look := stone.look("test "+name, 1, 1)
		banded, can := look.Texture.(interface {
			Within(maths.Vec, float64) maths.Light
		})
		if !can {
			t.Fatalf("%s: its texture has no band limit", name)
		}
		far := banded.Within(maths.V(0.3, 1.7, 2.9), farAway)
		if math.Abs(far.R-look.Colour.R) > gargoyleTol ||
			math.Abs(far.G-look.Colour.G) > gargoyleTol ||
			math.Abs(far.B-look.Colour.B) > gargoyleTol {
			t.Errorf("%s far away is %v, not its average %v", name, far,
				look.Colour)
		}
	}
}

func TestACreaturesHeldPointIsWhereItsRowHoldsIt(t *testing.T) {
	block, err := cities.Corbel(1, "granite")
	if err != nil {
		t.Fatal(err)
	}
	for name, creature := range gargoyleCreatures() {
		for number := range uint64(gargoyleSeeds) {
			pose := creature.pose(sow(number), block)
			want := maths.V(0, creature.Up, creature.Out*block.Proud)
			if got := pose.Out(creature.HeldAt()); got.Sub(want).Len() > 1e-6 {
				t.Errorf("%s seed %d: held at %v, want %v", name, number, got,
					want)
			}
		}
	}
}

func TestEachSpoutIsInFrontOfTheWallAndOverTheCorbelsFootAndASeedMovesIt(t *testing.T) {
	for _, name := range gargoyleKindNames(t) {
		first, err := Spouts(name, 1)
		if err != nil || len(first) != 1 {
			t.Fatalf("%s: %d spouts, %v", name, len(first), err)
		}
		again, _ := Spouts(name, 1)
		if first[0] != again[0] {
			t.Errorf("%s: the same seed moved the spout", name)
		}
		moved := false
		for number := uint64(2); number < gargoyleSeeds+2; number++ {
			each, _ := Spouts(name, number)
			moved = moved || each[0].Sub(first[0]).Len() > gargoyleTol
			if !(each[0].Z > 0) {
				t.Errorf("%s seed %d: the spout is inside the wall, at z %v",
					name, number, each[0].Z)
			}
		}
		if !moved {
			t.Errorf("%s: no seed moves the spout", name)
		}
	}
}

func TestACorniceHoldsItsRowOfGargoylesAlongTheWallEachWithASpout(t *testing.T) {
	for _, stone := range []string{"granite", "concrete"} {
		name := gargoyleCorniceName(stone)
		spouts, err := Spouts(name, 1)
		if err != nil || len(spouts) != gargoyleCornices {
			t.Fatalf("%s: %d spouts, %v", name, len(spouts), err)
		}
		for index := 1; index < len(spouts); index++ {
			if !(spouts[index].X > spouts[index-1].X) {
				t.Errorf("%s: spout %d is not along the wall", name, index)
			}
		}
		made := mustMake(t, name, 1)
		if len(made.native) < gargoyleCornices {
			t.Errorf("%s: %d carved pieces for %d gargoyles", name,
				len(made.native), gargoyleCornices)
		}
	}
}

func TestTheGargoylesOfACorniceAreNotFiveOfOne(t *testing.T) {
	spouts, err := Spouts(gargoyleCorniceName("granite"), 1)
	if err != nil {
		t.Fatal(err)
	}
	heights := map[float64]bool{}
	for _, spout := range spouts {
		heights[math.Round(spout.Y*1e6)] = true
	}
	if len(heights) < 2 {
		t.Errorf("five gargoyles all spout at one height: %v", spouts)
	}
}

func TestAKindThatIsNotAGargoyleHasNoSpoutAndANoStoneKindIsRefused(t *testing.T) {
	if _, err := Spouts("fern", 1); err == nil {
		t.Error("a fern has a spout")
	}
	if _, err := Spouts("gargoyle-orca-marzipan", 1); err == nil {
		t.Error("a marzipan gargoyle was found")
	}
	if _, err := Spouts("gargoyle-dodo-granite", 1); err == nil {
		t.Error("a dodo gargoyle was found")
	}
}

func TestAGargoyleWantsASownSeed(t *testing.T) {
	for _, name := range gargoyleKindNames(t) {
		row, err := kindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := makeOf(row, seed{}); err == nil {
			t.Errorf("%s was made from a seed nobody sowed", name)
		}
	}
}
