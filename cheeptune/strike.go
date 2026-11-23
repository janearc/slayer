package cheeptune

import (
	"math"
	"sort"
)

// strike is one voice sounding once: a note of a chord, a step of an arp,
// or a hit.
type strike struct {
	on     float64 // seconds from the start of the plan
	gate   float64 // seconds it is held
	midi   int
	hz     float64
	gain   float64 // the echo's gain times the note's loudness
	hit    string
	seed   uint64
	stolen float64 // seconds after on at which the voice is taken
	loud   float64 // the gain of the note's loudness mark alone
	glide  float64 // seconds a slide takes
	path   []slideStep
}

// slideStep is a note a strike slides to: when, counted from the strike's
// start, and the pitch and gain it moves to.
type slideStep struct {
	at, hz, gain float64
}

// slideAt is how far a strike has slid at an age: its pitch as a ratio
// to the pitch it started at, and its gain. Each step is reached in a
// straight line (in pitch, a straight line in semitones) over the glide.
func (st strike) slideAt(age float64) (ratio, gain float64) {
	ratio, gain = 1, st.gain
	for _, step := range st.path {
		if age < step.at {
			break
		}
		done := 1.0
		if st.glide > 0 {
			done = math.Min(1, (age-step.at)/st.glide)
		}
		to := step.hz / st.hz
		ratio = math.Exp2(math.Log2(ratio) +
			(math.Log2(to)-math.Log2(ratio))*done)
		gain += (step.gain - gain) * done
	}
	return ratio, gain
}

// strikes turns a channel's soundings into strikes: a chord becomes a
// voice a note, or a run of single strikes when the channel has an arp.
func strikes(vc *voice, score *Score) []strike {
	var out []strike
	c := vc.ch
	slides := engines[c.Kind].vibrato && !c.Arp.set
	glide := glideDefault
	if c.Glide != nil {
		glide = *c.Glide
	}
	sliding := false // the strike before is waiting for this note
	for _, n := range vc.notes {
		base := strike{on: n.on, gate: n.off - n.on, loud: n.loud,
			gain: n.gain * n.loud, hit: n.hitName, stolen: math.Inf(1),
			glide: glide}
		if sliding && len(n.pitches) == 1 {
			last := &out[len(out)-1]
			last.path = append(last.path, slideStep{at: n.on - last.on,
				hz: hz(n.pitches[0], c.Detune), gain: base.gain})
			last.gate = n.off - last.on
			sliding = n.slide
			continue
		}
		sliding = slides && n.slide
		switch {
		case n.hitName != "":
			out = append(out, base)
		case c.Arp.set && len(n.pitches) > 1:
			out = append(out, arpStrikes(base, n, c, score)...)
		default:
			for _, m := range n.pitches {
				st := base
				st.midi, st.hz = m, hz(m, c.Detune)
				out = append(out, st)
			}
		}
	}
	sort.SliceStable(out, func(a, b int) bool {
		return out[a].on < out[b].on
	})
	for i := range out {
		out[i].seed = uint64(i + 1)
	}
	return out
}

// arpStrikes breaks a chord into single strikes, one a step, cycling
// through its notes until the chord's time is up. A step counted in
// beats follows the tempo; one counted in ticks is the same length at
// any tempo.
func arpStrikes(base strike, n sounding, c *channel,
	score *Score) []strike {
	end := base.on + base.gate
	starts := func(k int) float64 {
		if c.Arp.beats.n > 0 {
			beat := n.onBeat + float64(k)*c.Arp.beats.float()
			return score.SecondsAt(beat)
		}
		ticks := arpTicksDefault
		if c.Arp.ticks > 0 {
			ticks = c.Arp.ticks
		}
		return base.on + float64(k*ticks)/tickRate
	}
	var out []strike
	for k := 0; starts(k) < end-arpSliver; k++ {
		st := base
		st.on = starts(k)
		st.gate = math.Min(starts(k+1), end) - st.on
		st.midi = n.pitches[k%len(n.pitches)]
		st.hz = hz(st.midi, c.Detune)
		out = append(out, st)
	}
	return out
}

// arpTicksDefault is the arp's step when the voicing sets an arp without
// a speed. Defined by the Game Boy machine: three driver ticks.
const arpTicksDefault = 3

// arpSliver is the shortest last step an arp bothers to strike, in
// seconds. Picked: under a millisecond is rounding, not a note.
const arpSliver = 0.001

// steal marks the strikes whose voices are taken because the channel is
// full: when a strike starts and limit voices are still sounding, the
// oldest one already let go is taken, or the oldest of all.
func steal(sts []strike, limit int, ring func(strike) float64) {
	var live []int
	for i := range sts {
		now := sts[i].on
		kept := live[:0]
		for _, j := range live {
			if sts[j].on+sts[j].length(ring) > now {
				kept = append(kept, j)
			}
		}
		live = kept
		if len(live) >= limit {
			at := victim(sts, live, now)
			sts[live[at]].stolen = now - sts[live[at]].on
			live = append(live[:at], live[at+1:]...)
		}
		live = append(live, i)
	}
}

// victim is which of the live voices is taken at a moment: the oldest
// already let go, or the oldest. live is in order of starting.
func victim(sts []strike, live []int, now float64) int {
	for at, j := range live {
		if sts[j].on+sts[j].gate <= now {
			return at
		}
	}
	return 0
}

// length is how long a strike sounds, in seconds: held, then ringing, or
// until it is taken.
func (st strike) length(ring func(strike) float64) float64 {
	whole := st.gate + ring(st)
	return math.Min(whole, st.stolen+stealFade)
}
