package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The loose skin of a bull's neck and shoulders lies in creases. Each
// here is short and uneven: it runs part of the way round the body,
// leans back as it goes down the flank, wanders, and lies differently on
// the two sides. None closes into a ring. Every place, depth and width
// is picked; a depth is in the density's units, and walrusCutDepth says
// how far in metres it moves the hide.

// walrusCreaseLine is one crease. At is where along the body it crosses
// the middle of the back, in rest metres, and lean how far back it lies
// for each radian round the body from there. It runs from the angle from
// to the angle to, in radians round the body from the top of the back.
//
// Depth is how deep it cuts at its deepest and width its half width
// along the body. Wander is how far it strays from its line, and phase
// where in its straying it begins.
type walrusCreaseLine struct {
	at, lean      float64
	from, to      float64
	depth, width  float64
	wander, phase float64
}

// walrusCreaseLines are the creases, from behind the head back to the
// fold behind the fore flipper.
var walrusCreaseLines = []walrusCreaseLine{
	{at: 1.26, lean: .05, from: 0, to: 2.2, depth: .09, width: .022,
		wander: .015, phase: .3},
	{at: 1.13, lean: .09, from: .3, to: 1.9, depth: .11, width: .028,
		wander: .020, phase: 1.7},
	{at: .99, lean: .12, from: 0, to: 1.5, depth: .10, width: .030,
		wander: .025, phase: 4.1},
	{at: .84, lean: .15, from: .6, to: 2.0, depth: .09, width: .030,
		wander: .020, phase: 2.6},
	{at: .58, lean: .10, from: 1.0, to: 2.1, depth: .07, width: .035,
		wander: .030, phase: 5.2},
}

const (
	// walrusCreaseBack and walrusCreaseFront are the ends of the stretch
	// of the body that creases, behind the fore flippers and at the back
	// of the head, in rest metres.
	walrusCreaseBack  = .20
	walrusCreaseFront = 1.40
	// walrusCreaseFade is the angle over which a crease shallows out at
	// each of its ends, in radians.
	walrusCreaseFade = .30
	// walrusCreaseTurns is how many times a crease strays back and forth
	// in one radian round the body.
	walrusCreaseTurns = 3.
)

// cut is how deep the crease cuts at x along the body and the angle round
// from the top of the back, on the side given, one for the right and
// minus one for the left. The two sides stray alike at the top of the
// back, where they meet, and apart from there down.
func (line walrusCreaseLine) cut(x, round, side float64) float64 {
	apart := side * math.Min(round, 1)
	middle := line.at - line.lean*round +
		line.wander*math.Sin(walrusCreaseTurns*round+line.phase+apart)
	off := (x - middle) / line.width
	ends := maths.Smooth(line.from-walrusCreaseFade, line.from, round) *
		(1 - maths.Smooth(line.to, line.to+walrusCreaseFade, round))
	return line.depth * ends * math.Exp(-off*off)
}

// walrusCreasesOf is the creases as a cut into the outline given: how
// deep they cut at a point, nought off the neck and shoulders.
func walrusCreasesOf(outline *walrusOutline) func(maths.Vec) float64 {
	return func(p maths.Vec) float64 {
		if p.X < walrusCreaseBack || p.X > walrusCreaseFront {
			return 0
		}
		round := math.Atan2(math.Abs(p.Z), p.Y-outline.at(p.X).centre)
		side := 1.
		if p.Z < 0 {
			side = -1
		}
		total := 0.
		for _, line := range walrusCreaseLines {
			total += line.cut(p.X, round, side)
		}
		return total
	}
}
