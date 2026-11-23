package skeleton

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Skin weighed by nearness: a point of the hide is held by every bone in
// proportion to how near the bone's middle line is, the distance divided
// by the bone's reach and raised to the fourth power. Near a joint two
// bones share the point, so the skin bends there without a seam. A bone
// that should hold only the skin close round it, as a limb bone inside the
// body wall, is given a short reach.

// nearHold is the distance, in reaches, inside which a bone holds a point
// as firmly as at its own line, so the weights stay finite on a bone.
const nearHold = .02

// Holder is one bone that holds the skin: its place in the skeleton, its
// middle line at rest, and its reach.
type Holder struct {
	bone  int
	line  []maths.Vec
	reach float64
}

// NewHolder is bone at place bone, with middle line line, holding the skin
// out to reach.
func NewHolder(bone int, line []maths.Vec, reach float64) Holder {
	return Holder{bone: bone, line: line, reach: reach}
}

// Nearness is how firmly each holder holds rest point p, as influences for
// Carry, one for each holder.
func Nearness(p maths.Vec, holders []Holder) []Influence {
	influences := make([]Influence, len(holders))
	for i, holder := range holders {
		far := lineDistance(p, holder.line) / holder.reach
		weight := 1 / math.Pow(far*far+nearHold*nearHold, 2)
		influences[i] = Influence{bone: holder.bone, weight: weight}
	}
	return influences
}

// lineDistance is how far p lies from a line of points; a line of one
// point is that point.
func lineDistance(p maths.Vec, line []maths.Vec) float64 {
	if len(line) == 1 {
		return p.Sub(line[0]).Len()
	}
	nearest := math.Inf(1)
	for i := 1; i < len(line); i++ {
		nearest = math.Min(nearest, segmentDistance(p, line[i-1], line[i]))
	}
	return nearest
}

// segmentDistance is how far p lies from the segment from a to b.
func segmentDistance(p, a, b maths.Vec) float64 {
	axis := b.Sub(a)
	share := p.Sub(a).Dot(axis) / axis.Dot(axis)
	share = math.Max(0, math.Min(1, share))
	foot := a.Add(axis.Scale(share))
	return p.Sub(foot).Len()
}
