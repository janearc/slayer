package cheeptune

import "math"

// The Game Boy, as far as a music driver sees it.
//
// A driver does not write samples. Once a tick it writes a frequency register
// and a volume to each channel, and the hardware does the rest. Everything
// that moves -- an envelope, vibrato, an arp -- moves in steps of one tick.
const (
	tickRate = 59.73 // driver ticks a second, the LCD's frame rate
	oversamp = 4     // the machine runs at this many times the output rate
	outRate  = 44100
	simRate  = outRate * oversamp
)

// rangeOf is the lowest and highest note a kind can sound, as midi numbers.
//
// The bottom is where the 11-bit register runs out: a pulse cannot go below
// 64 Hz and a wave below 32 Hz. The top is where a register step grows past
// about a sixth of a semitone, and the note stops being the note written.
func rangeOf(kind string) (lo, hi int) {
	switch kind {
	case "pulse":
		return 36, 108 // c2 to c8
	case "wave":
		return 24, 96 // c1 to c7
	}
	return 0, 127
}

// quantise is a frequency as the register can hold it. The pulse register's
// top is 131072 Hz and the wave's is 65536, a whole cycle of 32 steps.
func quantise(kind string, f float64) float64 {
	top := 131072.0
	if kind == "wave" {
		top = 65536
	}
	x := math.Round(2048 - top/f)
	x = math.Max(0, math.Min(2047, x))
	return top / (2048 - x)
}

// hz is a midi number, moved by cents, before the register gets it.
func hz(m int, cents float64) float64 {
	return 440 * math.Pow(2, (float64(m)-69)/12+cents/1200)
}

// noiseClock is how often the noise register steps, from the two numbers
// the hardware takes. A divisor of nought counts as a half.
func noiseClock(shift, divisor int) float64 {
	r := float64(divisor)
	if divisor == 0 {
		r = 0.5
	}
	return 524288 / r / math.Pow(2, float64(shift+1))
}

// tickState is what a driver wrote to one channel on one tick.
type tickState struct {
	hz    float64 // after quantising; nought when silent
	vol   float64 // nought to one, after quantising to the hardware's steps
	pitch float64 // fractional midi of hz, for the roll
	hit   string  // the hit sounding, on the noise channel
	shift int
	div   int
	short bool
}

// channelSim is one channel running: the driver's state and the
// oscillator's.
type channelSim struct {
	v     *voice
	next  int // index of the first note not yet started
	cur   int // the note sounding, or -1
	state tickState
	phase float64
	lfsr  uint16
	panL  float64
	panR  float64
}

// newChannelSim is a channel before its first tick, panned as voiced.
func newChannelSim(v *voice) *channelSim {
	c := &channelSim{v: v, cur: -1, lfsr: 0x7fff, panL: 1, panR: 1}
	switch v.ch.Pan {
	case "left":
		c.panR = 0
	case "right":
		c.panL = 0
	}
	return c
}

// tick is the driver's work for one tick at time t: which note sounds, and
// what frequency and volume that makes.
func (c *channelSim) tick(t float64, score *Score) {
	// a channel plays one thing at a time, so the latest note to have
	// started is the one sounding. hits struck together start at the
	// same moment: the first written is the one the channel plays.
	ns := c.v.notes
	for c.next < len(ns) && ns[c.next].on <= t {
		if c.cur < 0 || ns[c.next].on != ns[c.cur].on {
			c.cur = c.next
		}
		c.next++
	}
	c.state = tickState{}
	if c.cur < 0 {
		return
	}
	n := ns[c.cur]
	ch := c.v.ch
	age := t - n.on
	volume := 1.0
	if ch.Volume != nil {
		volume = *ch.Volume
	}

	if ch.Kind == "noise" {
		h := n.hit
		lv := 1.0
		if h.Volume != nil {
			lv = *h.Volume
		}
		lv *= 1 - age/h.Decay
		if lv <= 0 {
			return
		}
		c.state = tickState{
			vol:   steps16(lv * volume * n.gain),
			hz:    noiseClock(h.Shift, h.Divisor),
			shift: h.Shift, div: h.Divisor, short: h.Short,
			hit: n.hitName,
		}
		return
	}

	lv := envelopeAt(ch.Envelope, age, n.off-n.on)
	if lv <= 0 {
		return
	}

	// the arp: which of the chord's notes this tick plays.
	m := n.pitches[0]
	if k := len(n.pitches); k > 1 {
		var step int
		switch {
		case ch.Arp.beats.n > 0:
			beats := score.BeatsAt(t) - n.onBeat
			step = int(beats / ch.Arp.beats.float())
		default:
			per := 3
			if ch.Arp.ticks > 0 {
				per = ch.Arp.ticks
			}
			step = int(math.Round(age*tickRate)) / per
		}
		m = n.pitches[step%k]
	}

	cents := ch.Detune
	if vb := ch.Vibrato; vb != nil && age > vb.Delay {
		cents += vb.Depth * math.Sin(2*math.Pi*vb.Rate*(age-vb.Delay))
	}
	f := quantise(ch.Kind, hz(m, cents))
	vol := lv * volume * n.gain
	if ch.Kind == "wave" {
		vol = waveSteps(vol)
	} else {
		vol = steps16(vol)
	}
	c.state = tickState{
		hz: f, vol: vol, pitch: 69 + 12*math.Log2(f/440),
	}
}

// sample advances the oscillator one step at simRate and returns its output,
// from minus one to one before the pan.
func (c *channelSim) sample() float64 {
	s := c.state
	if s.vol == 0 {
		return 0
	}
	switch c.v.ch.Kind {
	case "pulse":
		c.phase += s.hz / simRate
		c.phase -= math.Floor(c.phase)
		if c.phase < *c.v.ch.Duty {
			return s.vol
		}
		return -s.vol
	case "wave":
		c.phase += s.hz / simRate
		c.phase -= math.Floor(c.phase)
		step := c.v.ch.table[int(c.phase*32)&31]
		return (float64(step) - 7.5) / 7.5 * s.vol
	}
	// noise: a linear feedback shift register, stepped at the clock rate.
	// bit nought xor bit one feeds bit fourteen, and bit six as well when
	// the short register is selected, which makes the period 127 steps.
	c.phase += s.hz / simRate
	for c.phase >= 1 {
		c.phase--
		fb := (c.lfsr ^ (c.lfsr >> 1)) & 1
		c.lfsr = (c.lfsr >> 1) | (fb << 14)
		if s.short {
			c.lfsr = (c.lfsr &^ (1 << 6)) | (fb << 6)
		}
	}
	if c.lfsr&1 == 0 {
		return s.vol
	}
	return -s.vol
}

// envelopeAt is a note's level at age seconds, gated for gate seconds.
// With no envelope a note is full while gated and silent after.
func envelopeAt(e *Envelope, age, gate float64) float64 {
	if e == nil {
		if age < gate {
			return 1
		}
		return 0
	}
	adsr := func(a float64) float64 {
		var r ramp
		if e.Attack > 0 {
			ad := e.Attack + e.Decay
			r = ramp{{0, 0}, {e.Attack, 1}, {ad, e.Sustain}}
		} else {
			r = ramp{{0, 1}, {e.Decay, e.Sustain}}
		}
		return r.at(a)
	}
	if age < gate {
		return adsr(age)
	}
	if e.Release <= 0 {
		return 0
	}
	return adsr(gate) * (1 - (age-gate)/e.Release)
}

// steps16 is a level as the four-bit volume the pulse and noise channels
// have.
func steps16(v float64) float64 {
	return math.Round(math.Max(0, math.Min(1, v))*15) / 15
}

// waveSteps is a level as the wave channel's volume, which has four
// settings and not sixteen: off, a quarter, a half, all.
func waveSteps(v float64) float64 {
	levels := []float64{0, 0.25, 0.5, 1}
	best := 0.0
	for _, l := range levels {
		if math.Abs(v-l) < math.Abs(v-best) {
			best = l
		}
	}
	return best
}
