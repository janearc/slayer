package hair

import "github.com/janearc/slayer/maths"

// GuideTail extends a guide with a second cubic beyond its Tip.
// At gives the fraction of the whole parameter at the join. Bend and
// Tip use the original root axes and length units. The second cubic's
// first handle is derived from the preceding guide to match tangents.
type GuideTail struct {
	Bend, Tip maths.Vec
	At        float64
}

func (tail GuideTail) active() bool { return tail != (GuideTail{}) }

func (guide Guide) tailStart() maths.Vec {
	share := (1 - guide.Tail.At) / guide.Tail.At
	return guide.Tip.Add(guide.Tip.Sub(guide.Bend).Scale(share))
}

func (guide Guide) validTail() bool {
	if !guide.Tail.active() {
		return true
	}
	return guide.Tail.At >= .2 && guide.Tail.At <= .85 &&
		guide.Tail.Bend.Len() <= 1 && guide.Tail.Tip.Len() <= 1 && guide.tailStart().Len() <= 1 &&
		guide.Tail.Tip.Sub(guide.Tail.Bend).Len() > leastVector
}

func (guide Guide) retaining(t float64) float64 {
	if !guide.Tail.active() {
		return 1
	}
	return 1 - maths.Smooth(guide.Tail.At-.07, guide.Tail.At+.14, t)
}
