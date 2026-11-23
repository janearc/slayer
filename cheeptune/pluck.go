package cheeptune

import "math"

// The pluck engine: a string, by the method of Karplus and Strong as
// Jaffe and Smith extended it. A delay line one period long is filled with
// a burst of noise and fed back into itself through a filter that takes a
// little off the top each time round, so the burst settles into a tone
// that dies away, the high end first.
//
// Three things make the pitch right. The loop's filter delays the sound
// by a part of a sample, which is taken off the line's length. The line
// holds a whole number of samples, so the part of a sample left over is
// made by an allpass filter. And both are worked out at the note's own
// frequency, not as low-frequency approximations. A test holds the pitch.

// pluck is a pluck channel's sound. Brightness is how much of the top the
// pluck itself has, nought (a thumb) to one (a pick). Damping is how fast
// the high end dies against the low, nought to one.
//
// Position is where along the string it is plucked, from nought (at the
// end) to a half (the middle, which takes out every even overtone).
// Decay is the seconds the note takes to fall sixty decibels while held,
// and release the seconds once it is let go.
type pluck struct {
	Brightness float64 `json:"brightness"`
	Damping    float64 `json:"damping"`
	Position   float64 `json:"position"`
	Decay      float64 `json:"decay"`
	Release    float64 `json:"release"`
}

// The ranges of a pluck. Picked.
const (
	pluckTimeLeast = 0.01
	// pluckDullest and pluckBrightest are the cutoff of the filter the
	// burst goes through at brightness nought and one, in hertz.
	pluckDullest   = 300
	pluckBrightest = 20000
	// pluckDampingMost is the loop filter's weight on the sample before
	// at damping one: a half, the two-sample average Karplus and Strong
	// used. Defined by their method.
	pluckDampingMost = 0.5
	// allpassLeast is the least part of a sample the allpass is asked
	// to make; under it the filter's pole nears the unit circle. Picked.
	allpassLeast = 0.1
)

// init puts the pluck engine in the studio's table.
func init() {
	register("pluck", engineRow{
		shaped: "a string has its own decay and release",
		check:  checkPluck, build: buildPluck})
}

// checkPluck is a pluck channel's own rules.
func checkPluck(c *channel, q *quarrel) {
	p := c.Pluck
	q.within("pluck brightness", p.Brightness, 0, 1)
	q.within("pluck damping", p.Damping, 0, 1)
	q.within("pluck position", p.Position, 0, 0.5)
	q.within("pluck decay", p.Decay, pluckTimeLeast, envelopeMost)
	q.within("pluck release", p.Release, pluckTimeLeast, envelopeMost)
}

// pluckEngine is a pluck channel ready to sound.
type pluckEngine struct{ pluck *pluck }

// buildPluck is the engine for a checked pluck channel.
func buildPluck(c *channel) engine { return &pluckEngine{pluck: c.Pluck} }

// ring is the string's release.
func (e *pluckEngine) ring(strike) float64 { return e.pluck.Release }

// stringLoop is a string tuned to a note: the line's length, the loop
// filter's weight and what it does at the note's frequency, and the
// allpass that makes up the part of a sample.
type stringLoop struct {
	length  int
	weight  float64 // the loop filter's weight on the sample before
	loss    float64 // the loop filter's gain at the note
	allpass float64
	period  float64 // seconds
}

// newStringLoop tunes a string to a frequency.
func newStringLoop(hertz, damping float64) stringLoop {
	turn := 2 * math.Pi * hertz / studioRate // radians a sample
	weight := damping * pluckDampingMost
	real := (1 - weight) + weight*math.Cos(turn)
	imaginary := weight * math.Sin(turn)
	late := math.Atan2(imaginary, real) / turn // samples
	whole := studioRate/hertz - late
	length := int(math.Floor(whole - allpassLeast))
	part := whole - float64(length)
	return stringLoop{length: length, weight: weight,
		loss:    math.Hypot(real, imaginary),
		allpass: math.Sin((1-part)*turn/2) / math.Sin((1+part)*turn/2),
		period:  1 / hertz}
}

// gain is what the loop is turned down by each time round so the note
// falls sixty decibels in so many seconds. It is never above one: a
// string whose filter alone kills it faster than asked dies at the
// filter's pace.
func (l stringLoop) gain(seconds float64) float64 {
	want := math.Exp(-sixtyDecibels * l.period / seconds)
	return math.Min(1, want/l.loss)
}

// burst is what the string starts with: one period of noise, dulled by
// the brightness, with the echo of the pluck's position taken out, scaled
// to a peak of one.
func (e *pluckEngine) burst(st strike, length int) []float64 {
	source := newNoise(st.seed)
	cutoff := pluckDullest *
		math.Pow(pluckBrightest/pluckDullest, e.pluck.Brightness)
	pole := math.Exp(-2 * math.Pi * cutoff / studioRate)
	raw := make([]float64, length)
	state := 0.0
	// twice round, so the filter's state at the start is the state the
	// end leaves and the burst has no step in it.
	for range 2 {
		for i := range raw {
			state = source.next()*(1-pole) + state*pole
			raw[i] = state
		}
	}
	apart := int(math.Round(e.pluck.Position * float64(length)))
	out := make([]float64, length)
	peak := 0.0
	for i := range out {
		out[i] = raw[i]
		if apart > 0 {
			out[i] -= raw[(i+length-apart)%length]
		}
		peak = math.Max(peak, math.Abs(out[i]))
	}
	for i := range out {
		out[i] /= peak
	}
	return out
}

// sound renders one note.
func (e *pluckEngine) sound(st strike, out []float64) {
	loop := newStringLoop(st.hz, e.pluck.Damping)
	line := e.burst(st, loop.length)
	held := loop.gain(e.pluck.Decay)
	free := math.Min(held, loop.gain(e.pluck.Release))
	letGo := int(st.gate * studioRate)
	at := 0
	var before, passedIn, passedOut float64
	for i := range out {
		y := line[at]
		out[i] = y
		dulled := (1-loop.weight)*y + loop.weight*before
		before = y
		passed := loop.allpass*dulled + passedIn - loop.allpass*passedOut
		passedIn, passedOut = dulled, passed
		gain := held
		if i >= letGo {
			gain = free
		}
		line[at] = passed * gain
		at++
		if at == loop.length {
			at = 0
		}
	}
}
