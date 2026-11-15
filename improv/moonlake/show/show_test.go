package show

import (
	"errors"
	"math"
	"strings"
	"testing"
)

// testScore and testSheet are cueshow's test score and sheet, read.
func testScore(t *testing.T) *Score {
	t.Helper()
	score, err := ReadScore("testdata/score.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return score
}

// testSheet is testdata's sheet read against its score.
func testSheet(t *testing.T) *Sheet {
	t.Helper()
	sheet, err := ReadSheet("testdata/cues.yaml", testScore(t))
	if err != nil {
		t.Fatal(err)
	}
	return sheet
}

// the beat lit at a moment is the score's: on a beat it is that beat, a
// moment between two beats is in the earlier, and the bar and the place
// in it follow from its index.
func TestTheBeatAtAMomentIsTheScores(t *testing.T) {
	score := testScore(t)
	for _, each := range []struct {
		seconds          float64
		index, bar, beat int
		since            float64
	}{
		{2.0, 0, 1, 1, 0},
		{2.49, 0, 1, 1, 0.49},
		{2.5, 1, 1, 2, 0},
		{4.25, 4, 2, 1, 0.25},
		{9.75, 15, 4, 4, 0.25},
	} {
		beat := score.BeatAt(each.seconds)
		if beat.Index() != each.index || beat.Bar() != each.bar ||
			beat.InBar() != each.beat ||
			!(math.Abs(beat.Since()-each.since) <= near) {
			t.Errorf("at %v: beat %d, bar %d.%d, %v since; want %d, %d.%d, %v",
				each.seconds, beat.Index(), beat.Bar(), beat.InBar(),
				beat.Since(), each.index, each.bar, each.beat, each.since)
		}
	}
	if beat := score.BeatAt(4.25); !beat.Downbeat() || beat.Struck() != 4.0 {
		t.Errorf("4.25 s: downbeat %v, struck %v", beat.Downbeat(),
			beat.Struck())
	}
}

// the anti-test: before the grid no beat is lit, and a beat just before
// the next is still the earlier, not the next.
func TestBeforeTheGridNoBeatIsLit(t *testing.T) {
	score := testScore(t)
	before := score.BeatAt(1.99)
	if before.On() || before.Bar() != 0 || before.InBar() != 0 ||
		before.Downbeat() {
		t.Errorf("before the grid: on %v, bar %d.%d", before.On(),
			before.Bar(), before.InBar())
	}
	if beat := score.BeatAt(2.4999); beat.InBar() != 1 {
		t.Errorf("just before beat two the beat is %d", beat.InBar())
	}
}

// a carried beat is marked as carried: measured says which beats the
// drums played.
func TestACarriedBeatIsMarkedCarried(t *testing.T) {
	score, err := ParseScore([]byte(`score:
  beats:
    times: [1, 2, 3, 4]
    measured: [1, 0, 0, 1]`))
	if err != nil {
		t.Fatal(err)
	}
	for seconds, want := range map[float64]bool{1: true, 2.5: false,
		3: false, 4.5: true} {
		if got := score.BeatAt(seconds).Measured(); got != want {
			t.Errorf("at %v measured is %v, not %v", seconds, got, want)
		}
	}
}

// the anti-test: a score with no measured list carries every beat, and
// a grid that is too short or does not rise is refused by name.
func TestAScoreWithoutAGridIsRefused(t *testing.T) {
	score, err := ParseScore([]byte("score: {beats: {times: [1, 2]}}"))
	if err != nil || score.BeatAt(1).Measured() {
		t.Errorf("no measured list: %v, measured %v", err,
			score.BeatAt(1).Measured())
	}
	for raw, says := range map[string]string{
		"score: {beats: {times: [1]}}":    "two or more",
		"score: {beats: {times: [2, 1]}}": "do not rise",
		"score: [":                        "the score",
	} {
		if _, err := ParseScore([]byte(raw)); err == nil ||
			!errors.Is(err, ErrShow) || !strings.Contains(err.Error(), says) {
			t.Errorf("%q gave %v, not a refusal saying %q", raw, err, says)
		}
	}
	if _, err := ReadScore("testdata/none.yaml"); err == nil {
		t.Error("a missing score was read")
	}
}

// a score's notes come back with their start, end and loudness, and a
// track it lacks is empty.
func TestAScoreKeepsItsNotes(t *testing.T) {
	score, err := ParseScore([]byte(`score:
  tracks:
    chant:
      - {at_sec: 1.5, dur: 0.25, vel: 70}
      - {sec: 3, dur: 1, vel: 50}
  duration: 9
  beats: {times: [1, 2]}`))
	if err != nil {
		t.Fatal(err)
	}
	notes := score.Track("chant")
	if len(notes) != 2 || notes[0].Start() != 1.5 || notes[0].End() != 1.75 ||
		notes[0].Vel() != 70 || notes[1].Start() != 3 {
		t.Errorf("the chant reads back as %+v", notes)
	}
	if score.Duration() != 9 || len(score.Beats()) != 2 {
		t.Errorf("duration %v, beats %v", score.Duration(), score.Beats())
	}
	if lead := score.Track("lead"); len(lead) != 0 {
		t.Errorf("a track the score lacks has %d notes", len(lead))
	}
}

// a cue fires at its bar and not before: the state after is the sheet's,
// and before it is what came earlier.
func TestACueFiresAtItsBarAndNotBefore(t *testing.T) {
	sheet := testSheet(t)
	before, err := sheet.At(2.99, 24)
	if err != nil {
		t.Fatal(err)
	}
	after, err := sheet.At(3.0, 24)
	if err != nil {
		t.Fatal(err)
	}
	// 1.1 sets rig 0.5, beams 4; 2.1 (4 s) fades the rig to 1
	if before.Rig() != 0.5 || before.Beams() != 4 {
		t.Errorf("in bar one: rig %v, beams %d", before.Rig(), before.Beams())
	}
	// 3.1 at 6 s sets moon 1 and its pulse; before it the moon is 0.2
	if moon, _ := sheet.At(5.99, 24); !(math.Abs(moon.Moon()-0.2) <= near) {
		t.Errorf("before its cue the moon is %v, not 0.2", moon.Moon())
	}
	if moon, _ := sheet.At(6.0, 24); !(math.Abs(moon.Moon()-1) <= near) {
		t.Errorf("on its cue the moon is %v, not 1", moon.Moon())
	}
	if after.Overhead() != 0 || after.Search() != 0 || after.StrobesOn() ||
		after.FloodsOn() {
		t.Errorf("parts no cue set moved: %v", after.Lines())
	}
}

// the anti-test: a sheet with a cue off the grid, or naming a figure
// lightshow does not play, is refused when it is read, by name.
func TestASheetOffTheGridIsRefusedByName(t *testing.T) {
	score := testScore(t)
	for raw, says := range map[string]string{
		`cues: [{at: "9.1", rig: 1}]`:               `"9.1"`,
		`cues: [{at: "1.1", figures: [star]}]`:      `"star"`,
		`cues: [{at: "1.1", banks: [apart]}]`:       `"apart"`,
		`cues: [{at: "1.1", search_moves: [spin]}]`: `"spin"`,
		`cues: [`: "the cue sheet",
	} {
		if _, err := ParseSheet([]byte(raw), score); err == nil ||
			!strings.Contains(err.Error(), says) {
			t.Errorf("%s gave %v, not a refusal naming %s", raw, err, says)
		}
	}
	if _, err := ParseSheet([]byte("cues: []"), nil); err == nil {
		t.Error("a sheet with no score was read")
	}
	if _, err := ReadSheet("testdata/none.yaml", score); err == nil {
		t.Error("a missing sheet was read")
	}
	if _, err := testSheet(t).At(1, 0); err == nil {
		t.Error("a frame rate of nought was taken")
	}
}

// every cue is fired at its moment with what it sets, as the metronome
// shows it, and says which parts of the rig it sets.
func TestEachCueIsFiredWithItsWords(t *testing.T) {
	fired := testSheet(t).Fired()
	if len(fired) != 7 {
		t.Fatalf("%d cues fired, not the sheet's 7", len(fired))
	}
	first := fired[2]
	if first.At() != 2.0 || first.Words() != "1.1  RIG 0.5  4 BEAMS  GREEN RED" ||
		first.Kill() || !first.Sets(PartLasers) {
		t.Errorf("1.1 fired at %v as %q, kill %v", first.At(), first.Words(),
			first.Kill())
	}
	if fired[0].Words() != "0.0s  " || !fired[0].Sets(PartMaster) ||
		!fired[0].Sets(PartMoon) {
		t.Errorf("the first cue fired as %q", fired[0].Words())
	}
}

// the anti-test: a cue sets only the parts it names; a kill is a rig of
// nought and nothing else.
func TestACueSetsOnlyWhatItNames(t *testing.T) {
	score := testScore(t)
	sheet, err := ParseSheet([]byte(`cues:
  - {at: "1.1", rig: 0, strobes: lead, mist: 0.5, sweep_bars: 2}
  - {at: "1.2", rig: 0.3}`), score)
	if err != nil {
		t.Fatal(err)
	}
	fired := sheet.Fired()
	if !fired[0].Kill() || fired[1].Kill() {
		t.Errorf("kills: %v, %v", fired[0].Kill(), fired[1].Kill())
	}
	if fired[0].Sets(PartMoon) || fired[0].Sets(PartOverhead) ||
		!fired[0].Sets(PartStrobes) {
		t.Errorf("1.1 sets the moon %v, the beam %v, the strobes %v",
			fired[0].Sets(PartMoon), fired[0].Sets(PartOverhead),
			fired[0].Sets(PartStrobes))
	}
	if words := fired[0].Words(); words !=
		"1.1  KILL  STROBES LEAD  SWEEP 2  MIST 0.5" {
		t.Errorf("1.1 says %q", words)
	}
}

// the lights a frame shows: the strobes fire on a loud lead note in the
// frame, the lasers with their pulse, the searchlights struck on the beat.
func TestTheLevelsOfAFrame(t *testing.T) {
	score := testScore(t)
	sheet, err := ParseSheet([]byte(`cues:
  - {at: "1.1", rig: 1, strobes: lead, hot: 80, hotter: 85, floods: ones,
     search: 0.5, search_beat: 1, pulse: "x...............", decay: 1}`),
		score)
	if err != nil {
		t.Fatal(err)
	}
	on, _ := sheet.At(2.0, 24)
	if !on.Strobe() || !on.StrobesOn() || on.Lasers() != 1 ||
		on.Floods() != 1 || on.Searchlights() != 0.5 || !on.FloodsOn() {
		t.Errorf("on the first beat: %v", on.Lines())
	}
	off, _ := sheet.At(2.25, 24)
	if off.Strobe() || off.Lasers() >= 0.5 || off.Searchlights() >= 0.5 {
		t.Errorf("half a beat on: strobe %v, lasers %v, search %v",
			off.Strobe(), off.Lasers(), off.Searchlights())
	}
	// the lead's 80 at 2.75 s strikes white; its 40 at 3.5 s strikes none
	if quiet, _ := sheet.At(3.5, 24); quiet.Strobe() {
		t.Error("a quiet lead note fired a strobe")
	}
	if loud, _ := sheet.At(2.74, 24); !loud.Strobe() {
		t.Error("a loud lead note inside the frame fired no strobe")
	}
}

// the anti-test: a moon behind the rim shows nothing, however bright its
// cue sets it.
func TestAMoonBehindTheRimIsDark(t *testing.T) {
	sheet, err := ParseSheet([]byte(`cues: [{at_sec: 0, moon: 1, moon_alt: 0}]`),
		testScore(t))
	if err != nil {
		t.Fatal(err)
	}
	if hidden, _ := sheet.At(1, 24); hidden.Moon() != 0 {
		t.Errorf("behind the rim the moon is %v", hidden.Moon())
	}
	if sheet.Score() == nil {
		t.Error("the sheet lost its score")
	}
}

// the strobes on the grid burn the first share of each subdivision,
// white on the evens and magenta on the odds; the lattice is
// lightshow's own; before the grid a grid strobe is dark.
func TestGridStrobesBurnTheirDuty(t *testing.T) {
	score := testScore(t)
	for _, each := range []struct {
		mode           string
		seconds        float64
		white, magenta int
	}{
		{"grid16", 2.0, 1, 0},
		{"grid16", 2.125, 0, 1},
		{"grid16", 2.1, 0, 0},
		{"grid8", 2.25, 0, 1},
		{"grid8", 1.0, 0, 0},
		{"lattice", 2.0, -1, -1},
		{"off", 2.0, 0, 0},
	} {
		sheet, err := ParseSheet([]byte(`cues: [{at_sec: 0, strobes: `+
			each.mode+`, duty: 0.3}]`), score)
		if err != nil {
			t.Fatal(err)
		}
		at, _ := sheet.At(each.seconds, 24)
		if at.white != each.white || at.magenta != each.magenta {
			t.Errorf("%s at %v: white %d magenta %d", each.mode,
				each.seconds, at.white, at.magenta)
		}
	}
}

// a moment says itself as cueshow's lines: every key, the lists taken in
// turn from their cues, and the bar, or intro before the grid.
func TestAMomentSaysItselfAsCueshowsLines(t *testing.T) {
	sheet, err := ParseSheet([]byte(`cues:
  - {at_sec: 0, master: 0.5, floods: lead, hot: 70}
  - {at: "2.1", figures: [scan, beam], figure_bars: 1, banks: [offset],
     search_moves: [chase, fan], search_move_bars: 1}`), testScore(t))
	if err != nil {
		t.Fatal(err)
	}
	early, _ := sheet.At(1, 24)
	if !strings.Contains(early.Lines(), "bar=intro\n") ||
		early.Master() != 0.5 {
		t.Errorf("before the grid:\n%s", early.Lines())
	}
	// the lead's 90 at 2.0 s fires the left flood, 80 at 2.75 the right
	if flood, _ := sheet.At(2.0, 24); flood.Floods() != 1 {
		t.Errorf("on a loud lead note the floods are %v", flood.Floods())
	}
	late, _ := sheet.At(6.0, 24)
	lines := late.Lines()
	for _, want := range []string{"figure=beam\n", "banks=offset\n",
		"search_move=fan\n", "bar=3.1\n", "flood_left=", "moon_alt="} {
		if !strings.Contains(lines, want) {
			t.Errorf("at 6 s the lines lack %q:\n%s", want, lines)
		}
	}
}

// the sheet's palettes are every colour its cues name, once each, and
// the beams take the beat's colour from the palette.
func TestThePalettesAreTheSheets(t *testing.T) {
	sheet := testSheet(t)
	if got := sheet.Palettes(); len(got) != 2 || got[0] != "green" ||
		got[1] != "red" {
		t.Errorf("palettes %v, not green and red", got)
	}
	red, ok := PaletteColour("red")
	if !ok || red != paletteColours["red"] {
		t.Errorf("red is %v (%v)", red, ok)
	}
	if _, ok := PaletteColour("mauve"); ok {
		t.Error("mauve is a palette colour")
	}
	// 1.1 sets [green, red]: beat two of bar one is red
	if at, _ := sheet.At(2.5, 24); at.Colour() != red {
		t.Errorf("on beat two the beams are %v, not red", at.Colour())
	}
}
