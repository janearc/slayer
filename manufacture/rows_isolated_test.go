package manufacture

import (
	"reflect"
	"testing"
)

// isolatedSeed is the seed the isolation tests make stones from; the
// geometry records of cobble and pebble are held at it.
const isolatedSeed = 5

// invented is a row no catalogue has: a stone that is not any other.
func invented() stone {
	row := stoneRows()["cobble"]
	row.long = span{1, 2}
	return row
}

// A row added to the table must not change how any other kind is made:
// each kind is made from its own row and its seed, and from nothing else
// in the table. This guards adding a row; it is not the guard for the
// one-ulp case, which TestAStonesBreakingIsWorkedOutAsConstants holds.
func TestAddingARowDoesNotChangeACobble(t *testing.T) {
	before := mustMake(t, "cobble", isolatedSeed).skins()
	rows := table()
	rows["invented-stone"] = invented()
	how, known := rows["cobble"]
	if !known {
		t.Fatal("cobble is not in the table")
	}
	after, err := makeOf(kind{name: "cobble", how: how}, sow(isolatedSeed))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after.skins(), before) {
		t.Error("adding a row changed the geometry of a cobble")
	}
}

// A stone's breaking is worked out as constants, exactly. Worked out in
// float64 from the same numbers it differs in the last place, which moved
// every pebble, cobble and campfire when plates were added.
func TestAStonesBreakingIsWorkedOutAsConstants(t *testing.T) {
	const exact = 1 - squarest
	broken := stoneBreaking()
	if broken.squareless != exact || broken.flatless != exact {
		t.Errorf("a stone's squareless %v and flatless %v are not %v",
			broken.squareless, broken.flatless, exact)
	}
	if complement(squarest) == exact {
		t.Error("float64 and constant arithmetic no longer differ, so the " +
			"plate's own working out may be made a constant")
	}
}
