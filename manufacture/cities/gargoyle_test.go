package cities

import (
	"math"
	"strings"
	"testing"
)

// gargoyleSeeds is how many seeds the gargoyle tests look at.
const gargoyleSeeds = 6

// gargoyleTol is how close two lengths are, in metres, to be the same.
const gargoyleTol = 1e-9

// corbelOf is a corbel drawn at the quick level, or the test's end.
func corbelOf(t *testing.T, seed Seed, stone string) (GargoyleBlock, []Rendered) {
	t.Helper()
	block, err := Corbel(seed, stone)
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := block.Design.Rendered("quick")
	if err != nil {
		t.Fatal(err)
	}
	return block, drawn
}

func TestACorbelsSlabTopIsTheFrameOriginAndItStandsOutOfTheWall(t *testing.T) {
	for seed := range Seed(gargoyleSeeds) {
		block, drawn := corbelOf(t, seed, "granite")
		var slabs int
		for _, one := range drawn {
			low, high := worldReach(one)
			if one.Mesh.Name == "corbel panel" {
				// the panel is set into the wall, by its back
				if math.Abs(low[2]+block.Back) > gargoyleTol {
					t.Errorf("seed %d: the panel is set %v into the wall, "+
						"want %v", seed, -low[2], block.Back)
				}
				continue
			}
			if low[2] < -gargoyleTol {
				t.Errorf("seed %d: %s reaches into the wall, to z %v", seed,
					one.Mesh.Name, low[2])
			}
			if one.Mesh.Name != "corbel slab" {
				continue
			}
			slabs++
			if math.Abs(high[1]) > gargoyleTol ||
				math.Abs(high[2]-block.Proud) > gargoyleTol {
				t.Errorf("seed %d: the slab tops at y %v and stands out %v, "+
					"want 0 and %v", seed, high[1], high[2], block.Proud)
			}
		}
		if slabs != 1 {
			t.Errorf("seed %d: %d slabs", seed, slabs)
		}
	}
}

func TestEachStepOfACorbelStandsOutLessThanTheOneAboveIt(t *testing.T) {
	_, drawn := corbelOf(t, 1, "concrete")
	last := math.Inf(1)
	var steps int
	for _, one := range drawn {
		if one.Mesh.Name != "corbel step" {
			continue
		}
		steps++
		_, high := worldReach(one)
		if !(high[2] < last) {
			t.Errorf("step %d stands out %v, not less than %v", steps,
				high[2], last)
		}
		last = high[2]
	}
	if steps < 2 {
		t.Errorf("%d steps make no bracket", steps)
	}
}

func TestACorbelAndItsCorniceAreOneStone(t *testing.T) {
	for _, stone := range []string{"granite", "concrete"} {
		_, drawn := corbelOf(t, 2, stone)
		for _, one := range drawn {
			if one.Material != drawn[0].Material {
				t.Errorf("%s corbel: %s is not the stone of the slab", stone,
					one.Mesh.Name)
			}
		}
		design, err := GargoyleCornice(2, stone)
		if err != nil {
			t.Fatal(err)
		}
		wall, err := design.Elements[0].Element.Rendered("quick")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(wall.Material.Name, stone) {
			t.Errorf("the %s cornice's wall is %q", stone, wall.Material.Name)
		}
	}
}

func TestACorniceHoldsOneMountForEachGargoyleOnItsCorbels(t *testing.T) {
	drawing := drawerOf(t, gargoyleCulture, 1)
	want := drawing.count("gargoyle-count", place{})
	creatures, err := GargoyleCreatures()
	if err != nil {
		t.Fatal(err)
	}
	for seed := range Seed(gargoyleSeeds) {
		design, err := GargoyleCornice(seed, "granite")
		if err != nil {
			t.Fatal(err)
		}
		if len(design.Mounts) != want {
			t.Fatalf("seed %d: %d mounts, want %d", seed, len(design.Mounts),
				want)
		}
		for index, mount := range design.Mounts {
			known := false
			for _, creature := range creatures {
				known = known || mount.Kind == GargoyleKind(creature, "granite")
			}
			if !known {
				t.Errorf("seed %d: mount %d is %q, of no creature the data "+
					"names", seed, index, mount.Kind)
			}
			if index > 0 && !(mount.Place.Origin.X >
				design.Mounts[index-1].Place.Origin.X) {
				t.Errorf("seed %d: mount %d is not along the wall", seed,
					index)
			}
		}
	}
}

func TestTheGargoylesStandUnderTheCorniceAndClearOfItsMoulding(t *testing.T) {
	drawing := drawerOf(t, gargoyleCulture, 1)
	height, course := drawing.fixed("wall-height"),
		drawing.fixed("course-height")
	design, err := GargoyleCornice(1, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, mount := range design.Mounts {
		if !(mount.Place.Origin.Y < height-course) {
			t.Errorf("a corbel top at %v is not under the course at %v",
				mount.Place.Origin.Y, height-course)
		}
	}
}

func TestACorniceNamesNoStoneItDoesNotHaveAndSaysWhichThereAre(t *testing.T) {
	_, err := GargoyleCornice(1, "marzipan")
	if err == nil || !strings.Contains(err.Error(), "granite") {
		t.Fatalf("marzipan: %v, want a refusal that lists the stones", err)
	}
	if _, err := Corbel(1, "marzipan"); err == nil {
		t.Fatal("a corbel of marzipan was drawn")
	}
}

func TestTheSameSeedIsTheSameCorniceAndAnotherDrawsItsOwnCreatures(t *testing.T) {
	one, err := GargoyleCornice(7, "granite")
	if err != nil {
		t.Fatal(err)
	}
	same, err := GargoyleCornice(7, "granite")
	if err != nil {
		t.Fatal(err)
	}
	for index := range one.Mounts {
		if one.Mounts[index] != same.Mounts[index] {
			t.Errorf("mount %d changed with the same seed", index)
		}
	}
	kinds := map[string]bool{}
	for seed := range Seed(gargoyleSeeds * gargoyleSeeds) {
		design, err := GargoyleCornice(seed, "granite")
		if err != nil {
			t.Fatal(err)
		}
		for _, mount := range design.Mounts {
			kinds[mount.Kind] = true
		}
	}
	creatures, _ := GargoyleCreatures()
	if len(kinds) != len(creatures) {
		t.Errorf("%d kinds over many seeds, of %d creatures: %v", len(kinds),
			len(creatures), kinds)
	}
}
