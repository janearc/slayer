package set

import (
	"strings"
	"testing"
)

const absoluteTree = "        position:\n" +
	"          east: {value: -12.5, unit: m}\n" +
	"          north: {value: 450, unit: m}\n" +
	"          up: {value: -3, unit: m}\n"

func TestThingReadsTypedWorldPositionInBothSettings(t *testing.T) {
	for _, setting := range []string{"ground", "space"} {
		t.Run(setting, func(t *testing.T) {
			plant := specimen("tree", absoluteTree)
			read, err := readV2("  setting: " + setting + "\n" +
				stood(plant))
			if err != nil {
				t.Fatal(err)
			}
			position, has := read.Things()[0].Position()
			if !has || position != (WorldPosition{east: -12.5, north: 450, up: -3}) {
				t.Fatalf("position = %+v, present %t", position, has)
			}
		})
	}
}

func TestThingWorldPositionIsAlsoReadByVersionOne(t *testing.T) {
	text := `"$schema": ` + EcosystemSchema + "\n" +
		"scene:\n  name: old\n" + defaultFilm + defaultCamera +
		"  set:\n    things:\n" + specimen("tree", absoluteTree)
	read, err := Read("old-position.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	position, has := read.Things()[0].Position()
	if !has || position != (WorldPosition{east: -12.5, north: 450, up: -3}) {
		t.Fatalf("v1 position = %+v, present %t", position, has)
	}
}

func TestThingWorldPositionRefusesAmbiguousPlacements(t *testing.T) {
	for _, test := range []struct {
		name, setting, extra, want string
	}{
		{"legacy east-north", "ground", "        east: 1\n        north: 2\n", "position and east"},
		{"terrain height", "ground", "        height: {value: 2, unit: m}\n", "position and height"},
		{"space place", "space", "        place: {at: origin}\n", "position and place"},
		{"space orbit", "space", "        around: tree\n", "position and around"},
	} {
		t.Run(test.name, func(t *testing.T) {
			plant := specimen("tree", absoluteTree+test.extra)
			_, err := readV2("  setting: " + test.setting + "\n" +
				stood(plant))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestThingWorldPositionRequiresThreeFiniteMetreCoordinates(t *testing.T) {
	for _, test := range []struct{ name, change string }{
		{"missing up", ""},
		{"wrong unit", "          up: {value: -3, unit: km}\n"},
		{"too far", "          up: {value: -1000000001, unit: m}\n"},
		{"nonfinite", "          up: {value: .nan, unit: m}\n"},
		{"unknown key", "          vertical: {value: -3, unit: m}\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			text := strings.Replace(absoluteTree,
				"          up: {value: -3, unit: m}\n", test.change, 1)
			plant := specimen("tree", text)
			if _, err := readV2(stood(plant)); err == nil {
				t.Fatal("accepted invalid world position")
			}
		})
	}
}

func TestLegacyTerrainPlacementRetainsItsRelativeHeight(t *testing.T) {
	plant := specimen("tree",
		"        east: 12\n        north: 25\n"+
			"        height: {value: 2.5, unit: m}\n")
	read, err := readV2("  setting: ground\n" + stood(plant))
	if err != nil {
		t.Fatal(err)
	}
	tree := read.Things()[0]
	if east, north := tree.At(); east != 12 || north != 25 {
		t.Fatalf("legacy map location = %g, %g", east, north)
	}
	if tree.Height() != 2.5 {
		t.Fatalf("legacy terrain-relative height = %g", tree.Height())
	}
	if _, has := tree.Position(); has {
		t.Fatal("legacy placement gained an absolute position")
	}
}
