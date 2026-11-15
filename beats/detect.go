package beats

import (
	"fmt"
	"math"
	"slices"
)

// Settings are authored analysis choices, not facts about the music.
// Tempo zero asks for an estimate; otherwise it is the target in BPM.
// Meter counts beats per bar; First trims the estimated grid to a chosen
// bar-one beat, in seconds. A negative First leaves that phase estimated.
type Settings struct {
	Tempo float64
	Meter int
	First float64
}

// Check refuses invalid authored settings before an audio file is opened.
func (with Settings) Check() error {
	if with.Meter < 1 || with.Meter > 32 || !finite(with.Tempo) ||
		with.Tempo < 0 || (with.Tempo != 0 &&
		(with.Tempo < leastTempo || with.Tempo > mostTempo)) ||
		!finite(with.First) {
		return fmt.Errorf("beats: invalid tempo, meter or first-beat setting")
	}
	return nil
}

// Authored analysis settings: ten-millisecond energy blocks keep the
// search small; the tempo band avoids very slow and very fast harmonics.
// These are not Ellis's front-end parameters or calibrated measurements.
const (
	hopSeconds = 0.01
	leastTempo = 60.0
	mostTempo  = 200.0
	minute     = 60.0  // Definition: seconds in a minute.
	stereo     = 2     // Input is interleaved left, right.
	tightness  = 100.0 // Chosen continuity penalty for normalized onsets.
)

// Detect estimates beats from stereo PCM. This deliberately small front
// end uses positive log-energy changes, not instrument transcription or
// source separation. Prefer drum stems when available; check full mixes
// with the visual metronome before trusting their tempo or bar phase.
func Detect(samples []float32, rate int, settings Settings) (*Grid, error) {
	if err := checkSound(samples, rate, settings); err != nil {
		return nil, err
	}
	hop := max(1, int(float64(rate)*hopSeconds))
	onsets := novelty(samples, hop)
	period := minute / settings.Tempo / hopSeconds
	if settings.Tempo == 0 {
		period = targetPeriod(onsets)
	}
	if !finite(period) || period <= 0 {
		return nil, fmt.Errorf("beats: no rhythmic onset evidence")
	}
	indices := pulsePath(onsets, period)
	grid := makeGrid(indices, onsets, rate, len(samples), settings)
	return grid, grid.Check()
}

// checkSound refuses incomplete stereo, invalid samples and bad settings.
func checkSound(samples []float32, rate int, with Settings) error {
	if err := with.Check(); err != nil {
		return err
	}
	if rate < 100 || rate%100 != 0 || len(samples)%stereo != 0 ||
		len(samples)/stereo < rate {
		return fmt.Errorf("beats: need a second of stereo and valid settings")
	}
	for _, sample := range samples {
		if !finite(float64(sample)) || math.Abs(float64(sample)) > 1 {
			return fmt.Errorf("beats: PCM sample outside -1..1")
		}
	}
	return nil
}

// novelty measures energy per channel before summing, so opposite-phase
// stereo drums do not cancel. A log difference reduces gain sensitivity.
func novelty(samples []float32, hop int) []float64 {
	blocks := (len(samples)/stereo + hop - 1) / hop
	onsets := make([]float64, blocks)
	previous, peak := 0.0, 0.0
	for block := range blocks {
		from, to := block*hop*stereo, min(len(samples), (block+1)*hop*stereo)
		energy := 0.0
		for _, sample := range samples[from:to] {
			energy += float64(sample) * float64(sample)
		}
		// One is the log1p reference energy, an authored compression choice.
		level := math.Log1p(energy)
		onsets[block] = max(0, level-previous)
		peak = max(peak, onsets[block])
		previous = level
	}
	if peak > 0 {
		for index := range onsets {
			onsets[index] /= peak
		}
	}
	return onsets
}

// targetPeriod picks the strongest mean autocorrelation in the authored
// tempo band. Half/double-tempo ambiguity is left visible, not certified.
func targetPeriod(onsets []float64) float64 {
	best, period := 0.0, 0
	lo := int(minute / mostTempo / hopSeconds)
	hi := min(len(onsets)-1, int(minute/leastTempo/hopSeconds))
	for lag := lo; lag <= hi; lag++ {
		score := 0.0
		for index := lag; index < len(onsets); index++ {
			score += onsets[index] * onsets[index-lag]
		}
		score /= float64(len(onsets) - lag)
		if score > best {
			best, period = score, lag
		}
	}
	return float64(period)
}

// pulsePath implements the onset/spacing objective in Ellis (2007),
// equations 2-4, with a squared log-ratio penalty. The implementation and
// energy front end are ours; no code or trained parameters were copied.
func pulsePath(onsets []float64, period float64) []int {
	cost := make([]float64, len(onsets))
	previous := make([]int, len(onsets))
	last := 0
	for index, onset := range onsets {
		cost[index], previous[index] = onset, -1
		lo := max(0, index-int(2*period))
		hi := index - max(1, int(period/2))
		for before := lo; before <= hi; before++ {
			ratio := math.Log(float64(index-before) / period)
			candidate := onset + cost[before] - tightness*ratio*ratio
			if candidate > cost[index] {
				cost[index], previous[index] = candidate, before
			}
		}
		if cost[index] > cost[last] {
			last = index
		}
	}
	var path []int
	for last >= 0 {
		path = append(path, last)
		last = previous[last]
	}
	slices.Reverse(path)
	return path
}

// makeGrid keeps unsupported pulses visibly counted through. A tenth of
// the strongest onset is an authored display threshold, not confidence.
func makeGrid(indices []int, onsets []float64, rate, count int,
	with Settings) *Grid {
	score := Score{Duration: float64(count/stereo) / float64(rate),
		Meter: with.Meter, Status: "estimated", Method: "energy-dp-v1"}
	for _, index := range indices {
		at := float64(index) * hopSeconds
		if with.First >= 0 && at < with.First-hopSeconds/2 {
			continue
		}
		mark := 0
		if onsets[index] >= 0.1 {
			mark = 1
		}
		score.Beats.Times = append(score.Beats.Times, at)
		score.Beats.Measured = append(score.Beats.Measured, mark)
	}
	return &Grid{Score: score}
}
