package show

import (
	"math"
	"strings"
)

// gridAt is a moment's place on the grid: beats from its start,
// fractional, or noGrid before the first drum.
func gridAt(seconds float64, beats []float64) float64 {
	if len(beats) < 2 || seconds < beats[0] {
		return noGrid
	}
	index := len(beats) - 2
	for i := range len(beats) - 1 {
		if beats[i] > seconds {
			index = i - 1
			break
		}
	}
	span := beats[index+1] - beats[index]
	return float64(index) + (seconds-beats[index])/span
}

// beatTime is a place on the grid, in beats, as seconds: the measured
// beat it is in, and the way to the next; past the last beat the last
// spacing carries on.
func beatTime(position float64, beats []float64) float64 {
	if position < 0 || len(beats) < 2 {
		return noGrid
	}
	i := int(math.Floor(position))
	if i >= len(beats)-1 {
		span := beats[len(beats)-1] - beats[len(beats)-2]
		return beats[len(beats)-1] + (position-float64(len(beats)-1))*span
	}
	return beats[i] + (position-float64(i))*(beats[i+1]-beats[i])
}

// ramp is a level gliding from one value to another, from a moment, over
// some seconds, eased in and out.
type ramp struct{ from, to, start, seconds float64 }

// held is a ramp resting at a value.
func held(value float64) ramp { return ramp{value, value, 0, 0} }

// at is the ramp's value at a moment.
func (r ramp) at(t float64) float64 {
	if r.seconds <= 0 || t >= r.start+r.seconds {
		if t < r.start {
			return r.from
		}
		return r.to
	}
	if t <= r.start {
		return r.from
	}
	share := (t - r.start) / r.seconds
	share = share * share * (3 - 2*share)
	return r.from + (r.to-r.from)*share
}

// glide sets a ramp off from where it is at a moment toward a value.
func (r *ramp) glide(to, start, seconds float64) {
	*r = ramp{from: r.at(start), to: to, start: start, seconds: seconds}
}

// bursts is how long ago each burst still showing began, newest first:
// a burst on the beat from, on the grid, and every bar of meter beats
// after, up to the moment at position. None before from, or without one.
func bursts(seconds, position, from, meter float64,
	beats []float64) []float64 {
	if from < 0 || position < from {
		return nil
	}
	var since []float64
	for n := math.Floor((position - from) / meter); n >= 0; n-- {
		ago := seconds - beatTime(from+meter*n, beats)
		if ago >= burstSpan {
			break
		}
		if ago >= 0 {
			since = append(since, ago)
		}
	}
	return since
}

// lit is how long ago the fires were lit, in seconds: since the fire
// cue's beat, from, on the grid; noGrid before it, or without one.
func lit(seconds, position, from float64, beats []float64) float64 {
	if from < 0 || position < from {
		return noGrid
	}
	return seconds - beatTime(from, beats)
}

// gated is the lasers' level under a pulse at a place on the grid, in
// beats, and which hit it is, counted from the grid's start: the last
// hit at or before the moment, fallen by the decay since. A pulse with
// no hits holds the fan dark; no pulse, or no grid, holds it full.
func gated(pulse string, decay, position float64) (float64, int) {
	steps := len(pulse)
	if steps == 0 || position < 0 {
		return 1, 0
	}
	hits := strings.Count(pulse, string(pulseHit))
	sixteenth := position * stepsPerBeat
	now := int(math.Floor(sixteenth))
	for back := range steps {
		step := now - back
		if step < 0 || pulse[step%steps] != pulseHit {
			continue
		}
		since := sixteenth - float64(step)
		bar, in := step/steps, step%steps
		return math.Exp(-since / decay),
			bar*hits + strings.Count(pulse[:in], string(pulseHit))
	}
	return 0, 0
}

// inTurn is which of a list holds at a place on the grid, taken in turn
// so many bars of meter beats apiece from a place, and how far through
// its bars it is; the first before the grid or the place, and nothing of
// an empty list.
func inTurn(list []string, bars, meter, from,
	position float64) (string, float64) {
	if len(list) == 0 {
		return "", 0
	}
	if bars <= 0 || position < 0 || position < from {
		return list[0], 0
	}
	turns := (position - from) / meter / bars
	k := int(math.Floor(turns))
	return list[k%len(list)], turns - float64(k)
}

// floodHit is when the floods fire, and its count, which picks its side.
type floodHit struct {
	at    float64
	count int
}

// floodSteps is how many flood hits fall in a beat, by mode, in a bar
// of meter beats: the ones fall once a bar.
func floodSteps(mode string, meter float64) (float64, bool) {
	perBeat, ok := map[string]float64{
		floodsGrid16: stepsPerBeat, floodsGrid8: eighthsPerBeat,
		floodsOnes: 1 / meter,
	}[mode]
	return perBeat, ok
}

// gridHits is the floods' last hits on the grid at a moment.
func gridHits(perBeat, seconds, position float64,
	beats []float64) []floodHit {
	if position < 0 {
		return nil
	}
	var hits []floodHit
	step := math.Floor(position * perBeat)
	for back := range floodsLookBack {
		place := step - float64(back)
		at := beatTime(place/perBeat, beats)
		if at >= 0 && at <= seconds {
			hits = append(hits, floodHit{at, int(place)})
		}
	}
	return hits
}

// leadHits is the lead's loud notes that may still show at a moment.
func leadHits(seconds float64, lead []scoreNote, hot int) []floodHit {
	var hits []floodHit
	for i, one := range lead {
		at := one.start
		if at <= seconds && at > seconds-leadLookBack && one.vel >= hot {
			hits = append(hits, floodHit{at, i})
		}
	}
	return hits
}

// flooded is the two floods' levels at a moment: each hit fires one
// side, the sides in turn, and a hit falls by floodFlash. A bar is
// meter beats.
func flooded(mode string, seconds, position, meter float64,
	beats []float64, lead []scoreNote, hot int) (left, right float64) {
	var hits []floodHit
	if perBeat, ok := floodSteps(mode, meter); ok {
		hits = gridHits(perBeat, seconds, position, beats)
	} else if mode == floodsLead {
		hits = leadHits(seconds, lead, hot)
	}
	for _, hit := range hits {
		level := math.Exp(-(seconds - hit.at) / floodFlash)
		if hit.count%2 == 0 {
			left = math.Max(left, level)
		} else {
			right = math.Max(right, level)
		}
	}
	return left, right
}
