package host

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/cheeptune"
	"github.com/janearc/slayer/interfaces/movie"
)

// The tune's numbers the package picked: the quasar slice, 2026-11-12.
const (
	// kickFall is how long a kick's strength takes to fall by e, in
	// seconds: about a frame and a half at twenty-four a second.
	kickFall = 0.06
	// loudWindow is the span, in seconds, over which loudness is
	// measured: a beat's worth of a fast tune.
	loudWindow = 0.1
	// beatSlack is how far short of a beat, in beats, a moment counts
	// as on it: far under a sample, over a float's rounding.
	beatSlack = 1e-9
	// stereo is defined: a sound's samples a moment, left and right.
	stereo = 2
)

// Tune is what the music does when: its beat from the score, its parts
// from the cut sheet, its loudness from its own sound, and which parts
// kick.
type Tune struct {
	score   *cheeptune.Score
	parts   []cheeptune.Cut
	loud    []float64
	rate    float64
	kicking map[string]bool
	sound   movie.Sound
}

// NewTune is a tune from its score, its cut sheet and its own sound, with
// the parts that kick named. It refuses a sheet with no parts, a part
// named to kick that the sheet lacks, and a sound with no samples.
func NewTune(score *cheeptune.Score, sheet *cheeptune.CutSheet,
	sound movie.Sound, kicking []string) (Tune, error) {
	if sheet == nil || len(sheet.Parts) == none {
		return Tune{}, fmt.Errorf("host: a tune with no parts: give " +
			"its cut sheet")
	}
	named := map[string]bool{}
	for _, part := range sheet.Parts {
		named[part.Name] = true
	}
	kicks := map[string]bool{}
	for _, name := range kicking {
		if !named[name] {
			return Tune{}, fmt.Errorf("host: no part is called %q: "+
				"name parts the sheet has", name)
		}
		kicks[name] = true
	}
	if sound.Length() == none {
		return Tune{}, fmt.Errorf("host: a tune with no sound: give " +
			"the tune's own")
	}
	tune := Tune{score: score, parts: sheet.Parts,
		rate: float64(sound.Rate()), kicking: kicks, sound: sound}
	tune.loud = loudness(sound, int(loudWindow*tune.rate))
	return tune, nil
}

// Sound is the tune's own sound.
func (tune Tune) Sound() movie.Sound { return tune.sound }

// Seconds is how long the tune is.
func (tune Tune) Seconds() float64 { return tune.sound.Duration().Seconds() }

// Part is the part playing at seconds in: the last that has started.
func (tune Tune) Part(seconds float64) string {
	playing := tune.parts[none].Name
	for _, part := range tune.parts {
		if part.Second <= seconds {
			playing = part.Name
		}
	}
	return playing
}

// At is the moment seconds in, the host speaking or not.
func (tune Tune) At(seconds float64, speaking bool) Moment {
	part := tune.Part(seconds)
	kick := float64(none)
	if tune.kicking[part] {
		// a moment on a beat may come back a hair short of it
		beats := tune.score.BeatsAt(seconds) + beatSlack
		since := (beats - math.Floor(beats)) * tune.beatSeconds(seconds)
		kick = math.Exp(-since / kickFall)
	}
	return Moment{Part: part, Kick: kick, Loudness: tune.loudAt(seconds),
		Speaking: speaking}
}

// beatSeconds is how long the beat is at seconds in.
func (tune Tune) beatSeconds(seconds float64) float64 {
	beats := tune.score.BeatsAt(seconds)
	return tune.score.SecondsAt(math.Floor(beats)+one) -
		tune.score.SecondsAt(math.Floor(beats))
}

// loudAt is the loudness at seconds in, a share of the loudest window.
func (tune Tune) loudAt(seconds float64) float64 {
	at := int(seconds * tune.rate)
	if at < none || at >= len(tune.loud) {
		return none
	}
	return tune.loud[at]
}

// loudness is each sample's loudness: the root mean square of both
// channels over a window ending there, as a share of the loudest.
func loudness(sound movie.Sound, window int) []float64 {
	samples := sound.Samples()
	count := len(samples) / stereo
	window = max(window, one)
	energy := make([]float64, count+one)
	for i := range count {
		left, right := float64(samples[stereo*i]),
			float64(samples[stereo*i+one])
		energy[i+one] = energy[i] + (left*left+right*right)/two
	}
	loud, top := make([]float64, count), float64(none)
	for i := range count {
		from := max(none, i+one-window)
		loud[i] = math.Sqrt((energy[i+one] - energy[from]) /
			float64(i+one-from))
		top = math.Max(top, loud[i])
	}
	if top > none {
		for i := range loud {
			loud[i] /= top
		}
	}
	return loud
}
