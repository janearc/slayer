package island

import "github.com/janearc/slayer/terrain"

// edgeIndex narrows a seam lookup to leaves whose actual edge lies on the
// queried lattice line. Interior vertices no longer scan every selected leaf.
// It retains four references per leaf and lives for one prepared view only.
type edgeIndex struct {
	leaves               []terrain.Patch
	vertical, horizontal map[int][]int
}

// newEdgeIndex records each leaf's two row and two column boundaries.
func newEdgeIndex(leaves []terrain.Patch) edgeIndex {
	index := edgeIndex{leaves: leaves, vertical: make(map[int][]int), horizontal: make(map[int][]int)}
	for ordinal, patch := range leaves {
		index.vertical[patch.MinColumn] = append(index.vertical[patch.MinColumn], ordinal)
		index.vertical[patch.MaxColumn] = append(index.vertical[patch.MaxColumn], ordinal)
		index.horizontal[patch.MinRow] = append(index.horizontal[patch.MinRow], ordinal)
		index.horizontal[patch.MaxRow] = append(index.horizontal[patch.MaxRow], ordinal)
	}
	return index
}

// coarser preserves the previous scan's rule: the lowest level wins, with
// original leaf order breaking ties where several edges meet at a corner.
func (index edgeIndex) coarser(key lattice, level int) *terrain.Patch {
	chosen := -1
	consider := func(ordinal int, within bool) {
		patch := index.leaves[ordinal]
		if !within || patch.Level >= level {
			return
		}
		if chosen < 0 || patch.Level < index.leaves[chosen].Level ||
			(patch.Level == index.leaves[chosen].Level && ordinal < chosen) {
			chosen = ordinal
		}
	}
	for _, ordinal := range index.vertical[key.column] {
		patch := index.leaves[ordinal]
		consider(ordinal, key.row >= patch.MinRow && key.row <= patch.MaxRow)
	}
	for _, ordinal := range index.horizontal[key.row] {
		patch := index.leaves[ordinal]
		consider(ordinal, key.column >= patch.MinColumn && key.column <= patch.MaxColumn)
	}
	if chosen < 0 {
		return nil
	}
	return &index.leaves[chosen]
}
