package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// pillarsFirstTip is the place of the first tip in the tips of a pillars.
const pillarsFirstTip = 0 // defined: the first place of a slice

// pillarsTested is the pillars the tests below hold, read from the chosen
// numbers.
func pillarsTested(t *testing.T) Pillars {
	t.Helper()
	return Pillars{Count: pickedCount(t, "pillars-count"),
		Height: picked(t, "pillars-height"),
		Width:  picked(t, "pillars-width"),
		Thick:  unity,
		Seed:   uint64(pickedCount(t, "pillars-seed"))}
}

// pillarsAt is the point at a distance out along x and a height up.
func pillarsAt(out, height float64) maths.Vec {
	return maths.V(out, height, nought)
}

// Pillars have a dense wall along the bottom, columns above it that end
// below the top, and thin fog between; nothing beyond the box.
func TestPillarsStandOnAWall(t *testing.T) {
	pillars := pillarsTested(t)
	dense, thin := picked(t, "pillars-dense"), picked(t, "pillars-thin")
	wall := maths.V(picked(t, "pillars-wall-x"), picked(t, "pillars-wall-y"),
		picked(t, "pillars-wall-z"))
	if density := pillars.Density(wall); density < dense {
		t.Errorf("the wall is %v", density)
	}
	// somewhere at half height a column is dense, and most of the row is fog
	middle := picked(t, "pillars-half-height")
	columns, fog := pillarsRow(t, pillars, middle, dense, thin)
	if columns < pickedCount(t, "pillars-columns-least") ||
		fog < pickedCount(t, "pillars-fog-least") {
		t.Errorf("at half height %d dense and %d thin of 100", columns, fog)
	}
	topHeight := picked(t, "pillars-top-height")
	top := pillars.Density(pillarsAt(nought, topHeight))
	if top > thin {
		t.Errorf("the top is %v", top)
	}
	depth := picked(t, "pillars-below-depth")
	below := pillars.Density(pillarsAt(nought, -depth))
	out := picked(t, "pillars-outside-out")
	outside := pillars.Density(pillarsAt(out, middle))
	if below != nought || outside != nought {
		t.Error("gas beyond the box")
	}
}

// pillarsRow counts, along a row at a height, the points denser than dense
// and the points thinner than thin.
func pillarsRow(t *testing.T, pillars Pillars, height, dense,
	thin float64) (columns, fog int) {
	t.Helper()
	edge := picked(t, "pillars-box-half-width")
	step := picked(t, "pillars-row-step")
	for out := picked(t, "pillars-row-start"); out < edge; out += step {
		switch density := pillars.Density(pillarsAt(out, height)); {
		case density > dense:
			columns++
		case density < thin:
			fog++
		}
	}
	return columns, fog
}

// The tips are the tops of the columns, one each, inside the box and
// above half height; filaments thin the gas inside a column in
// streaks without emptying it.
func TestPillarsHaveTipsAndFilaments(t *testing.T) {
	pillars := pillarsTested(t)
	tips := pillars.Tips()
	if len(tips) != pillars.Count {
		t.Fatalf("%d tips", len(tips))
	}
	edge := picked(t, "pillars-box-half-width")
	under := maths.V(nought, -picked(t, "pillars-under-tip"), nought)
	middle := picked(t, "pillars-half-height")
	gasLeast := picked(t, "pillars-tip-gas-least")
	for _, tip := range tips {
		if tip.Y < middle || tip.Y > pillars.Height ||
			!(math.Abs(tip.X) <= edge) || !(math.Abs(tip.Z) <= edge) {
			t.Errorf("a tip at %v", tip)
		}
		below := tip.Add(under)
		if density := pillars.Density(below); density < gasLeast {
			t.Errorf("under the tip at %v the gas is %v", tip, density)
		}
	}
	pillarsStreaks(t, pillars, tips[pillarsFirstTip])
}

// pillarsStreaks checks that filaments run a column from dense to thin and
// back, along the first tip's column.
func pillarsStreaks(t *testing.T, pillars Pillars, tip maths.Vec) {
	t.Helper()
	streaked := pillars
	streaked.Filaments = picked(t, "pillars-filaments")
	low, high, dense := float64(unity), float64(nought), nought
	step := picked(t, "pillars-column-step")
	top := tip.Y - double
	for height := picked(t, "pillars-column-start"); height < top; height += step {
		at := maths.V(tip.X, height, tip.Z)
		density := streaked.Density(at)
		low, high = math.Min(low, density), math.Max(high, density)
		if pillars.Density(at) > picked(t, "pillars-dense") {
			dense++
		}
	}
	if dense < pickedCount(t, "pillars-dense-points-least") ||
		!(high > picked(t, "pillars-filaments-high-least") &&
			low < picked(t, "pillars-filaments-low-share")*high) {
		t.Errorf("filaments run the column from %.2f to %.2f over %d dense "+
			"points", low, high, dense)
	}
}
