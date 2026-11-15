//go:build draw

package cheeptune

import (
	"math"
	"strings"
	"testing"
)

// analogSaw is a lead of one saw through a filter, held.
func analogSaw(filter string) string {
	return `{"kind": "analog", "role": "lead", ` + held + `,
		"analog": {"oscillators": [{"shape": "saw"}]` + filter + `}}`
}

func TestTheFiltersCutoffMovesTheSpectrumAsStated(t *testing.T) {
	sc := slow(`{"role": "lead", "bars": ["c3:4"]}`)
	top := func(filter string) float64 {
		p := render(t, sc, studioVoicing(analogSaw(filter), ""))
		s := newSpectrum(seconds(p, 1, 2), p.Rate)
		own, _ := s.near(hz(48, 0), 5)
		return decibels(s.most(4000, 8000) / own)
	}
	open := top("")
	shut := top(`, "filter": {"cutoff": 500, "resonance": 0}`)
	// a two-pole filter falls twelve decibels an octave: three octaves
	// above 500 Hz the saw's top is about thirty-six down.
	if open-shut < 30 {
		t.Errorf("a 500 Hz cutoff takes %.1f dB off the top", open-shut)
	}
	// the anti-test: a cutoff far above the band leaves it alone.
	wide := top(`, "filter": {"cutoff": 20000, "resonance": 0}`)
	if math.Abs(open-wide) > 3 {
		t.Errorf("a 20 kHz cutoff moves the top by %.1f dB", open-wide)
	}
	// following the note: the same filter on a note two octaves up
	// opens two octaves with follow at one; and c3, an octave under
	// middle c, closes an octave.
	follows := `, "filter": {"cutoff": 500, "resonance": 0, "follow": 1}`
	f := Filter{Cutoff: 500, Follow: 1}
	if got := f.cutoff(middleC+24, 0); math.Abs(got-2000) > 1 {
		t.Errorf("following, two octaves up the cutoff is %v", got)
	}
	if top(follows) > shut-3 {
		t.Errorf("following a note an octave under middle c did not " +
			"close the filter")
	}
	// and its envelope opens it by its amount, in octaves.
	f = Filter{Cutoff: 500}
	if got := f.cutoff(middleC, 3); math.Abs(got-4000) > 1 {
		t.Errorf("opened three octaves the cutoff is %v", got)
	}
}

// string is a lead on a plucked string.
func plucked(extra string) string {
	return `{"kind": "pluck", "role": "lead"` + extra + `, "pluck":
		{"brightness": 0.8, "damping": 0.3, "position": 0.2, "decay": 6,
		 "release": 2}}`
}

// cents is how far a frequency is from a wanted one.
func cents(got, want float64) float64 {
	return centsAnOctave * math.Log2(got/want)
}

func TestAPluckedStringIsInTuneLowMiddleAndHigh(t *testing.T) {
	for _, note := range []string{"e2", "a4", "c7"} {
		m, _ := pitch(note)
		p := render(t, slow(`{"role": "lead", "bars": ["`+note+`:4"]}`),
			studioVoicing(plucked(""), ""))
		s := newSpectrum(seconds(p, 0.2, 1.6), p.Rate)
		_, at := s.near(hz(m, 0), hz(m, 0)*0.03)
		if off := cents(at, hz(m, 0)); math.Abs(off) > pluckCents {
			t.Errorf("%s on a string is %.2f cents off", note, off)
		}
	}
	// the anti-test: a string moved ten cents measures as ten cents
	// off, so the measure would see a mistuned loop.
	p := render(t, slow(`{"role": "lead", "bars": ["a4:4"]}`),
		studioVoicing(plucked(`, "detune": 10`), ""))
	s := newSpectrum(seconds(p, 0.2, 1.6), p.Rate)
	_, at := s.near(440, 15)
	if off := cents(at, 440); math.Abs(off-10) > pluckCents {
		t.Errorf("a string moved 10 cents measures %.2f", off)
	}
}

func TestAStringPluckedInTheMiddleHasNoEvenOvertones(t *testing.T) {
	middle := strings.Replace(plucked(""), `"position": 0.2`,
		`"position": 0.5`, 1)
	p := render(t, slow(`{"role": "lead", "bars": ["a2:4"]}`),
		studioVoicing(middle, ""))
	s := newSpectrum(seconds(p, 0.05, 0.6), p.Rate)
	first, _ := s.near(110, 5)
	second, _ := s.near(220, 5)
	third, _ := s.near(330, 5)
	if second > first/10 || third < second {
		t.Errorf("plucked in the middle: overtones %.3f %.3f %.3f",
			first, second, third)
	}
}

// struck is a lead on a struck object.
func struck(modal string) string {
	return `{"kind": "modal", "role": "lead", "modal": ` + modal + `}`
}

// ringing is how strong the partial at a frequency is over a tenth of a
// second from a moment.
func ringing(p *Piece, at, hertz float64) float64 {
	s := newSpectrum(seconds(p, at, at+0.1), p.Rate)
	strength, _ := s.near(hertz, 15)
	return strength
}

func TestAModalSetsPartialsAreAtItsRatiosAndDieInTheirTimes(t *testing.T) {
	// three modes of our own, so the stated numbers are in the test.
	modes := []Mode{{1, 2, 1}, {2.76, 1, 0.8}, {5.4, 0.5, 0.6}}
	p := render(t, slow(`{"role": "lead", "bars": ["a3:4"]}`),
		studioVoicing(struck(`{"hardness": 1, "modes": [
			{"ratio": 1, "decay": 2, "level": 1},
			{"ratio": 2.76, "decay": 1, "level": 0.8},
			{"ratio": 5.4, "decay": 0.5, "level": 0.6}]}`), ""))
	s := newSpectrum(seconds(p, 0.02, 0.4), p.Rate)
	for _, mode := range modes {
		want := 220 * mode.Ratio
		_, at := s.near(want, 20)
		if math.Abs(cents(at, want)) > 5 {
			t.Errorf("the mode at %v times the note is at %.1f Hz", mode.Ratio, at)
		}
		// between a tenth and four tenths of a second a mode falls by
		// sixty decibels times three tenths over its time.
		fell := decibels(ringing(p, 0.1, want) / ringing(p, 0.4, want))
		stated := 60 * 0.3 / mode.Decay
		if math.Abs(fell-stated) > decayTolerance*stated {
			t.Errorf("the mode at %v fell %.1f dB in 0.3 s, stated %.1f",
				mode.Ratio, fell, stated)
		}
	}
	// the anti-test: nothing rings at the whole multiples a string
	// would have.
	for _, not := range []float64{440, 660} {
		if strongAt(s, not, -40) {
			t.Errorf("a partial at %v Hz, which the modes do not hold",
				not)
		}
	}
}

func TestAHardBlowRingsTheHighModesAndASoftOneDoesNot(t *testing.T) {
	top := func(hardness string) float64 {
		p := render(t, slow(`{"role": "lead", "bars": ["a3:4"]}`),
			studioVoicing(struck(`{"set": "bar", "decay": 2,
				"hardness": `+hardness+`}`), ""))
		s := newSpectrum(seconds(p, 0, 0.2), p.Rate)
		own, _ := s.near(220, 10)
		high, _ := s.near(220*barRatios(6, 0, 1)[3], 20)
		return decibels(high / own)
	}
	if hard, soft := top("1"), top("0"); hard-soft < 12 {
		t.Errorf("the fourth mode is %.1f dB hard and %.1f dB soft",
			hard, soft)
	}
}

func TestAHandOnAStruckThingStopsItAndWithoutOneItRingsOn(t *testing.T) {
	late := func(release string) float64 {
		p := render(t, slow(`{"role": "lead", "bars": ["a3:1 -:3"]}`),
			studioVoicing(struck(`{"set": "bowl", "decay": 8,
				"hardness": 0.5`+release+`}`), ""))
		return decibels(rms(seconds(p, 2, 2.2)) / rms(seconds(p, 0.5, 0.7)))
	}
	if free := late(""); free < -15 {
		t.Errorf("left alone, the bowl is %.1f dB down a second on", free)
	}
	if damped := late(`, "release": 0.3`); damped > -60 {
		t.Errorf("damped, the bowl is only %.1f dB down", damped)
	}
}

// kit is a drum channel with a kick and a hat.
const kit = `{"kind": "drum", "role": "perc", "drums": {
	"k": {"tone": 50, "sweep": 24, "fall": 0.05, "decay": 0.4,
	      "click": 0.3},
	"h": {"noise": 1, "noise_decay": 0.05, "colour": 8000}}}`

func TestAKickFallsToItsToneAndAHatHasNoBottom(t *testing.T) {
	p := render(t, slow(`{"role": "perc", "bars": ["k:2 h:2"]}`),
		studioVoicing(kit, ""))
	// once the fall is over the kick is at its tone.
	s := newSpectrum(seconds(p, 0.1, 0.35), p.Rate)
	if _, at := s.near(50, 30); math.Abs(at-50) > 4 {
		t.Errorf("the kick lands on %.1f Hz, want 50", at)
	}
	// in its first hundredth it is well above: two octaves up at the
	// start, so most of that window is over 100 Hz.
	early := frequency(seconds(p, 0, 0.02), p.Rate)
	if early < 100 {
		t.Errorf("the kick starts at %.0f Hz, no higher than it ends",
			early)
	}
	// the hat, at two seconds, has nothing under a kilohertz.
	s = newSpectrum(seconds(p, 2, 2.05), p.Rate)
	if got := decibels(s.most(20, 1000) / s.most(6000, 16000)); got > -20 {
		t.Errorf("the hat's bottom is %.1f dB against its top", got)
	}
}
