package job

import (
	"math"
	"testing"
	"time"
)

func TestFrameTimeMatchesTheWholeClock(t *testing.T) {
	for _, example := range []struct {
		name, text string
	}{
		{"still at zero", still},
		{"still at twenty seconds", still + "at: {value: 20, unit: s}\n"},
		{"film", film + "start: {value: 40, unit: s}\n"},
		{"stretched film", still + "fps: 6\nduration: {value: 30, unit: s}\n" +
			"start: {value: 45, unit: s}\nscene-duration: {value: 3600, unit: s}\n"},
		{"short film", still + "fps: 1\nduration: {value: 0.001, unit: s}\n"},
	} {
		t.Run(example.name, func(t *testing.T) {
			clock := read(t, example.text).Clock()
			for index, want := range clock.Times() {
				got, ok := clock.FrameTime(index)
				if !ok || got != want {
					t.Fatalf("frame %d: %s (%v), want %s", index, got, ok, want)
				}
			}
			for _, index := range []int{-1, clock.Frames(), math.MaxInt} {
				if got, ok := clock.FrameTime(index); ok || got != 0 {
					t.Errorf("invalid frame %d: %s (%v)", index, got, ok)
				}
			}
		})
	}
}

func TestFrameTimeDoesNotMaterializeALongFilm(t *testing.T) {
	clock := read(t, still+"fps: 120\nduration: {value: 3600, unit: s}\n"+
		"start: {value: 45, unit: s}\nscene-duration: {value: 86400, unit: s}\n").Clock()
	index := clock.Frames() - 1
	got, ok := clock.FrameTime(index)
	want := 45*time.Second + time.Duration(index)*(24*time.Hour/time.Duration(clock.Frames()))
	if !ok || got != want {
		t.Fatalf("frame %d: %s (%v), want %s", index, got, ok, want)
	}
	if got, ok := (Clock{}).FrameTime(0); ok || got != 0 {
		t.Fatalf("unchecked zero clock: %s (%v)", got, ok)
	}
}
