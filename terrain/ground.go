package terrain

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// Ground is a window's heights as a mesh to stand a scene on: x east and
// z south from the window's first point, y up from the lowest height in
// it, which it says, so a scene stands near its own origin.
//
// two triangles a cell, each facing up, shaded smooth: each corner's
// normal is the ground's slope there, read across the corners either
// side, so the light rolls across the cells and not in steps.
func (window Window) Ground() (*mesh.Mesh, float64, error) {
	lowest := window.heights[0]
	for _, height := range window.heights {
		lowest = min(lowest, height)
	}
	points := make([]maths.Vec, 0, len(window.heights))
	normals := make([]maths.Vec, 0, len(window.heights))
	for row := range window.down {
		for column := range window.across {
			points = append(points, maths.V(float64(column)*window.step,
				window.At(column, row)-lowest, float64(row)*window.step))
			east, south := window.slope(column, row)
			normals = append(normals, maths.V(-east, 1, -south))
		}
	}
	faces := make([][3]int, 0, 2*(window.across-1)*(window.down-1))
	for row := range window.down - 1 {
		for column := range window.across - 1 {
			near := row*window.across + column
			far := near + window.across
			faces = append(faces, [3]int{near, far + 1, near + 1},
				[3]int{near, far, far + 1})
		}
	}
	made, err := mesh.New(points, faces, normals, true)
	return made, lowest, err
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
