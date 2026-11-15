package cuesheet_test

import (
	"errors"
	"testing"
	"time"

	"github.com/janearc/slayer/beats"
	"github.com/janearc/slayer/cuesheet"
)

// TestMeasuredClockKeepsOriginalBars checks trimmed playback time against
// the audio's irregular grid and protects the clock from later grid edits.
func TestMeasuredClockKeepsOriginalBars(t *testing.T) {
	grid := &beats.Grid{Score: beats.Score{Duration: 4, Meter: 4,
		Status: "estimated", Beats: beats.Pulse{
			Times: []float64{1, 1.5, 2.25, 3}, Measured: []int{1, 1, 0, 1}}}}
	clock, err := cuesheet.Measured(grid, 1500*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	grid.Score.Beats.Times[2] = 99 // Editor changes cannot change this clock.
	if clock.Meter() != 4 || clock.Bars() != 1 ||
		clock.Beats(375*time.Millisecond) != 1.5 ||
		clock.Moment(2) != 750*time.Millisecond {
		t.Fatal("trim or original bar position was lost")
	}
	notes, err := clock.Notes("beat")
	if err != nil || len(notes) != 3 || notes[0].On != 0 {
		t.Fatalf("trimmed beat voice: %v, %v", notes, err)
	}
	// anti-test of TestMeasuredClockKeepsOriginalBars: no invented bass notes.
	if _, err := clock.Notes("bass"); !errors.Is(err, cuesheet.ErrVoice) {
		t.Fatalf("invented an instrument: %v", err)
	}
	if _, err := cuesheet.Measured(nil, 0); err == nil {
		t.Fatal("accepted absent grid")
	}
}
