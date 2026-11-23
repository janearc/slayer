package cheeptune

import "math"

// The studio's envelope: the same four numbers as the Game Boy's, read as
// a curve and not as straight lines, because a struck or plucked thing
// dies away by a ratio and not by a ruler.
//
// Attack is a straight rise to full. Decay is the seconds the level takes
// to cover all but a thousandth (sixty decibels) of the way from full to
// the sustain. Release is the seconds it takes to fall sixty decibels from
// wherever it was when the note was let go, after which it is silent.

// gated is the envelope of a channel that gives none.
var gated = &envelope{Attack: gateAttack, Sustain: 1, Release: gateRelease}

// orGated is the envelope, or the gate when there is none.
func orGated(e *envelope) *envelope {
	if e == nil {
		return gated
	}
	return e
}

// held is the level at age seconds of a note still held.
func (e *envelope) held(age float64) float64 {
	if age < e.Attack {
		return age / e.Attack
	}
	if e.Decay <= 0 {
		return e.Sustain
	}
	fall := math.Exp(-sixtyDecibels * (age - e.Attack) / e.Decay)
	return e.Sustain + (1-e.Sustain)*fall
}

// level is the level at age seconds of a note held for gate seconds.
func (e *envelope) level(age, gate float64) float64 {
	if age < gate {
		return e.held(age)
	}
	since := age - gate
	if since >= e.Release {
		return 0
	}
	return e.held(gate) * math.Exp(-sixtyDecibels*since/e.Release)
}

// envelopeRun steps an envelope a sample at a time, working the curve out
// once a block and moving in a straight line between.
type envelopeRun struct {
	envelope *envelope
	gate     float64
	sample   int
	level    float64
	step     float64
}

// newEnvelopeRun is an envelope at the first sample of a note.
func newEnvelopeRun(e *envelope, gate float64) *envelopeRun {
	e = orGated(e)
	return &envelopeRun{envelope: e, gate: gate, level: e.level(0, gate)}
}

// next is the level at this sample; it then moves on one.
func (r *envelopeRun) next() float64 {
	if r.sample%envelopeBlock == 0 {
		ahead := float64(r.sample+envelopeBlock) / studioRate
		target := r.envelope.level(ahead, r.gate)
		r.step = (target - r.level) / envelopeBlock
	}
	now := r.level
	r.level += r.step
	r.sample++
	return now
}

// bendRun is a channel's vibrato as a ratio on the frequency, stepped the
// way an envelope is.
type bendRun struct {
	vibrato *vibrato
	strike  *strike // for its slides; nil when the note has none
	sample  int
	ratio   float64
	step    float64
}

// newBendRun is a note's vibrato and slides at its first sample; with
// neither the ratio stays at one.
func newBendRun(vb *vibrato, st *strike) *bendRun {
	b := &bendRun{vibrato: vb, ratio: 1}
	if len(st.path) > 0 {
		b.strike = st
	}
	return b
}

// at is the ratio at age seconds, worked out exactly.
func (b *bendRun) at(age float64) float64 {
	ratio := 1.0
	if b.strike != nil {
		ratio, _ = b.strike.slideAt(age)
	}
	vb := b.vibrato
	if vb == nil || age <= vb.Delay {
		return ratio
	}
	cents := vb.Depth * math.Sin(2*math.Pi*vb.Rate*(age-vb.Delay))
	return ratio * math.Exp2(cents/centsAnOctave)
}

// next is the ratio at this sample; it then moves on one.
func (b *bendRun) next() float64 {
	if b.vibrato == nil && b.strike == nil {
		return 1
	}
	if b.sample%envelopeBlock == 0 {
		ahead := float64(b.sample+envelopeBlock) / studioRate
		b.step = (b.at(ahead) - b.ratio) / envelopeBlock
	}
	now := b.ratio
	b.ratio += b.step
	b.sample++
	return now
}

// centsAnOctave is defined: twelve hundred.
const centsAnOctave = 1200
