package kelp

import (
	"testing"

	"github.com/janearc/slayer/manufacture/branching"
)

type fixedSeed struct{}

func (fixedSeed) Draw(int, int, int) float64       { return .5 }
func (fixedSeed) Swing(int, int, int) float64      { return 0 }
func (s fixedSeed) Sprout(int, int) branching.Seed { return s }
func (s fixedSeed) OnPiece(int) branching.Seed     { return s }
func sow(uint64) branching.Seed                    { return fixedSeed{} }

var kelpRead = func() kelpBook {
	book, err := readKelp(kelpRaw)
	if err != nil {
		panic(err)
	}
	return book
}()

// TestKelpLeansWithACurrent checks that the same seed in a current hangs
// a stipe's top farther downstream than in still water, and that the
// swell moves it with time.
func TestKelpLeansWithACurrent(test *testing.T) {
	still := kelpPlant{book: &kelpRead, from: sow(4), water: kelpRead.Conditions["slack"]}
	moving := kelpPlant{book: &kelpRead, from: sow(4), water: kelpRead.Conditions["surge"]}
	top := still.top()
	if still.drift(top).X != 0 {
		test.Fatal("still water moved a stipe")
	}
	if moving.drift(top).X <= 0 {
		test.Fatal("a current did not lean a stipe downstream")
	}
	later := moving
	later.clock = moving.water.SwellPeriod / 4
	if later.drift(top).X == moving.drift(top).X {
		test.Fatal("the swell did not move with time")
	}
	if moving.drift(0).Len() != 0 {
		test.Fatal("the foot moved")
	}
}

func TestProfilesRequireASeedFactory(t *testing.T) {
	if _, err := Read("kelp.yaml", kelpRaw, nil); err == nil {
		t.Fatal("accepted a missing seed factory")
	}
}
