package manufacture

import (
	"math"
	"testing"
)

func TestASuperblocksAccretionIsAskedOrDrawnFromItsSeed(t *testing.T) {
	got, err := SuperblockAccretion(SuperblockAsk{Accretion: 0.5}, 9)
	if err != nil || got != 0.5 {
		t.Errorf("accretion half came out %v, %v", got, err)
	}
	drawn, err := SuperblockAccretion(SuperblockAsk{Drawn: true}, 9)
	again, _ := SuperblockAccretion(SuperblockAsk{Drawn: true}, 9)
	if err != nil || drawn < 0 || drawn > 1 || drawn != again {
		t.Errorf("a drawn accretion came out %v then %v, %v", drawn, again,
			err)
	}
}

func TestASuperblocksAccretionOutsideNoughtToOneIsRefused(t *testing.T) {
	for _, value := range []float64{-1, 2, math.NaN()} {
		if _, err := SuperblockAccretion(SuperblockAsk{Accretion: value},
			1); err == nil {
			t.Errorf("accretion %v was taken", value)
		}
	}
}
