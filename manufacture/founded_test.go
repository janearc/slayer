package manufacture

import (
	"testing"

	"github.com/janearc/slayer/manufacture/cities"
)

// fallingToTheDoor is a ground that falls three in ten along z, toward the
// front wall of a terrace, which is at positive z.
func fallingToTheDoor(_, z float64) float64 { return -0.3 * z }

func TestATerraceFoundedAtItsDoorsStandsLowerThanOneOnTheHighestGround(t *testing.T) {
	row, err := KindNamed("terrace-portofino")
	if err != nil {
		t.Fatal(err)
	}
	highest, err := Founded(row, fallingToTheDoor, cities.FoundHighest)
	if err != nil {
		t.Fatal(err)
	}
	door, err := Founded(row, fallingToTheDoor, cities.FoundAtDoor)
	if err != nil {
		t.Fatal(err)
	}
	high, err := MakeWithComplexity(highest, sow(1), "hull")
	if err != nil {
		t.Fatal(err)
	}
	low, err := MakeWithComplexity(door, sow(1), "hull")
	if err != nil {
		t.Fatal(err)
	}
	if !(low.Bounds().Max.Y < high.Bounds().Max.Y-1) {
		t.Errorf("founded at its door a terrace is %v high, and %v on the "+
			"highest ground", low.Bounds().Max.Y, high.Bounds().Max.Y)
	}
}

func TestOnlyATerraceOnSomeGroundCanBeFoundedAtItsDoor(t *testing.T) {
	yurt, err := KindNamed("yurt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Founded(yurt, fallingToTheDoor, cities.FoundAtDoor); err == nil {
		t.Error("a yurt was founded at its door")
	}
	condominium, err := KindNamed("condominium")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Founded(condominium, fallingToTheDoor,
		cities.FoundAtDoor); err == nil {
		t.Error("a condominium was founded at its door")
	}
	terrace, err := KindNamed("terrace-portofino")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Founded(terrace, nil, cities.FoundAtDoor); err == nil {
		t.Error("a terrace was founded on no ground")
	}
}

func TestTheRunOfATerraceIsTheLengthOfTheRowItsKindMakes(t *testing.T) {
	const number = 42
	run, err := RunOf("terrace-portofino", number)
	if err != nil {
		t.Fatal(err)
	}
	row, err := KindNamed("terrace-portofino")
	if err != nil {
		t.Fatal(err)
	}
	made, err := MakeWithComplexity(row, sow(number), "hull")
	if err != nil {
		t.Fatal(err)
	}
	bounds := made.Bounds()
	if length := bounds.Max.X - bounds.Min.X; length > run.Length ||
		length < run.Length-run.Length/10 {
		t.Errorf("the row made is %v m long and its run says %v", length,
			run.Length)
	}
	if _, err := RunOf("yurt", number); err == nil {
		t.Error("the run of a yurt was given")
	}
}
