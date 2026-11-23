package cheeptune

import (
	"sort"
	"strings"
)

// The fm engine: two to four sine operators. A carrier is heard; a
// modulator is not heard, it swings the phase of the operator it feeds,
// and that puts partials either side of the carrier at steps of the
// modulator's frequency. Whole-number ratios give a harmonic tone; a ratio
// off a whole number gives the uneven partials of a bell.

// fmSound is an fm channel's sound.
type fmSound struct {
	Arrangement string     `json:"arrangement"`
	Operators   []operator `json:"operators"`
	Feedback    float64    `json:"feedback"`
}

// operator is one sine: its frequency as a ratio to the note, its level
// (how loud a carrier is, or how hard a modulator swings what it feeds)
// and its own envelope.
type operator struct {
	Ratio    float64   `json:"ratio"`
	Level    float64   `json:"level"`
	Envelope *envelope `json:"envelope"`
}

// arrangement is how operators are joined: for each operator, counted
// from nought, the operators that modulate it, and which are heard. A
// modulator always has a higher number than what it feeds, so one pass
// from the last operator to the first has every input ready.
type arrangement struct {
	operators []int   // how many operators it may have
	feeds     [][]int // feeds[k] modulate operator k
	carriers  []int
}

// arrangements are the named ways to join operators.
//
//	chain  4 > 3 > 2 > 1, and 1 is heard
//	fan    2, 3 and 4 each into 1, and 1 is heard
//	pairs  2 > 1 and 4 > 3, and 1 and 3 are heard
//	sum    nothing modulates anything, and all are heard
var arrangements = map[string]arrangement{
	"chain": {operators: []int{2, 3, 4},
		feeds: [][]int{{1}, {2}, {3}, {}}, carriers: []int{0}},
	"fan": {operators: []int{2, 3, 4},
		feeds: [][]int{{1, 2, 3}, {}, {}, {}}, carriers: []int{0}},
	"pairs": {operators: []int{4},
		feeds: [][]int{{1}, {}, {3}, {}}, carriers: []int{0, 2}},
	"sum": {operators: []int{2, 3, 4},
		feeds: [][]int{{}, {}, {}, {}}, carriers: []int{0, 1, 2, 3}},
}

// fmDepth is how far a modulator at level one swings a phase, in cycles:
// two cycles, which is four pi radians. Picked: past that the tone is
// noise at any ratio.
const fmDepth = 2

// fmFeedbackDepth is how far the last operator at feedback one swings its
// own phase, in cycles. Picked: about two radians, where a sine has
// become close to a saw and has not yet broken into noise.
const fmFeedbackDepth = 0.32

// fmRatioMost is the highest ratio an operator has. Picked: a thirty-
// second partial of middle c is past eight kilohertz.
const fmRatioMost = 32

// init puts the fm engine in the studio's table.
func init() {
	register("fm", engineRow{vibrato: true,
		shaped: "each operator has its own envelope",
		check:  checkFM, build: buildFM})
}

// arrangementNames lists the arrangements, for an error.
func arrangementNames() string {
	names := make([]string, 0, len(arrangements))
	for name := range arrangements {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// checkFM is an fm channel's own rules.
func checkFM(c *channel, q *quarrel) {
	f := c.FM
	arr, ok := arrangements[f.Arrangement]
	if !ok {
		q.fail("fm arrangement %q; arrangements are %s", f.Arrangement,
			arrangementNames())
		return
	}
	fits := false
	for _, n := range arr.operators {
		fits = fits || n == len(f.Operators)
	}
	if !fits {
		q.fail("fm arrangement %s takes %v operators and this has %d",
			f.Arrangement, arr.operators, len(f.Operators))
	}
	for i, op := range f.Operators {
		name := "fm operator " + string(rune('1'+i))
		q.within(name+" ratio", op.Ratio, fmRatioLeast, fmRatioMost)
		q.within(name+" level", op.Level, 0, 1)
		checkEnvelope(q, name+" envelope", op.Envelope)
	}
	q.within("fm feedback", f.Feedback, 0, 1)
}

// fmRatioLeast is the lowest ratio an operator has. Picked: an eighth is
// three octaves under the note, low enough for a slow wobble.
const fmRatioLeast = 0.125

// fmEngine is an fm channel ready to sound.
type fmEngine struct {
	channel *channel
	arr     arrangement
}

// buildFM is the engine for a checked fm channel.
func buildFM(c *channel) engine {
	return &fmEngine{channel: c, arr: arrangements[c.FM.Arrangement]}
}

// ring is the longest release among the operators that are heard.
func (e *fmEngine) ring(strike) float64 {
	longest := 0.0
	for _, k := range e.arr.carriers {
		if k < len(e.channel.FM.Operators) {
			env := orGated(e.channel.FM.Operators[k].Envelope)
			longest = max(longest, env.Release)
		}
	}
	return longest
}

// sound renders one note: the operators from last to first, each fed by
// the ones above it, the heard ones summed.
func (e *fmEngine) sound(st strike, out []float64) {
	ops := e.channel.FM.Operators
	count := len(ops)
	var phase, was, value [4]float64
	var runs [4]*envelopeRun
	for k := range ops {
		runs[k] = newEnvelopeRun(ops[k].Envelope, st.gate)
	}
	bend := newBendRun(e.channel.Vibrato, &st)
	feedback := e.channel.FM.Feedback * fmFeedbackDepth
	last := count - 1
	for i := range out {
		step := st.hz * bend.next() / studioRate
		for k := last; k >= 0; k-- {
			swing := 0.0
			for _, from := range e.arr.feeds[k] {
				if from < count {
					swing += value[from]
				}
			}
			swing *= fmDepth
			if k == last {
				swing += feedback * (value[k] + was[k]) / 2
			}
			was[k] = value[k]
			value[k] = sine(phase[k]+swing) * ops[k].Level *
				runs[k].next()
			phase[k] += step * ops[k].Ratio
			if phase[k] >= 1 {
				phase[k]--
			}
		}
		sum := 0.0
		for _, k := range e.arr.carriers {
			if k < count {
				sum += value[k]
			}
		}
		out[i] = sum
	}
}
