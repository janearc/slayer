package searching

import (
	"math"
	"testing"

	"github.com/janearc/slayer/beats"
)

// TestFourBeatMotionNeverTeleports checks both sides of every velocity
// change and independently prescribed motion in each beat's middle.
func TestFourBeatMotionNeverTeleports(t *testing.T) {
	edit := &Edit{Plan: Plan{BeatMotion: []float64{1, 0.45, 0.18, 0.03}}}
	for beat := 0; beat < 32; beat++ {
		at := float64(beat)
		if math.Abs(edit.motion(at+1e-7)-edit.motion(at-1e-7)) > 2.1e-7 {
			t.Fatalf("camera jumps at beat %d", beat)
		}
	}
	for index, wanted := range []float64{1, 0.45, 0.18, 0.03} {
		from, to := float64(index)+0.25, float64(index)+0.75
		if got := (edit.motion(to) - edit.motion(from)) / (to - from); math.Abs(got-wanted) > 1e-12 {
			t.Fatalf("beat %d velocity %g, wanted %g", index+1, got, wanted)
		}
	}
}

// TestLockHoldsDirectionAndRadioWaits checks story events independently
// of rendering; a moving camera must not keep the instrument searching.
func TestLockHoldsDirectionAndRadioWaits(t *testing.T) {
	times := make([]float64, 600)
	marks := make([]int, 600)
	for i := range times {
		times[i] = float64(i) * 0.64
		marks[i] = 1
	}
	edit := &Edit{Plan: Plan{FPS: 6, SearchCutBars: 4, BeatMotion: []float64{1, .45, .18, .03},
		Azimuth: 25, SearchDays: 6}, Grid: &beats.Grid{Score: beats.Score{
		Duration: 384, Meter: 4, Status: "estimated", Beats: beats.Pulse{Times: times, Measured: marks}}},
		Lock: 123, RadioOne: 154, RadioTwo: 185, Reveal: 216, Coda: 370}
	for _, at := range []float64{123, 130, 150} {
		state := edit.At(at)
		if state.Aim != 25 || state.Chapter != "lock" || state.Flash != 0 {
			t.Fatalf("lock not held: %+v", state)
		}
	}
	if edit.At(153).Site == "radio-hawaii" {
		t.Fatal("radio began before its cue")
	}
	if edit.At(161).Aim != 25 || edit.At(192).Aim != 25 {
		t.Fatal("arrays did not reach shared direction")
	}
	if edit.At(122).Aim == 25 {
		t.Fatal("search was already fixed on the target")
	}
}

// TestCueUsesIrregularSongPositions keeps a local timing change in the
// score rather than replacing it with one tempo for the whole film.
func TestCueUsesIrregularSongPositions(t *testing.T) {
	edit := &Edit{Grid: &beats.Grid{Score: beats.Score{
		Meter: 4, Beats: beats.Pulse{Times: []float64{1, 1.5, 2, 2.5, 3.2, 3.9, 4.6, 5.3}},
	}}}
	for cue, wanted := range map[string]float64{"1.1": 1, "1.4": 2.5, "2.1": 3.2, "2.4": 5.3} {
		got, err := edit.cue(cue)
		if err != nil || math.Abs(got-wanted) > 1e-12 {
			t.Fatalf("cue %s: %g, %v; wanted %g", cue, got, err, wanted)
		}
	}
	for _, cue := range []string{"0.1", "1.0", "1.5", "2", "2.3.1", "NaN.1"} {
		if _, err := edit.cue(cue); err == nil {
			t.Fatalf("accepted malformed bar.beat %q", cue)
		}
	}
}
