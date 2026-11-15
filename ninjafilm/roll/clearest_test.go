package roll_test

import (
	"testing"

	"github.com/janearc/slayer/ninjafilm/mocks"
	"github.com/janearc/slayer/ninjafilm/roll"
	"go.uber.org/mock/gomock"
	"pgregory.net/rapid"
)

// A roll's clearest is its lay's, asked of its own records: the roll
// passes the question on and adds nothing.
func TestARollsClearestIsItsLays(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ctrl := gomock.NewController(t)
		records := drawnRecords(t)
		clearest := lamps(t, "clearest", whole)
		laid := mocks.NewMockLay(ctrl)
		laid.EXPECT().Clearest(records).Return(clearest).Times(once)
		made := loaded(t, ctrl, roll.NewStock(super35(t), laid, records),
			grainySeed)
		if got := made.Clearest(); got != clearest {
			t.Fatalf("the lay passes %v, the roll says %v", clearest, got)
		}
	})
}

// Anti: rating a roll does not move its clearest. A push or a pull
// moves where light falls on the curve, not where the curve ends.
func TestRatingARollDoesNotMoveItsClearest(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ctrl := gomock.NewController(t)
		records := drawnRecords(t)
		stock := roll.NewStock(super35(t), stacked(t), records)
		made := loaded(t, ctrl, stock, grainySeed)
		stops := float64(rapid.IntRange(-stopsMost, stopsMost).
			Draw(t, "stops"))
		if made.Rated(stops).Clearest() != made.Clearest() {
			t.Fatalf("%v stops moved the clearest from %v to %v", stops,
				made.Clearest(), made.Rated(stops).Clearest())
		}
	})
}
