//go:build draw

package cheeptune

import (
	"bytes"
	"math"
	"strings"
	"testing"
	"time"
)

func TestTheStudioPlaysAMarkedNoteLouderOrSofterByItsGain(t *testing.T) {
	loud := func(bar string) float64 {
		p := render(t, marked(bar), studioVoicing(fmPair("2"), ""))
		return rms(seconds(p, 0.5, 1.5))
	}
	plain := loud("a4:4")
	for mark, gain := range Marks {
		got := loud("a4:4@"+mark) / plain
		if math.Abs(got-gain) > 0.01 {
			t.Errorf("a note marked %s is %.3f of an unmarked one, "+
				"want %v", mark, got, gain)
		}
	}
}

func TestTheGameBoyPlaysAMarkedScoreExactlyAsAnUnmarkedOne(t *testing.T) {
	plain := render(t, marked("a4:2 c5:2"), voicing(lead, ""))
	loud := render(t, marked("a4:2@pp c5:2@ff"), voicing(lead, ""))
	if !bytes.Equal(pcmBytes(plain.Left), pcmBytes(loud.Left)) {
		t.Errorf("the marks changed what the Game Boy played")
	}
	// the anti-test: a different note does change it, so the
	// comparison can tell.
	other := render(t, marked("a4:2 d5:2"), voicing(lead, ""))
	if bytes.Equal(pcmBytes(plain.Left), pcmBytes(other.Left)) {
		t.Errorf("two different scores compare as the same")
	}
}

// bothMachines is the Game Boy and the studio each playing turning.
var bothMachines = map[string][]byte{
	"gameboy": voicing(lead+`, {"kind": "noise", "role": "perc",
		"hits": {"k": {"shift": 3, "divisor": 2, "decay": 0.1}}}`, ""),
	"studio": studioVoicing(fmPair("2")+`,`+click(`"reverb": 0`), ""),
}

func TestBothMachinesThePlayheadAndTheSheetHonourATempoChange(
	t *testing.T) {
	for name, v := range bothMachines {
		p := render(t, turning(doubled), v)
		// the fourth bar's hit is at ten seconds, not twelve.
		if at := rms(seconds(p, 10, 10.05)); at < 0.001 {
			t.Errorf("%s: nothing at 10 s, where bar 4 begins", name)
		}
		if quiet := rms(seconds(p, 9.8, 9.95)); quiet >
			rms(seconds(p, 10, 10.05))/5 {
			t.Errorf("%s: sound just before bar 4", name)
		}
		// the lead's note in bar 3 is two fast beats, eight to nine.
		s := newSpectrum(seconds(p, 8.2, 8.9), p.Rate)
		if _, at := s.near(440, 30); math.Abs(at-440) > 5 {
			t.Errorf("%s: the lead is not sounding at eight and a half seconds", name)
		}
		// the playhead: nine seconds is beat ten, half way into the
		// third beat of bar three.
		_, bar, beats, _, ok := p.At(9 * time.Second)
		if !ok || math.Abs(beats-10) > 1e-9 || math.Abs(bar-0.5) > 1e-9 {
			t.Errorf("%s: at 9 s the playhead says beat %v, bar %v",
				name, beats, bar)
		}
		if p.Score.Seconds() != 12 {
			t.Errorf("%s: the piece is %v s", name, p.Score.Seconds())
		}
	}
	s, err := ParseScore(turning(doubled))
	if err != nil {
		t.Fatal(err)
	}
	sheet := func(second string) []byte {
		return []byte(`{"group": "g", "mood": "two words", "feeling": "f",
			"intent": "i", "listen_for": "l", "voicing": "@dmg-pad",
			"tempo": 60, "meter": 4, "seconds": 12, "parts": [
			{"name": "a", "bar": 1, "second": 0, "change": "c",
			 "feels": "f"},
			{"name": "b", "bar": 4, "second": ` + second + `,
			 "change": "c", "feels": "f"}]}`)
	}
	if _, err := ParseCuts(sheet("10"), s); err != nil {
		t.Errorf("a sheet with the changed second: %v", err)
	}
	_, err = ParseCuts(sheet("12"), s)
	wantErr(t, err, "bar 4 starts at 10.00 s")
}

func TestTheDelaysRepeatsFollowATempoChange(t *testing.T) {
	echo := `, "delay": {"beats": "1", "feedback": 0, "tone": 1,
		"level": 1}`
	sc := []byte(strings.Replace(string(turning(doubled)),
		`"k -:3", "k -:3", "k -:3", "k -:3"`,
		`"k -:3", "-:4", "k -:3", "-:4"`, 1))
	sc = []byte(strings.Replace(string(sc), `"a4:2 -:2"`, `"-:4"`, 1))
	v := studioVoicing(fmPair("2")+`,`+click(`"delay": 1`), echo)
	p := render(t, sc, v)
	loud := func(at float64) float64 {
		return rms(seconds(p, at, at+0.02))
	}
	// the hit at nought comes back a slow beat later, at one second;
	// the hit at eight comes back a fast beat later, at eight and a
	// half, and not at nine.
	if loud(1) < loud(0)/2 || loud(8.5) < loud(8)/2 {
		t.Errorf("repeats at 1 s and 8.5 s are %.4f and %.4f", loud(1),
			loud(8.5))
	}
	if loud(9) > loud(8.5)/50 || loud(0.5) > loud(1)/50 {
		t.Errorf("a repeat off its own tempo's beat")
	}
}
