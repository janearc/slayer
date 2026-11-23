package trace

// What a march takes as a touch of a surface, and the lift that clears it.

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// leaveTouches is how many touches off a marched surface a ray leaving
// it starts. A march stops within one touch of the surface, on either
// side of it, so a start two touches along the normal is a touch clear.
const leaveTouches = 2

// touched is how close to its surface a march takes a point as met:
// touch's share of the larger of size, the shape's reach, and away, the
// point's distance from the shape's own origin, in one frame's lengths.
// It is the one place the share is worked, for the march and the lift.
func touched(size, away float64) float64 {
	return touch * max(size, away)
}

// marchReach is a shape's reach as touched measures it: that of its box,
// or nothing for a shape with no edge, which is measured where the march
// is.
func marchReach(box maths.Box) float64 {
	size := reach(box)
	if math.IsInf(size, 0) {
		return 0
	}
	return size
}

// marchedThing is one thing a march draws, as the lift sees it: its box
// in its own space, widened as boxes are, its reach in the world's
// lengths, and its space in the world's.
type marchedThing struct {
	box  maths.Box
	size float64
	at   maths.Subspace
}

// holds is whether a point of the world lies in the thing's box, which a
// march of it never ends outside.
func (each marchedThing) holds(point maths.Vec) bool {
	local := each.at.In(point)
	return local.X >= each.box.Min.X && local.X <= each.box.Max.X &&
		local.Y >= each.box.Min.Y && local.Y <= each.box.Max.Y &&
		local.Z >= each.box.Min.Z && local.Z <= each.box.Max.Z
}

// marchedThings are the things a march draws: those that no shape
// answers for exactly, and that the lift must clear.
type marchedThings []marchedThing

// marchedBy is the things among those placed that are marched.
func marchedBy(things []placedThing) marchedThings {
	var found marchedThings
	for _, each := range things {
		if _, exact := each.thing.Shape.(shape.Hitter); exact {
			continue
		}
		box := each.thing.Shape.Bounds()
		found = append(found, marchedThing{box: widened(box),
			size: each.scale * marchReach(box), at: each.at})
	}
	return found
}

// clear is how far off a marched surface a ray leaving it starts, from a
// point: of each marched thing the point is in the box of, leaveTouches
// of touched, for its reach and the point's distance from the origin of
// its own space, which is the march's own measure in the world's
// lengths; the most of them. Nothing where nothing is marched.
//
// It is only the marched part of the lift. The lift a ray gets is
// lifting's: the larger of this and the scene-wide share, lift of the
// larger of the spread and the point's distance from the world's origin.
//
// A ball of radius one at its own origin has a reach of two root three,
// 3.46, and is met within 3.5 millionths of a metre; clear is 6.9
// millionths there, at the world's origin or a hundred kilometres out.
//
// At the origin the lift is that, over the scene's 3.5 millionths. A
// hundred kilometres out the scene-wide share is a millionth of that
// distance, a tenth of a metre, and that is the lift, of every surface
// there.
//
// A ball whose own middle is a hundred kilometres from its origin is
// met within a tenth of a metre, and clear is two tenths. A point
// outside every marched box is cleared of nothing marched.
func (these marchedThings) clear(point maths.Vec) float64 {
	var most float64
	for _, each := range these {
		if !each.holds(point) {
			continue
		}
		away := point.Sub(each.at.Origin).Len()
		most = max(most, leaveTouches*touched(each.size, away))
	}
	return most
}
