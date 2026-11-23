package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// gapLimit is the most of a straight step that the gap in a tube may take,
// as a share of the step, so that a short step is not cut through.
const gapLimit = 0.45

// opened is a closed ring cut open: the ring is cut in the middle of its
// longest step, where a bender would cut it, the two ends a gap apart, and
// the path runs from one end round the ring to the other.
func opened(ring []maths.Vec, gap float64) []maths.Vec {
	longest, best := 0, 0.0
	for index, at := range ring {
		if run := ring[(index+1)%len(ring)].Sub(at).Len(); run > best {
			longest, best = index, run
		}
	}
	way := ring[(longest+1)%len(ring)].Sub(ring[longest]).Unit()
	reach := math.Min(gap/half, best*gapLimit)
	centre := ring[longest].Add(way.Scale(best / half))
	path := []maths.Vec{centre.Add(way.Scale(reach))}
	for step := 1; step <= len(ring); step++ {
		path = append(path, ring[(longest+step)%len(ring)])
	}
	return append(path, centre.Sub(way.Scale(reach)))
}

// tubeRing is one ring of a letter as tube: bent to the ring, which is set
// at an origin in the face's frame, standing off the board's front by the
// culture's share of the em, with its posts and its electrodes.
func (job signJob) tubeRing(line textLine, origin maths.Vec,
	ring []maths.Vec, glass *thing.Material) []placedElement {
	drawing := job.drawing
	stand := line.em * drawing.fixed("sign-tube-stand")
	path := make([]maths.Vec, len(ring))
	for index, at := range ring {
		path[index] = maths.V(origin.X+at.X, origin.Y+at.Y, stand)
	}
	open := opened(path, line.em*drawing.fixed("sign-tube-gap"))
	parts := []placedElement{{Element: solid{Name: "sign tube", Form: meshForm{
		sweep(open, false, line.em*drawing.fixed("sign-tube-radius"))},
		Material: glass}, Place: standingAt(maths.Vec{})}}
	parts = append(parts, job.standoffs(open, line.em, stand)...)
	return append(parts, job.electrodes(open, line.em)...)
}

// standoffs are the posts that hold a tube off its board: one at the start
// of the path and another each time the path has gone the culture's pitch
// further, each a square post from the board's front to the tube's middle.
func (job signJob) standoffs(path []maths.Vec, em, stand float64) []placedElement {
	drawing := job.drawing
	post, pitch := em*drawing.fixed("sign-tube-post"),
		em*drawing.fixed("sign-tube-pitch")
	var parts []placedElement
	gone, next := 0.0, 0.0
	for index, at := range path {
		if index > 0 {
			gone += at.Sub(path[index-1]).Len()
		}
		if gone < next {
			continue
		}
		parts = append(parts, strut("sign standoff", maths.V(at.X, at.Y, 0),
			maths.V(at.X, at.Y, stand), post, job.frame))
		next = gone + pitch
	}
	return parts
}

// electrodes are the two ends of an open tube: at each end a short dark
// housing that carries on along the tube, and a lead from its tip back
// into the board.
func (job signJob) electrodes(path []maths.Vec, em float64) []placedElement {
	drawing := job.drawing
	size, length := em*drawing.fixed("sign-electrode-size"),
		em*drawing.fixed("sign-electrode-length")
	last := len(path) - 1
	var parts []placedElement
	for _, end := range [][2]maths.Vec{{path[0], path[1]},
		{path[last], path[last-1]}} {
		tip := end[0].Add(end[0].Sub(end[1]).Unit().Scale(length))
		parts = append(parts, strut("sign electrode", end[0], tip, size,
			job.steel), strut("sign lead", maths.V(tip.X, tip.Y, 0), tip,
			size, job.frame))
	}
	return parts
}
