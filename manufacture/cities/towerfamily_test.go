package cities

import (
	"math"
	"testing"
)

// The landmarks of the world are not data. Each is a tower of high
// irregularity with a few features pinned, and these tests say that the
// one generator can make each family.

func TestTheGeneratorMakesATransamericaATaperToASpire(t *testing.T) {
	for seed := range Seed(6) {
		tower := towerOf(t, seed, 1, Pins{Plan: planRectangle, Shafts: 1,
			Taper: &yes, Setbacks: &no, Twist: &no, Spire: &yes,
			Stack: &no, Nest: &no})
		segment := tower.Shafts[0].Segments[0]
		if !(segment.Top.Width < segment.Bottom.Width) {
			t.Errorf("seed %d: no taper", seed)
		}
		drawn := drawnTower(t, tower, "hull", 0)
		if len(named(drawn, "spire")) != 1 {
			spires := len(named(drawn, "spire"))
			t.Errorf("seed %d: %d spires", seed, spires)
		}
		spire, _ := reach(named(drawn, "spire")[0])
		_, top := reach(named(drawn, "spire")[0])
		if top.Y <= tower.height(0) || spire.Y < tower.height(0)-1 {
			t.Errorf("seed %d: the spire is not on top of the tower", seed)
		}
	}
}

func TestTheGeneratorMakesAPetronasTwoStarsStepsAndASkybridge(t *testing.T) {
	for seed := range Seed(6) {
		tower := towerOf(t, seed, 1, Pins{Plan: planStar, Shafts: 2,
			Span: SpanSkybridge, Setbacks: &yes, Taper: &no, Twist: &no})
		if len(tower.Shafts) != 2 || tower.Span != SpanSkybridge {
			t.Fatalf("seed %d: %d shafts joined by %s", seed,
				len(tower.Shafts), tower.Span)
		}
		bottom, _, err := tower.Shafts[0].layer(0)
		if err != nil || len(bottom) != 2 {
			t.Fatalf("seed %d: a star is %d parts, %v", seed, len(bottom),
				err)
		}
		steps := len(tower.Shafts[0].Segments) - 1
		stepped := 0
		for index, segment := range tower.Shafts[0].Segments {
			if index > 0 && segment.Bottom.Width <
				tower.Shafts[0].Segments[index-1].Top.Width {
				stepped++
			}
		}
		hulls := drawnTower(t, tower, "hull", 0)
		if stepped != steps || len(named(hulls,
			"bridge")) != 1 {
			t.Errorf("seed %d: %d of %d joins stepped back", seed, stepped,
				steps)
		}
	}
}

func TestTheGeneratorMakesABurjAYWithItsWingsSteppingBackInTurn(t *testing.T) {
	spiralled := 0
	for seed := range Seed(6) {
		tower := towerOf(t, seed, 1, Pins{Plan: planY, Shafts: 1,
			Setbacks: &yes, Taper: &no, Twist: &no})
		shaft := tower.Shafts[0]
		bottom, _, err := shaft.layer(0)
		if err != nil || len(bottom) != wingCount {
			t.Fatalf("seed %d: a Y is %d parts, %v", seed, len(bottom), err)
		}
		last := shaft.Segments[len(shaft.Segments)-1].Bottom.Wings
		first := shaft.Segments[0].Bottom.Wings
		if len(shaft.Segments) >= wingCount+1 && last != first &&
			last[0] != last[1] {
			spiralled++
		}
	}
	if spiralled == 0 {
		t.Error("no Y stepped its wings back one at a time")
	}
}

func TestTheGeneratorMakesAMarinaBayThreeSlabsAndADeckAcrossTheirTops(t *testing.T) {
	for seed := range Seed(6) {
		tower := towerOf(t, seed, 1, Pins{Plan: planRectangle, Shafts: 3,
			Span: SpanDeck, Taper: &no, Setbacks: &no, Twist: &no})
		drawn := drawnTower(t, tower, "hull", 0)
		decks := named(drawn, "deck")
		if len(decks) != 1 {
			t.Fatalf("seed %d: %d decks", seed, len(decks))
		}
		low, high := xRange(decks[0])
		first, last := tower.Shafts[0], tower.Shafts[2]
		if !(low < first.Centre.X && high > last.Centre.X) ||
			math.Abs(tower.height(0)-tower.height(2)) > tolerance {
			t.Errorf("seed %d: a deck from %v to %v over shafts at %v and "+
				"%v, heights %v and %v", seed, low, high, first.Centre.X,
				last.Centre.X, tower.height(0), tower.height(2))
		}
		_, deckTop := reach(decks[0])
		if deckTop.Y < tower.height(0) {
			t.Errorf("seed %d: the deck is under the tops", seed)
		}
	}
}
