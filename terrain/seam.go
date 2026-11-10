package terrain

import (
	"fmt"
	"math"
	"slices"
	"sort"
)

// StitchTo returns an independent sample grid whose shared edge follows its
// coarser neighbour. Both grids must be from the same source placement. This
// joins source-space height fields under an affine world mapping; a curved
// world needs stitching after projection instead.
//
// Every coarse edge knot within the overlap must already be a fine knot.
// Missing coarse heights remain missing. The returned shift is the largest
// changed sample height; the caller must include it in its error allowance.
func (grid PatchGrid) StitchTo(coarse PatchGrid, limitBytes int64) (PatchGrid, float64, error) {
	if !grid.usableSeamGrid() || !coarse.usableSeamGrid() ||
		grid.west != coarse.west || grid.north != coarse.north ||
		grid.spacing != coarse.spacing || grid.rowStep != coarse.rowStep || grid.reference != coarse.reference ||
		grid.patch.Level <= coarse.patch.Level {
		return PatchGrid{}, 0, fmt.Errorf("terrain: incompatible seam grids")
	}
	var fineAxis, coarseAxis []int
	var fineIndices, coarseIndices seamEdge
	switch {
	case grid.patch.MaxColumn == coarse.patch.MinColumn:
		fineAxis, coarseAxis = grid.rows, coarse.rows
		fineIndices = grid.edge(true, len(grid.columns)-1)
		coarseIndices = coarse.edge(true, 0)
	case grid.patch.MinColumn == coarse.patch.MaxColumn:
		fineAxis, coarseAxis = grid.rows, coarse.rows
		fineIndices = grid.edge(true, 0)
		coarseIndices = coarse.edge(true, len(coarse.columns)-1)
	case grid.patch.MaxRow == coarse.patch.MinRow:
		fineAxis, coarseAxis = grid.columns, coarse.columns
		fineIndices = grid.edge(false, len(grid.rows)-1)
		coarseIndices = coarse.edge(false, 0)
	case grid.patch.MinRow == coarse.patch.MaxRow:
		fineAxis, coarseAxis = grid.columns, coarse.columns
		fineIndices = grid.edge(false, 0)
		coarseIndices = coarse.edge(false, len(coarse.rows)-1)
	default:
		return PatchGrid{}, 0, fmt.Errorf("terrain: seam grids are not adjacent")
	}
	if fineAxis[0] < coarseAxis[0] ||
		fineAxis[len(fineAxis)-1] > coarseAxis[len(coarseAxis)-1] {
		return PatchGrid{}, 0, fmt.Errorf("terrain: coarse edge does not cover fine edge")
	}
	for _, knot := range coarseAxis {
		if knot >= fineAxis[0] && knot <= fineAxis[len(fineAxis)-1] &&
			!containsIndex(fineAxis, knot) {
			return PatchGrid{}, 0, fmt.Errorf("terrain: fine edge misses a coarse knot")
		}
	}
	if limitBytes < 0 || int64(len(grid.heights)) > limitBytes/(gridHeightBytes+gridAvailabilityBytes) {
		return PatchGrid{}, 0, fmt.Errorf("terrain: seam exceeds output byte budget")
	}
	joined := grid
	joined.heights = slices.Clone(grid.heights)
	joined.present = slices.Clone(grid.present)
	largest := 0.0
	for i, knot := range fineAxis {
		height, present := coarse.edgeHeight(knot, coarseAxis, coarseIndices)
		index := fineIndices.at(i)
		// Stitching does not fill source holes in either grid.
		if !grid.present[index] || !present {
			joined.present[index] = false
			continue
		}
		shift := math.Abs(height - grid.heights[index])
		if !finiteCoordinate(height) || !finiteCoordinate(shift) {
			return PatchGrid{}, 0, fmt.Errorf("terrain: non-finite seam interpolation")
		}
		largest = max(largest, shift)
		joined.heights[index] = height
	}
	return joined, largest, nil
}

// containsIndex reports whether a sorted axis holds a knot.
func containsIndex(axis []int, knot int) bool {
	index := sort.SearchInts(axis, knot)
	return index < len(axis) && axis[index] == knot
}

// usableSeamGrid rejects unconstructed grids before array indexing.
func (grid PatchGrid) usableSeamGrid() bool {
	return grid.patch.valid() && len(grid.columns) >= 2 && len(grid.rows) >= 2 &&
		len(grid.heights)/len(grid.columns) == len(grid.rows) &&
		len(grid.heights)%len(grid.columns) == 0 &&
		len(grid.present) == len(grid.heights)
}

type seamEdge struct{ first, stride int }

// at is the grid index of the edge's index-th sample.
func (edge seamEdge) at(index int) int { return edge.first + index*edge.stride }

// edge walks the grid along a fixed column (vertical) or row.
func (grid PatchGrid) edge(vertical bool, fixed int) seamEdge {
	if vertical {
		return seamEdge{first: fixed, stride: len(grid.columns)}
	}
	return seamEdge{first: fixed * len(grid.columns), stride: 1}
}

// edgeHeight interpolates between adjacent native knots, preserving exact
// endpoint values and refusing interpolation across missing source heights.
func (grid PatchGrid) edgeHeight(knot int, axis []int, indices seamEdge) (float64, bool) {
	next := sort.SearchInts(axis, knot)
	if next < len(axis) && axis[next] == knot {
		index := indices.at(next)
		return grid.heights[index], grid.present[index]
	}
	before, after := indices.at(next-1), indices.at(next)
	share := float64(knot-axis[next-1]) / float64(axis[next]-axis[next-1])
	return (1-share)*grid.heights[before] + share*grid.heights[after],
		grid.present[before] && grid.present[after]
}
