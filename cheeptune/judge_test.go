package cheeptune

import (
	"testing"
)

// The arithmetic DESIGN.md's "Judging a keypress" tells a game to use, as a
// test, so the document cannot drift from what the numbers do.
//
// It is here rather than in a game because the fold is the part everybody
// gets wrong, and getting it wrong turns the most forgiving judgement into
// the harshest one.

func TestTheSameWindowIsMoreOfABeatAtAFasterTempo(t *testing.T) {
	// why a window is in seconds: a player's hand does not get quicker
	// because the tune did.
	//
	// so a fixed window eats more of a beat as the beat gets shorter. at
	// 115 a beat is 522 ms and 45 either side is 0.09 of it; at 172 a
	// beat is 349 and the same 45 is 0.13. the window has not moved; the
	// beat has come to meet it.
	slow := 60 / 115.0
	fast := 60 / 172.0
	const window = 0.045 // 45 ms either side
	if window/slow >= window/fast {
		t.Fatalf("a fixed window should be a smaller share of a "+
			"slower beat: %.3f of one at 115, %.3f at 172",
			window/slow, window/fast)
	}
	if got := window / fast; got > 0.5 {
		t.Errorf("at 172 a 45 ms window is %.2f of a beat, which "+
			"would overlap the next one", got)
	}
}
