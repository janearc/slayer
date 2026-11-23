package cheeptune

import (
	"math"
	"testing"
)

// What the acid voice is held to. Set from what it measured on
// 2026-11-12, with room, by the writer of the studio; not chosen by ear.
const (
	// ladderSteeper is how many decibels more than the two-pole filter
	// the ladder takes off three octaves above the cutoff. Four poles
	// against two is thirty-six in theory.
	ladderSteeper = 30
	// ladderPeakOff is how far the ladder's peak may sit from the
	// stated cutoff, as a share.
	ladderPeakOff = 0.12
	// ladderPeakTall is the least the peak stands over the low end at
	// resonance 0.9, in decibels.
	ladderPeakTall = 12
)

// hissed is white noise at a level through a filter, as samples.
func hissed(f lowPass, level float64) []float32 {
	source := newNoise(7)
	out := make([]float32, waveLength)
	for i := range out {
		drive := source.next() * level
		out[i] = float32(f.low(drive))
	}
	return out
}

// powerFloor keeps a log off nought, far under anything measured: the
// spectrum's own floor is too high for a sound eighty decibels down.
const powerFloor = 1e-30

// power is the mean strength between two frequencies, in decibels.
func power(s *spectrum, from, to float64) float64 {
	sum, count := 0.0, 0
	for i := s.bin(from); i <= s.bin(to); i++ {
		sum += s.strength[i] * s.strength[i]
		count++
	}
	return 10 * math.Log10(sum/float64(count)+powerFloor)
}

// tallest is the frequency of the strongest part of a spectrum, each
// part a sixth of an octave wide so noise does not pick it.
func tallest(s *spectrum, from, to float64) (hertz, level float64) {
	level = math.Inf(-1)
	for at := from; at < to; at *= math.Exp2(1.0 / 24) {
		if got := power(s, at/math.Exp2(1.0/12), at*math.Exp2(1.0/12)); got > level {
			hertz, level = at, got
		}
	}
	return hertz, level
}

func TestTheLadderIsSteeperThanTheStateFilter(t *testing.T) {
	fall := func(f lowPass) float64 {
		f.tune(500, 0)
		s := newSpectrum(hissed(f, 0.05), studioRate)
		return power(s, 3500, 4500) - power(s, 60, 200)
	}
	ladder, state := fall(&ladderFilter{}), fall(newStateFilter())
	if state-ladder < ladderSteeper {
		t.Errorf("three octaves up the ladder is %.1f dB down and the "+
			"state filter %.1f", ladder, state)
	}
	// the anti-test: the state filter is the gentle one, about
	// thirty-six down there, and is not mistaken for a ladder.
	if state < -45 || state > -28 {
		t.Errorf("the state filter is %.1f dB down three octaves up",
			state)
	}
}

func TestTheLaddersPeakSitsAtItsCutoff(t *testing.T) {
	for _, cutoff := range []float64{300, 1000, 4000} {
		f := &ladderFilter{}
		f.tune(cutoff, 0.9)
		s := newSpectrum(hissed(f, 0.02), studioRate)
		at, level := tallest(s, 60, 16000)
		if math.Abs(at-cutoff) > ladderPeakOff*cutoff {
			t.Errorf("a %v Hz ladder peaks at %.0f Hz", cutoff, at)
		}
		if tall := level - power(s, 60, cutoff/3); tall < ladderPeakTall {
			t.Errorf("a %v Hz ladder's peak is %.1f dB tall", cutoff, tall)
		}
	}
	// the anti-test: with no resonance nothing stands over the low end.
	f := &ladderFilter{}
	f.tune(1000, 0)
	s := newSpectrum(hissed(f, 0.02), studioRate)
	if _, level := tallest(s, 100, 16000); level-power(s, 100, 300) > 4 {
		t.Errorf("a ladder with no resonance has a peak")
	}
}

func TestTheLadderSingsAndNeverBlowsUp(t *testing.T) {
	for _, cutoff := range []float64{20, 500, 8000, 21000} {
		for _, resonance := range []float64{0.86, 1} {
			f := &ladderFilter{}
			f.tune(cutoff, resonance)
			// a second of a square five times over full scale.
			for i := range studioRate {
				x := 5.0
				if i/50%2 == 1 {
					x = -5
				}
				if y := f.low(x); math.IsNaN(y) || math.Abs(y) > 1 {
					t.Fatalf("cutoff %v resonance %v gave %v", cutoff,
						resonance, y)
				}
			}
			// at the resonance the acid voicings use, it stops when
			// its input stops: it squelches and does not whistle on.
			if resonance == 1 {
				continue
			}
			last := 0.0
			for range studioRate / 2 {
				last = f.low(0)
			}
			if math.Abs(last) > 0.001 {
				t.Errorf("cutoff %v: still at %v half a second after "+
					"the input stopped", cutoff, last)
			}
		}
	}
}

func TestAStrikeSlidesInAStraightLineOfSemitones(t *testing.T) {
	st := strike{hz: 100, gain: 1, glide: 0.1, path: []slideStep{
		{at: 1, hz: 400, gain: 0.5}, {at: 2, hz: 100, gain: 1}}}
	for age, want := range map[float64][2]float64{
		0.5: {1, 1}, 1.05: {2, 0.75}, 1.5: {4, 0.5}, 2.05: {2, 0.75},
		3: {1, 1},
	} {
		ratio, gain := st.slideAt(age)
		if math.Abs(ratio-want[0]) > 1e-9 || math.Abs(gain-want[1]) > 1e-9 {
			t.Errorf("at %v s: ratio %v gain %v, want %v", age, ratio,
				gain, want)
		}
	}
	// with no glide time the step is at once.
	st.glide = 0
	if ratio, _ := st.slideAt(1); ratio != 4 {
		t.Errorf("a glide of nought is at %v at its step", ratio)
	}
}

func TestASlideIsReadAndItsMistakesRefusedByName(t *testing.T) {
	s, err := ParseScore(slow(`{"role": "bass",
		"bars": ["a2:1/2> a3:1/2@ff> a2 -:2", "a2:4~", "a2:2> c3:2"]}`))
	if err != nil {
		t.Fatal(err)
	}
	ev := s.Parts[0].Events()
	if !ev[0].Slide || !ev[1].Slide || ev[2].Slide || ev[1].Loud != 1.58 {
		t.Errorf("slides read as %+v", ev[:3])
	}
	if !ev[3].Slide || ev[3].Length != 6 {
		t.Errorf("a tied note that then slides is %+v", ev[3])
	}
	for bar, fragment := range map[string]string{
		"a2:2> -:2":       "the slide before it lands on a single note",
		"a2:2> (c3 e3):2": "the slide before it lands on a single note",
		"(c3 e3):2> a2:2": "only a single note slides",
		"-:2> a2:2":       "only a single note slides",
		"a2:2 a2:2>":      "slides into a bar that is not there",
		"a2:4>~":          "a note slides (>) or ties (~), not both",
		"a2:4~>":          "a note slides (>) or ties (~), not both",
	} {
		_, err := ParseScore(slow(`{"role": "bass", "bars": ["` + bar +
			`"]}`))
		wantErr(t, err, fragment)
	}
	_, err = ParseScore(slow(`{"role": "perc", "bars": ["k:2> k:2"]}`))
	wantErr(t, err, "only a single note slides")
}

func TestTheAcidFieldsKeepTheirRules(t *testing.T) {
	filter := func(inner string) string {
		return studioChannelOf("analog", "bass", `"analog": {
			"oscillators": [{"shape": "saw"}], "filter": {"cutoff": 500`+
			inner+`}}`)
	}
	drum := func(inner string) string {
		return studioChannelOf("drum", "perc", `"drums": {"k": {`+inner+
			`}}`)
	}
	for doc, fragment := range map[string]string{
		filter(`, "poles": 3`): "filter poles 3; a filter has 2 or 4",
		filter(`, "accent": 1`): "filter accent and snap need a filter " +
			"envelope to act on",
		filter(`, "snap": 2, "envelope": {"decay": 1}`): "filter snap 2 " +
			"is outside 0 to 0.95",
		studioChannelOf("analog", "bass", goodAnalog+`, "glide": 5`): "glide 5 is outside 0.001 to 2",
		studioChannelOf("pluck", "bass", goodPluck+`, "glide": 0.1`): "glide is not for a pluck channel; fm and analog slide",
		studioChannelOf("analog", "bass", `"analog": {"oscillators":
			[{"shape": "saw"}], "fold": 2}`): "analog fold 2 is outside 0 to 1",
		studioChannelOf("analog", "bass", `"analog": {"oscillators":
			[{"shape": "saw"}], "overdrive": 2}`): "analog overdrive 2 is outside 0 to 1",
		drum(`"tone": 400, "decay": 0.1, "struck": "anvil"`): `drum k struck "anvil"; sets are bar, bowl, marimba, plate, tube`,
		drum(`"tone": 400, "decay": 0.1, "struck": "bar", "sweep": 3,
			"fall": 0.1`): "a struck drum needs a tone and takes no sweep",
		drum(`"tone": 400, "decay": 0.1, "hardness": 0.5`): "hardness is for a struck drum",
	} {
		_, err := ParseVoicing(studioVoicing(doc, ""))
		wantErr(t, err, fragment)
	}
	_, err := ParseVoicing(voicing(`{"kind": "pulse", "role": "lead",
		"duty": 0.5, "glide": 0.1}`, ""))
	wantErr(t, err, "glide is only for the studio machine")
	// the good ones read.
	for _, doc := range []string{
		filter(`, "poles": 4, "accent": 1, "snap": 0.5,
			"envelope": {"decay": 0.2}`),
		studioChannelOf("analog", "bass", goodAnalog+`, "glide": 0.1`),
		studioChannelOf("fm", "bass", goodFM+`, "glide": 0.1`),
		drum(`"tone": 400, "decay": 0.1, "struck": "bar", "hardness": 1`),
	} {
		if _, err := ParseVoicing(studioVoicing(doc, "")); err != nil {
			t.Errorf("%s: %v", doc, err)
		}
	}
}
