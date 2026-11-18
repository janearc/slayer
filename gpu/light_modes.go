//go:build darwin

package gpu

// The two choices a light grid makes by traversal mode. They read the
// mode, which is Metal's (sampled.go), so they are built only beside it;
// what they choose between is in reach.go and sky_share.go, which build
// everywhere.

// cellsFor is how many cells a mode's light grids have along their
// longest side: the lamp store's for fast light, the smoke source
// grid's for the rest, which only the cached mode reads.
func cellsFor(mode traversal) int {
	if mode.stored() {
		return reachCells
	}
	return lightCells
}

// shareSkyFor shares the sky's looks among a pixel's samples for a
// mode that asks it, at so many samples along a pixel's side.
//
// The looks are shared only where they are dealt evenly: where a pixel's
// samples divide sixteen (one, two or four along a side), or sixteen
// divides them (eight, sixteen). At three a pixel would take nine of the
// sixteen in a row, and at five some twice, so there every sample takes
// all sixteen, as the exact tracer does (review of 2026-11-12).
func (encoded *scene) shareSkyFor(mode traversal, side int) {
	if mode == fasterTraversal && encoded.reach != nil &&
		sharesEvenly(side) {
		encoded.reach.shared = max(1, side)
	}
}
