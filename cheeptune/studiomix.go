package cheeptune

import (
	"fmt"
	"math"
	"sort"
)

// studioChannel is one channel of a plan, ready to render.
type studioChannel struct {
	channel *Channel
	label   string
	engine  engine
	strikes []strike
}

// newStudioChannel turns a voice of the plan into strikes and an engine,
// and takes voices where the channel is full.
func newStudioChannel(vc *voice, score *Score) *studioChannel {
	c := vc.ch
	sc := &studioChannel{channel: c, label: vc.label,
		engine:  engines[c.Kind].build(c),
		strikes: strikes(vc, score)}
	if !engines[c.Kind].percussive {
		limit := voicesDefault
		if c.Voices > 0 {
			limit = c.Voices
		}
		steal(sc.strikes, limit, sc.engine.ring)
	}
	return sc
}

// end is the second the channel's last strike falls silent.
func (sc *studioChannel) end() float64 {
	last := 0.0
	for _, st := range sc.strikes {
		last = math.Max(last, st.on+st.length(sc.engine.ring))
	}
	return last
}

// render is the channel alone, one sample a slot, over n samples.
func (sc *studioChannel) render(n int) []float64 {
	out := make([]float64, n)
	volume := 1.0
	if sc.channel.Volume != nil {
		volume = *sc.channel.Volume
	}
	var scratch []float64
	for _, st := range sc.strikes {
		count := int(st.length(sc.engine.ring) * studioRate)
		first := int(math.Round(st.on * studioRate))
		count = min(count, n-first)
		if count <= 0 {
			continue
		}
		if cap(scratch) < count {
			scratch = make([]float64, count)
		}
		one := scratch[:count]
		sc.engine.sound(st, one)
		fadeTaken(st, one)
		for i, x := range one {
			gain := st.gain
			if len(st.path) > 0 {
				_, gain = st.slideAt(float64(i) / studioRate)
			}
			out[first+i] += x * gain * volume
		}
	}
	return out
}

// fadeTaken fades the end of a strike whose voice was taken, or whose
// ring ran out, over the steal fade, so no strike ends on a step.
func fadeTaken(st strike, one []float64) {
	fade := min(int(math.Round(stealFade*studioRate)), len(one))
	for i := range fade {
		one[len(one)-1-i] *= float64(i) / float64(fade)
	}
}

// places is a place from minus one (left) to one (right) as the gains of
// the two sides, at equal power: a sound is as loud wherever it sits.
func places(place *float64) (left, right float64) {
	at := 0.0
	if place != nil {
		at = *place
	}
	angle := (at + 1) * math.Pi / 4
	return math.Cos(angle), math.Sin(angle)
}

// studioMix is the mix being summed: the two sides and the two sends.
type studioMix struct {
	left, right []float64
	toReverb    []float64
	toDelay     []float64
}

// add places one channel's samples in the mix and its sends.
func (m *studioMix) add(c *Channel, x []float64) {
	left, right := places(c.Place)
	send := Send{}
	if c.Send != nil {
		send = *c.Send
	}
	for i, v := range x {
		m.left[i] += v * left
		m.right[i] += v * right
		m.toReverb[i] += v * send.Reverb
		m.toDelay[i] += v * send.Delay
	}
}

// effects runs the reverb and the delay over what was sent and adds what
// they return. The delay's return is also sent to the reverb, at the
// reverb's own level, so repeats sit in the same room as the rest.
func (m *studioMix) effects(v *Voicing, s *Score) {
	var delay *repeatsBank
	var room *reverbRun
	if v.Delay != nil {
		delay = newRepeatsBank(v.Delay, s)
	}
	if v.Reverb != nil {
		room = newReverbRun(v.Reverb)
	}
	for i := range m.left {
		sent := m.toReverb[i]
		if delay != nil {
			back := delay.step(i, m.toDelay[i])
			m.left[i] += back * centre
			m.right[i] += back * centre
			sent += back * delayIntoRoom
		}
		if room != nil {
			l, r := room.step(sent)
			m.left[i] += l
			m.right[i] += r
		}
	}
}

// centre is the gain of each side for a sound in the middle, at equal
// power. Defined: the square root of a half.
var centre = math.Sqrt(0.5)

// delayIntoRoom is how much of the delay's return goes on to the reverb.
// Picked: a quarter, enough that a repeat is not drier than its source.
const delayIntoRoom = 0.25

// render runs the studio over a plan.
func (studio) render(pl *Plan) (*Piece, error) {
	s, v := pl.score, pl.voicing
	length := s.Seconds()
	channels := make([]*studioChannel, len(pl.voices))
	end := length
	for i := range pl.voices {
		channels[i] = newStudioChannel(&pl.voices[i], s)
		end = math.Max(end, channels[i].end())
	}
	end += effectsTail(v, s)
	end = math.Min(end, length+tailMost)
	n := int(math.Round(end * studioRate))
	mix := &studioMix{left: make([]float64, n), right: make([]float64, n),
		toReverb: make([]float64, n), toDelay: make([]float64, n)}
	p := &Piece{Score: s, Voicing: v, Rate: studioRate,
		Ticks: make([][]Tick, len(channels))}
	for i, sc := range channels {
		mix.add(sc.channel, sc.render(n))
		p.Labels = append(p.Labels, sc.label)
		p.Ticks[i] = sc.ticks(length, s.Loop)
	}
	mix.effects(v, s)
	master := masterDefault
	if v.Master != nil {
		master = *v.Master
	}
	keep := n
	if s.Loop {
		keep = int(math.Round(length * studioRate))
	}
	p.Left = settle(mix.left, keep, master)
	p.Right = settle(mix.right, keep, master)
	for i := range p.Left {
		p.Peak = math.Max(p.Peak, math.Abs(float64(p.Left[i])))
		p.Peak = math.Max(p.Peak, math.Abs(float64(p.Right[i])))
	}
	if p.Peak > 1 {
		return nil, fmt.Errorf("the mix peaks at %.3f and clips; lower "+
			"the voicing's master, which is %v, or a channel's volume",
			p.Peak, master)
	}
	p.measure()
	return p, nil
}

// effectsTail is how long the reverb and the delay ring on, in seconds.
func effectsTail(v *Voicing, s *Score) float64 {
	tail := 0.0
	if v.Delay != nil {
		tail += newRepeatsBank(v.Delay, s).tail()
	}
	if v.Reverb != nil {
		tail += v.Reverb.Decay
	}
	return tail
}

// settle is one side of the finished mix: scaled by the master, and for
// a loop folded, everything past the loop's length added back onto its
// start, as many times round as it takes. A folded loop is exactly what
// the piece sounds like on its second pass and every pass after.
func settle(x []float64, keep int, master float64) []float32 {
	out := make([]float32, keep)
	folded := make([]float64, keep)
	for i, v := range x {
		folded[i%keep] += v
	}
	for i, v := range folded {
		out[i] = float32(v * master)
	}
	return out
}

// ticks is what the channel played, a mark a driver tick for each voice
// sounding, so the roll and the films draw a studio piece as they draw a
// Game Boy one. A chord is several marks at one moment. A mark is as
// strong as the strike's gain while held and fades over its ring.
func (sc *studioChannel) ticks(length float64, loop bool) []Tick {
	var out []Tick
	for _, st := range sc.strikes {
		sounds := st.length(sc.engine.ring)
		first := int(math.Ceil(st.on * tickRate))
		for k := first; ; k++ {
			at := float64(k) / tickRate
			age := at - st.on
			if age >= sounds && k > first {
				break
			}
			vol := st.gain
			if age > st.gate && sounds > st.gate {
				vol *= 1 - (age-st.gate)/(sounds-st.gate)
			}
			if loop {
				at = math.Mod(at, length)
			}
			out = append(out, Tick{At: at, Pitch: float64(st.midi),
				Vol: math.Max(vol, tickFaintest), Hit: st.hit})
		}
	}
	sort.SliceStable(out, func(a, b int) bool {
		return out[a].At < out[b].At
	})
	return out
}

// tickFaintest is the least strength a mark is given, so a voice at the
// very end of its ring is still drawn. Picked.
const tickFaintest = 0.05
