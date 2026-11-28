package hair

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// Guide shapes a rooted cubic curve. Lead, Bend and Tip are measured
// in units of the strand's Length along the forward, outward and
// transverse axes of the root frame. A zero guide uses the angular shaft.
type Guide struct {
	Lead, Bend, Tip maths.Vec
	Wave            GuideWave
	Tail            GuideTail
}

func (guide Guide) active() bool { return guide != (Guide{}) }

func (guide Guide) valid() bool {
	return !guide.active() ||
		(guide.Lead.X > 0 && guide.Bend.X > guide.Lead.X && guide.Tip.X > guide.Bend.X &&
			guide.Lead.Y >= 0 && guide.Lead.Len() <= 1 && guide.Bend.Len() <= 1 && guide.Tip.Len() <= 1 &&
			guide.Wave.valid() && guide.validTail())
}

func (guide Guide) point(t float64) maths.Vec {
	var centre maths.Vec
	switch {
	case !guide.Tail.active():
		left := 1 - t
		return guide.Lead.Scale(3 * t * left * left).
			Add(guide.Bend.Scale(3 * t * t * left)).
			Add(guide.Tip.Scale(t * t * t)).Add(guide.Wave.offset(t))
	case t <= guide.Tail.At:
		centre = cubic(maths.Vec{}, guide.Lead, guide.Bend, guide.Tip, t/guide.Tail.At)
	default:
		centre = cubic(guide.Tip, guide.tailStart(), guide.Tail.Bend, guide.Tail.Tip,
			(t-guide.Tail.At)/(1-guide.Tail.At))
	}
	return centre.Add(guide.Wave.offset(t))
}

func cubic(root, lead, bend, tip maths.Vec, t float64) maths.Vec {
	left := 1 - t
	return root.Scale(left * left * left).Add(lead.Scale(3 * t * left * left)).
		Add(bend.Scale(3 * t * t * left)).Add(tip.Scale(t * t * t))
}

type shaftFrame struct{ along, up, side maths.Vec }

func (frame shaftFrame) out(point maths.Vec) maths.Vec {
	return frame.along.Scale(point.X).Add(frame.up.Scale(point.Y)).Add(frame.side.Scale(point.Z))
}

// Clumps gives nearby chart roots a shared target for the free part of
// their shaft. Spacing is in chart units. It jitters cell centres so that
// the clusters vary in spacing and root footprint with the seed.
type Clumps struct {
	Seed    uint64
	Spacing float64
}

// Target returns the nearest jittered guide centre. A nonpositive or
// unrepresentable spacing leaves the point at its original place.
func (clumps Clumps) Target(point maths.Vec) maths.Vec {
	return clumps.BlendTarget(point, 0)
}

// BlendTarget feathers the join between neighbouring clump targets.
// Feather is the distance from the closest target over which other
// targets contribute. With zero feather each target has a hard boundary.
func (clumps Clumps) BlendTarget(point maths.Vec, feather float64) maths.Vec {
	if !(clumps.Spacing > 0 && finite(clumps.Spacing)) {
		return point
	}
	at := point.Scale(1 / clumps.Spacing)
	if !(at.Len() < 1<<48) {
		return point
	}
	feather /= clumps.Spacing

	column, row, depth := int64(math.Floor(at.X)), int64(math.Floor(at.Y)), int64(math.Floor(at.Z))
	best, distance := at, math.Inf(1)
	var nearby [27]clumpTarget
	count := 0
	for z := depth - 1; z <= depth+1; z++ {
		for y := row - 1; y <= row+1; y++ {
			for x := column - 1; x <= column+1; x++ {
				target := clumps.centre(x, y, z)
				delta := target.Sub(at)
				square := delta.Dot(delta)
				if square < distance {
					best, distance = target, square
				}
				nearby[count] = clumpTarget{target, square}
				count++
			}
		}
	}
	if !(feather > 0 && finite(feather)) {
		return best.Scale(clumps.Spacing)
	}
	distance = math.Sqrt(distance)
	var sum maths.Vec
	total := 0.
	for _, neighbour := range nearby {
		weight := 1 - maths.Smooth(0, feather, math.Sqrt(neighbour.squared)-distance)
		sum = sum.Add(neighbour.point.Scale(weight))
		total += weight
	}
	return sum.Scale(clumps.Spacing / total)
}

type clumpTarget struct {
	point   maths.Vec
	squared float64
}

func (clumps Clumps) centre(x, y, z int64) maths.Vec {
	return maths.V(
		float64(x)+.29+.42*noise.Hash3(clumps.Seed, x, y, z),
		float64(y)+.29+.42*noise.Hash3(clumps.Seed+1, x, y, z),
		float64(z)+.29+.42*noise.Hash3(clumps.Seed+2, x, y, z),
	)
}
