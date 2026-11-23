package cities

import (
	"fmt"
	"math"
)

// fullTurn is a whole turn about a point, in radians.
const fullTurn = 2 * math.Pi

// outlineCorner is one corner of an outline, on the ground plane: how far along
// the x axis and the z axis it stands from the outline's own origin.
type outlineCorner struct{ X, Z float64 }

// outline is a closed shape on the ground plane, said as the corners a
// renderer joins in order. A level says how many a curved outline has.
type outline interface {
	Points(level Detail) ([]outlineCorner, error)
}

// outlineCircle is a circle about its own origin. Its side count is data: each
// detail level is cut into as many flat sides as the data names.
type outlineCircle struct{ Radius float64 }

// Points is the circle's corners, counter-clockwise from its x axis.
func (circle outlineCircle) Points(level Detail) ([]outlineCorner, error) {
	if !(circle.Radius > 0) {
		return nil, fmt.Errorf("cities: a circle needs a positive radius, "+
			"not %v", circle.Radius)
	}
	sides, err := sidesAt(level)
	if err != nil {
		return nil, err
	}
	corners := make([]outlineCorner, sides)
	for index := range corners {
		angle := fullTurn * float64(index) / float64(sides)
		corners[index] = outlineCorner{circle.Radius * math.Cos(angle),
			circle.Radius * math.Sin(angle)}
	}
	return corners, nil
}

// polygon is an outline of the corners it is given, the same at every
// level.
type polygon struct{ Corners []outlineCorner }

// Points is the polygon's corners, a copy of those it was given. A
// polygon of fewer than three corners encloses nothing and is refused.
func (polygon polygon) Points(Detail) ([]outlineCorner, error) {
	if len(polygon.Corners) < leastSides {
		return nil, fmt.Errorf("cities: a polygon needs at least %d "+
			"corners, not %d", leastSides, len(polygon.Corners))
	}
	return append([]outlineCorner(nil), polygon.Corners...), nil
}

// convex is whether every turn of the outline, corner to corner, is the
// same way. A straight run of corners is no turn either way.
func convex(corners []outlineCorner) bool {
	var turn float64
	for index := range corners {
		first := corners[index]
		second := corners[(index+1)%len(corners)]
		third := corners[(index+2)%len(corners)]
		cross := (second.X-first.X)*(third.Z-second.Z) -
			(second.Z-first.Z)*(third.X-second.X)
		if cross*turn < 0 {
			return false
		}
		if turn == 0 {
			turn = cross
		}
	}
	return true
}
