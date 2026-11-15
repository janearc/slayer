package cheeptune

import (
	"testing"
)

// perc is a one-bar score of one perc bar at sixty a minute.
func perc(bar string) []byte {
	return slow(`{"role": "perc", "bars": ["` + bar + `"]}`)
}

func TestHitsStruckTogetherAreReadAsOneMomentWithALengthAndAMark(
	t *testing.T) {
	s, err := ParseScore(perc("(k h):1/2@f k:1/2 (k s h) -:2"))
	if err != nil {
		t.Fatal(err)
	}
	ev := s.Parts[0].Events()
	if len(ev) != 6 {
		t.Fatalf("%d events, want two, one and three", len(ev))
	}
	if ev[0].Hit != "k" || ev[1].Hit != "h" || ev[0].Beats != 0 ||
		ev[1].Beats != 0 || ev[0].Length != 0.5 || ev[1].Loud != 1.26 {
		t.Errorf("the first stack is %+v %+v", ev[0], ev[1])
	}
	if ev[3].Beats != 1 || ev[5].Beats != 1 || ev[5].Hit != "h" {
		t.Errorf("the second stack is %+v", ev[3:])
	}
}

func TestAStackThatIsNotOneIsRefusedByName(t *testing.T) {
	for bar, fragment := range map[string]string{
		"(k c4):4":     "a chord is notes or hits, not both",
		"(k k):4":      "the hit k is in the chord twice",
		"(k):4":        "a chord has two notes or more",
		"(k h):4~":     "a hit does not tie",
		"(k h):2> k:2": "only a single note slides",
	} {
		_, err := ParseScore(perc(bar))
		wantErr(t, err, fragment)
	}
	// the near miss: a chord of notes is still a chord of notes.
	s, err := ParseScore(slow(`{"role": "pad", "bars": ["(c4 e4):4"]}`))
	if err != nil || len(s.Parts[0].Events()) != 1 {
		t.Errorf("a chord of notes: %v", err)
	}
}

func TestEveryBuiltInStudioVoicingHasTheSameSixDrums(t *testing.T) {
	_, voicings := Builtin()
	seen := 0
	for _, name := range voicings {
		v, err := ReadVoicing(name)
		if err != nil {
			t.Fatal(err)
		}
		if v.Machine != "studio" {
			continue
		}
		seen++
		for i, c := range v.Channels {
			if !v.Percussive(i) {
				continue
			}
			if got := len(c.Drums); got != len(DrumLetters) {
				t.Errorf("%s has %d drums, want the %d letters", name,
					got, len(DrumLetters))
			}
			// an open hat is longer than a closed one, where both are
			// hiss: the one thing about a letter that can be measured.
			h, o := c.Drums["h"], c.Drums["o"]
			if longest(o) <= longest(h) {
				t.Errorf("%s: o lasts %v s and h %v s", name, longest(o),
					longest(h))
			}
		}
	}
	if seen != 8 {
		t.Errorf("%d studio voicings, want eight", seen)
	}
	if len(drumLetters) != len(DrumLetters) {
		t.Errorf("the letters and their order disagree")
	}
}

// longest is how long a drum sounds, by its longer decay.
func longest(d Drum) float64 { return max(d.Decay, d.NoiseDecay) }
