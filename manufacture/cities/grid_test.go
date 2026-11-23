package cities

import (
	"math"
	"slices"
	"testing"
)

// gridSeed is a seed for the grid tests, which no grid depends on.
const gridSeed = 5

// sanFrancisco is the San Francisco row as the data says it.
func sanFrancisco(t *testing.T) Item {
	t.Helper()
	item, err := ItemOf("terrace-san-francisco")
	if err != nil || !item.Blocked {
		t.Fatalf("the data gives no blocks: %v %+v", err, item)
	}
	return item
}

func TestAGridLaysTwoFacingRowsOnEveryBlock(t *testing.T) {
	item := sanFrancisco(t)
	box := Quarter{West: 0, South: 0, East: 2040, North: 760}
	laid, err := grid{}.Sites(Ground{Box: &box}, item, gridSeed, 0)
	if err != nil || len(laid.Sites) == 0 || len(laid.Sites)%2 != 0 {
		t.Fatalf("%d rows, %v: want an even, nonzero number", len(laid.Sites),
			err)
	}
	turn := item.Block.Turn
	for index, site := range laid.Sites {
		want := turn
		if index%2 == 1 {
			want = math.Mod(turn+halfTurn, fullDegrees)
		}
		if !site.Faced || math.Abs(site.Heading-want) > 1e-9 {
			t.Fatalf("row %d faces %v, want %v", index, site.Heading, want)
		}
	}
}

func TestNeighbouringQuartersLayEveryBlockOnceAndNoneTwice(t *testing.T) {
	item := sanFrancisco(t)
	whole := Quarter{West: 0, South: 0, East: 2040, North: 1100}
	west := Quarter{West: 0, South: 0, East: 2040, North: 760}
	east := Quarter{West: 0, South: 760, East: 2040, North: 1100}
	each := func(box Quarter) []Site {
		laid, err := grid{}.Sites(Ground{Box: &box}, item, gridSeed, 0)
		if err != nil {
			t.Fatal(err)
		}
		return laid.Sites
	}
	both := append(each(west), each(east)...)
	if len(both) != len(each(whole)) {
		together := len(each(whole))
		t.Errorf("two quarters lay %d rows, the two together %d", len(both),
			together)
	}
	for _, site := range each(west) {
		if slices.Contains(each(east), site) {
			t.Fatalf("a row at %+v is laid by both quarters", site)
		}
	}
}

func TestAGridTakesFewerBySeedAndIsTheSameEachTime(t *testing.T) {
	item := sanFrancisco(t)
	box := Quarter{West: 0, South: 0, East: 2040, North: 760}
	first, _ := grid{}.Sites(Ground{Box: &box}, item, gridSeed, 10)
	again, _ := grid{}.Sites(Ground{Box: &box}, item, gridSeed, 10)
	if len(first.Sites) != 10 || !slices.Equal(first.Sites, again.Sites) {
		t.Errorf("%d rows, and the same again is %v", len(first.Sites),
			slices.Equal(first.Sites, again.Sites))
	}
}

func TestAGridRefusesWhatItCannotLay(t *testing.T) {
	item := sanFrancisco(t)
	box := Quarter{West: 0, South: 0, East: 500, North: 500}
	crowded := item
	crowded.Block.RowDepth = item.Block.Depth
	cases := map[string]struct {
		ground Ground
		item   Item
	}{
		"no box":         {Ground{}, item},
		"no blocks":      {Ground{Box: &box}, Item{Kind: "yurt"}},
		"rows too deep":  {Ground{Box: &box}, crowded},
		"a pitched item": {Ground{Box: &box}, Item{Kind: "winnebago", Pitched: true}},
	}
	for name, each := range cases {
		if _, err := (grid{}).Sites(each.ground, each.item, gridSeed,
			0); err == nil {
			t.Errorf("%s: laid", name)
		}
	}
}

func TestEveryLotOfASanFranciscoRowHasAnEraAndNeighboursDiffer(t *testing.T) {
	drawing, err := newDrawer("san-francisco", 1000)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for bay := range 15 {
		seen[drawing.eraAt(place{Bay: bay})] = true
	}
	delete(seen, "")
	if len(seen) < 2 {
		t.Errorf("a row of fifteen lots drew %v, want more than one era", seen)
	}
	if _, err := Terrace("san-francisco", 1000); err != nil {
		t.Errorf("a row of San Francisco was not drawn: %v", err)
	}
	other, _ := newDrawer("portofino", 1000)
	if other.eraAt(place{}) != "" {
		t.Error("a culture with no eras drew one")
	}
}
