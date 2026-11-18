package rig

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

// The floods are the left pass white and the right through the gel, each
// at the gain times its flash.
func TestFloodsAreTheTwoPassesAtTheirFlashes(t *testing.T) {
	got := Floods([2]string{"left", "right"}, 0.5, 0.25, 2)
	want := []Layer{{"left", 1, [3]float64{1, 1, 1}},
		{"right", 0.5, [3]float64{1, 0.25, 0.85}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Floods gave %v, want %v", got, want)
	}
}

// The overhead is its own pass at the gain times the cue's level, in the
// daylight colour.
func TestOverheadIsTheGainTimesTheLevel(t *testing.T) {
	got := Overhead(0.5, 4)
	want := Layer{"overhead", 2, [3]float64{0.62, 0.8, 1.0}}
	if got != want {
		t.Errorf("Overhead gave %v, want %v", got, want)
	}
}

// A layer adds its pass into the frame by its amount and colour, channel
// by channel, and keeps what was there.
func TestAddLaysThePassIntoTheFrame(t *testing.T) {
	layer := Layer{"pass", 2, [3]float64{0.5, 1, 0.25}}
	into := []float64{0, 0, 0, 1, 1, 1}
	if err := layer.Add(into, []float64{1, 2, 3, 4, 5, 6}); err != nil {
		t.Fatal(err)
	}
	want := []float64{1, 4, 1.5, 5, 11, 4}
	for i := range want {
		if math.Abs(into[i]-want[i]) > 1e-12 {
			t.Errorf("the frame is %v, want %v", into, want)
			break
		}
	}
}

// A pass of another size than the frame is refused by name and adds
// nothing.
func TestAddRefusesAPassOfAnotherSize(t *testing.T) {
	layer := Layer{"floor", 1, [3]float64{1, 1, 1}}
	into := []float64{1, 1, 1}
	err := layer.Add(into, []float64{1, 1, 1, 1, 1, 1})
	if err == nil || !strings.Contains(err.Error(), "floor") ||
		!strings.Contains(err.Error(), "6 numbers") ||
		!strings.Contains(err.Error(), "3") {
		t.Errorf("a pass of 6 numbers for a frame of 3 gave %v", err)
	}
	if into[0] != 1 || into[1] != 1 || into[2] != 1 {
		t.Errorf("a refused pass changed the frame: %v", into)
	}
}

// A strobe cued on fires, cued off stays dark, and any cue above one is
// still on.
func TestStrobeFiresAsCued(t *testing.T) {
	for frame := 0; frame < 50; frame++ {
		if !StrobeFires(1, 1, frame) || !StrobeFires(2, 7, frame) {
			t.Fatalf("a strobe cued on was dark at frame %d", frame)
		}
		if StrobeFires(1, 0, frame) {
			t.Fatalf("a strobe cued off fired at frame %d", frame)
		}
	}
}

// A strobe left to its luck fires on a little over a quarter of frames,
// the same frames every time, and the two seeds do not fire together.
func TestStrobeFiresOnItsOwnLuckWhenNotCued(t *testing.T) {
	const frames = 20000
	fired, together := 0, 0
	for frame := 0; frame < frames; frame++ {
		white, magenta := StrobeFires(1, -1, frame), StrobeFires(2, -1, frame)
		if white != StrobeFires(1, -1, frame) {
			t.Fatalf("frame %d fired differently twice", frame)
		}
		if white {
			fired++
		}
		if white && magenta {
			together++
		}
	}
	if share := float64(fired) / frames; math.Abs(share-0.28) > 0.02 {
		t.Errorf("the strobe fired on %v of frames, want about 0.28", share)
	}
	// independent strobes at 0.28 coincide on 0.0784 of frames
	if share := float64(together) / frames; math.Abs(share-0.0784) > 0.02 {
		t.Errorf("both strobes fired on %v of frames, want about 0.0784",
			share)
	}
}

// The strobe clock runs twelve frames a second, rounding to the nearest,
// halves going up.
func TestStrobeFrameRoundsTwelveAFrameSecond(t *testing.T) {
	for _, each := range []struct {
		seconds float64
		frame   int
	}{{0, 0}, {0.04, 0}, {0.05, 1}, {1.0 / 24, 1}, {1, 12}, {2.5, 30}} {
		if got := StrobeFrame(each.seconds); got != each.frame {
			t.Errorf("%v seconds is frame %d, want %d", each.seconds, got,
				each.frame)
		}
	}
}

// The lattice's number is nought for the origin of seed nought, stays
// from nought up to one, repeats, and changes with each of its inputs.
func TestLatticeIsAHashFromNoughtToOne(t *testing.T) {
	if got := Lattice(0, 0, 0); got != 0 {
		t.Errorf("the lattice at the origin with seed 0 is %v, want 0", got)
	}
	seen := map[float64]bool{}
	for x := int64(-20); x < 20; x++ {
		for y := int64(-20); y < 20; y++ {
			got := Lattice(x, y, 3)
			if got < 0 || got >= 1 {
				t.Fatalf("Lattice(%d, %d) is %v, outside [0, 1)", x, y, got)
			}
			if got != Lattice(x, y, 3) {
				t.Fatalf("Lattice(%d, %d) gave two numbers", x, y)
			}
			seen[got] = true
		}
	}
	if len(seen) < 1000 {
		t.Errorf("1600 lattice points gave only %d different numbers", len(seen))
	}
	if Lattice(5, 6, 1) == Lattice(5, 6, 2) || Lattice(5, 6, 1) == Lattice(6, 5, 1) {
		t.Error("the lattice ignored its seed or the order of its points")
	}
}
