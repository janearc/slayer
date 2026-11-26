package session

import "testing"

// TestASecondAskForTheSameKindIsTheFirstAnswer checks that the row of a
// kind is worked out once: a second ask is given the same row, which a
// row made afresh would not be.
func TestASecondAskForTheSameKindIsTheFirstAnswer(t *testing.T) {
	stage := testStaging()
	first, err := stage.construction("cobble", "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := stage.construction("cobble", "")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Error("the second ask worked the row out again")
	}
}

// TestAKindThatDiffersIsWorkedOutAfresh is the anti-test of the key: a
// kind other than the first is its own row.
func TestAKindThatDiffersIsWorkedOutAfresh(t *testing.T) {
	stage := testStaging()
	stone, err := stage.construction("cobble", "")
	if err != nil {
		t.Fatal(err)
	}
	can, err := stage.construction("can", "")
	if err != nil {
		t.Fatal(err)
	}
	if stone == can {
		t.Error("a can was given the cobble's row")
	}
	again, err := stage.construction("cobble", "")
	if err != nil || again != stone {
		t.Error("asking for the can lost the cobble's row")
	}
}

// TestARecipeThereIsNoneOfIsRefusedInTheSameWordsEachTime checks the
// refusal is kept, and said as it was the first time, and that a recipe
// that differs is refused for its own name.
func TestARecipeThereIsNoneOfIsRefusedInTheSameWordsEachTime(t *testing.T) {
	stage := testStaging()
	_, first := stage.construction("cobble", "no such recipe")
	_, second := stage.construction("cobble", "no such recipe")
	if first == nil || second == nil {
		t.Fatal("a recipe there is none of was not refused")
	}
	if first != second {
		t.Errorf("refused as %q, then as %q", first, second)
	}
	_, other := stage.construction("cobble", "another")
	if other == nil || other.Error() == first.Error() {
		t.Errorf("a recipe that differs was refused as %v", other)
	}
}

// TestAnUnknownKindIsRefusedInTheSameWordsEachTime checks a kind there is
// none of is refused as manufacture refuses it, each time it is asked.
func TestAnUnknownKindIsRefusedInTheSameWordsEachTime(t *testing.T) {
	stage := testStaging()
	_, first := stage.construction("no such kind", "")
	_, second := stage.construction("no such kind", "")
	if first == nil || second == nil || first.Error() != second.Error() {
		t.Errorf("refused as %v, then as %v", first, second)
	}
}
