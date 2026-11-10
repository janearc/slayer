package terrain

import (
	"fmt"
	"strconv"
)

// Patch is a stable rectangle of native sample indices within one source.
// Source identity belongs alongside it; a render crop is not part of this key.
// Adjacent patches share their endpoint samples, not their interior cells.
type Patch struct {
	MinColumn, MaxColumn int
	MinRow, MaxRow       int
	Level                int
}

// RootPatch spans the complete source sample lattice without decoding it.
func (raster *Raster) RootPatch() (Patch, error) {
	if !raster.placed || raster.width < 2 || raster.height < 2 {
		return Patch{}, fmt.Errorf("terrain: source has no placed two-dimensional sample lattice")
	}
	return Patch{MaxColumn: raster.width - 1, MaxRow: raster.height - 1}, nil
}

// valid checks an index rectangle before midpoint arithmetic or source reads.
func (patch Patch) valid() bool {
	return patch.MinColumn >= 0 && patch.MinRow >= 0 &&
		patch.MaxColumn > patch.MinColumn && patch.MaxRow > patch.MinRow &&
		patch.Level >= 0 && patch.Level < strconv.IntSize
}

// Children splits source cells while retaining exactly shared sample edges.
// A narrow source strip splits only along its remaining nonterminal axis.
func (patch Patch) Children() ([]Patch, error) {
	if !patch.valid() {
		return nil, fmt.Errorf("terrain: invalid native patch")
	}
	columns := splitRange(patch.MinColumn, patch.MaxColumn)
	rows := splitRange(patch.MinRow, patch.MaxRow)
	if len(columns) == 1 && len(rows) == 1 {
		return nil, nil
	}
	if patch.Level+1 >= strconv.IntSize {
		return nil, fmt.Errorf("terrain: patch subdivision exceeds the native index depth")
	}
	var children []Patch
	for _, row := range rows {
		for _, column := range columns {
			children = append(children, Patch{MinColumn: column[0], MaxColumn: column[1],
				MinRow: row[0], MaxRow: row[1], Level: patch.Level + 1})
		}
	}
	return children, nil
}

// splitRange uses a difference midpoint, so large positive indices cannot
// add-overflow.
func splitRange(first, last int) [][2]int {
	if last-first <= 1 {
		return [][2]int{{first, last}}
	}
	middle := first + (last-first)/2
	return [][2]int{{first, middle}, {middle, last}}
}
