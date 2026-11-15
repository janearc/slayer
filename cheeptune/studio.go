package cheeptune

import (
	"fmt"
	"sort"
	"strings"
)

// The studio: the second machine.
//
// The design, written before the code. The Game Boy is thin for four
// reasons: one family of timbres, one voice a channel, no room, and every
// note as loud as the next. The studio keeps the score format and the split
// between score and voicing, and takes each limit away.
//
// What the machine is:
//
//   - Kinds. Each channel has one engine, chosen by its kind: fm (sine
//     operators that bend each other's phase), analog (band-limited
//     oscillators into a resonant low-pass filter), pluck (a string, by
//     Karplus and Strong's delay line), modal (a struck object as a bank
//     of decaying resonators) and drum (hits made of a swept tone and
//     filtered noise). An engine is a row in the engines table in
//     engine.go; a sixth is a file and a row.
//   - Limits. Up to studioChannels channels. Any number of them may play
//     the same role, which is how sounds are layered: an echo is always
//     late, and a layer is not. The Game Boy keeps its own rule, one
//     channel a role.
//   - Chords. A chord sounds every note at once, each its own voice with
//     its own envelope. A channel holds at most its voices (eight unless
//     the voicing says); past that the oldest voice that is already
//     releasing is taken, or the oldest of all when none is. A channel
//     with an arp breaks the chord into single strikes as before.
//   - Checking. A field that belongs to another kind, or to the Game Boy,
//     is refused by name, and a number outside its range says its range.
//   - Rendering. Nothing is simulated a tick at a time. Every note is
//     rendered whole by its engine into its channel, at the output rate;
//     the channels are placed between left and right, sent to one shared
//     reverb and one shared delay, summed, and scaled by the master. A
//     looping piece has its tail folded back onto its start, so the loop
//     is exactly periodic. A mix that clips is refused with its peak.
//
// The files: studio.go (this: the documents and their rules),
// studiomix.go (a plan to samples), strike.go (notes to voices),
// envelope.go, engine.go and one file an engine, reverb.go, repeats.go.

// studio is the second machine.
type studio struct{}

// Send is how much of a channel goes to each shared effect, nought to one.
type Send struct {
	Reverb float64 `json:"reverb"`
	Delay  float64 `json:"delay"`
}

// Reverb is the studio's one room: a feedback delay network of eight
// lines. Size is how big the room is, nought to one; decay is the seconds
// a sound takes to fall sixty decibels; damping is how much faster the high
// end dies, nought to one; level is how loud the room is in the mix.
type Reverb struct {
	Size    float64 `json:"size"`
	Decay   float64 `json:"decay"`
	Damping float64 `json:"damping"`
	Level   float64 `json:"level"`
}

// Repeats is the studio's one delay: what is sent to it comes back every
// so many beats, each time quieter by the feedback and duller by the tone
// (one is bright, nought is dark).
type Repeats struct {
	Beats    string  `json:"beats"`
	Feedback float64 `json:"feedback"`
	Tone     float64 `json:"tone"`
	Level    float64 `json:"level"`

	beats frac
}

// kinds are the studio's engines, sorted.
func (studio) kinds() []string {
	names := make([]string, 0, len(engines))
	for name := range engines {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// percussive is whether a kind plays hits.
func (studio) percussive(kind string) bool {
	return engines[kind].percussive
}

// check is the studio's rules on a voicing.
func (m studio) check(v *Voicing) ([]error, map[string]int) {
	var errs []error
	fail := func(f string, a ...any) {
		errs = append(errs, fmt.Errorf(f, a...))
	}
	if len(v.Channels) > studioChannels {
		fail("the studio has %d channels and the voicing has %d; "+
			"remove %d", studioChannels, len(v.Channels),
			len(v.Channels)-studioChannels)
	}
	plays := map[string]int{}
	for i := range v.Channels {
		c := &v.Channels[i]
		if _, ok := engines[c.Kind]; !ok {
			fail("channel %d: unknown kind %q; the studio's kinds "+
				"are %s", i+1, c.Kind, strings.Join(m.kinds(), ", "))
			continue
		}
		at := fmt.Sprintf("channel %d (%s)", i+1, c.Kind)
		errs = append(errs, c.does(at)...)
		errs = append(errs, c.checkStudio(at)...)
		if c.Role != "" {
			plays[c.Role]++
		}
	}
	errs = append(errs, v.checkMix()...)
	return errs, plays
}

// checkMix is the voicing's master, reverb and delay.
func (v *Voicing) checkMix() []error {
	q := &quarrel{}
	if v.Master != nil {
		q.within("master", *v.Master, masterLeast, masterMost)
	}
	if r := v.Reverb; r != nil {
		q.within("reverb size", r.Size, 0, 1)
		q.within("reverb decay", r.Decay, reverbDecayLeast, reverbDecayMost)
		q.within("reverb damping", r.Damping, 0, 1)
		q.within("reverb level", r.Level, 0, 1)
	}
	if d := v.Delay; d != nil {
		beats, err := parseFrac(d.Beats)
		if err != nil {
			q.fail("delay beats: %v; write beats as \"3/4\"", err)
		}
		d.beats = beats
		q.within("delay feedback", d.Feedback, 0, repeatsFeedbackMost)
		q.within("delay tone", d.Tone, 0, 1)
		q.within("delay level", d.Level, 0, 1)
	}
	for i, c := range v.Channels {
		if c.Send == nil {
			continue
		}
		if c.Send.Reverb > 0 && v.Reverb == nil {
			q.fail("channel %d sends to a reverb and the voicing has "+
				"none; give the voicing a reverb", i+1)
		}
		if c.Send.Delay > 0 && v.Delay == nil {
			q.fail("channel %d sends to a delay and the voicing has "+
				"none; give the voicing a delay", i+1)
		}
	}
	return q.errs
}

// gameboyField names the first Game Boy field a channel carries, or
// nothing.
func (c *Channel) gameboyField() string {
	for _, f := range []struct {
		name string
		set  bool
	}{
		{"duty", c.Duty != nil}, {"table", c.Table != ""},
		{"hits", c.Hits != nil}, {"pan", c.Pan != ""},
	} {
		if f.set {
			return f.name
		}
	}
	return ""
}

// checkStudio is one studio channel's fields against its kind.
func (c *Channel) checkStudio(at string) []error {
	q := &quarrel{at: at + ": "}
	if name := c.gameboyField(); name != "" {
		q.fail("%s is only for the gameboy machine%s", name,
			map[string]string{
				"pan":  "; a studio channel has place, -1 to 1",
				"hits": "; a drum channel has drums",
			}[name])
	}
	if c.Place != nil {
		q.within("place", *c.Place, -1, 1)
	}
	if c.Send != nil {
		q.within("send reverb", c.Send.Reverb, 0, 1)
		q.within("send delay", c.Send.Delay, 0, 1)
	}
	if c.Voices != 0 {
		q.within("voices", float64(c.Voices), 1, voicesMost)
	}
	q.within("detune", c.Detune, -detuneMost, detuneMost)
	eng := engines[c.Kind]
	for _, f := range c.engineFields() {
		if f.set && f.kind != c.Kind {
			q.fail("%s is only for a channel of kind %s", f.name, f.kind)
		}
		if !f.set && f.kind == c.Kind {
			q.fail("a channel of kind %s needs %s", c.Kind, f.name)
		}
	}
	c.checkShaping(q, eng)
	if !q.failed() {
		eng.check(c, q)
	}
	return q.errs
}

// engineField is one engine's own object on a channel.
type engineField struct {
	name, kind string
	set        bool
}

// engineFields is each engine's object and whether the channel has it.
func (c *Channel) engineFields() []engineField {
	return []engineField{
		{"fm", "fm", c.FM != nil},
		{"analog", "analog", c.Analog != nil},
		{"pluck", "pluck", c.Pluck != nil},
		{"modal", "modal", c.Modal != nil},
		{"drums", "drum", c.Drums != nil},
	}
}

// checkShaping is the envelope, vibrato and arp: which kinds may carry
// each, and their ranges.
func (c *Channel) checkShaping(q *quarrel, eng engineRow) {
	if c.Envelope != nil && !eng.envelope {
		q.fail("envelope is not for a %s channel; %s", c.Kind, eng.shaped)
	}
	if c.Vibrato != nil && !eng.vibrato {
		q.fail("vibrato is not for a %s channel; fm and analog bend",
			c.Kind)
	}
	if c.Glide != nil {
		if !eng.vibrato {
			q.fail("glide is not for a %s channel; fm and analog slide",
				c.Kind)
		}
		q.within("glide", *c.Glide, glideLeast, glideMost)
	}
	if eng.percussive && (c.Arp.set || c.Detune != 0 || c.Voices != 0) {
		q.fail("arp, detune and voices are for pitched channels")
	}
	checkEnvelope(q, "envelope", c.Envelope)
	if vb := c.Vibrato; vb != nil {
		q.within("vibrato rate", vb.Rate, vibratoRateLeast, vibratoRateMost)
		q.within("vibrato depth", vb.Depth, 0, detuneMost)
		q.within("vibrato delay", vb.Delay, 0, envelopeMost)
	}
}

// checkEnvelope is an envelope's ranges, when there is one.
func checkEnvelope(q *quarrel, name string, e *Envelope) {
	if e == nil {
		return
	}
	q.within(name+" attack", e.Attack, 0, envelopeMost)
	q.within(name+" decay", e.Decay, 0, envelopeMost)
	q.within(name+" sustain", e.Sustain, 0, 1)
	q.within(name+" release", e.Release, 0, envelopeMost)
}

// fits is whether a studio channel can sound an event.
func (studio) fits(c *Channel, e event) error {
	if engines[c.Kind].percussive {
		if e.hit == "" {
			return fmt.Errorf("a note on a drum channel; it plays hits")
		}
		if _, ok := c.Drums[e.hit]; !ok {
			return fmt.Errorf("hit %q is not one of %s", e.hit,
				strings.Join(drumNames(c.Drums), ", "))
		}
		return nil
	}
	if e.hit != "" {
		return fmt.Errorf("hit %q on a pitched channel", e.hit)
	}
	for _, m := range e.pitches {
		if m < studioLowest || m > studioHighest {
			return fmt.Errorf("%s is outside the studio's range %s to "+
				"%s", NoteName(m), NoteName(studioLowest),
				NoteName(studioHighest))
		}
	}
	return nil
}

// quarrel collects what is wrong with one thing, each problem prefixed
// with where it is.
type quarrel struct {
	at   string
	errs []error
}

// fail records a problem.
func (q *quarrel) fail(f string, a ...any) {
	q.errs = append(q.errs, fmt.Errorf(q.at+f, a...))
}

// failed is whether anything has been recorded.
func (q *quarrel) failed() bool { return len(q.errs) > 0 }

// within records a number outside its range, by name, with the range.
func (q *quarrel) within(name string, x, least, most float64) {
	if x < least || x > most || x != x {
		q.fail("%s %v is outside %v to %v", name, x, least, most)
	}
}
