package cheeptune

import (
	"math"
)

// The analog engine: one or two oscillators and some noise, summed, driven
// if asked, through a resonant low-pass filter whose cutoff follows the
// note and is opened by its own envelope, then shaped by the channel's
// envelope, folded and overdriven if asked on the way.
//
// With one saw or square, a four-pole filter near the top of its
// resonance, a short filter envelope, accents and slides, it is an acid
// bass line.
//
// The oscillators are band-limited by correcting each jump (a saw's fall,
// a square's two edges) and each corner (a triangle's two) with a short
// polynomial, the method called PolyBLEP, so a high note does not fold its
// overtones back under itself. A test holds how little is left.

// analog is an analog channel's sound.
type analog struct {
	Oscillators []oscillator `json:"oscillators"`
	Noise       float64      `json:"noise"`
	Drive       float64      `json:"drive"`
	Filter      *filter      `json:"filter"`

	// Fold and Overdrive distort the sound after the filter and before
	// the channel's envelope, so a distorted note still swells and dies
	// as its envelope says. Fold bends the wave back on itself, which is
	// harsh and buzzing; overdrive pushes it into a soft clipper, which
	// is thick and loud. Both give full scale out for full scale in.
	Fold      float64 `json:"fold"`
	Overdrive float64 `json:"overdrive"`
}

// oscillator is one band-limited wave: saw, square (with a width) or
// triangle, moved by cents and by whole octaves. Levels are a balance:
// the oscillators together are as loud as one.
type oscillator struct {
	Shape  string   `json:"shape"`
	Width  *float64 `json:"width"`
	Detune float64  `json:"detune"`
	Octave int      `json:"octave"`
	Level  *float64 `json:"level"`
}

// filter is the low-pass the oscillators go through. Cutoff is in hertz
// for middle c; follow is how much of the note's distance from middle c
// the cutoff moves with it, nought to one; amount is how many octaves the
// filter's envelope opens the cutoff at its peak (or closes it, below
// nought).
type filter struct {
	Cutoff    float64   `json:"cutoff"`
	Resonance float64   `json:"resonance"`
	Follow    float64   `json:"follow"`
	Amount    float64   `json:"amount"`
	Envelope  *envelope `json:"envelope"`

	// Poles is two (a gentle slope, twelve decibels an octave; the
	// default) or four (the ladder: twice as steep, and its resonance
	// squelches). Accent is how many more octaves the envelope opens
	// the cutoff on a note marked ff, and a soft mark closes it by its
	// share. Snap is how much shorter the envelope's decay is on a note
	// marked ff, nought to one.
	Poles  int     `json:"poles"`
	Accent float64 `json:"accent"`
	Snap   float64 `json:"snap"`
}

// shape is one cycle of a wave at a phase from nought to one, given the
// phase's step a sample and a square's width.
type shape func(phase, step, width float64) float64

// shapes are the waves an oscillator has.
var shapes = map[string]shape{
	"saw":      sawWave,
	"square":   squareWave,
	"triangle": triangleWave,
}

// The ranges of an analog channel. Picked.
const (
	oscillatorsMost  = 2
	widthLeast       = 0.05
	widthMost        = 0.95
	octaveMost       = 3
	cutoffLeast      = 20
	cutoffMost       = 20000
	filterAmountMost = 8
	// driveMost is how many times over the signal is pushed into the
	// soft clipper at drive one.
	driveMost = 8
	// middleC is the note a filter's cutoff is written for. Defined.
	middleC = 60
	// foldMost is how many quarter turns past the first a wave at full
	// scale is bent through at fold one.
	foldMost = 4
	// snapMost is the most of a decay an accent takes away.
	snapMost = 0.95
)

// init puts the analog engine in the studio's table.
func init() {
	register("analog", engineRow{envelope: true, vibrato: true,
		check: checkAnalog, build: buildAnalog})
}

// checkAnalog is an analog channel's own rules.
func checkAnalog(c *channel, q *quarrel) {
	a := c.Analog
	if n := len(a.Oscillators); n < 1 || n > oscillatorsMost {
		q.fail("analog has %d oscillators; it takes 1 or %d", n,
			oscillatorsMost)
	}
	for i, o := range a.Oscillators {
		name := "analog oscillator " + string(rune('1'+i))
		if _, ok := shapes[o.Shape]; !ok {
			q.fail("%s shape %q; shapes are saw, square, triangle",
				name, o.Shape)
		}
		if o.Width != nil {
			if o.Shape != "square" {
				q.fail("%s: width is only for a square", name)
			}
			q.within(name+" width", *o.Width, widthLeast, widthMost)
		}
		q.within(name+" detune", o.Detune, -detuneMost, detuneMost)
		q.within(name+" octave", float64(o.Octave), -octaveMost,
			octaveMost)
		if o.Level != nil {
			q.within(name+" level", *o.Level, 0, 1)
		}
	}
	q.within("analog noise", a.Noise, 0, 1)
	q.within("analog drive", a.Drive, 0, 1)
	q.within("analog fold", a.Fold, 0, 1)
	q.within("analog overdrive", a.Overdrive, 0, 1)
	if f := a.Filter; f != nil {
		q.within("filter cutoff", f.Cutoff, cutoffLeast, cutoffMost)
		q.within("filter resonance", f.Resonance, 0, 1)
		q.within("filter follow", f.Follow, 0, 1)
		q.within("filter amount", f.Amount, -filterAmountMost,
			filterAmountMost)
		checkEnvelope(q, "filter envelope", f.Envelope)
		if f.Amount != 0 && f.Envelope == nil {
			q.fail("filter amount needs a filter envelope to move by")
		}
		if f.Poles != 0 && f.Poles != 2 && f.Poles != 4 {
			q.fail("filter poles %d; a filter has 2 or 4", f.Poles)
		}
		q.within("filter accent", f.Accent, 0, filterAmountMost)
		q.within("filter snap", f.Snap, 0, snapMost)
		if (f.Accent != 0 || f.Snap != 0) && f.Envelope == nil {
			q.fail("filter accent and snap need a filter envelope " +
				"to act on")
		}
	}
}

// polyStep is the correction for a jump of two down at phase nought: a
// short polynomial either side of the jump, nought elsewhere.
func polyStep(phase, step float64) float64 {
	switch {
	case phase < step:
		x := phase / step
		return x + x - x*x - 1
	case phase > 1-step:
		x := (phase - 1) / step
		return x*x + x + x + 1
	}
	return 0
}

// polyCorner is the correction for a corner at phase nought, the integral
// of polyStep's.
func polyCorner(phase, step float64) float64 {
	switch {
	case phase < step:
		x := phase/step - 1
		return -x * x * x / 3
	case phase > 1-step:
		x := (phase-1)/step + 1
		return x * x * x / 3
	}
	return 0
}

// wrapped is a phase moved on and brought back inside nought to one.
func wrapped(phase float64) float64 { return phase - math.Floor(phase) }

// sawWave rises from minus one to one and jumps back.
func sawWave(phase, step, _ float64) float64 {
	return 2*phase - 1 - polyStep(phase, step)
}

// squareWave is one for the width's share of the cycle and minus one for
// the rest.
func squareWave(phase, step, width float64) float64 {
	naive := -1.0
	if phase < width {
		naive = 1
	}
	return naive + polyStep(phase, step) -
		polyStep(wrapped(phase+1-width), step)
}

// triangleWave rises from minus one to one over the first half and falls
// back over the second.
func triangleWave(phase, step, _ float64) float64 {
	naive := 4*phase - 1
	if phase >= 0.5 {
		naive = 3 - 4*phase
	}
	return naive + 4*step*(polyCorner(phase, step)-
		polyCorner(wrapped(phase+0.5), step))
}

// analogEngine is an analog channel ready to sound.
type analogEngine struct {
	channel *channel
	waves   []shape
	ratios  []float64 // each oscillator's frequency over the note's
	levels  []float64 // each oscillator's share of the sum
	widths  []float64
}

// buildAnalog is the engine for a checked analog channel.
func buildAnalog(c *channel) engine {
	e := &analogEngine{channel: c}
	total := 0.0
	for _, o := range c.Analog.Oscillators {
		level, width := 1.0, 0.5
		if o.Level != nil {
			level = *o.Level
		}
		if o.Width != nil {
			width = *o.Width
		}
		e.waves = append(e.waves, shapes[o.Shape])
		e.ratios = append(e.ratios,
			math.Exp2(float64(o.Octave)+o.Detune/centsAnOctave))
		e.levels = append(e.levels, level)
		e.widths = append(e.widths, width)
		total += level
	}
	for i := range e.levels {
		if total > 0 {
			e.levels[i] /= total
		}
	}
	return e
}

// ring is the channel envelope's release.
func (e *analogEngine) ring(strike) float64 {
	return orGated(e.channel.Envelope).Release
}

// cutoff is the filter's cutoff for a note with its envelope opening it
// by so many octaves.
func (f *filter) cutoff(midi int, opened float64) float64 {
	octaves := f.Follow*float64(midi-middleC)/12 + opened
	return math.Max(cutoffLeast, f.Cutoff*math.Exp2(octaves))
}

// filterBlock is how many samples the filter keeps a cutoff before it is
// tuned again. Picked: eight, so a fast envelope on a ringing filter
// moves without a zipper.
const filterBlock = 8

// accent is how hard a note is accented by its loudness mark: one at
// ff, nought with no mark, and below nought for a soft mark, no lower
// than minus one.
func accent(loud float64) float64 {
	if loud <= 0 {
		return 0
	}
	return math.Max(-1, math.Log(loud)/math.Log(marks["ff"]))
}

// opening is the filter's envelope and amount for one note: as written,
// moved by the note's accent.
func (f *filter) opening(st strike) (*envelopeRun, float64) {
	hard := accent(st.loud)
	amount := f.Amount + f.Accent*hard
	envelope := f.Envelope
	if envelope != nil && f.Snap != 0 && hard > 0 {
		snapped := *envelope
		snapped.Decay *= 1 - f.Snap*hard
		envelope = &snapped
	}
	return newEnvelopeRun(envelope, st.gate), amount
}

// newLowPass is the filter a channel asks for, at rest.
func (f *filter) newLowPass() lowPass {
	if f.Poles == 4 {
		return &ladderFilter{}
	}
	return newStateFilter()
}

// after is the distortion after the filter: the fold, then the
// overdrive, each averaged over the step so it folds little back, each
// scaled so full scale in is full scale out.
//
// It was first put after the channel's envelope too. Measured on a tune
// that flattened every note into a steady buzz (the lead's peak stood
// 9 dB over its mean, against 13 for the driven voicing that was kept),
// it was tossed. Before the envelope, a note keeps its shape.
type after struct {
	fold, over  *softClip
	turns, push float64
	over1       float64
}

// newAfter is the distortion a channel asks for, or nil for none.
func (a *analog) newAfter() *after {
	if a.Fold == 0 && a.Overdrive == 0 {
		return nil
	}
	d := &after{}
	if a.Fold > 0 {
		d.turns = math.Pi / 2 * (1 + a.Fold*foldMost)
		d.fold = &softClip{curve: folder(d.turns)}
	}
	if a.Overdrive > 0 {
		d.push = 1 + a.Overdrive*driveMost
		d.over = &softClip{curve: tangent}
		d.over1 = 1 / math.Tanh(d.push)
	}
	return d
}

// step takes one sample and gives it distorted.
func (d *after) step(x float64) float64 {
	if d.fold != nil {
		x = d.fold.step(x)
	}
	if d.over != nil {
		x = d.over.step(x*d.push) * d.over1
	}
	return x
}

// sound renders one note.
func (e *analogEngine) sound(st strike, out []float64) {
	a := e.channel.Analog
	amp := newEnvelopeRun(e.channel.Envelope, st.gate)
	bend := newBendRun(e.channel.Vibrato, &st)
	hiss := newNoise(st.seed)
	var filter lowPass
	var opens *envelopeRun
	amount := 0.0
	if a.Filter != nil {
		filter = a.Filter.newLowPass()
		opens, amount = a.Filter.opening(st)
	}
	push := 1 + a.Drive*driveMost
	clip := &softClip{curve: tangent}
	distort := a.newAfter()
	// each oscillator starts a little apart, so two of them do not
	// begin every note as one.
	phases := make([]float64, len(e.waves))
	for k := range phases {
		phases[k] = wrapped(float64(k) * oscillatorSpread)
	}
	for i := range out {
		slid := bend.next()
		bent := st.hz * slid / studioRate
		sum := a.Noise * hiss.next()
		for k, wave := range e.waves {
			step := bent * e.ratios[k]
			sum += e.levels[k] * wave(phases[k], step, e.widths[k])
			phases[k] = wrapped(phases[k] + step)
		}
		if a.Drive > 0 {
			sum = clip.step(sum * push)
		}
		if filter != nil {
			opened := opens.next()
			if i%filterBlock == 0 {
				filter.tune(a.Filter.cutoff(st.midi, amount*opened)*
					math.Pow(slid, a.Filter.Follow), a.Filter.Resonance)
			}
			sum = filter.low(sum)
		}
		if distort != nil {
			sum = distort.step(sum)
		}
		out[i] = sum * amp.next()
	}
}

// oscillatorSpread is how far apart in the cycle a channel's oscillators
// start. Picked: a little over a third, so two never start together.
const oscillatorSpread = 0.37

// softClip is the drive stage: a hyperbolic tangent, which bends a loud
// signal over smoothly and so adds overtones.
//
// Bending a signal adds overtones without limit, and the ones past half
// the rate fold back. So the tangent is not taken at each sample. It is
// averaged over the straight line from the sample before to this one,
// which is its antiderivative's difference over the step (the method
// called antiderivative anti-aliasing, first order). A test holds what
// that leaves against the plain tangent.
type softClip struct {
	curve  curve
	before float64 // the sample before
	area   float64 // the antiderivative there
	primed bool
}

// clipFlat is the step under which the line is too short to divide by,
// and the tangent at its middle is used. Picked.
const clipFlat = 1e-6

// tangentArea is the antiderivative of the hyperbolic tangent, the log
// of the hyperbolic cosine, written so a large input does not overflow.
func tangentArea(x float64) float64 {
	size := math.Abs(x)
	return size + math.Log1p(math.Exp(-2*size)) - math.Ln2
}

// step takes one sample and gives it clipped.
func (c *softClip) step(x float64) float64 {
	if !c.primed {
		c.area, c.primed = c.curve.area(0), true
	}
	area := c.curve.area(x)
	out := c.curve.shape((x + c.before) / 2)
	if run := x - c.before; math.Abs(run) > clipFlat {
		out = (area - c.area) / run
	}
	c.before, c.area = x, area
	return out
}
