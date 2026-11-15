//go:build draw

package cheeptune

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

// strongAt says whether a spectrum holds a partial at a frequency: a peak
// within five hertz, at least so many decibels under the strongest.
func strongAt(s *spectrum, hertz, floor float64) bool {
	strength, _ := s.near(hertz, 5)
	return decibels(strength/s.most(20, 20000)) > floor
}

func TestAnFMPairAtAWholeRatioHasItsPartialsWhereTheSumSaysTheyAre(
	t *testing.T) {
	// a carrier at 440 and a modulator at twice that: partials at 440
	// plus and minus whole steps of 880, which is every odd multiple.
	p := render(t, slow(`{"role": "lead", "bars": ["a4:4"]}`),
		studioVoicing(fmPair("2"), ""))
	s := newSpectrum(seconds(p, 1, 2.5), p.Rate)
	for _, want := range []float64{440, 1320, 2200} {
		if !strongAt(s, want, -45) {
			t.Errorf("no partial at %v Hz", want)
		}
	}
	// the anti-test: nothing at the even multiples in between.
	for _, not := range []float64{880, 1760} {
		if strongAt(s, not, -70) {
			t.Errorf("a partial at %v Hz, where a 1 to 2 pair has none",
				not)
		}
	}
}

func TestAnFMPairOffAWholeRatioHasPartialsOffTheHarmonics(t *testing.T) {
	// a modulator at 1.41 times 440 puts partials at 440 plus and minus
	// 620.4: 1060.4 and (folded through nought) 180.4, neither a
	// multiple of 440.
	p := render(t, slow(`{"role": "lead", "bars": ["a4:4"]}`),
		studioVoicing(fmPair("1.41"), ""))
	s := newSpectrum(seconds(p, 1, 2.5), p.Rate)
	for _, want := range []float64{440, 1060.4, 180.4} {
		if !strongAt(s, want, -45) {
			t.Errorf("no partial at %v Hz", want)
		}
	}
	for _, not := range []float64{880, 1320} {
		if strongAt(s, not, -60) {
			t.Errorf("a partial at %v Hz, a harmonic the pair lacks", not)
		}
	}
}

func TestAChordOfThreeNotesHoldsThreeFundamentalsAtOnce(t *testing.T) {
	sine := `{"kind": "fm", "role": "pad", "fm": {"arrangement": "sum",
		"operators": [{"ratio": 1, "level": 1, ` + held + `},
		{"ratio": 1, "level": 0, ` + held + `}]}`
	sc := slow(`{"role": "pad", "bars": ["(c4 e4 g4):4"]}`)
	p := render(t, sc, studioVoicing(sine+`}`, ""))
	// every tenth of a second in the middle holds all three.
	for at := 1.0; at < 3; at += 0.5 {
		s := newSpectrum(seconds(p, at, at+0.1), p.Rate)
		for _, want := range []float64{261.63, 329.63, 392.0} {
			strength, _ := s.near(want, 8)
			if decibels(strength/s.most(20, 20000)) < -6 {
				t.Errorf("at %v s the chord has no %v Hz", at, want)
			}
		}
	}
	// the anti-test: with an arp the same chord is one note at a time.
	p = render(t, sc, studioVoicing(sine+`, "arp": "1"}`, ""))
	s := newSpectrum(seconds(p, 0.3, 0.7), p.Rate)
	if !strongAt(s, 261.63, -6) || strongAt(s, 329.63, -30) {
		t.Errorf("an arp's first beat is not c4 alone")
	}
}

// burst is a score of one short hit and three bars of nothing.
var burst = slow(`{"role": "perc", "bars": ["k:1/4 -:15/4", "-:4", "-:4",
	"-:4"]}`)

// click is a drum channel whose one hit is a short burst of noise.
func click(send string) string {
	return `{"kind": "drum", "role": "perc", "send": {` + send + `},
		"drums": {"k": {"noise": 1, "noise_decay": 0.01, "colour": 200}}}`
}

// fallsBy is the second after which the sound left in a piece is so many
// decibels under all of it: Schroeder's backward sum.
func fallsBy(p *Piece, from, drop float64) float64 {
	start := int(from * float64(p.Rate))
	total := 0.0
	for _, v := range p.Left[start:] {
		total += float64(v) * float64(v)
	}
	left := total
	for i, v := range p.Left[start:] {
		left -= float64(v) * float64(v)
		if left <= total*math.Pow(10, -drop/10) {
			return from + float64(i)/float64(p.Rate)
		}
	}
	return math.Inf(1)
}

func TestTheReverbsTailFallsSixtyDecibelsNearItsStatedTime(t *testing.T) {
	for _, decay := range []float64{1, 3} {
		room := `, "reverb": {"size": 0.8, "decay": ` +
			strings.TrimRight(strings.TrimRight(
				strconvF(decay), "0"), ".") +
			`, "damping": 0, "level": 1}`
		p := render(t, burst, studioVoicing(click(`"reverb": 1`), room))
		// thirty decibels of the backward sum is half the decay.
		got := 2 * (fallsBy(p, 0.1, 30) - 0.1)
		if math.Abs(got-decay) > 0.15*decay {
			t.Errorf("a %v s reverb falls sixty decibels in %.2f s",
				decay, got)
		}
		// and it is silent after: the piece ends, and its last tenth
		// of a second is seventy decibels under its loudest.
		tail := p.Left[len(p.Left)-p.Rate/10:]
		if decibels(rms(tail)/p.Peak) > -70 {
			t.Errorf("a %v s reverb is not silent at the end", decay)
		}
	}
	// the anti-test: with nothing sent, the hit is over in its own time.
	room := `, "reverb": {"size": 0.8, "decay": 3, "damping": 0,
		"level": 1}`
	p := render(t, burst, studioVoicing(click(`"reverb": 0`), room))
	if got := fallsBy(p, 0.1, 30); got > 0.2 {
		t.Errorf("a dry hit rings until %.2f s", got)
	}
}

func TestTheDelaysRepeatsFallOnTheBeat(t *testing.T) {
	echo := `, "delay": {"beats": "1", "feedback": 0.5, "tone": 1,
		"level": 1}`
	p := render(t, burst, studioVoicing(click(`"delay": 1`), echo))
	loud := func(at float64) float64 {
		return rms(seconds(p, at, at+0.02))
	}
	// a beat is a second: the hit at nought comes back at one, two and
	// three, each time at half, and the half-beats between are empty.
	for beat := 1.0; beat <= 3; beat++ {
		ratio := loud(beat) / loud(beat-1)
		if beat == 1 {
			ratio /= 2 // the dry hit and its first return are equal
		}
		if math.Abs(ratio-0.5) > 0.1 {
			t.Errorf("repeat %v is %.2f of the one before", beat, ratio)
		}
		if quiet := loud(beat - 0.5); quiet > loud(beat)/100 {
			t.Errorf("sound at beat %v, between repeats", beat-0.5)
		}
	}
}

// strconvF is a number as a document writes it.
func strconvF(x float64) string {
	return strconv.FormatFloat(x, 'f', 3, 64)
}
