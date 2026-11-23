//go:build draw

package cheeptune

import (
	"bytes"
	"math"
	"testing"
)

func TestAnAccentOpensTheFilterAndShortensItsDecay(t *testing.T) {
	f := &filter{Cutoff: 500, Amount: 3, Accent: 1.5, Snap: 0.5,
		Envelope: &envelope{Decay: 0.2}}
	_, plain := f.opening(strike{loud: 1, gate: 1})
	run, loud := f.opening(strike{loud: marks["ff"], gate: 1})
	_, soft := f.opening(strike{loud: marks["pp"], gate: 1})
	if plain != 3 || math.Abs(loud-4.5) > 1e-9 || soft != 1.5 {
		t.Errorf("amounts: plain %v, ff %v, pp %v", plain, loud, soft)
	}
	if got := run.envelope.Decay; math.Abs(got-0.1) > 1e-9 {
		t.Errorf("an ff note's decay is %v, want half of 0.2", got)
	}
	if accent(0) != 0 || accent(1) != 0 || accent(0.001) != -1 {
		t.Errorf("accents: %v %v %v", accent(0), accent(1), accent(0.001))
	}
	if f.Envelope.Decay != 0.2 {
		t.Errorf("the accent changed the voicing's own envelope")
	}
	// heard: the same note marked ff has more top in its first tenth
	// of a second than its gain alone gives.
	acid := func(accent string) string {
		return `{"kind": "analog", "role": "bass", ` + held + `,
		"analog": {"oscillators": [{"shape": "saw"}],
		"filter": {"poles": 4, "cutoff": 300, "resonance": 0.5,
		"amount": 2` + accent + `, "envelope": {"attack": 0,
		"decay": 0.3, "sustain": 0, "release": 0.1}}}}`
	}
	top := func(channel, bar string) float64 {
		p := render(t, slow(`{"role": "bass", "bars": ["`+bar+`"]}`),
			studioVoicing(channel, ""))
		s := newSpectrum(seconds(p, 0.01, 0.1), p.Rate)
		return power(s, 2000, 6000) - power(s, 80, 400)
	}
	with := acid(`, "accent": 2`)
	if gain := top(with, "a2:4@ff") - top(with, "a2:4"); gain < 6 {
		t.Errorf("an accent brightens the note by %.1f dB", gain)
	}
	// the anti-test: with no accent set, a mark is gain and nothing
	// else, and the balance of top to bottom does not move.
	without := acid("")
	if gain := top(without, "a2:4@ff") - top(without, "a2:4"); math.Abs(gain) > 1 {
		t.Errorf("with no accent a mark moved the top by %.1f dB", gain)
	}
}

// sliding is a bass of one plain saw whose level falls to a fifth while
// a note is held, so a note that restarts is heard to.
const sliding = `{"kind": "analog", "role": "bass",
	"envelope": {"attack": 0.002, "decay": 0.3, "sustain": 0.2,
	"release": 0.05}, "analog": {"oscillators": [{"shape": "triangle"}]}}`

func TestASlideGlidesToTheNextNoteWithoutStartingItAgain(t *testing.T) {
	play := func(bar string) *Piece {
		return render(t, slow(`{"role": "bass", "bars": ["`+bar+`"]}`),
			studioVoicing(sliding, ""))
	}
	slid := play("a2:1> a3:1 -:2")
	pitch := func(p *Piece, from, to float64) float64 {
		return frequency(seconds(p, from, to), p.Rate)
	}
	if got := pitch(slid, 0.5, 0.9); math.Abs(got-110) > 1 {
		t.Errorf("before the slide the note is %.1f Hz", got)
	}
	// thirty milliseconds into a sixty millisecond glide it is between.
	if got := pitch(slid, 1.02, 1.04); got < 125 || got > 200 {
		t.Errorf("half way through the glide the note is %.1f Hz", got)
	}
	if got := pitch(slid, 1.3, 1.9); math.Abs(got-220) > 1 {
		t.Errorf("after the slide the note is %.1f Hz", got)
	}
	// it did not start again: it is as quiet after as before.
	before, after := rms(seconds(slid, 0.8, 0.95)), rms(seconds(slid, 1.1, 1.2))
	if after > before*1.2 {
		t.Errorf("the slid note started again: %.4f then %.4f", before,
			after)
	}
	// the anti-test: the same notes with no slide start again, at
	// once, and loud.
	struck := play("a2:1 a3:1 -:2")
	if got := pitch(struck, 1.02, 1.06); math.Abs(got-220) > 8 {
		t.Errorf("with no slide the second note begins at %.1f Hz", got)
	}
	if rms(seconds(struck, 1.02, 1.1)) < 2*rms(seconds(struck, 0.8, 0.95)) {
		t.Errorf("with no slide the second note did not start again")
	}
}

func TestAChannelsGlideIsHowLongItsSlidesTake(t *testing.T) {
	// with a glide of four tenths of a second, two tenths in the note
	// is still between; with the sixty milliseconds a channel has
	// without one, it arrived long before.
	long := `{"kind": "analog", "role": "bass", "glide": 0.4, ` + held +
		`, "analog": {"oscillators": [{"shape": "triangle"}]}}`
	short := `{"kind": "analog", "role": "bass", ` + held +
		`, "analog": {"oscillators": [{"shape": "triangle"}]}}`
	at := func(channel string) float64 {
		p := render(t, slow(`{"role": "bass", "bars": ["a2:1> a3:3"]}`),
			studioVoicing(channel, ""))
		return frequency(seconds(p, 1.18, 1.22), p.Rate)
	}
	if got := at(long); got < 140 || got > 175 {
		t.Errorf("half way through a long glide the note is %.1f Hz", got)
	}
	if got := at(short); math.Abs(got-220) > 8 {
		t.Errorf("long after a short glide the note is %.1f Hz", got)
	}
}

func TestWhatCannotSlidePlaysASlidNoteAsANewOne(t *testing.T) {
	plain, slid := "a2:2 a3:2", "a2:2> a3:2"
	// the Game Boy: byte for byte the same.
	bass := `{"kind": "wave", "role": "bass", "table": "triangle"}`
	a := render(t, slow(`{"role": "bass", "bars": ["`+plain+`"]}`),
		voicing(bass, ""))
	b := render(t, slow(`{"role": "bass", "bars": ["`+slid+`"]}`),
		voicing(bass, ""))
	if !bytes.Equal(pcmBytes(a.Left), pcmBytes(b.Left)) {
		t.Errorf("a slide changed what the Game Boy played")
	}
	// a plucked string: the same.
	string := studioChannelOf("pluck", "bass", goodPluck)
	a = render(t, slow(`{"role": "bass", "bars": ["`+plain+`"]}`),
		studioVoicing(string, ""))
	b = render(t, slow(`{"role": "bass", "bars": ["`+slid+`"]}`),
		studioVoicing(string, ""))
	if !bytes.Equal(pcmBytes(a.Left), pcmBytes(b.Left)) {
		t.Errorf("a slide changed what a plucked string played")
	}
	// the anti-test: on a channel that slides, it does change.
	a = render(t, slow(`{"role": "bass", "bars": ["`+plain+`"]}`),
		studioVoicing(sliding, ""))
	b = render(t, slow(`{"role": "bass", "bars": ["`+slid+`"]}`),
		studioVoicing(sliding, ""))
	if bytes.Equal(pcmBytes(a.Left), pcmBytes(b.Left)) {
		t.Errorf("a slide changed nothing on an analog channel")
	}
}

func TestFoldAndOverdriveDistortAndStayInsideFullScale(t *testing.T) {
	voice := func(extra string) string {
		return `{"kind": "analog", "role": "bass", ` + held + `,
			"analog": {"oscillators": [{"shape": "triangle"}]` + extra +
			`}}`
	}
	play := func(extra string) *Piece {
		v := []byte(`{"name": "v", "machine": "studio", "master": 1,
			"channels": [` + voice(extra) + `]}`)
		return render(t, slow(`{"role": "bass", "bars": ["a2:4"]}`), v)
	}
	top := func(p *Piece) float64 {
		s := newSpectrum(seconds(p, 1, 2), p.Rate)
		// the overtones from the third to the thirteenth, against the
		// note itself. a soft clipper is smooth, so what it adds is
		// here and not far up.
		return power(s, 300, 1500) - power(s, 80, 150)
	}
	plain := play("")
	for _, extra := range []string{`, "overdrive": 1`, `, "fold": 1`,
		`, "fold": 0.5, "overdrive": 0.5`} {
		p := play(extra)
		// centred, a channel at full scale is 0.707 a side.
		if p.Peak > centre*1.02 {
			t.Errorf("%s peaks at %.3f, over a full-scale channel",
				extra, p.Peak)
		}
		if gain := top(p) - top(plain); gain < 6 {
			t.Errorf("%s adds %.1f dB of overtones to a triangle", extra,
				gain)
		}
	}
	// overdrive makes a quiet thing loud; that is what it is for.
	if over := play(`, "overdrive": 1`); rms(seconds(over, 1, 2)) <
		1.2*rms(seconds(plain, 1, 2)) {
		t.Errorf("overdrive did not thicken the note")
	}
	// the anti-test: with neither, the channel is exactly as before.
	if a, b := play(""), play(`, "fold": 0, "overdrive": 0`); !bytes.Equal(
		pcmBytes(a.Left), pcmBytes(b.Left)) {
		t.Errorf("nought of each changed the sound")
	}
}

func TestAStruckDrumIsMetalAndAPlainOneIsATone(t *testing.T) {
	kit := func(struck string) string {
		return `{"kind": "drum", "role": "perc", "drums": {
			"k": {"tone": 400, "decay": 0.5` + struck + `}}}`
	}
	second := func(channel string) float64 {
		p := render(t, slow(`{"role": "perc", "bars": ["k -:3"]}`),
			studioVoicing(channel, ""))
		s := newSpectrum(seconds(p, 0, 0.2), p.Rate)
		own, _ := s.near(400, 15)
		mode, _ := s.near(400*plateRatios[1], 15)
		return decibels(mode / own)
	}
	if got := second(kit(`, "struck": "plate", "hardness": 0.9`)); got < -15 {
		t.Errorf("a struck plate's second mode is %.1f dB", got)
	}
	if got := second(kit("")); got > -40 {
		t.Errorf("a plain tone has something at the plate's mode: %.1f dB",
			got)
	}
}
