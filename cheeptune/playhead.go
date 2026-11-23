package cheeptune

import (
	"math"
	"time"
)

// playhead says where a piece of music is, at an offset from its first
// sample. It is the interface miami's app depends on; tune does not import
// miami and miami does not import tune, and the method set is what joins
// them.
type playhead interface {
	At(offset time.Duration) (beat, bar, beats, level float64, ok bool)
}

var _ playhead = (*Piece)(nil)

// levelWindow is the span the level is measured over, centred on the offset.
// It reads half of it ahead of the offset, which is fine: the audio exists
// already, and a picture that lags the sound is worse than one that does not.
const levelWindow = 50 * time.Millisecond

// At is where the piece is at an offset from its first sample.
//
// beat and bar run from nought up to one inside the current beat and bar.
// beats counts from the first sample and only grows, across the loop seam
// of a looping piece too.
//
// level is the loudness around the offset against the loudest moment in
// the piece. ok is false before nought and past the end of a piece that
// does not loop, and then every number is nought.
func (p *Piece) At(offset time.Duration) (beat, bar, beats, level float64,
	ok bool) {
	secs := offset.Seconds()
	n := len(p.Left)
	if secs < 0 || n == 0 {
		return 0, 0, 0, 0, false
	}
	idx := int(secs * float64(p.Rate))
	if idx >= n {
		if !p.Score.Loop {
			return 0, 0, 0, 0, false
		}
	}
	beats = p.Score.BeatsAt(secs)
	beat = beats - math.Floor(beats)
	m := float64(p.Score.Meter)
	bar = (beats - m*math.Floor(beats/m)) / m
	return beat, bar, beats, p.levelAt(idx), true
}

// measure builds what levelAt needs: a running sum of the mean square of
// both channels, and the loudest window in the piece to scale against.
func (p *Piece) measure() {
	n := len(p.Left)
	p.energy = make([]float64, n+1)
	for i := range n {
		l, r := float64(p.Left[i]), float64(p.Right[i])
		p.energy[i+1] = p.energy[i] + (l*l+r*r)/2
	}
	p.levelMax = 0
	hop := p.Rate / 100
	for i := 0; i < n; i += hop {
		p.levelMax = math.Max(p.levelMax, p.rms(i))
	}
}

// rms is the root mean square over the window centred on sample i. A
// looping piece wraps; a piece that does not counts silence past its ends.
func (p *Piece) rms(i int) float64 {
	n := len(p.Left)
	half := int(levelWindow.Seconds()*float64(p.Rate)) / 2
	sum := func(a, b int) float64 { // samples a up to b, inside the piece
		a, b = max(a, 0), min(b, n)
		if a >= b {
			return 0
		}
		return p.energy[b] - p.energy[a]
	}
	a, b := i-half, i+half
	total := sum(a, b)
	if p.Score.Loop {
		if a < 0 {
			total += sum(n+a, n)
		}
		if b > n {
			total += sum(0, b-n)
		}
	}
	return math.Sqrt(total / float64(2*half))
}

// levelAt is the level at sample i against the loudest window in the piece.
func (p *Piece) levelAt(i int) float64 {
	if p.levelMax == 0 {
		return 0
	}
	if p.Score.Loop {
		i %= len(p.Left)
	}
	return math.Min(1, p.rms(i)/p.levelMax)
}
