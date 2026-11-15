package cheeptune

import (
	"math"
	"strings"
	"testing"
)

// marked is a one-bar score of one lead bar at sixty a minute.
func marked(bar string) []byte {
	return slow(`{"role": "lead", "bars": ["` + bar + `"]}`)
}

func TestALoudnessMarkIsReadOnANoteAChordAndAHit(t *testing.T) {
	s, err := ParseScore(slow(
		`{"role": "lead", "bars": ["c4@p (c4 e4):2@ff c4"]}`,
		`{"role": "perc", "bars": ["k@f k:1/2@pp k:1/2 -:2"]}`))
	if err != nil {
		t.Fatal(err)
	}
	lead, perc := s.Parts[0].Events(), s.Parts[1].Events()
	for i, want := range []float64{0.4, 1.58, 1} {
		if lead[i].Loud != want {
			t.Errorf("lead event %d is at %v, want %v", i+1,
				lead[i].Loud, want)
		}
	}
	if perc[0].Loud != 1.26 || perc[1].Loud != 0.25 || perc[2].Loud != 1 {
		t.Errorf("the hits are at %v %v %v", perc[0].Loud, perc[1].Loud,
			perc[2].Loud)
	}
	// a mark sits before the tie, and the tied note is one event.
	s, err = ParseScore(slow(
		`{"role": "lead", "bars": ["c4:4@p~", "c4:4"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if ev := s.Parts[0].Events(); len(ev) != 1 || ev[0].Loud != 0.4 ||
		ev[0].Length != 8 {
		t.Errorf("a marked tied note is %+v", ev)
	}
}

func TestALoudnessMarkThatIsNotOneIsRefusedByName(t *testing.T) {
	for bar, fragment := range map[string]string{
		"c4:4@loud":   `loudness mark "loud"; marks are pp, p, mp, mf, f, ff`,
		"c4:4@":       `loudness mark ""`,
		"-:4@p":       "a rest has no loudness to mark",
		"c4:2 -:2@ff": "a rest has no loudness to mark",
	} {
		_, err := ParseScore(marked(bar))
		wantErr(t, err, fragment)
	}
	_, err := ParseScore(slow(
		`{"role": "lead", "bars": ["c4:4~", "c4:4@f"]}`))
	wantErr(t, err, "a tied note keeps the loudness it began with")
	// the near miss: an unmarked landing is fine.
	if _, err := ParseScore(slow(
		`{"role": "lead", "bars": ["c4:4@f~", "c4:4"]}`)); err != nil {
		t.Errorf("a tie from a marked note: %v", err)
	}
}

// turning is a score of two bars at sixty a minute and two at twice
// that, with a hit at the start of each bar.
func turning(tempos string) []byte {
	return []byte(`{"name": "t", "tempo": 60, "meter": 4, "loop": false,
		"tempos": ` + tempos + `, "parts": [
		{"role": "perc", "bars": ["k -:3", "k -:3", "k -:3", "k -:3"]},
		{"role": "lead", "bars": ["-:4", "-:4", "a4:2 -:2", "-:4"]}]}`)
}

const doubled = `[{"bar": 3, "tempo": 120}]`

func TestATempoChangeMovesEverySecondAfterIt(t *testing.T) {
	s, err := ParseScore(turning(doubled))
	if err != nil {
		t.Fatal(err)
	}
	// two bars of four seconds, then two of two.
	for beats, want := range map[float64]float64{0: 0, 4: 4, 8: 8, 9: 8.5,
		12: 10, 16: 12} {
		if got := s.SecondsAt(beats); math.Abs(got-want) > 1e-9 {
			t.Errorf("beat %v is at %v s, want %v", beats, got, want)
		}
		if got := s.BeatsAt(want); math.Abs(got-beats) > 1e-9 {
			t.Errorf("second %v is beat %v, want %v", want, got, beats)
		}
	}
	if s.Seconds() != 12 {
		t.Errorf("the piece is %v s, want 12", s.Seconds())
	}
	// the anti-test: without the list the same bars are sixteen
	// seconds and every beat is a second.
	s, err = ParseScore(turning(`[]`))
	if err != nil {
		t.Fatal(err)
	}
	if s.Seconds() != 16 || s.SecondsAt(9) != 9 || s.BeatsAt(9) != 9 {
		t.Errorf("with no changes the piece is %v s", s.Seconds())
	}
}

func TestATempoChangeOutOfPlaceIsRefusedByName(t *testing.T) {
	for tempos, fragment := range map[string]string{
		`[{"bar": 1, "tempo": 90}]`:  "bar 1 is not after bar 1",
		`[{"bar": 9, "tempo": 90}]`:  "bar 9 is not in the score",
		`[{"bar": 2, "tempo": 900}]`: "tempo 900 is outside 40 to 300",
		`[{"bar": 3, "tempo": 90}, {"bar": 2, "tempo": 80}]`: "bar 2 is " +
			"not after bar 3",
	} {
		_, err := ParseScore(turning(tempos))
		wantErr(t, err, fragment)
	}
}

func TestALoopWithATempoChangeGoesRoundInBeatsAndSeconds(t *testing.T) {
	s, err := ParseScore([]byte(strings.Replace(string(turning(doubled)),
		`"loop": false`, `"loop": true`, 1)))
	if err != nil {
		t.Fatal(err)
	}
	if got := s.SecondsAt(17); got != 13 {
		t.Errorf("a beat into the second pass is at %v s, want 13", got)
	}
	if got := s.BeatsAt(20.5); got != 25 {
		t.Errorf("20.5 s is beat %v, want 25", got)
	}
}

func TestCheckSaysWhereTheTempoChanges(t *testing.T) {
	s, err := ParseScore(turning(doubled))
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	describeScore(&out, s)
	if !strings.Contains(out.String(), "from bar 3, 120 bpm") {
		t.Errorf("check does not say the change:\n%s", out.String())
	}
}
