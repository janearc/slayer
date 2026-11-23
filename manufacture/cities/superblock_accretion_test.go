package cities

import (
	"math"
	"testing"
)

// accretionSeeds is how many seeds the accretion's tests draw from.
const accretionSeeds = 6

// berthSeed is a seed that grown over keeps berths on, as the roofs of
// the picture's parcel have always done.
const berthSeed = 6

func TestAccretionOfAValueIsTheValue(t *testing.T) {
	for _, value := range []float64{0, 0.25, halfGrown, 1} {
		got, err := AccretionOf(Accretion{Value: value}, 3)
		if err != nil || got != value {
			t.Errorf("accretion %v came out %v, %v", value, got, err)
		}
	}
}

func TestAccretionOfADrawnOneIsTheSeedsAndTheBlocks(t *testing.T) {
	for seed := range Seed(accretionSeeds) {
		first, err := AccretionOf(Accretion{Drawn: true}, seed)
		if err != nil {
			t.Fatal(err)
		}
		again, _ := AccretionOf(Accretion{Drawn: true}, seed)
		ask := kowloon()
		ask.Accretion = Accretion{Drawn: true}
		block := growSuperblock(t, ask, seed)
		if first < 0 || first > 1 || first != again ||
			first != block.Accretion {
			t.Errorf("seed %d: drawn %v then %v, and the block grown at %v",
				seed, first, again, block.Accretion)
		}
	}
}

func TestAccretionOfWhatIsOutsideNoughtToOneIsRefused(t *testing.T) {
	for _, value := range []float64{-0.1, 1.1, math.NaN(), math.Inf(1)} {
		if got, err := AccretionOf(Accretion{Value: value}, 1); err == nil {
			t.Errorf("accretion %v was taken as %v", value, got)
		}
	}
}

func TestAPlannedBlockKeepsNoBerthForAPartAndAGrownOverOneDoes(t *testing.T) {
	for seed := range Seed(accretionSeeds) {
		ask := kowloon().at(0)
		ask.Berths = true
		if got := len(growSuperblock(t, ask, seed).Berths); got != 0 {
			t.Errorf("seed %d: a planned block keeps %d berths", seed, got)
		}
	}
	grown := kowloon()
	grown.Berths = true
	if len(growSuperblock(t, grown, berthSeed).Berths) == 0 {
		t.Error("a block grown over keeps no berth for a part")
	}
}
