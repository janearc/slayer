package ocean

import (
	"math"
	"testing"

	"github.com/janearc/slayer/oceantex"
)

// reelSea is the reel sea's: 7 m/s, fully developed over 500 km, deep,
// running north.
var reelSea = oceantex.Sea{Wind: 7, Fetch: 500000, Toward: math.Pi / 2}

// The sheet's spacing is its extent over one fewer than its samples: the
// reel sea's 512 m at 513 a side is 1 m, Nyquist pi.
func TestGlintSlopesUseTheSheetSpacing(t *testing.T) {
	got, err := GlintSlopes(reelSea, 512, 513)
	if err != nil {
		t.Fatal(err)
	}
	want, err := reelSea.SlopeBudget(1)
	if err != nil {
		t.Fatal(err)
	}
	if got != want || got.Nyquist != math.Pi {
		t.Fatalf("budget %+v, want %+v", got, want)
	}
	if _, err := GlintSlopes(reelSea, 512, 1); err == nil {
		t.Fatal("a sheet of one sample gave a budget")
	}
}

// A measurement as much as a test: the slope the reel sea's sheet
// carries from cell to cell, against the spectrum's resolved part at
// its spacing. The sheet samples cascades finer than its cells, so
// the two need not agree; the sheet carries at least half.
func TestGlintSlopesAgainstTheSheet(t *testing.T) {
	const extent, samples = 512, 513
	field, err := NewHeightField(HeightFieldConfig{Sea: reelSea,
		Extent: extent, Samples: samples, Seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	heights, n, _, _, _, _ := field.Pack()
	step := float64(extent) / float64(samples-1)
	var sum float64
	for row := 0; row+1 < n; row++ {
		for column := 0; column+1 < n; column++ {
			here := heights[row*n+column]
			east := (heights[row*n+column+1] - here) / step
			south := (heights[(row+1)*n+column] - here) / step
			sum += east*east + south*south
		}
	}
	carried := sum / float64((n-1)*(n-1))
	budget, err := GlintSlopes(reelSea, extent, samples)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("sheet carries %.6g; spectrum resolves %.6g; Cox and Munk %.6g;"+
		" unresolved %.6g", carried, budget.Resolved, budget.Total,
		budget.Unresolved)
	if !(carried > budget.Resolved/2) {
		t.Errorf("the sheet carries %v, under half the resolved %v",
			carried, budget.Resolved)
	}
}
