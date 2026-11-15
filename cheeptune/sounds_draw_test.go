//go:build draw

package cheeptune

import (
	"math"
	"os"
	"regexp"
	"strings"
	"testing"
)

// soundsPage is the text of SOUNDS.md.
func soundsPage(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("SOUNDS.md")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// workedSounds is the instruments SOUNDS.md gives, by name.
func workedSounds(t *testing.T) map[string]*Voicing {
	t.Helper()
	blocks := regexp.MustCompile("(?s)```json\n(.*?)```").
		FindAllStringSubmatch(soundsPage(t), -1)
	out := map[string]*Voicing{}
	for _, block := range blocks {
		v, err := ParseVoicing([]byte(block[1]))
		if err != nil {
			t.Fatalf("an instrument in SOUNDS.md does not read: %v\n%s",
				err, block[1])
		}
		out[v.Name] = v
	}
	return out
}

// sounded is one of the page's instruments playing a bar.
func sounded(t *testing.T, v *Voicing, bar string) *Piece {
	t.Helper()
	s, err := ParseScore(slow(`{"role": "` + v.Channels[0].Role +
		`", "bars": ["` + bar + `"]}`))
	if err != nil {
		t.Fatal(err)
	}
	p, err := Render(s, v)
	if err != nil {
		t.Fatalf("%s: %v", v.Name, err)
	}
	return p
}

func TestThePageGivesSixInstrumentsAndEachOneSounds(t *testing.T) {
	sounds := workedSounds(t)
	if len(sounds) != 7 {
		t.Fatalf("SOUNDS.md has %d instruments, want seven", len(sounds))
	}
	for name, v := range sounds {
		bar := "(a3 c#4 e4):2 a4:2"
		if v.Channels[0].Role == "bass" {
			bar = "a2:1/2> a3:1/2@ff a2:1 e2:2"
		}
		if v.Percussive(0) {
			bar = "k s h t"
		}
		p := sounded(t, v, bar)
		if p.Peak < 0.02 || p.Peak > 1 {
			t.Errorf("%s peaks at %v", name, p.Peak)
		}
	}
}

func TestEachWorkedInstrumentIsWhatThePageSaysItIs(t *testing.T) {
	sounds := workedSounds(t)
	partial := func(name string, multiple float64) float64 {
		p := sounded(t, sounds[name], "a4:4")
		s := newSpectrum(seconds(p, 0.02, 0.5), p.Rate)
		strength, _ := s.near(440*multiple, 12)
		return decibels(strength / s.most(20, 20000))
	}
	// the bell clangs: partials either side of its carrier at steps of
	// three and a half, which are at 2.5 and 4.5 times the note.
	if partial("glass-bell", 2.5) < -45 || partial("glass-bell", 4.5) < -45 {
		t.Errorf("the bell has no partials off the whole numbers")
	}
	// the keys have one clear pitch: a second overtone, and nothing
	// half way to it.
	if partial("soft-keys", 2) < -40 || partial("soft-keys", 1.5) > -60 {
		t.Errorf("the keys: second %.1f dB, between %.1f dB",
			partial("soft-keys", 2), partial("soft-keys", 1.5))
	}
	// the pad swells: a twentieth of a second in it is far quieter
	// than at eight tenths. and it is dark.
	pad := sounded(t, sounds["warm-pad"], "a3:4")
	early, late := rms(seconds(pad, 0.03, 0.07)), rms(seconds(pad, 0.8, 1))
	if early > late/3 {
		t.Errorf("the pad does not swell: %v then %v", early, late)
	}
	s := newSpectrum(seconds(pad, 1, 2), pad.Rate)
	if top := decibels(s.most(6000, 16000) / s.most(20, 2000)); top > -35 {
		t.Errorf("the pad's top is only %.1f dB down", top)
	}
	// the harp is in tune, rings on after its note, and plucked in
	// the middle has almost no second overtone.
	harp := sounded(t, sounds["harp"], "a3:1 -:3")
	s = newSpectrum(seconds(harp, 0.1, 1.5), harp.Rate)
	first, at := s.near(220, 8)
	second, _ := s.near(440, 8)
	if math.Abs(cents(at, 220)) > pluckCents || second > first/8 {
		t.Errorf("the harp is at %.2f Hz, second overtone %.3f of first",
			at, second/first)
	}
	if rms(seconds(harp, 1.5, 2)) < rms(seconds(harp, 0.5, 1))/20 {
		t.Errorf("the harp does not ring on after its note")
	}
	// the bowl hums on long after a short note, with an overtone at
	// the ring's 2.83.
	bowl := sounded(t, sounds["singing-bowl"], "a3:1 -:3")
	if rms(seconds(bowl, 2, 2.5)) < rms(seconds(bowl, 0.5, 1))/10 {
		t.Errorf("the bowl has stopped by two seconds")
	}
	s = newSpectrum(seconds(bowl, 0.02, 0.4), bowl.Rate)
	if strength, _ := s.near(220*ringRatios(2)[1], 10); decibels(
		strength/s.most(20, 20000)) < -40 {
		t.Errorf("the bowl has no overtone at 2.83")
	}
	// the acid line squelches: in its first twentieth of a second the
	// filter is open and the top is there; a fifth of a second on it
	// has closed. and its filter is the ladder.
	acid := sounded(t, sounds["acid-line"], "a2:4")
	open := newSpectrum(seconds(acid, 0, 0.05), acid.Rate)
	shut := newSpectrum(seconds(acid, 0.3, 0.5), acid.Rate)
	bright := func(s *spectrum) float64 {
		return power(s, 2500, 8000) - power(s, 80, 300)
	}
	if bright(open)-bright(shut) < 8 {
		t.Errorf("the acid line's filter does not close: %.1f then %.1f",
			bright(open), bright(shut))
	}
	if sounds["acid-line"].Channels[0].Analog.Filter.Poles != 4 {
		t.Errorf("the acid line is not on the four-pole filter")
	}
	// the kit's kick lands on its tone.
	kit := sounded(t, sounds["soft-kit"], "k -:3")
	s = newSpectrum(seconds(kit, 0.1, 0.3), kit.Rate)
	if _, at := s.near(50, 30); math.Abs(at-50) > 5 {
		t.Errorf("the kit's kick lands on %.1f Hz", at)
	}
}

func TestTheAcidBarThePageGivesIsABarAndSlidesAndAccents(t *testing.T) {
	_, after, found := strings.Cut(soundsPage(t), "a bar for it")
	if !found {
		t.Fatal("SOUNDS.md has no acid bar")
	}
	bar := regexp.MustCompile("(?s)`(a1:1/4.*?)`").FindStringSubmatch(after)
	if bar == nil {
		t.Fatal("the acid bar is not quoted")
	}
	s, err := ParseScore([]byte(`{"name": "x", "tempo": 138, "meter": 4,
		"parts": [{"role": "bass", "bars": ["` +
		strings.Join(strings.Fields(bar[1]), " ") + `"]}]}`))
	if err != nil {
		t.Fatalf("the page's acid bar: %v", err)
	}
	slides, accents := 0, 0
	for _, e := range s.Parts[0].Events() {
		if e.Slide {
			slides++
		}
		if e.Loud > 1 {
			accents++
		}
	}
	if slides < 3 || accents < 3 {
		t.Errorf("the acid bar has %d slides and %d accents", slides,
			accents)
	}
	p, err := Render(s, workedSounds(t)["acid-line"])
	if err != nil || p.Peak > 1 {
		t.Errorf("the acid bar through the acid line: %v", err)
	}
}
