package maths_test

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// The step is zero up to the start and one from the end, with the cubic
// between. Worked by hand, a quarter of the way is 0.15625 and halfway is a
// half. Each case of smooth.csv holds between any two, not only zero and one.
func TestSmoothIsTheStepWithNoCorner(test *testing.T) {
	for _, row := range read(test, smoothFile) {
		start, end := float(test, row, "start"), float(test, row, "end")
		x, want := float(test, row, "x"), float(test, row, "want")
		if got := maths.Smooth(start, end, x); got != want {
			test.Errorf("%s: Smooth(%v, %v, %v) is %v, want %v", row["name"],
				start, end, x, got, want)
		}
	}
}
