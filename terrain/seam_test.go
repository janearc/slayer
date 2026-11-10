package terrain

import (
	"slices"
	"testing"
)

func seamGrids() (PatchGrid, PatchGrid) {
	coarse := PatchGrid{patch: Patch{MaxColumn: 8, MaxRow: 8},
		spacing: 1, columns: []int{0, 4, 8}, rows: []int{0, 4, 8},
		heights: []float64{0, 0, 0, 4, 4, 4, 0, 0, 0},
		present: []bool{true, true, true, true, true, true, true, true, true}}
	fine := PatchGrid{patch: Patch{MinColumn: 8, MaxColumn: 12, MaxRow: 4, Level: 1},
		spacing: 1, columns: []int{8, 10, 12}, rows: []int{0, 2, 4},
		heights: []float64{10, 10, 10, 10, 10, 10, 10, 10, 10},
		present: []bool{true, true, true, true, true, true, true, true, true}}
	return fine, coarse
}

func TestTerrainSeamMatchesCoarseSegmentsWithoutMutatingCache(t *testing.T) {
	fine, coarse := seamGrids()
	joined, shift, err := fine.StitchTo(coarse, 81)
	if err != nil {
		t.Fatal(err)
	}
	want := []float64{0, 10, 10, 2, 10, 10, 4, 10, 10}
	if !slices.Equal(joined.heights, want) || shift != 10 {
		t.Fatalf("seam %v, shift %v", joined.heights, shift)
	}
	for _, height := range fine.heights {
		if height != 10 {
			t.Fatal("cached grid mutated")
		}
	}
	if coarse.heights[5] != 4 {
		t.Fatal("neighbour mutated")
	}
}

func TestTerrainSeamKeepsMissingHeightsMissing(t *testing.T) {
	fine, coarse := seamGrids()
	coarse.present[2] = false
	joined, _, err := fine.StitchTo(coarse, 81)
	if err != nil {
		t.Fatal(err)
	}
	if joined.present[0] || joined.present[3] || !joined.present[6] {
		t.Fatalf("availability %v", joined.present)
	}
}

func TestTerrainSeamRefusesInsufficientBudgetOrMissingKnot(t *testing.T) {
	fine, coarse := seamGrids()
	if _, _, err := fine.StitchTo(coarse, 80); err == nil {
		t.Fatal("seam allocated beyond budget")
	}
	// A coarse bend at row 1 is absent from the fine edge's 0,2,4 knots.
	coarse.rows = []int{0, 1, 8}
	if _, _, err := fine.StitchTo(coarse, 81); err == nil {
		t.Fatal("missing coarse bend accepted")
	}
}

func TestTerrainSeamHandlesEverySharedSide(t *testing.T) {
	for _, row := range []struct {
		name                                             string
		fineColumns, fineRows, coarseColumns, coarseRows []int
		coarseHeights                                    []float64
		changed                                          [3]int
	}{
		{"east", []int{8, 10, 12}, []int{0, 2, 4}, []int{0, 4, 8}, []int{0, 4, 8},
			[]float64{0, 0, 0, 4, 4, 4, 0, 0, 0}, [3]int{0, 3, 6}},
		{"west", []int{0, 2, 4}, []int{0, 2, 4}, []int{4, 8, 12}, []int{0, 4, 8},
			[]float64{0, 0, 0, 4, 4, 4, 0, 0, 0}, [3]int{2, 5, 8}},
		{"south", []int{0, 2, 4}, []int{8, 10, 12}, []int{0, 4, 8}, []int{0, 4, 8},
			[]float64{0, 4, 0, 0, 4, 0, 0, 4, 0}, [3]int{0, 1, 2}},
		{"north", []int{0, 2, 4}, []int{0, 2, 4}, []int{0, 4, 8}, []int{4, 8, 12},
			[]float64{0, 4, 0, 0, 4, 0, 0, 4, 0}, [3]int{6, 7, 8}},
	} {
		t.Run(row.name, func(t *testing.T) {
			fine, coarse := seamGrids()
			fine.columns, fine.rows = row.fineColumns, row.fineRows
			coarse.columns, coarse.rows = row.coarseColumns, row.coarseRows
			coarse.heights = row.coarseHeights
			for _, g := range []*PatchGrid{&fine, &coarse} {
				g.patch.MinColumn, g.patch.MaxColumn = g.columns[0], g.columns[2]
				g.patch.MinRow, g.patch.MaxRow = g.rows[0], g.rows[2]
			}
			joined, _, err := fine.StitchTo(coarse, 81)
			if err != nil {
				t.Fatal(err)
			}
			want := []float64{10, 10, 10, 10, 10, 10, 10, 10, 10}
			for i, index := range row.changed {
				want[index] = float64(i * 2)
			}
			if !slices.Equal(joined.heights, want) {
				t.Fatalf("%v, want %v", joined.heights, want)
			}
		})
	}
}
