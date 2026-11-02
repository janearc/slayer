package terrain

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// windowOf is a window of heights given row by row, so many metres
// apart.
func windowOf(step float64, rows ...[]float64) Window {
	made := Window{step: step, across: len(rows[0]), down: len(rows)}
	for _, row := range rows {
		made.heights = append(made.heights, row...)
	}
	return made
}

// a slope, step on step, blurred by any width stays the slope inside it, since
// a gaussian weighs either side alike; a flat window stays flat; zero is the
// window as it is.
func TestABlurKeepsASlope(t *testing.T) {
	var rows [][]float64
	for range 9 {
		var row []float64
		for column := range 30 {
			row = append(row, 3*float64(column))
		}
		rows = append(rows, row)
	}
	slope := windowOf(4, rows...)
	blurred, err := slope.Smoothed(6)
	if err != nil {
		t.Fatal(err)
	}
	for column := 8; column < 22; column++ {
		if got := blurred.At(column, 4); math.Abs(got-3*float64(column)) >
			1e-9 {
			t.Errorf("column %d at %g, want %g", column, got,
				3*float64(column))
		}
	}
	same, err := slope.Smoothed(0)
	if err != nil || &same.heights[0] != &slope.heights[0] {
		t.Errorf("nought changed the window: %v", err)
	}
	for _, width := range []float64{-1, math.NaN(), math.Inf(1)} {
		if _, err := slope.Smoothed(width); err == nil {
			t.Errorf("a blur %g wide", width)
		}
	}
}

// a step of ten metres blurred by a cell's width becomes a ramp: still five at
// the step, nearer zero and ten away from it, and no height past the step's
// own.
func TestABlurRoundsAStep(t *testing.T) {
	row := make([]float64, 20)
	for column := 10; column < 20; column++ {
		row[column] = 10
	}
	step := windowOf(4, row, row, row)
	blurred, err := step.Smoothed(4)
	if err != nil {
		t.Fatal(err)
	}
	mid := (blurred.At(9, 1) + blurred.At(10, 1)) / 2
	if math.Abs(mid-5) > 1e-9 {
		t.Errorf("at the step %g, want 5", mid)
	}
	if !(blurred.At(9, 1) > 0.5 && blurred.At(9, 1) < 5) ||
		!(blurred.At(3, 1) < 1e-6) || !(blurred.At(16, 1) > 10-1e-6) {
		t.Errorf("the ramp %g %g %g", blurred.At(3, 1), blurred.At(9, 1),
			blurred.At(16, 1))
	}
	for _, height := range blurred.heights {
		if height < -1e-9 || height > 10+1e-9 {
			t.Fatalf("a height %g past the step's", height)
		}
	}
}

// the ground is shaded smooth, each corner's normal the ground's slope
// there: a slope rising three in four to the east leans its normals
// west, alike at every corner, edges too.
func TestTheGroundsNormalsAreItsSlope(t *testing.T) {
	var rows [][]float64
	for range 4 {
		rows = append(rows, []float64{0, 3, 6, 9, 12})
	}
	ground, _, err := windowOf(4, rows...).Ground()
	if err != nil {
		t.Fatal(err)
	}
	if !ground.Smooth() {
		t.Fatal("the ground is shaded flat")
	}
	want := maths.V(-0.75, 1, 0).Unit()
	for _, at := range []maths.Vec{maths.V(2, 1.5, 6), maths.V(13, 9.75, 2),
		maths.V(0.5, 0.375, 11)} {
		got := ground.Normal(at)
		if got.Sub(want).Len() > 1e-9 {
			t.Errorf("at %v the normal is %v, want %v", at, got, want)
		}
	}
}
