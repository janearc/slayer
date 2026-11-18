package show

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
)

// mothTempo, mothMeter and mothBars are Moth Light's, as its score says
// them: "tempo": 63, "meter": 3, sixteen bars in each part, in
// renders/animations/2026-11-12-fable-music-library/moth-light.json.
const (
	mothTempo = 63.0
	mothMeter = 3
	mothBars  = 16
)

// mothSeconds is Moth Light's length, its cut sheet's "seconds": 45.71.
const mothSeconds = 45.71

// secondsPerMinute turns a tempo in beats a minute into a beat's length.
const secondsPerMinute = 60.0

// cutSheetPlaces is the second each part of Moth Light's cut sheet
// starts at, by bar: flutter at bar 7, 17.14 s, and thin at bar 13,
// 34.29 s. The sheet writes them to the hundredth.
var cutSheetPlaces = map[int]float64{1: 0, 7: 17.14, 13: 34.29}

// cutSheetRounding is the most a time written to the hundredth is off.
const cutSheetRounding = 0.005

// mothScore is Moth Light's grid as its score writes it, not as audio
// measures it: a beat every 60/63 s from nought, three to a bar, for
// sixteen bars.
func mothScore(t *testing.T) *Score {
	t.Helper()
	var times []string
	for beat := range mothMeter * mothBars {
		times = append(times, fmt.Sprintf("%.6f",
			float64(beat)*secondsPerMinute/mothTempo))
	}
	score, err := ParseScore(fmt.Appendf(nil, "score:\n  meter: %d\n"+
		"  duration: %g\n  beats:\n    times: [%s]\n", mothMeter,
		mothSeconds, strings.Join(times, ", ")))
	if err != nil {
		t.Fatal(err)
	}
	return score
}

// a score in three reads, and its bars land where Moth Light's cut sheet
// says its parts begin.
func TestAScoreInThreeLandsOnItsCutSheet(t *testing.T) {
	score := mothScore(t)
	if score.Meter() != mothMeter {
		t.Fatalf("meter %d, not %d", score.Meter(), mothMeter)
	}
	for bar, want := range cutSheetPlaces {
		got, err := score.BarTime(bar, 1)
		if err != nil || !(math.Abs(got-want) <= cutSheetRounding) {
			t.Errorf("bar %d starts at %v (%v), not %v", bar, got, err, want)
		}
		beat := score.BeatAt(want + cutSheetRounding)
		if beat.Bar() != bar || beat.InBar() != 1 || !beat.Downbeat() {
			t.Errorf("at %v the beat is %d.%d", want, beat.Bar(),
				beat.InBar())
		}
	}
	last, err := score.BarTime(mothBars, mothMeter)
	if err != nil || last != score.Beats()[mothBars*mothMeter-1] {
		t.Errorf("the last beat is %v (%v)", last, err)
	}
}

// the anti-test: in three there is no fourth beat and no seventeenth
// bar, and the refusal says the score's own count.
func TestAScoreInThreeHasNoFourthBeat(t *testing.T) {
	score := mothScore(t)
	for _, place := range [][2]int{{1, 4}, {mothBars + 1, 1}} {
		_, err := score.BarTime(place[0], place[1])
		if !errors.Is(err, ErrShow) ||
			!strings.Contains(err.Error(), "bars 1 to 16, beats 1 to 3") {
			t.Errorf("bar %d beat %d: %v", place[0], place[1], err)
		}
	}
	if beat := score.BeatAt(score.Beats()[3]); beat.Bar() != 2 ||
		beat.InBar() != 1 {
		t.Errorf("the fourth beat is %d.%d, not 2.1", beat.Bar(),
			beat.InBar())
	}
}

// the lights count bars of three too: the fourth beat is bar two's
// first, a list taken two bars apiece turns after six beats, the fires
// burst every three beats and the floods' ones fall every three.
func TestTheLightsCountBarsOfThree(t *testing.T) {
	sheet, err := ParseSheet([]byte("cues: []\n"), mothScore(t))
	if err != nil {
		t.Fatal(err)
	}
	moment, err := sheet.At(sheet.Score().Beats()[3], 24)
	if err != nil || !strings.Contains(moment.Lines(), "bar=2.1\n") {
		t.Errorf("on the fourth beat (%v):\n%s", err, moment.Lines())
	}
	looks := []string{"fan", "scan"}
	if got, _ := inTurn(looks, 2, mothMeter, 0, 6); got != "scan" {
		t.Errorf("six beats into bars of three the list is at %s", got)
	}
	beats := steady(40)
	got := bursts(beatTime(3, beats), 3, 0, mothMeter, beats)
	if len(got) != 2 || got[0] != 0 || !(math.Abs(got[1]-1.5) <= near) {
		t.Errorf("three beats on, the bursts are %v", got)
	}
	_, right := flooded("ones", beatTime(3, beats), 3, mothMeter, beats,
		nil, 0)
	if !(math.Abs(right-1) <= near) {
		t.Errorf("on the second one of three, right is %v", right)
	}
}

// the anti-test: a meter under one is refused by name.
func TestAMeterUnderOneIsRefused(t *testing.T) {
	_, err := ParseScore([]byte("score:\n  meter: -3\n  beats:\n" +
		"    times: [1, 2]\n"))
	if !errors.Is(err, ErrShow) || !strings.Contains(err.Error(),
		"meter is -3") {
		t.Errorf("meter -3: %v", err)
	}
}
