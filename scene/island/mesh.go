package island

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/frame"
	"github.com/janearc/slayer/terrain"
)

// surface matches unequal-resolution edges after spherical projection. The
// coarse edge's actual world-space segments are authoritative, so a curved
// mapping cannot reopen the gap left by source-height-only stitching.
func (book *vertexBook) surface(patch terrain.Patch,
	origin frame.Relative,
) (*mesh.Mesh, error) {
	side := book.terrain.detail.Side
	step := (patch.MaxColumn - patch.MinColumn) / (side - 1)
	points := make([]maths.Vec, side*side)
	var normals []maths.Vec
	if book.normals != nil {
		normals = make([]maths.Vec, side*side)
	}
	for row := range side {
		for column := range side {
			key := lattice{patch.MinColumn + column*step, patch.MinRow + row*step}
			value, err := book.conform(key, patch.Level)
			if err != nil {
				return nil, err
			}
			if normals != nil {
				normals[row*side+column], err = book.conformValue(key, patch.Level, book.normalAt)
				if err != nil {
					return nil, err
				}
			}
			points[row*side+column], err = origin.Point(value)
			if err != nil {
				return nil, err
			}
		}
	}
	faces := make([][3]int, 0, 2*(side-1)*(side-1))
	for row := range side - 1 {
		for column := range side - 1 {
			near := row*side + column
			far := near + side
			faces = append(faces, [3]int{near, far + 1, near + 1}, [3]int{near, far, far + 1})
		}
	}
	return mesh.New(points, faces, normals, normals != nil)
}

// conform follows coarser edge knots recursively. Each recursive call has a
// strictly smaller level; several refinement levels may meet without a loop.
// The source book remains unchanged, and every patch sharing a knot computes
// the same complete world point before render-origin subtraction.
func (book *vertexBook) conform(key lattice, level int) (maths.Vec, error) {
	return book.conformValue(key, level, book.at)
}

// conformValue uses the same coarse-edge knots for geometry and optional
// shading normals. Mesh construction normalizes the blended normal vectors.
func (book *vertexBook) conformValue(key lattice, level int, at func(lattice) (maths.Vec, error)) (maths.Vec, error) {
	coarse := book.edges.coarser(key, level)
	if coarse == nil {
		return at(key)
	}
	step := (coarse.MaxColumn - coarse.MinColumn) / (book.terrain.detail.Side - 1)
	first, last := key, key
	var portion float64
	if key.column == coarse.MinColumn || key.column == coarse.MaxColumn {
		first.row = coarse.MinRow + (key.row-coarse.MinRow)/step*step
		last.row = min(first.row+step, coarse.MaxRow)
		if first.row == key.row {
			return book.conformValue(key, coarse.Level, at)
		}
		portion = float64(key.row-first.row) / float64(last.row-first.row)
	} else {
		first.column = coarse.MinColumn + (key.column-coarse.MinColumn)/step*step
		last.column = min(first.column+step, coarse.MaxColumn)
		if first.column == key.column {
			return book.conformValue(key, coarse.Level, at)
		}
		portion = float64(key.column-first.column) / float64(last.column-first.column)
	}
	begin, err := book.conformValue(first, coarse.Level, at)
	if err != nil {
		return maths.Vec{}, err
	}
	end, err := book.conformValue(last, coarse.Level, at)
	if err != nil {
		return maths.Vec{}, err
	}
	return begin.Add(end.Sub(begin).Scale(portion)), nil
}
