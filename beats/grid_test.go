package beats_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/beats"
)

// clicks is an authored recording: a drum every half second after an intro,
// with opposite stereo polarity and a two-second break in the middle.
func clicks() []float32 {
	const rate, duration = 1000, 12
	samples := make([]float32, rate*duration*2)
	for tick := 2; tick < 24; tick++ {
		if tick >= 10 && tick < 14 {
			continue
		}
		for offset := 0; offset < 15; offset++ {
			index := tick*500 + offset
			samples[index*2] = 0.8
			samples[index*2+1] = -0.8
		}
	}
	return samples
}

// TestPulseTimes checks an independent 120 BPM click schedule, including
// opposite-phase stereo which must not cancel in onset analysis.
func TestPulseTimes(t *testing.T) {
	grid, err := beats.Detect(clicks(), 1000,
		beats.Settings{Tempo: 120, Meter: 4, First: -1})
	if err != nil {
		t.Fatal(err)
	}
	if grid.Score.Status != "estimated" {
		t.Fatal("analysis certified itself")
	}
	times := grid.Score.Beats.Times
	if len(times) < 20 {
		t.Fatalf("lost pulses across break: %v", times)
	}
	for index, at := range times {
		if math.Abs(at-(1+float64(index)*0.5)) > 0.011 {
			t.Fatalf("beat %d at %f, expected %f", index, at,
				1+float64(index)*0.5)
		}
		if at >= 5 && at < 7 && grid.Score.Beats.Measured[index] != 0 {
			t.Fatal("claimed measured beat in silent break")
		}
	}
	// anti-test of TestPulseTimes: silent audio has no rhythm to certify.
	if _, err := beats.Detect(make([]float32, 24000), 1000,
		beats.Settings{Tempo: 120, Meter: 4, First: -1}); err == nil {
		t.Fatal("silence became a beat grid")
	}
}

// TestIrregularInterpolation checks published times, rather than deriving
// expectations from the clock's implementation or assuming constant BPM.
func TestIrregularInterpolation(t *testing.T) {
	grid := &beats.Grid{Score: beats.Score{Duration: 4, Meter: 4,
		Status: "estimated", Beats: beats.Pulse{
			Times: []float64{1, 1.5, 2.25, 3}, Measured: []int{1, 1, 0, 1}}}}
	if err := grid.Check(); err != nil {
		t.Fatal(err)
	}
	if grid.BeatAt(1.875) != 1.5 || grid.SecondAt(2.5) != 2.625 ||
		grid.BeatAt(0) != -2 {
		t.Fatal("irregular grid was replaced with a fixed tempo")
	}
	// anti-test of TestIrregularInterpolation: duplicates cannot be inverted.
	grid.Score.Beats.Times[2] = 1.5
	if grid.Check() == nil {
		t.Fatal("duplicate beat accepted")
	}
}
