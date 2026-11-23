// Package beams makes the dimensioned solids used by coastal structures.
package beams

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// Box returns a rectangular solid centred and oriented at the given pose.
func Box(at, size maths.Vec, turn maths.Orientation, worn *thing.Material) (thing.Thing, error) {
	half := size.Scale(.5)
	x, y, z := half.X, half.Y, half.Z
	points := []maths.Vec{
		maths.V(-x, -y, -z), maths.V(x, -y, -z),
		maths.V(x, y, -z), maths.V(-x, y, -z),
		maths.V(-x, -y, z), maths.V(x, -y, z),
		maths.V(x, y, z), maths.V(-x, y, z),
	}
	faces := [][3]int{{0, 3, 2}, {0, 2, 1}, {4, 5, 6}, {4, 6, 7},
		{0, 4, 7}, {0, 7, 3}, {1, 2, 6}, {1, 6, 5},
		{0, 1, 5}, {0, 5, 4}, {3, 7, 6}, {3, 6, 2}}
	solid, err := mesh.New(points, faces, nil, false)
	return thing.Thing{Shape: solid, Motion: thing.Still(maths.NewSubspace(at, turn, 1)), Material: worn}, err
}

// Span makes a horizontal deck between two points on a reference plane.
func Span(from, to maths.Vec, width, deck, thickness, overlap float64,
	worn *thing.Material) (thing.Thing, error) {
	way := to.Sub(from)
	at := from.Add(to).Scale(.5)
	turn := maths.Rotation(maths.Up, math.Atan2(-way.Z, way.X))
	return Box(at.Add(maths.V(0, deck-thickness*.5, 0)),
		maths.V(way.Len()+overlap, thickness, width), turn, worn)
}

// Rod returns a square-section beam with endpoints from and to.
func Rod(from, to maths.Vec, width float64, worn *thing.Material) (thing.Thing, error) {
	way := to.Sub(from)
	return Box(from.Add(to).Scale(.5), maths.V(width, way.Len(), width), maths.Upright(way), worn)
}
