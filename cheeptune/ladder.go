package cheeptune

import "math"

// The ladder: a four-pole low-pass filter, twenty-four decibels an
// octave, with its output fed back to its input. It is the filter of the
// bass machines acid music is played on: the feedback makes a peak at the
// cutoff that, turned up, squelches and then whistles.
//
// It is four one-pole filters in a row. What goes into the first is the
// input less the fed-back output, through a hyperbolic tangent. Every
// stage after that is an average of a bounded thing, so nothing in the
// filter can pass one whatever the resonance or the cutoff: it sings and
// does not blow up.
//
// It is run twice a sample, at twice the rate, which keeps the peak
// close to the stated cutoff high up. Tests hold the slope, the peak's
// place and the bound.

// lowPass is a filter the analog engine can put its sound through.
type lowPass interface {
	// tune sets the cutoff in hertz and the resonance, nought to one.
	tune(hertz, resonance float64)
	// low takes one sample and gives the low-passed one.
	low(x float64) float64
}

// low is the state filter's low output.
func (f *stateFilter) low(x float64) float64 {
	out, _, _ := f.step(x)
	return out
}

// The ladder's numbers. Picked, except where it says.
const (
	// ladderRuns is how many times a sample the ladder is run.
	ladderRuns = 2
	// ladderFeedback is the feedback at resonance one. Four is where
	// an ideal ladder begins to sing on its own (defined by its four
	// poles); a little under keeps a held note a note.
	ladderFeedback = 3.9
	// ladderMakeUp is how much of the level the feedback takes away is
	// put back at the input.
	ladderMakeUp = 0.5
)

// ladderFilter is a ladder at rest or running.
type ladderFilter struct {
	pole     float64 // each stage's share of its new input
	feedback float64
	stage    [4]float64
}

// tune sets the cutoff and the resonance.
func (f *ladderFilter) tune(hertz, resonance float64) {
	hertz = math.Min(hertz, filterCeiling*studioRate)
	f.pole = 1 - math.Exp(-2*math.Pi*hertz/(studioRate*ladderRuns))
	f.feedback = resonance * ladderFeedback
}

// low takes one sample and gives the low-passed one.
func (f *ladderFilter) low(x float64) float64 {
	x *= 1 + f.feedback*ladderMakeUp
	for range ladderRuns {
		in := math.Tanh(x - f.feedback*f.stage[3])
		for k := range f.stage {
			f.stage[k] += f.pole * (in - f.stage[k])
			in = f.stage[k]
		}
	}
	return f.stage[3]
}

// curve is a shaping curve and its antiderivative, for a clipper that
// averages the curve over each step (see softClip).
type curve struct {
	shape func(x float64) float64
	area  func(x float64) float64
}

// tangent is the soft clipper's curve.
var tangent = curve{shape: math.Tanh, area: tangentArea}

// folder is a wave folder's curve for a number of turns: a sine, which
// past a quarter turn bends the wave back on itself.
func folder(turns float64) curve {
	return curve{
		shape: func(x float64) float64 { return math.Sin(turns * x) },
		area:  func(x float64) float64 { return -math.Cos(turns*x) / turns },
	}
}
