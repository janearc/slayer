package cheeptune

import (
	"fmt"
	"math"
	"time"
)

// Piece is a score rendered through a voicing: the samples, what the driver
// wrote on every tick, and enough of the score to say where a moment falls.
type Piece struct {
	Score   *Score
	Voicing *Voicing
	Rate    int       // samples a second
	Left    []float32 // one to minus one
	Right   []float32
	Peak    float64 // the largest sample, after the output filter

	// Ticks holds, for each channel in voicing order, what the driver
	// wrote on each tick inside the piece. The roll is drawn from this.
	Ticks  [][]Tick
	Labels []string

	notes    map[string][]Note // each played role's notes, one pass
	energy   []float64         // prefix sums of the mean square of both channels
	levelMax float64
	name     string // what Open called it, for a cooked file
	key      string // the hash of the documents it came from
}

// Tick is one driver tick on one channel.
type Tick struct {
	At    float64 // seconds from the first sample
	Pitch float64 // fractional midi; nought on the noise channel
	Vol   float64 // nought to one, as quantised
	Hit   string
}

// Render binds a score to a voicing and runs the machine. It fails if the pair
// does not fit, and it fails if the mix clips: a clipped sample is a wrong
// sample, and saying so is better than shipping it.
func Render(s *Score, v *Voicing) (*Piece, error) {
	pl, err := Bind(s, v)
	if err != nil {
		return nil, err
	}
	return pl.Render()
}

// Render runs the voicing's machine over a plan.
func (pl *Plan) Render() (*Piece, error) {
	return machines[pl.voicing.Machine].render(pl)
}

// render runs the Game Boy over a plan.
func (gameboy) render(pl *Plan) (*Piece, error) {
	s := pl.score
	length := s.Seconds()

	// a looping piece is run twice and the second pass kept. whatever
	// the first pass leaves ringing past its end -- an echo, a release --
	// is then already sounding at the start of the second.
	//
	// the notes that start there cut it off the way the hardware would,
	// and the seam sounds like the middle of the loop.
	voices := pl.voices
	start, end := 0.0, length
	if s.Loop {
		voices = make([]voice, len(pl.voices))
		for i, vc := range pl.voices {
			vc2 := vc
			vc2.notes = make([]sounding, 0, 2*len(vc.notes))
			vc2.notes = append(vc2.notes, vc.notes...)
			for _, n := range vc.notes {
				n.on += length
				n.off += length
				n.onBeat += float64(s.Bars() * s.Meter)
				vc2.notes = append(vc2.notes, n)
			}
			voices[i] = vc2
		}
		start, end = length, 2*length
	} else {
		end = length + tailOf(pl.voices, length)
	}

	sims := make([]*channelSim, len(voices))
	p := &Piece{
		Score: s, Voicing: pl.voicing, Rate: outRate,
		Ticks: make([][]Tick, len(voices)),
	}
	for i := range voices {
		sims[i] = newChannelSim(&voices[i])
		p.Labels = append(p.Labels, voices[i].label)
	}
	p.notes = notesOf(pl.voices)

	n0 := int(math.Round(start * simRate))
	n1 := int(math.Round(end * simRate))
	hiL := make([]float32, n1-n0)
	hiR := make([]float32, n1-n0)
	perTick := simRate / tickRate
	tickN := -1
	for n := 0; n < n1; n++ {
		if k := int(float64(n) / perTick); k != tickN {
			tickN = k
			t := float64(k) / tickRate
			for i, c := range sims {
				c.tick(t, s)
				if n >= n0 {
					st := c.state
					p.Ticks[i] = append(p.Ticks[i], Tick{
						At: t - start, Pitch: st.pitch,
						Vol: st.vol, Hit: st.hit,
					})
				}
			}
		}
		var l, r float64
		for _, c := range sims {
			x := c.sample()
			l += x * c.panL
			r += x * c.panR
		}
		if n >= n0 {
			// fixed headroom: four channels at full volume are full
			// scale. a mix that is quiet is quiet on purpose.
			hiL[n-n0] = float32(l / 4)
			hiR[n-n0] = float32(r / 4)
		}
	}

	p.Left = highpass(decimate(hiL, s.Loop), s.Loop)
	p.Right = highpass(decimate(hiR, s.Loop), s.Loop)
	if s.Loop {
		seam(p.Left)
		seam(p.Right)
	}
	for i := range p.Left {
		p.Peak = math.Max(p.Peak, math.Abs(float64(p.Left[i])))
		p.Peak = math.Max(p.Peak, math.Abs(float64(p.Right[i])))
	}
	if p.Peak > 1 {
		return nil, fmt.Errorf("the mix peaks at %.3f and clips; "+
			"lower a channel's volume", p.Peak)
	}
	p.measure()
	return p, nil
}

// tailOf is how long anything sounds past the end of the piece: the latest of
// every note's release or decay, echoes included, less the piece's length.
func tailOf(vs []voice, length float64) float64 {
	last := 0.0
	for _, v := range vs {
		for _, n := range v.notes {
			end := n.off
			if e := v.ch.Envelope; e != nil {
				end += e.Release
			}
			if n.hit != nil {
				end = math.Max(end, n.on+n.hit.Decay)
			}
			last = math.Max(last, end)
		}
	}
	return math.Max(0, last-length)
}

// decimate low-passes a signal at simRate and keeps every oversamp'th
// sample. The filter is a windowed sinc cut at 18 kHz. What it lets fold
// back lands above 18 kHz, where nothing is listening.
//
// A looping signal wraps at its edges, so the seam is filtered like the
// rest; a signal that does not loop is silent past its ends.
func decimate(x []float32, loop bool) []float32 {
	h := firTaps()
	half := len(h) / 2
	out := make([]float32, len(x)/oversamp)
	for i := range out {
		c := i * oversamp
		var acc float64
		for k, w := range h {
			j := c + k - half
			if j < 0 || j >= len(x) {
				if !loop {
					continue
				}
				j = ((j % len(x)) + len(x)) % len(x)
			}
			acc += w * float64(x[j])
		}
		out[i] = float32(acc)
	}
	return out
}

var taps []float64

// firTaps is the decimation filter, a Blackman-windowed sinc of 127 taps
// that sums to one, built once.
func firTaps() []float64 {
	if taps != nil {
		return taps
	}
	const n = 127
	fc := 18000.0 / simRate
	h := make([]float64, n)
	sum := 0.0
	for i := range h {
		m := float64(i - n/2)
		sinc := 2 * fc
		if m != 0 {
			sinc = math.Sin(2*math.Pi*fc*m) / (math.Pi * m)
		}
		w := 0.42 - 0.5*math.Cos(2*math.Pi*float64(i)/(n-1)) +
			0.08*math.Cos(4*math.Pi*float64(i)/(n-1))
		h[i] = sinc * w
		sum += h[i]
	}
	for i := range h {
		h[i] /= sum
	}
	taps = h
	return taps
}

// highpass is the output capacitor: a one-pole filter that takes away the DC a
// pulse wave carries. A looping signal is run through twice and the second
// pass kept, so the filter's state at the seam is the state the loop leaves.
func highpass(x []float32, loop bool) []float32 {
	const a = 0.996
	out := make([]float32, len(x))
	var xp, yp float64
	passes := 1
	if loop {
		passes = 2
	}
	for range passes {
		for i, v := range x {
			y := float64(v) - xp + a*yp
			xp, yp = float64(v), y
			out[i] = float32(y)
		}
	}
	return out
}

// seamFade is how long a looping piece takes to fade in at its start and
// out at its end.
const seamFade = 5 * time.Millisecond

// seam fades a looping piece to nought at both ends.
//
// Keeping the second pass makes the envelopes and echoes at the seam right,
// but not the phase. The oscillators and the tick grid run on in absolute
// time, and a piece is not periodic in its own length, so its last sample
// does not lead into its first.
//
// Measured on dusk, the jump there was about forty times the step a
// continuous render takes at the same place (render_test.go).
//
// A short raised-cosine fade turns the jump into a dip nobody hears. It also
// covers afplay, where each pass is its own process and consecutive passes
// meet within a latency's jitter, leaving a gap or an overlap. A player that
// loops to the sample would want a crossfade into the third pass instead.
func seam(x []float32) {
	n := min(int(seamFade.Seconds()*outRate), len(x)/2)
	for i := range n {
		g := float32(0.5 - 0.5*math.Cos(math.Pi*float64(i)/float64(n)))
		x[i] *= g
		x[len(x)-1-i] *= g
	}
}

// LoopPeriod is how long one pass of a looping piece lasts, measured from
// the samples, so a player restarting the file is in step with the file. It
// is nought for a piece that does not loop, so passing it on as a period
// never loops a piece written to play once.
func (p *Piece) LoopPeriod() time.Duration {
	if !p.Score.Loop {
		return 0
	}
	return p.Duration()
}

// Duration is how long the rendered audio lasts.
func (p *Piece) Duration() time.Duration {
	return time.Duration(float64(len(p.Left)) / float64(p.Rate) *
		float64(time.Second))
}

// Note is one note a role plays, in seconds from the piece's first sample:
// where it starts and where its release starts, and the hit it strikes
// when the role is a drum's.
type Note struct {
	On, Off float64
	Hit     string
	// Loud is the gain the note's loudness mark gives, from 0.25 at pp
	// to 1.58 at ff; an unmarked note is 1, mf's.
	Loud float64
}

// Notes are the notes a role plays, once through, in order; none for a
// role the piece does not play. An echo is not its role's: it is heard,
// but the score did not write it.
func (p *Piece) Notes(role string) []Note {
	return append([]Note(nil), p.notes[role]...)
}

// notesOf is each played role's notes, from its own channel, not its
// echo's.
func notesOf(voices []voice) map[string][]Note {
	notes := map[string][]Note{}
	for _, vc := range voices {
		if vc.ch.Role == "" {
			continue
		}
		for _, n := range vc.notes {
			loud := n.loud
			if loud <= 0 {
				loud = Marks["mf"]
			}
			notes[vc.ch.Role] = append(notes[vc.ch.Role],
				Note{On: n.on, Off: n.off, Hit: n.hitName, Loud: loud})
		}
	}
	return notes
}
