package cheeptune

import "math"

// The delay: what is sent comes back a fixed number of beats later, and
// what comes back is fed in again, quieter by the feedback and through a
// low-pass filter set by the tone, so each repeat is duller than the last.

// repeatsDarkest and repeatsBrightest are the filter's cutoff at tone
// nought and tone one, in hertz. Picked.
const (
	repeatsDarkest   = 500
	repeatsBrightest = 18000
)

// repeatsRun is a delay running.
type repeatsRun struct {
	line     []float64
	at       int
	feedback float64
	pole     float64
	state    float64
	level    float64
}

// newRepeatsRun is a delay at rest, its length the beats at this tempo.
func newRepeatsRun(d *Repeats, beatSeconds float64) *repeatsRun {
	n := int(math.Round(d.beats.float() * beatSeconds * studioRate))
	cutoff := repeatsDarkest *
		math.Pow(repeatsBrightest/repeatsDarkest, d.Tone)
	return &repeatsRun{line: make([]float64, max(n, 1)),
		feedback: d.Feedback, level: d.Level,
		pole: math.Exp(-2 * math.Pi * cutoff / studioRate)}
}

// step takes one sample of what was sent and gives what comes back.
func (r *repeatsRun) step(x float64) float64 {
	y := r.line[r.at]
	r.state = y*(1-r.pole) + r.state*r.pole
	r.line[r.at] = x + r.state*r.feedback
	r.at++
	if r.at == len(r.line) {
		r.at = 0
	}
	return y * r.level
}

// tail is how long the repeats take to fall sixty decibels after the
// last input, in seconds.
func (r *repeatsRun) tail() float64 {
	one := float64(len(r.line)) / studioRate
	if r.feedback <= 0 {
		return one
	}
	turns := math.Ceil(-sixtyDecibels / math.Log(r.feedback))
	return one * (turns + 1)
}

// repeatsBank is the delay across a piece whose tempo changes: a delay
// line for each stretch of one tempo, its length that tempo's beats. A
// line takes what is sent only during its own stretch and goes on
// returning its repeats after, so the repeats of each stretch fall on
// that stretch's beat and no line ever changes length while sounding.
type repeatsBank struct {
	runs   []*repeatsRun
	starts []int // the sample each line's stretch starts at
}

// newRepeatsBank is the delay for a score, a line a tempo.
func newRepeatsBank(d *Repeats, s *Score) *repeatsBank {
	bank := &repeatsBank{}
	for _, span := range s.tempoSpans() {
		bank.runs = append(bank.runs, newRepeatsRun(d, span.beatSeconds))
		bank.starts = append(bank.starts,
			int(math.Round(span.second*studioRate)))
	}
	return bank
}

// step takes the sample sent at a moment and gives what comes back from
// every line.
func (b *repeatsBank) step(sample int, x float64) float64 {
	back := 0.0
	for k, run := range b.runs {
		mine := sample >= b.starts[k] &&
			(k+1 == len(b.runs) || sample < b.starts[k+1])
		if mine {
			back += run.step(x)
		} else {
			back += run.step(0)
		}
	}
	return back
}

// tail is the longest any line rings on, in seconds.
func (b *repeatsBank) tail() float64 {
	longest := 0.0
	for _, run := range b.runs {
		longest = math.Max(longest, run.tail())
	}
	return longest
}
