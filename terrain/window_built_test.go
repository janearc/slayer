package terrain

import (
	"errors"
	"math"
	"testing"
)

// builtAcross, builtDown and builtStep are a small window to build: three
// points by two, a metre apart. picked: the smallest that has a row below
// another and a column beside another.
const (
	builtAcross = 3
	builtDown   = 2
	builtStep   = 1.0
)

// overBudget is a count of points on a side whose square, at eight bytes a
// point, is past the 64 MiB the window may hold. defined: 64 MiB is 8,388,608
// points, and 3,000 squared is 9,000,000.
const overBudget = 3000

// A built window has the heights asked of its function, at the columns and
// rows asked, and the first point and spacing it was given.
func TestBuildWindowHoldsWhatItIsGiven(t *testing.T) {
	window, err := BuildWindow(0, 10, builtAcross, builtDown, builtStep,
		func(column, row int) (float64, error) {
			return float64(10*row + column), nil
		})
	if err != nil {
		t.Fatal(err)
	}
	across, down, step := window.Size()
	if across != builtAcross || down != builtDown || step != builtStep {
		t.Fatalf("size %d %d %g", across, down, step)
	}
	if window.At(2, 1) != 12 || window.At(0, 0) != 0 || window.At(1, 1) != 11 {
		t.Fatalf("heights %v", window.heights)
	}
	if window.west != 0 || window.north != 10 {
		t.Fatalf("first point %g %g", window.west, window.north)
	}
}

// Anti: a built window reads back from bytes the same, so the cache keeps it.
func TestBuildWindowSurvivesBytes(t *testing.T) {
	window, err := BuildWindow(0, 10, builtAcross, builtDown, builtStep,
		func(column, row int) (float64, error) { return 0.5, nil })
	if err != nil {
		t.Fatal(err)
	}
	read, err := WindowFromBytes(window.Bytes())
	if err != nil || read.At(1, 1) != 0.5 || read.north != 10 {
		t.Fatalf("read %+v %v", read, err)
	}
}

// The caller's error comes out as it went in.
func TestBuildWindowPassesTheCallersError(t *testing.T) {
	broken := errors.New("no height here")
	_, err := BuildWindow(0, 0, builtAcross, builtDown, builtStep,
		func(int, int) (float64, error) { return 0, broken })
	if !errors.Is(err, broken) {
		t.Fatalf("got %v", err)
	}
}

// A height that is not a number is refused, not kept.
func TestBuildWindowRefusesHeightsThatAreNotNumbers(t *testing.T) {
	for _, bad := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, err := BuildWindow(0, 0, builtAcross, builtDown, builtStep,
			func(int, int) (float64, error) { return bad, nil })
		if err == nil {
			t.Errorf("%g was kept", bad)
		}
	}
}

// A window over the payload budget is refused before a height is asked for.
func TestBuildWindowRefusesOverBudgetUnasked(t *testing.T) {
	asked := 0
	_, err := BuildWindow(0, 0, overBudget, overBudget, builtStep,
		func(int, int) (float64, error) { asked++; return 0, nil })
	if err == nil || asked != 0 {
		t.Fatalf("got %v after asking %d times", err, asked)
	}
}

// Anti: a window one small step under the budget is not refused for size.
func TestBuildWindowAcceptsWhatFits(t *testing.T) {
	_, err := BuildWindow(0, 0, builtAcross, builtDown, builtStep,
		func(int, int) (float64, error) { return 0, nil })
	if err != nil {
		t.Fatal(err)
	}
}

// A window of a single column, a step that is nothing, or no function is
// refused.
func TestBuildWindowRefusesAShapelessWindow(t *testing.T) {
	zero := func(int, int) (float64, error) { return 0, nil }
	for name, build := range map[string]func() (Window, error){
		"one column": func() (Window, error) {
			return BuildWindow(0, 0, 1, builtDown, builtStep, zero)
		},
		"no step": func() (Window, error) {
			return BuildWindow(0, 0, builtAcross, builtDown, 0, zero)
		},
		"no function": func() (Window, error) {
			return BuildWindow(0, 0, builtAcross, builtDown, builtStep, nil)
		},
	} {
		if _, err := build(); err == nil {
			t.Errorf("%s was built", name)
		}
	}
}
