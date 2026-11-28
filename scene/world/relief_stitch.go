package world

import (
	"fmt"

	"github.com/janearc/slayer/maths"
)

// ReliefEdge names a patch edge in its face-local grid. Its direction is
// immaterial to a stitch; the same vertex keys can be read in either order.
type ReliefEdge uint8

const (
	ReliefLowX ReliefEdge = iota
	ReliefHighX
	ReliefLowY
	ReliefHighY
)

// ReliefEdgeKeys are the endpoints and next-level midpoint of an edge.
// Midpoint is a topological key; its field-sampled point generally does
// not lie on the straight edge of a coarse triangle.
type ReliefEdgeKeys struct {
	First, Midpoint, Last ReliefVertex
}

// EdgeKeys names a patch edge and its midpoint one level finer. It refuses
// an unknown edge or a midpoint beyond the supported patch depth.
func (p ReliefPatch) EdgeKeys(edge ReliefEdge) (ReliefEdgeKeys, error) {
	if int(p.level) >= reliefDeepest {
		return ReliefEdgeKeys{}, fmt.Errorf("world: cannot split a deepest relief edge")
	}
	width := 1 << (p.level + 1)
	x, y := p.x*2, p.y*2
	var ax, ay, bx, by, mx, my int
	switch edge {
	case ReliefLowX:
		ax, ay, bx, by, mx, my = x, y, x, y+2, x, y+1
	case ReliefHighX:
		ax, ay, bx, by, mx, my = x+2, y, x+2, y+2, x+2, y+1
	case ReliefLowY:
		ax, ay, bx, by, mx, my = x, y, x+2, y, x+1, y
	case ReliefHighY:
		ax, ay, bx, by, mx, my = x, y+2, x+2, y+2, x+1, y+2
	default:
		return ReliefEdgeKeys{}, fmt.Errorf("world: unknown relief patch edge")
	}
	return ReliefEdgeKeys{First: reliefVertex(p.face, width, ax, ay),
		Midpoint: reliefVertex(p.face, width, mx, my),
		Last:     reliefVertex(p.face, width, bx, by)}, nil
}

// ReliefConstraints holds vertex positions for stitching projected edges.
// It does not balance a quadtree, construct triangles, or establish an
// error bound. A builder must add constraints from coarser levels to finer
// ones, then use Point for every occurrence of a key on both sides.
type ReliefConstraints struct {
	field  *Relief
	sea    bool
	points map[ReliefVertex]maths.Vec
}

// NewReliefConstraints starts a set of constraints for one field and sea
// choice. The field and sea choice must stay the same for all joined faces.
func NewReliefConstraints(field *Relief, sea bool) (*ReliefConstraints, error) {
	if field == nil {
		return nil, fmt.Errorf("world: relief constraints need a field")
	}
	return &ReliefConstraints{field: field, sea: sea,
		points: make(map[ReliefVertex]maths.Vec)}, nil
}

// Point gives a constrained vertex position in metres, or samples the
// shared field where no constraint was placed.
func (c *ReliefConstraints) Point(vertex ReliefVertex) (maths.Vec, error) {
	if c == nil || c.field == nil {
		return maths.Vec{}, fmt.Errorf("world: relief constraints need a field")
	}
	if point, ok := c.points[vertex]; ok {
		return point, nil
	}
	return c.field.Point(vertex, c.sea)
}

// Stitch places an edge's fine midpoint on the straight segment between
// its coarse endpoints, after spherical projection and displacement. It
// returns the metres moved from the field sample, for the caller's error
// allowance. It refuses conflicting prior constraints rather than choosing
// one. The caller must keep the coarse edge and the fine vertices in the
// actual mesh; this operation alone cannot close a crack.
func (c *ReliefConstraints) Stitch(edge ReliefEdgeKeys) (float64, error) {
	if c == nil || c.field == nil {
		return 0, fmt.Errorf("world: relief constraints need a field")
	}
	if edge.First == edge.Last || edge.Midpoint == edge.First || edge.Midpoint == edge.Last {
		return 0, fmt.Errorf("world: relief edge has repeated vertices")
	}
	first, err := c.Point(edge.First)
	if err != nil {
		return 0, err
	}
	last, err := c.Point(edge.Last)
	if err != nil {
		return 0, err
	}
	underlying, err := c.field.Point(edge.Midpoint, c.sea)
	if err != nil {
		return 0, err
	}
	joined := first.Add(last).Scale(0.5)
	if prior, exists := c.points[edge.Midpoint]; exists && prior != joined {
		return 0, fmt.Errorf("world: conflicting relief edge constraints")
	}
	c.points[edge.Midpoint] = joined
	return joined.Sub(underlying).Len(), nil
}
