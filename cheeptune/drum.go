package cheeptune

import (
	"math"
	"sort"
	"strings"
)

// The drum engine: a hit is a tone whose pitch falls, a click at its
// front, and noise through a high-pass filter, each with its own decay.
// A kick is the tone and the click; a hat is the noise alone; a snare is
// both; a tom is the tone with a small fall.

// drumMeanings are the hits a studio drum channel may have, and what each
// is. A letter means the same kind of hit in every studio voicing, so a
// score's drums move between voicings without being rewritten: a voicing
// decides what a kick sounds like, not which letter is the kick. Picked
// by the writer of the studio, 2026-11-12, after a composer found one
// letter was a tom in one voicing and an open hat in another.
var drumMeanings = map[string]string{
	"k": "the kick, or what stands in for it: the low, heavy hit",
	"s": "the snare or clap: the hit on the back beat",
	"h": "the closed hat or a tick: short and high",
	"o": "the open hat: high, and longer than h",
	"t": "the tom: a pitched drum between the kick and the snare",
	"b": "the bell: a pitched or metallic accent that rings a little",
}

// drumLetters is the letters in the order a kit is described.
var drumLetters = []string{"k", "s", "h", "o", "t", "b"}

// drum is one hit on a drum channel, named by the letter the score uses.
type drum struct {
	Tone       float64  `json:"tone"`        // hertz the tone lands on
	Sweep      float64  `json:"sweep"`       // semitones it starts above
	Fall       float64  `json:"fall"`        // seconds the fall takes
	Decay      float64  `json:"decay"`       // seconds the tone rings
	Click      float64  `json:"click"`       // nought to one
	Noise      float64  `json:"noise"`       // nought to one
	NoiseDecay float64  `json:"noise_decay"` // seconds the noise rings
	Colour     float64  `json:"colour"`      // hertz the noise is cut below
	Volume     *float64 `json:"volume"`

	// Struck names a mode set (modes.go). With it the hit's tone is not
	// a sine but that object struck at the tone's pitch, as hard as
	// hardness says, ringing for the decay: a tick, a ratchet, an anvil.
	Struck   string  `json:"struck"`
	Hardness float64 `json:"hardness"`
}

// The ranges of a hit. Picked, as the rest of the studio's are.
const (
	drumToneLeast   = 20
	drumToneMost    = 2000
	drumSweepMost   = 60
	drumTimeLeast   = 0.001
	drumTimeMost    = 8
	drumColourLeast = 20
	drumColourMost  = 16000
	// clickSeconds is how long a click lasts.
	clickSeconds = 0.003
	// fallSharpness is how many time constants a fall's seconds hold.
	fallSharpness = 5
	// drumBite is the high-pass filter's resonance setting, flat.
	drumBite = 0
)

// init puts the drum engine in the studio's table.
func init() {
	register("drum", engineRow{percussive: true,
		shaped: "each drum carries its own decay",
		check:  checkDrums, build: buildDrums})
}

// drumNames lists a channel's hits, sorted.
func drumNames(drums map[string]drum) []string {
	names := make([]string, 0, len(drums))
	for name := range drums {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// checkDrums is a drum channel's own rules.
func checkDrums(c *channel, q *quarrel) {
	if len(c.Drums) == 0 {
		q.fail("a drum channel with no drums plays nothing")
	}
	for _, name := range drumNames(c.Drums) {
		d := c.Drums[name]
		at := "drum " + name
		if _, ok := drumMeanings[name]; !ok {
			q.fail("%s: a studio drum is one of %s (kick, snare, "+
				"closed hat, open hat, tom, bell), so a score's drums "+
				"mean the same in every voicing", at,
				strings.Join(drumLetters, ", "))
		}
		if d.Tone == 0 && d.Noise == 0 {
			q.fail("%s has neither a tone nor noise and is silent", at)
		}
		if d.Tone != 0 {
			q.within(at+" tone", d.Tone, drumToneLeast, drumToneMost)
			q.within(at+" decay", d.Decay, drumTimeLeast, drumTimeMost)
		}
		if _, ok := modeSets[d.Struck]; d.Struck != "" && !ok {
			q.fail("%s struck %q; sets are %s", at, d.Struck, setNames())
		}
		if d.Struck != "" && (d.Tone == 0 || d.Sweep != 0) {
			q.fail("%s: a struck drum needs a tone and takes no sweep",
				at)
		}
		if d.Struck == "" && d.Hardness != 0 {
			q.fail("%s: hardness is for a struck drum", at)
		}
		q.within(at+" hardness", d.Hardness, 0, 1)
		q.within(at+" sweep", d.Sweep, 0, drumSweepMost)
		if d.Sweep != 0 {
			q.within(at+" fall", d.Fall, drumTimeLeast, drumTimeMost)
		}
		q.within(at+" click", d.Click, 0, 1)
		q.within(at+" noise", d.Noise, 0, 1)
		if d.Noise != 0 {
			q.within(at+" noise_decay", d.NoiseDecay, drumTimeLeast,
				drumTimeMost)
			q.within(at+" colour", d.Colour, drumColourLeast,
				drumColourMost)
		}
		if d.Volume != nil {
			q.within(at+" volume", *d.Volume, 0, 1)
		}
	}
}

// drumEngine is a drum channel ready to sound.
type drumEngine struct{ drums map[string]drum }

// buildDrums is the engine for a checked drum channel.
func buildDrums(c *channel) engine { return &drumEngine{drums: c.Drums} }

// ring is how long the hit lasts past its written length, which a hit
// ignores: it lasts as long as its longest decay.
func (e *drumEngine) ring(st strike) float64 {
	d := e.drums[st.hit]
	return math.Max(0, math.Max(d.Decay, d.NoiseDecay)-st.gate)
}

// sound renders one hit.
func (e *drumEngine) sound(st strike, out []float64) {
	d := e.drums[st.hit]
	volume := 1.0
	if d.Volume != nil {
		volume = *d.Volume
	}
	source := newNoise(st.seed)
	cut := newStateFilter()
	cut.tune(math.Max(d.Colour, drumColourLeast), drumBite)
	phase := 0.0
	var bank []resonator
	var blow []float64
	if d.Struck != "" {
		object := &modalEngine{modal: &modal{Hardness: d.Hardness},
			modes: setModes(modeSets[d.Struck], d.Decay)}
		bank, blow = object.bank(d.Tone), object.blow()
	}
	for i := range out {
		age := float64(i) / studioRate
		sum := 0.0
		if bank != nil {
			x := 0.0
			if i < len(blow) {
				x = blow[i]
			}
			for k := range bank {
				sum += bank[k].step(x)
			}
		} else if d.Tone > 0 {
			above := d.Sweep
			if d.Fall > 0 {
				above *= math.Exp(-fallSharpness * age / d.Fall)
			}
			phase += d.Tone * math.Exp2(above/12) / studioRate
			sum += sine(phase) * math.Exp(-sixtyDecibels*age/d.Decay)
		}
		hiss := source.next()
		if age < clickSeconds {
			sum += d.Click * hiss * (1 - age/clickSeconds)
		}
		if d.Noise > 0 {
			_, _, high := cut.step(hiss)
			sum += d.Noise * high *
				math.Exp(-sixtyDecibels*age/d.NoiseDecay)
		}
		out[i] = sum * volume
	}
}
