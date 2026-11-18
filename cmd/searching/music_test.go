package main

import (
	"github.com/janearc/slayer/cheeptune"
	"testing"
)

// TestCompositionClockPreservesSilentBarsAndTempo checks independent
// expected seconds across a tempo change, without synthesizing a long song.
func TestCompositionClockPreservesSilentBarsAndTempo(t *testing.T) {
	score, err := cheeptune.ParseScore([]byte(`{"name":"clock","tempo":120,"meter":4,"tempos":[{"bar":2,"tempo":60}],"parts":[{"role":"lead","bars":["-:4","c4:4"]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	grid, err := compositionGrid(score, 6, "audio.wav", "hash")
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []float64{0, .5, 1, 1.5, 2, 3, 4, 5} {
		if grid.Score.Beats.Times[i] != want || grid.Score.Beats.Measured[i] != 0 {
			t.Fatalf("beat %d: %v", i, grid.Score.Beats)
		}
	}
	if grid.Score.Status != "estimated" {
		t.Fatal("claimed editorial approval")
	}
	if _, err := compositionGrid(score, 4, "audio.wav", "hash"); err == nil {
		t.Fatal("accepted audio shorter than its grid")
	}
}
