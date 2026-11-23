//go:build draw

package cheeptune

import (
	"math"
	"testing"
	"time"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestThePlayheadCountsBeatsAndBars(t *testing.T) {
	// 120 a minute is a beat every half second.
	p := render(t,
		score(4, false, `{"role": "lead", "bars": ["c4:4", "c4:4"]}`),
		voicing(lead, ""))
	beat, bar, beats, _, ok := p.At(1250 * time.Millisecond)
	if !ok || !near(beats, 2.5) || !near(beat, 0.5) || !near(bar, 0.625) {
		t.Errorf("at 1.25 s: beat %v bar %v beats %v ok %v; "+
			"want 0.5 0.625 2.5 true", beat, bar, beats, ok)
	}
	_, bar, beats, _, _ = p.At(2500 * time.Millisecond)
	if !near(beats, 5) || !near(bar, 0.25) {
		t.Errorf("at 2.5 s, into the second bar: bar %v beats %v; "+
			"want 0.25 5", bar, beats)
	}
}

func TestOutsideThePieceEverythingIsNought(t *testing.T) {
	p := render(t,
		score(4, false, `{"role": "lead", "bars": ["c4:4"]}`),
		voicing(lead, ""))
	for _, off := range []time.Duration{-time.Millisecond, 10 * time.Second} {
		beat, bar, beats, level, ok := p.At(off)
		if ok || beat != 0 || bar != 0 || beats != 0 || level != 0 {
			t.Errorf("at %v: %v %v %v %v %v; want all nought and not ok",
				off, beat, bar, beats, level, ok)
		}
	}
}

func TestALoopKeepsCountingPastItsSeam(t *testing.T) {
	p := render(t,
		score(4, true, `{"role": "lead", "bars": ["c4:4"]}`),
		voicing(lead, ""))
	// the piece is two seconds. seven and a quarter is three loops in.
	beat, bar, beats, level, ok := p.At(7250 * time.Millisecond)
	if !ok || !near(beats, 14.5) || !near(beat, 0.5) ||
		!near(bar, 0.625) {
		t.Errorf("at 7.25 s: beat %v bar %v beats %v ok %v; "+
			"want 0.5 0.625 14.5 true", beat, bar, beats, ok)
	}
	if level < 0.5 {
		t.Errorf("level %.3f three loops in, where the note sounds", level)
	}
}

func TestLevelIsCentredOnTheOffset(t *testing.T) {
	// silence, then a note on beat three at 1 s. the window is 50 ms
	// wide and centred, so it hears the note 25 ms before it starts and
	// not 35 ms before. a window that trailed would hear neither.
	p := render(t,
		score(4, false, `{"role": "lead", "bars": ["-:2 c5:2"]}`),
		voicing(lead, ""))
	// the note starts on the first driver tick at or past one second.
	start := math.Ceil(1*tickRate) / tickRate
	at := func(s float64) float64 {
		_, _, _, l, _ := p.At(time.Duration(s * float64(time.Second)))
		return l
	}
	if l := at(start - 0.035); l != 0 {
		t.Errorf("35 ms before the note, level %.3f; want nought", l)
	}
	if l := at(start - 0.015); l == 0 {
		t.Errorf("15 ms before the note, level is nought; a centred " +
			"window hears ahead")
	}
	if l := at(start + 0.5); l < 0.9 {
		t.Errorf("inside the note, level %.3f; want near one", l)
	}
}
