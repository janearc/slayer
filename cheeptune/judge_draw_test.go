//go:build draw

package cheeptune

import (
	"math"
	"testing"
	"time"
)

// missedBy is how far a press was from the nearest beat, in seconds, the way
// the design doc says to work it out.
func missedBy(p *Piece, at time.Duration) (float64, bool) {
	beat, _, _, _, ok := p.At(at)
	if !ok {
		return 0, false
	}
	off := beat
	if off > 0.5 {
		off--
	}
	return math.Abs(off) * p.Score.BeatSeconds(), true
}

func TestAPressOnTheBeatMissesByNothing(t *testing.T) {
	p := render(t,
		score(4, true, `{"role": "lead", "bars": ["c4:4"]}`),
		voicing(lead, ""))
	// 120 a minute: beats at 0, 0.5, 1.0 seconds
	for _, at := range []time.Duration{0, 500 * time.Millisecond,
		1500 * time.Millisecond} {
		got, ok := missedBy(p, at)
		if !ok || got > 1e-9 {
			t.Errorf("a press at %v missed by %v", at, got)
		}
	}
}

func TestAPressJustEarlyIsJustEarly(t *testing.T) {
	// the fold: 20 ms before a beat is 20 ms out, not 480. without it
	// every early press reads as the latest possible press.
	p := render(t,
		score(4, true, `{"role": "lead", "bars": ["c4:4"]}`),
		voicing(lead, ""))
	got, ok := missedBy(p, 980*time.Millisecond)
	if !ok {
		t.Fatal("the piece was not playing")
	}
	if math.Abs(got-0.02) > 1e-6 {
		t.Errorf("20 ms early came back as %v s", got)
	}
}

func TestAPressJustLateIsJustLate(t *testing.T) {
	p := render(t,
		score(4, true, `{"role": "lead", "bars": ["c4:4"]}`),
		voicing(lead, ""))
	got, _ := missedBy(p, 1020*time.Millisecond)
	if math.Abs(got-0.02) > 1e-6 {
		t.Errorf("20 ms late came back as %v s", got)
	}
}
