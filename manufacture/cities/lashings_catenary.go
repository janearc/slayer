package cities

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// A cable that hangs from two points under its own weight takes the shape
// of a catenary, a cosh, and not a parabola. The two points may be at
// different heights, as the anchors of a lashing are.

// The bounds a catenary's scale is searched between, as shares of the run:
// a cable pulled nearly straight has a scale thousands of runs long, and
// one that is nearly all slack has a scale a thousandth of a run. The
// search halves the gap this many times.
const (
	tautestScale  = 1e3
	slackestScale = 1e-3
	scaleSteps    = 80
)

// hung is a cable hanging between two points. Its middle is as far under
// the straight line between them as it was given to be, for a cable
// between points at one height.
type hung struct {
	from, to maths.Vec
	// run is the distance between the points along the ground.
	run float64
	// scale is the catenary's own length: how much cable weighs what
	// the tension is at its lowest point.
	scale float64
	// turn is how far along the run the cable is lowest, and rise how
	// far the curve is lifted to meet the first point.
	turn, rise float64
}

// hang is a cable between two points that sags by a distance under the
// line between points at the height of the first, which the span of the
// two points and the sag fix the catenary from. It refuses points one
// above the other, and a sag that is not above nought.
func hang(from, to maths.Vec, sag float64) (hung, error) {
	run := math.Hypot(to.X-from.X, to.Z-from.Z)
	if !(run > 0) || !(sag > 0) {
		return hung{}, fmt.Errorf("cities: a cable hangs between two points "+
			"apart on the ground, with a sag above nought, not %v to %v "+
			"and %v", from, to, sag)
	}
	scale := catenaryScale(run, sag)
	height := to.Y - from.Y
	turn := run/2 - scale*math.Asinh(height/
		(2*scale*math.Sinh(run/(2*scale))))
	return hung{from: from, to: to, run: run, scale: scale, turn: turn,
		rise: from.Y - scale*math.Cosh(turn/scale)}, nil
}

// catenaryScale is the scale of the catenary that sags by a distance over
// a run, found by bisection: the sag falls as the scale grows.
func catenaryScale(run, sag float64) float64 {
	low, high := slackestScale*run, tautestScale*run
	for range scaleSteps {
		middle := (low + high) / 2
		if middle*(math.Cosh(run/(2*middle))-1) > sag {
			low = middle
		} else {
			high = middle
		}
	}
	return (low + high) / 2
}

// at is the point of the cable a share of the way along it from the first
// point to the second, measured along the ground.
func (hung hung) at(share float64) maths.Vec {
	along := share * hung.run
	line := hung.from.Add(hung.to.Sub(hung.from).Scale(share))
	return maths.V(line.X, hung.scale*math.Cosh((along-hung.turn)/
		hung.scale)+hung.rise, line.Z)
}

// lowest is the lowest height the cable reaches, which is at an end if
// the lowest part of the curve is not between them.
func (hung hung) lowest() float64 {
	return hung.at(math.Min(math.Max(hung.turn/hung.run, 0), 1)).Y
}

// points is the cable as that many points, evenly along the ground, the
// two ends among them. It gives at least the two ends.
func (hung hung) points(links int) []maths.Vec {
	links = max(links, 1)
	points := make([]maths.Vec, links+1)
	for index := range points {
		points[index] = hung.at(float64(index) / float64(links))
	}
	return points
}
