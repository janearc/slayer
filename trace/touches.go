package trace

// Whether a ray meets a thing, which a shadow ray asks and a look does
// not: it needs no distance, and a shape that can say whether in less
// work than where says so by the interface here.

import (
	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/maths"
)

// toucher is a shape that says whether a ray meets it between near and
// far, in its own lengths, which can take less work than saying where:
// shape.Hitter's answer to whether.
type toucher interface {
	Meets(ray maths.Ray, near, far float64) bool
}

// The sea says it in that way.
var _ toucher = (*arid.Sea)(nil)

// touches is whether a ray meets this thing between near and far, in the
// world's distances, as meet finds it. A shape that can say whether
// without saying where (toucher) is asked that.
func (placed *placedThing) touches(ray maths.Ray, near, far float64) bool {
	if solid, has := placed.thing.Shape.(toucher); has {
		return solid.Meets(placed.at.RayIn(ray), near/placed.scale,
			far/placed.scale)
	}
	_, met := placed.meet(ray, near, far)
	return met
}
