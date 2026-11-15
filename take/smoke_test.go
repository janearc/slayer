package take_test

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm"
	"github.com/janearc/slayer/ninjafilm/frame"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/take"
)

// the smoke tests film a small scene of slayer's through ninjafilm's
// doors and slayer's take, as a client would: this file is outside the
// package and can reach nothing that is not exported. doc/take.md shows
// the same code.
//
// slayer's frames and ninjafilm's are two types of one shape, until the
// film moves out of slayer. render/carry takes one across.

// the take is aligned to time: frame n of a take opens at n over the
// rate, however long the take
func TestATakeNeverDrifts(t *testing.T) {
	pace, err := take.Cranked(16, math.Pi, 1)
	if err != nil {
		t.Fatal(err)
	}
	still := thing.Still(maths.Subspace{})
	shot, err := take.Take(still, pace, time.Hour)
	if err != nil || shot.Frames() != 57600 {
		t.Fatalf("an hour at sixteen is %d frames, %v", shot.Frames(),
			err)
	}
	for _, number := range []int{0, 1, 1000, 57599} {
		opens := time.Duration(number) * time.Second / 16
		want := opens + time.Second/64
		if got := shot.Glimpses(number)[0].At(); got != want {
			t.Errorf("frame %d is caught at %v, want %v", number,
				got, want)
		}
	}
}

// a roll rated a stop over prints a grey card clearer, and one rated a
// stop under prints it denser: a client's way to match a stock to a
// scene's light
func TestARollIsRatedOverAndUnder(t *testing.T) {
	roll, err := ninjafilm.Roll("smoke", 1921)
	if err != nil {
		t.Fatal(err)
	}
	card := frame.New(1, 1)
	card.Pix[0] = frame.Pixel{R: 0.125, G: 0.125, B: 0.125, Cover: 1}
	shown := func(stops float64) float32 {
		printed, err := roll.Rated(stops).Print(0, []*frame.Frame{card})
		if err != nil {
			t.Fatal(err)
		}
		return printed.At(0, 0).G
	}
	under, rated, over := shown(-1), shown(0), shown(1)
	if under >= rated || rated >= over {
		t.Errorf("a grey card prints %v under, %v rated, %v over",
			under, rated, over)
	}
}
