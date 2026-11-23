package cheeptune

import "math"

// The reverb is a feedback delay network: eight delay lines whose outputs
// are mixed by a Householder matrix and fed back into all eight.
//
// Why this and not combs and allpasses in a row (Schroeder's, Freeverb's):
// here the decay time is a number each line is given, since a line of L
// samples is turned down by exactly what sixty decibels in the stated
// time comes to over L samples (Jot's rule), so the stated decay is the
// measured one and a test holds it. The mixing matrix makes the echoes
// dense without the metallic ring parallel combs have.

// reverbLines is how many delay lines the network has. Picked: eight is
// the usual size, dense enough and cheap.
const reverbLines = 8

// reverbLengths are the lines' lengths in samples for the largest room,
// each a prime so no two share an echo. Picked: 32 to 79 milliseconds.
var reverbLengths = [reverbLines]int{1423, 1637, 1877, 2137, 2411, 2729,
	3089, 3491}

// reverbSmallest is the share of those lengths a room of size nought has.
// Picked: a fifth, ten milliseconds at the shortest.
const reverbSmallest = 0.2

// reverbDullest is the damping filter's pole at damping one. Picked.
const reverbDullest = 0.7

// reverbRun is a reverb running.
type reverbRun struct {
	lines [reverbLines][]float64
	at    [reverbLines]int
	gain  [reverbLines]float64
	dull  float64
	state [reverbLines]float64
	level float64
}

// newReverbRun is a room at rest.
func newReverbRun(r *reverb) *reverbRun {
	run := &reverbRun{dull: r.Damping * reverbDullest, level: r.Level}
	scale := reverbSmallest + (1-reverbSmallest)*r.Size
	for k, full := range reverbLengths {
		n := nextPrime(int(float64(full) * scale))
		run.lines[k] = make([]float64, n)
		seconds := float64(n) / studioRate
		run.gain[k] = math.Exp(-sixtyDecibels * seconds / r.Decay)
	}
	return run
}

// nextPrime is the first prime at or above n.
func nextPrime(n int) int {
	for ; ; n++ {
		prime := n > 1
		for d := 2; d*d <= n; d++ {
			if n%d == 0 {
				prime = false
				break
			}
		}
		if prime {
			return n
		}
	}
}

// step takes one sample of what was sent and gives the room's left and
// right. The lines come out, are damped and turned down, are mixed (each
// less a quarter of their sum, the Householder matrix for eight) and go
// back in with the new sample.
func (r *reverbRun) step(x float64) (left, right float64) {
	var out [reverbLines]float64
	sum := 0.0
	for k := range r.lines {
		y := r.lines[k][r.at[k]]
		r.state[k] = y*(1-r.dull) + r.state[k]*r.dull
		out[k] = r.state[k] * r.gain[k]
		sum += out[k]
	}
	sum *= 2.0 / reverbLines
	for k := range r.lines {
		r.lines[k][r.at[k]] = x + out[k] - sum
		r.at[k]++
		if r.at[k] == len(r.lines[k]) {
			r.at[k] = 0
		}
	}
	left = out[0] - out[2] + out[4] - out[6]
	right = out[1] - out[3] + out[5] - out[7]
	return left * r.level / 2, right * r.level / 2
}
