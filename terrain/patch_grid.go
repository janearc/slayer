package terrain

import (
	"fmt"
	"math/bits"
	"strconv"
)

// PatchGrid stores bounded source samples on explicit native-index axes.
// Coarse index spacing may be nonuniform when a source span is not divisible.
type PatchGrid struct {
	patch                Patch
	reference            SourceReference
	west, north, spacing float64
	rowStep              float64
	columns, rows        []int
	heights              []float64
	present              []bool
}

// Size is the actual sample count, bounded by the requested side resolution.
func (grid PatchGrid) Size() (int, int) { return len(grid.columns), len(grid.rows) }

// Reference is the source metadata retained with this sampled grid.
func (grid PatchGrid) Reference() SourceReference { return grid.reference }

// Patch returns the complete native footprint represented by this sample grid.
func (grid PatchGrid) Patch() Patch { return grid.patch }

// At returns native source coordinates, height and availability.
// Reference identifies coordinate units; declared point registration is honored.
// Outside the result grid is absent. Missing heights do not become flat ground.
func (grid PatchGrid) At(column, row int) (east, north, height float64, available bool) {
	if column < 0 || row < 0 || column >= len(grid.columns) || row >= len(grid.rows) {
		return 0, 0, 0, false
	}
	index := row*len(grid.columns) + column
	offset := 0.5
	if grid.reference.RasterType == 2 {
		offset = 0
	}
	east = grid.west + (float64(grid.columns[column])+offset)*grid.spacing
	rowSpacing := grid.rowStep
	if rowSpacing == 0 {
		rowSpacing = grid.spacing
	}
	north = grid.north - (float64(grid.rows[row])+offset)*rowSpacing
	return east, north, grid.heights[index], grid.present[index]
}

// ReadPatch samples a source rectangle without inventing a new grid origin.
// limitBytes covers height/mask payloads and both native-index axes.
// This skips native samples; it is not a filtered overview or error bound.
func (raster *Raster) ReadPatch(patch Patch, side int, limitBytes int64) (PatchGrid, error) {
	across, down, count, err := raster.patchShape(patch, side, limitBytes)
	if err != nil {
		return PatchGrid{}, err
	}
	grid := PatchGrid{patch: patch, reference: raster.reference, west: raster.west, north: raster.north, spacing: raster.spacing,
		rowStep: raster.rowSpacing(),
		columns: nativeAxis(patch.MinColumn, patch.MaxColumn, across),
		rows:    nativeAxis(patch.MinRow, patch.MaxRow, down),
		heights: make([]float64, count), present: make([]bool, count)}
	for row, sourceRow := range grid.rows {
		for column, sourceColumn := range grid.columns {
			index := row*across + column
			grid.heights[index], grid.present[index], err = raster.sample(sourceColumn, sourceRow)
			if err != nil {
				return PatchGrid{}, err
			}
		}
	}
	return grid, nil
}

// patchShape is shared by fresh reads and cache hits, so neither bypasses the
// caller's current output allowance or serves a grid from a closed source.
func (raster *Raster) patchShape(patch Patch, side int, limit int64) (across, down, count int, err error) {
	if raster == nil || raster.closed || !raster.placed || !patch.valid() ||
		patch.MaxColumn >= raster.width || patch.MaxRow >= raster.height || side < 2 {
		return 0, 0, 0, fmt.Errorf("terrain: invalid or closed source patch")
	}
	across = min(side, patch.MaxColumn-patch.MinColumn+1)
	down = min(side, patch.MaxRow-patch.MinRow+1)
	count, err = patchCapacity(across, down, limit)
	return across, down, count, err
}

// patchCapacity checks every returned array product before allocating any axis.
func patchCapacity(across, down int, limit int64) (int, error) {
	const indexBytes = strconv.IntSize / byteBits
	limit = min(limit, addressableBytes)
	if limit < 0 || across < 2 || down < 2 || int64(across) > limit/indexBytes {
		return 0, fmt.Errorf("terrain: patch exceeds its output byte budget")
	}
	left := limit - int64(across)*indexBytes
	if int64(down) > left/indexBytes {
		return 0, fmt.Errorf("terrain: patch axes exceed their output byte budget")
	}
	left -= int64(down) * indexBytes
	if int64(across) > left/(gridHeightBytes+gridAvailabilityBytes)/int64(down) {
		return 0, fmt.Errorf("terrain: patch samples exceed their output byte budget")
	}
	return across * down, nil
}

// nativeAxis selects reproducible endpoints/interior native sample indices.
// Wide integer arithmetic avoids overflow in index*sourceSpan before division.
func nativeAxis(first, last, count int) []int {
	axis := make([]int, count)
	span, steps := uint64(last-first), uint64(count-1)
	for index := range count {
		high, low := bits.Mul64(uint64(index), span)
		share, _ := bits.Div64(high, low, steps)
		axis[index] = first + int(share)
	}
	return axis
}
