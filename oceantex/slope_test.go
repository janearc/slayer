package oceantex

import (
	"math"
	"testing"
)

// glintSea is the reel sea's: 7 m/s, fully developed over 500 km, deep,
// running north.
var glintSea = Sea{Wind: 7, Fetch: 500000, Toward: math.Pi / 2}

// Cox and Munk's clean-surface slope at 7 m/s is 0.003 + 0.00512 * 7.
func TestCoxMunkSlopeAtSevenMetresASecond(t *testing.T) {
	if got := coxMunkSlope(7); math.Abs(got-0.03884) > 1e-12 {
		t.Fatalf("Cox and Munk at 7 m/s = %v, want 0.03884", got)
	}
}

// Anti-test: the slope is both ways together, not the along-wind
// variance alone, and a calm or backward wind leaves only the offset.
func TestCoxMunkSlopeIsBothWaysAndCalmIsTheOffset(t *testing.T) {
	along, _ := roughness(7)
	if coxMunkSlope(7) == along {
		t.Fatal("the slope is the along-wind variance alone")
	}
	for _, wind := range []float64{0, -3} {
		if got := coxMunkSlope(wind); got != 0.003 {
			t.Errorf("wind %v gives %v, want the calm 0.003", wind, got)
		}
	}
}

// The finer the grid, the more of the sea's slope it resolves.
func TestResolvedSlopeRisesWithFinerGrids(t *testing.T) {
	last := 0.0
	for _, spacing := range []float64{4, 2, 1, 0.5, 0.25} {
		got := glintSea.resolvedSlope(spacing)
		if !(got > last) {
			t.Errorf("a %v m grid resolves %v, no more than the coarser %v",
				spacing, got, last)
		}
		last = got
	}
}

// Anti-test: a calm sea, or one with no fetch, resolves nothing at any
// grid; and a grid coarser than the sea's longest waves resolves almost
// nothing of what a 1 m grid does.
func TestResolvedSlopeOfNoSeaIsNothing(t *testing.T) {
	for _, sea := range []Sea{{Wind: 0, Fetch: 500000}, {Wind: 7}} {
		if got := sea.resolvedSlope(1); got != 0 {
			t.Errorf("%+v resolves %v", sea, got)
		}
	}
	coarse, fine := glintSea.resolvedSlope(1000), glintSea.resolvedSlope(1)
	if !(coarse < fine/1000) {
		t.Errorf("a 1 km grid resolves %v of a 1 m grid's %v", coarse, fine)
	}
}

// The integral agrees with the sum over a patch's own grid, which holds
// the same waves: 512 m at 512 a side, 1 m cells, against a 1 m grid.
func TestResolvedSlopeMatchesAPatchSum(t *testing.T) {
	patch, err := newPatch(glintSea, 512, 512, 1)
	if err != nil {
		t.Fatal(err)
	}
	along, across := patch.resolved()
	got, want := glintSea.resolvedSlope(1), along+across
	if math.Abs(got/want-1) > 0.02 {
		t.Errorf("integral %v, patch sum %v", got, want)
	}
}

// At the reel sea's 1 m grid the budget is Cox and Munk's total, the
// spectrum's resolved part, and exactly the rest.
func TestSlopeBudgetAtTheReelSeaGrid(t *testing.T) {
	budget, err := glintSea.SlopeBudget(1)
	if err != nil {
		t.Fatal(err)
	}
	if budget.Total != coxMunkSlope(7) ||
		budget.Resolved != glintSea.resolvedSlope(1) ||
		budget.Unresolved != budget.Total-budget.Resolved ||
		budget.Nyquist != math.Pi || budget.Floored || budget.Exceeds {
		t.Fatalf("budget at 1 m = %+v", budget)
	}
	if !(budget.Resolved > 0 && budget.Unresolved > slopeFloor) {
		t.Fatalf("budget at 1 m = %+v", budget)
	}
}

// Anti-test: a grid fine enough to resolve more than Cox and Munk
// measured says so, and the unresolved part is the floor, not a
// negative difference.
func TestSlopeBudgetSaysWhenTheSpectrumExceedsCoxMunk(t *testing.T) {
	budget, err := glintSea.SlopeBudget(1e-6)
	if err != nil {
		t.Fatal(err)
	}
	if !budget.Exceeds || !budget.Floored ||
		budget.Unresolved != slopeFloor || !(budget.Resolved > budget.Total) {
		t.Fatalf("budget at a micrometre = %+v", budget)
	}
	for _, spacing := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, err := glintSea.SlopeBudget(spacing); err == nil {
			t.Errorf("spacing %v accepted", spacing)
		}
	}
	if _, err := (Sea{Wind: math.NaN(), Fetch: 1}).SlopeBudget(1); err == nil {
		t.Error("a wind that is not a number gave a budget")
	}
}
