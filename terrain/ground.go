package terrain

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// Hole is a rectangle of a window that its ground leaves undrawn, in
// metres east and south of the window's first point, for a finer window
// laid over that part of it.
type Hole struct {
	East, South, Across, Down float64
}

// holds is whether a point, in metres east and south of the window's first
// point, is in the hole or on its edge.
func (hole Hole) holds(east, south float64) bool {
	return east >= hole.East && east <= hole.East+hole.Across &&
		south >= hole.South && south <= hole.South+hole.Down
}

// Lowest is the lowest height in the window.
func (window Window) Lowest() float64 {
	lowest := window.heights[0]
	for _, height := range window.heights {
		lowest = min(lowest, height)
	}
	return lowest
}

// Ground is a window's heights as a mesh to stand a scene on: x east and
// z south from the window's first point, y up from the lowest height in
// it, which it says, so a scene stands near its own origin.
//
// two triangles a cell, each facing up, shaded smooth: each corner's
// normal is the ground's slope there, read across the corners either
// side, so the light rolls across the cells and not in steps.
func (window Window) Ground() (*mesh.Mesh, float64, error) {
	lowest := window.Lowest()
	made, err := window.GroundOn(0, 0, lowest, nil)
	return made, lowest, err
}

// GroundOn is the window's ground as Ground makes it, but stood where a
// scene says: its first point east and south of the scene's origin by the
// metres given, and y up from the floor given, which is the scene's own
// lowest height and not the window's. A cell whose four corners are all
// in the hole is left out, so a finer window can be laid there; a nil hole
// leaves out none.
func (window Window) GroundOn(east, south, floor float64,
	hole *Hole) (*mesh.Mesh, error) {
	points := make([]maths.Vec, 0, len(window.heights))
	normals := make([]maths.Vec, 0, len(window.heights))
	for row := range window.down {
		for column := range window.across {
			point := maths.V(east+float64(column)*window.step,
				window.At(column, row)-floor,
				south+float64(row)*window.step)
			points = append(points, point)
			rise, fall := window.slope(column, row)
			normals = append(normals, maths.V(-rise, 1, -fall))
		}
	}
	faces := make([][3]int, 0, 2*(window.across-1)*(window.down-1))
	for row := range window.down - 1 {
		for column := range window.across - 1 {
			if window.cellInHole(column, row, hole) {
				continue
			}
			near := row*window.across + column
			far := near + window.across
			faces = append(faces, [3]int{near, far + 1, near + 1},
				[3]int{near, far, far + 1})
		}
	}
	return mesh.New(points, faces, normals, true)
}

// cellInHole is whether all four corners of the cell whose first corner
// is at a column and a row are in the hole, which may be nil.
func (window Window) cellInHole(column, row int, hole *Hole) bool {
	if hole == nil {
		return false
	}
	for _, corner := range [][2]int{{column, row}, {column + 1, row},
		{column, row + 1}, {column + 1, row + 1}} {
		if !hole.holds(float64(corner[0])*window.step,
			float64(corner[1])*window.step) {
			return false
		}
	}
	return true
}

// Rise is how steeply the ground climbs at a point of the window, across
// and down, in metres a metre along its steepest way: the east and south
// rises its slope reads, taken together.
func (window Window) Rise(across, down int) float64 {
	east, south := window.slope(across, down)
	return math.Hypot(east, south)
}

// slope is how fast the ground rises east and south at a point of the
// window, metres a metre: across the points either side, or the one
// beside it at the window's edge.
func (window Window) slope(column, row int) (east, south float64) {
	west, eastward := max(0, column-1), min(window.across-1, column+1)
	north, southward := max(0, row-1), min(window.down-1, row+1)
	east = (window.At(eastward, row) - window.At(west, row)) /
		(float64(eastward-west) * window.step)
	south = (window.At(column, southward) - window.At(column, north)) /
		(float64(southward-north) * window.step)
	return east, south
}
