package show

import (
	"strings"
	"testing"
)

// TestACueSetsItsFadedAndPlainNumbers checks the levels and numbers a cue names reach the lines of a moment, and the ones it does not name stay where they were.
func TestACueSetsItsFadedAndPlainNumbers(t *testing.T) {
	sheet, err := parseSheet([]byte(`cues:
  - {at: "1.1", rig: 1, mist: 0.5, exposure: 1.5, spots: 0.25, spot_bars: 2,
     search: 0.4, search_bars: 4, banks_bars: 2, sweep_bars: 1, decay: 0.5, split: true,
     pulse: "x..............."}`), testScore(t))
	if err != nil {
		t.Fatal(err)
	}
	// Beat 2 of the grid, position 2, falls at 3.0 s.
	moment, _ := sheet.At(3.0, 24)
	lines := moment.Lines()
	for _, want := range []string{
		"mist=0.500\n", "exposure=1.500\n", "spots=0.250\n", "search=0.400\n",
		// spot_bars 2: phase 2/4/2 = 0.25, and the aim is 2 * (1 - |2*0.25 - 1|) = 1
		"spot_aim=1.0000\n",
		// search_bars 4: phase 2/4/4 = 0.125
		"search_phase=0.1250\n",
		// sweep_bars 1: 2/4/1 = 0.5 of the way round
		"sweep=0.5000\n",
	} {
		if !strings.Contains(lines, want) {
			t.Errorf("lines lack %q:\n%s", want, lines)
		}
	}
	plain, _ := sheet.At(2.0, 24)
	if !strings.Contains(plain.Lines(), "parity=0\n") {
		t.Errorf("a split pulse on its first hit should be parity 0:\n%s", plain.Lines())
	}
	before, _ := sheet.At(1.0, 24)
	if !strings.Contains(before.Lines(), "mist=0.120\n") {
		t.Errorf("before the cue the mist is not its default 0.12:\n%s", before.Lines())
	}
}

// TestACueSetsOverheadAndWithoutSplitHasNoParity checks the overhead fades to the cue's level, and a pulse that is not split has no parity.
func TestACueSetsOverheadAndWithoutSplitHasNoParity(t *testing.T) {
	sheet, err := parseSheet([]byte(`cues:
  - {at: "1.1", overhead: 0.75, pulse: "x..............."}`), testScore(t))
	if err != nil {
		t.Fatal(err)
	}
	moment, _ := sheet.At(3.0, 24)
	if got := moment.Overhead(); got != 0.75 {
		t.Errorf("overhead is %v, want 0.75", got)
	}
	if !strings.Contains(moment.Lines(), "parity=-1\n") {
		t.Errorf("a pulse that is not split has a parity:\n%s", moment.Lines())
	}
}

// TestLinesListTheBurstsStillShowing checks the fire bursts are written as the seconds since each began, newest first.
func TestLinesListTheBurstsStillShowing(t *testing.T) {
	sheet, err := parseSheet([]byte(`cues:
  - {at: "1.1", fire: 1}`), testScore(t))
	if err != nil {
		t.Fatal(err)
	}
	// Position 5 is at 4.5 s. The fire is cued at 2.0 s, so it bursts at
	// positions 0 and 4 (2.0 s and 4.0 s): 0.5 s and 2.5 s ago.
	moment, _ := sheet.At(4.5, 24)
	lines := moment.Lines()
	for _, want := range []string{"fire_since=0.5000,2.5000\n", "fire_lit=2.5000\n"} {
		if !strings.Contains(lines, want) {
			t.Errorf("lines lack %q:\n%s", want, lines)
		}
	}
}

// TestGridHitsBeforeTheGridAreNone checks a position before the grid has no flood hits.
func TestGridHitsBeforeTheGridAreNone(t *testing.T) {
	if got := gridHits(4, 3, -1, steady(40)); got != nil {
		t.Fatalf("hits before the grid: %v", got)
	}
}

// TestParseScoreMeterAndStatus checks a score of another meter keeps
// it, an unstated meter is four, and the status and meter are reported.
func TestParseScoreMeterAndStatus(t *testing.T) {
	three, err := parseScore([]byte("score:\n  meter: 3\n  beats:\n    times: [1, 2]\n"))
	if err != nil || three.Meter() != 3 {
		t.Fatalf("meter 3: %v", err)
	}
	score, err := parseScore([]byte("score:\n  status: estimated\n  beats:\n    times: [1, 2]\n"))
	if err != nil {
		t.Fatal(err)
	}
	if score.Meter() != 4 || !score.Estimated() {
		t.Errorf("meter %d, estimated %v; want 4 and true", score.Meter(), score.Estimated())
	}
	score, err = parseScore([]byte("score:\n  status: verified\n  meter: 4\n  beats:\n    times: [1, 2]\n"))
	if err != nil || score.Estimated() {
		t.Errorf("verified score: estimated=%v err=%v", err == nil && score.Estimated(), err)
	}
}
