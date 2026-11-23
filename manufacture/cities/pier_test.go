package cities

import (
	"math"
	"reflect"
	"testing"
)

// pierSeeds is how many seeds the pier tests look at.
const pierSeeds = 8

// otherDistrict is a seed in another district from seed one: a pier's
// size is drawn once for a district, so seeds beside each other agree.
const otherDistrict = 1000

// drawnPier is a pier drawn at the quick level, or the test's end.
func drawnPier(t *testing.T, seed Seed) []Rendered {
	t.Helper()
	design, err := Pier(seed)
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := design.Rendered("quick")
	if err != nil {
		t.Fatal(err)
	}
	return drawn
}

func TestAPiersDeckStandsOverTheSeaAndItsPilesGoBelowIt(t *testing.T) {
	drawing := drawerOf(t, pierCulture, 1)
	height := drawing.fixed("deck-height")
	depth := drawing.fixed("pile-depth")
	for seed := range Seed(pierSeeds) {
		var decks, piles int
		for _, one := range drawnPier(t, seed) {
			low, high := worldReach(one)
			switch one.Mesh.Name {
			case "deck":
				decks++
				if math.Abs(high[1]-height) > heightTol {
					t.Errorf("seed %d: the deck's top is %v, want %v", seed,
						high[1], height)
				}
			case "pile":
				piles++
				if math.Abs(low[1]+depth) > heightTol ||
					high[1] >= height {
					t.Errorf("seed %d: a pile from %v to %v", seed, low[1],
						high[1])
				}
			}
		}
		if decks != 1 || piles < 10 {
			t.Errorf("seed %d: %d decks and %d piles", seed, decks, piles)
		}
	}
}

func TestAPiersShedStandsOnItsDeckWithTheApronClear(t *testing.T) {
	apron := drawerOf(t, pierCulture, 1).fixed("shed-apron")
	for seed := range Seed(pierSeeds) {
		var deckLow, deckHigh, shedLow, shedHigh [3]float64
		for _, one := range drawnPier(t, seed) {
			switch one.Mesh.Name {
			case "deck":
				deckLow, deckHigh = worldReach(one)
			case "shed":
				shedLow, shedHigh = worldReach(one)
			}
		}
		if shedLow[0]-deckLow[0] < apron-heightTol ||
			deckHigh[0]-shedHigh[0] < apron-heightTol ||
			shedLow[2] < deckLow[2] || shedHigh[2] > deckHigh[2] ||
			math.Abs(shedLow[1]-deckHigh[1]) > heightTol {
			t.Errorf("seed %d: a shed from %v to %v on a deck from %v to %v",
				seed, shedLow, shedHigh, deckLow, deckHigh)
		}
	}
}

func TestPiersDifferBySeedAndRepeatBySeed(t *testing.T) {
	first, err := Pier(1)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Pier(1)
	if err != nil {
		t.Fatal(err)
	}
	other, err := Pier(otherDistrict)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.Elements, again.Elements) {
		t.Error("one seed drew two piers")
	}
	if reflect.DeepEqual(first.Elements, other.Elements) {
		t.Error("two seeds drew the same pier")
	}
}
