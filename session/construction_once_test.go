package session

import "testing"

// TestASecondAskForTheSameKindIsTheFirstAnswer checks that the row of a
// kind is worked out once: a second ask is given the same row, which a
// row made afresh would not be.
func TestASecondAskForTheSameKindIsTheFirstAnswer(t *testing.T) {
	stage := testStaging()
	first, err := stage.construction("spruce", "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := stage.construction("spruce", "")
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
	spruce, err := stage.construction("spruce", "")
	if err != nil {
		t.Fatal(err)
	}
	fern, err := stage.construction("fern", "")
	if err != nil {
		t.Fatal(err)
	}
	if spruce == fern {
		t.Error("a fern was given the spruce's row")
	}
	again, err := stage.construction("spruce", "")
	if err != nil || again != spruce {
		t.Error("asking for the fern lost the spruce's row")
	}
}

// TestARecipeThereIsNoneOfIsRefusedInTheSameWordsEachTime checks the
// refusal is kept, and said as it was the first time, and that a recipe
// that differs is refused for its own name.
func TestARecipeThereIsNoneOfIsRefusedInTheSameWordsEachTime(t *testing.T) {
	stage := testStaging()
	_, first := stage.construction("spruce", "no such recipe")
	_, second := stage.construction("spruce", "no such recipe")
	if first == nil || second == nil {
		t.Fatal("a recipe there is none of was not refused")
	}
	if first != second {
		t.Errorf("refused as %q, then as %q", first, second)
	}
	_, other := stage.construction("spruce", "another")
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
