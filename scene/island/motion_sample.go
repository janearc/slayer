package island

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/terrain"
)

// surfaceSampler locates one endpoint's regular lattice faces without reading
// source heights again. Values refer to its immutable prepared meshes.
type surfaceSampler struct {
	prepared Prepared
	leaves   map[terrain.Patch]int
	root     terrain.Patch
	side     int
}

// sampler checks the regular face layout before any interpolation.
func (ground *Terrain) sampler(prepared Prepared) (surfaceSampler, error) {
	if len(prepared.Patches) != len(prepared.Surfaces) {
		return surfaceSampler{}, fmt.Errorf("island: inconsistent transition endpoint")
	}
	sample := surfaceSampler{prepared: prepared, leaves: make(map[terrain.Patch]int), root: ground.root, side: ground.detail.Side}
	for index, patch := range prepared.Patches {
		if prepared.Surfaces[index].FaceCount() != 2*(sample.side-1)*(sample.side-1) {
			return surfaceSampler{}, fmt.Errorf("island: terrain transition needs regular lattice faces")
		}
		sample.leaves[patch] = index
	}
	return sample, nil
}

// leaf follows the same source quadtree, assigning shared boundaries to their
// upper-index child except at the complete region's maximum edge.
func (sample surfaceSampler) leaf(key lattice) (terrain.Patch, int, error) {
	patch := sample.root
	for level := 0; level <= sample.root.Level+20; level++ {
		if index, present := sample.leaves[patch]; present {
			return patch, index, nil
		}
		children, err := patch.Children()
		if err != nil {
			return terrain.Patch{}, 0, err
		}
		found := false
		for _, child := range children {
			column := key.column >= child.MinColumn && (key.column < child.MaxColumn || key.column == child.MaxColumn && child.MaxColumn == sample.root.MaxColumn)
			row := key.row >= child.MinRow && (key.row < child.MaxRow || key.row == child.MaxRow && child.MaxRow == sample.root.MaxRow)
			if column && row {
				patch, found = child, true
				break
			}
		}
		if !found {
			break
		}
	}
	return terrain.Patch{}, 0, fmt.Errorf("island: transition endpoint does not cover its lattice")
}

// at evaluates the endpoint's actual triangles, including their stitched edges.
// The diagonal and barycentric weights match vertexBook.surface exactly.
func (sample surfaceSampler) at(key lattice) (maths.Vec, maths.Vec, error) {
	patch, index, err := sample.leaf(key)
	if err != nil {
		return maths.Vec{}, maths.Vec{}, err
	}
	step := float64(patch.MaxColumn-patch.MinColumn) / float64(sample.side-1)
	x, y := float64(key.column-patch.MinColumn)/step, float64(key.row-patch.MinRow)/step
	column, row := min(int(math.Floor(x)), sample.side-2), min(int(math.Floor(y)), sample.side-2)
	x, y = x-float64(column), y-float64(row)
	face := 2 * (row*(sample.side-1) + column)
	u, v := y, x-y
	if y > x {
		face++
		u, v = y-x, x
	}
	mesh := sample.prepared.Surfaces[index]
	a, b, c := mesh.Corners(face)
	point := a.Add(b.Sub(a).Scale(u)).Add(c.Sub(a).Scale(v)).Add(sample.prepared.Origin)
	var normal maths.Vec
	if normals, smooth := mesh.CornerNormals(face); smooth {
		normal = normals[0].Scale(1 - u - v).Add(normals[1].Scale(u)).Add(normals[2].Scale(v))
	}
	return point, normal, nil
}

// commonPatches forms the finest leaf needed by either endpoint. Endpoint
// coverage is inherited down the tree; no source region can disappear.
func commonPatches(root terrain.Patch, from, to map[terrain.Patch]int, limit int) ([]terrain.Patch, error) {
	var leaves []terrain.Patch
	var visit func(terrain.Patch, bool, bool) error
	visit = func(patch terrain.Patch, first, second bool) error {
		_, a := from[patch]
		_, b := to[patch]
		first, second = first || a, second || b
		if first && second {
			if len(leaves) >= limit {
				return fmt.Errorf("island: common terrain patches exceed transition allowance")
			}
			leaves = append(leaves, patch)
			return nil
		}
		children, err := patch.Children()
		if err != nil || len(children) == 0 {
			return fmt.Errorf("island: incomplete transition endpoint coverage")
		}
		for _, child := range children {
			if err := visit(child, first, second); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(root, false, false); err != nil {
		return nil, err
	}
	return leaves, nil
}
