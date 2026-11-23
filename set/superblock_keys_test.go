package set_test

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// superblockSet is a set in space of one thing of a kind, saying what it
// is given under the thing.
func superblockSet(kind, says string) string {
	return `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: superblock
  setting: space
  light: none
  film: {width: 1, height: 1, rays: 1}
  cameras:
    away:
      title: away
      orientation: {look_at: [0, 0, 0]}
      location: {at: [0, -100, 0]}
  set:
    things:
      block:
        kind: ` + kind + `
        seed: 1
        east: 0
        north: 0
` + says
}

func TestASuperblockSaysItsParcelItsCeilingAndItsHabit(t *testing.T) {
	source := superblockSet(
		"superblock", "        parcel:\n"+
			"          width: {value: 210, unit: m}\n"+
			"          depth: {value: 120, unit: m}\n"+
			"        ceiling: 14\n        habit: accreted\n"+
			"        parts: {winnebago: 1}\n")
	read, err := set.Read("superblock.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	block := read.Things()[0]
	if block.ParcelWidth() != 210 || block.ParcelDepth() != 120 ||
		block.Ceiling() != 14 || block.Habit() != "accreted" ||
		block.Parts()["winnebago"] != 1 {
		t.Errorf("the superblock says %v by %v, %d, %q, %v",
			block.ParcelWidth(), block.ParcelDepth(), block.Ceiling(),
			block.Habit(), block.Parts())
	}
}

func TestASuperblockThatSaysNothingLeavesItAllToTheData(t *testing.T) {
	source := superblockSet(
		"superblock", "")
	read, err := set.Read("superblock.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	block := read.Things()[0]
	if block.ParcelWidth() != 0 || block.ParcelDepth() != 0 ||
		block.Ceiling() != 0 || block.Habit() != "" {
		t.Errorf("a bare superblock says %v by %v, %d, %q",
			block.ParcelWidth(), block.ParcelDepth(), block.Ceiling(),
			block.Habit())
	}
}

func TestASuperblockSaysItsPeopleTheirCrowdingAndItsAccretion(
	t *testing.T) {
	source := superblockSet(
		"superblock", "        occupants: 9000\n        crowding: 1.5\n"+
			"        accretion: 0.25\n")
	read, err := set.Read("superblock.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	block := read.Things()[0]
	grown, drawn := block.Accretion()
	if block.Occupants() != 9000 || block.Crowding() != 1.5 ||
		grown != 0.25 || drawn {
		t.Errorf("the superblock says %d, %v, %v and %v",
			block.Occupants(), block.Crowding(), grown, drawn)
	}
}

func TestASuperblockDrawsItsAccretionUnlessItSaysOne(t *testing.T) {
	for _, says := range []string{"", "        accretion: drawn\n"} {
		source := superblockSet(
			"superblock", says)
		read, err := set.Read("superblock.yaml", []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		if _, drawn := read.Things()[0].Accretion(); !drawn {
			t.Errorf("%q does not draw its accretion", says)
		}
	}
}

func TestASuperblockThatSaysItWrongIsRefused(t *testing.T) {
	for name, says := range map[string]string{
		"a habit nobody has":          "        habit: lattice\n",
		"a ceiling of none":           "        ceiling: 0\n",
		"a part that is a superblock": "        parts: {superblock: 1}\n",
		"a parcel in feet": "        parcel:\n" +
			"          width: {value: 210, unit: ft}\n",
		"a parcel of nothing": "        parcel:\n" +
			"          width: {value: 0, unit: m}\n",
		"an accretion past one":  "        accretion: 1.5\n",
		"an accretion of a word": "        accretion: lots\n",
		"nobody in it":           "        occupants: 0\n",
		"half a person":          "        occupants: 10.5\n",
		"no crowding":            "        crowding: 0\n",
	} {
		source := superblockSet(
			"superblock", says)
		_, err := set.Read("superblock.yaml", []byte(source))
		if err == nil || !strings.Contains(err.Error(), "superblock.yaml") {
			t.Errorf("%s was read: %v", name, err)
		}
	}
}

func TestAParcelOnAThingThatIsNotASuperblockIsRefused(t *testing.T) {
	for _, says := range []string{"        ceiling: 14\n",
		"        habit: accreted\n", "        accretion: 0.5\n",
		"        occupants: 900\n", "        crowding: 1\n",
		"        parcel: {width: {value: 9, unit: m}}\n"} {
		source := superblockSet("yurt",
			says)
		_, err := set.Read("superblock.yaml", []byte(source))
		if err == nil || !strings.Contains(err.Error(), "only a superblock") {
			t.Errorf("%q was read on a yurt: %v", says, err)
		}
	}
}

func TestTheSuperblockSetsOfThePicturesRead(t *testing.T) {
	for _, name := range []string{"superblock-kowloon.yaml",
		"superblock-planned.yaml", "superblock-half.yaml",
		"superblock-accreted.yaml"} {
		raw, err := os.ReadFile("../sets/" + name)
		if err != nil {
			t.Fatal(err)
		}
		read, err := set.Read(name, raw)
		if err != nil {
			t.Fatal(err)
		}
		if got := len(read.Things()); got != 1 {
			t.Errorf("%s: %d things in the set, want the one superblock",
				name, got)
		}
	}
}
