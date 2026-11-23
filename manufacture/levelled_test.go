package manufacture

import (
	"testing"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
)

func TestEachKindIsLevelledByItsLadder(t *testing.T) {
	for name, want := range map[string]string{"spruce": treeLadder,
		"norway": treeLadder, "terrace-portofino": buildingLadder,
		"tower": buildingLadder, "condominium": buildingLadder,
		"soho": kindLadder} {
		row, err := KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		got, ladder, err := LadderOf(row)
		if err != nil || got != want || len(ladder) == 0 {
			t.Errorf("%s is levelled by %s, %v: %v", name, got, ladder, err)
		}
	}
}

func TestOnlyBuildingsAreDrawnPartByPart(t *testing.T) {
	for name, want := range map[string]bool{"terrace-groningen": true,
		"tower": true, "spruce": false, "condominium": false} {
		row, err := KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		if LevelsParts(row) != want {
			t.Errorf("%s drawn part by part: %v", name, !want)
		}
	}
}

func TestAKindNotDrawnPartByPartIsRefusedALevelledMaking(t *testing.T) {
	row, err := KindNamed("spruce")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := MakeLevelled(row, sow(1), constantPass("quick"),
		maths.Subspace{}); err == nil {
		t.Error("a spruce was made part by part")
	}
}

func TestLevelledIsNotARepresentation(t *testing.T) {
	for _, name := range Complexities() {
		if name == Levelled {
			t.Error("levelled is named a representation")
		}
	}
	row, err := KindNamed("tower")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := MakeWithComplexity(row, sow(1), Levelled); err == nil {
		t.Error("a tower was made at the level levelled")
	}
}

func TestATerraceLevelledFromNearIsFinerThanFromFar(t *testing.T) {
	row, err := KindNamed("terrace-portofino")
	if err != nil {
		t.Fatal(err)
	}
	ladder, err := cities.Ladder(buildingLadder)
	if err != nil {
		t.Fatal(err)
	}
	faces := func(from maths.Vec) int {
		eye := strategies.Eye{Place: maths.Subspace{Origin: from,
			Orientation: maths.LookAtUp(from, maths.V(20, 5, 0),
				maths.Up)}, Wide: 1, Across: 960, Down: 540}
		leveller, err := cities.NewLeveller(eye, ladder)
		if err != nil {
			t.Fatal(err)
		}
		made, err := MakeLevelled(row, sow(1), cities.Pass{
			Leveller: leveller, Sees: eye.Sees, Finest: "demo"},
			maths.Subspace{})
		if err != nil {
			t.Fatal(err)
		}
		parts := 0
		for _, count := range made.groupParts {
			parts += count
		}
		return parts
	}
	near, far := faces(maths.V(0, 2, 15)), faces(maths.V(0, 50, 4000))
	if !(near > far) {
		t.Errorf("a terrace near is %d parts, far %d", near, far)
	}
}
