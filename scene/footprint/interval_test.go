package footprint

import (
	"math"
	"testing"
)

// TestIntervalArithmeticKeepsIndeterminateResultsUnknown checks that a
// result with no number in it is the whole line, never an empty range.
func TestIntervalArithmeticKeepsIndeterminateResultsUnknown(t *testing.T) {
	whole := interval{math.Inf(-1), math.Inf(1)}
	cases := map[string]interval{
		"a NaN bound":             enclosed(math.NaN(), 1),
		"zero times infinity":     (interval{0, 1}).mul(interval{math.Inf(1), math.Inf(1)}),
		"division across zero":    (interval{1, 2}).div(interval{-1, 1}),
		"division by zero":        (interval{1, 2}).div(interval{0, 1}),
		"infinity minus infinity": (interval{math.Inf(1), math.Inf(1)}).sub(interval{math.Inf(1), math.Inf(1)}),
	}
	for name, got := range cases {
		if got != whole {
			t.Errorf("%s gave %v, want the whole line", name, got)
		}
	}
}

// TestIntervalOperationsWidenOutward checks that a sum and a quotient
// contain the exact result and are wider than it by a step on each side.
func TestIntervalOperationsWidenOutward(t *testing.T) {
	sum := exact(1).add(exact(2))
	if !(sum.low < 3 && sum.high > 3 && sum.high-sum.low < 1e-14) {
		t.Errorf("1 + 2 gave %v", sum)
	}
	quotient := (interval{6, 6}).div(interval{2, 3})
	if !(quotient.low < 2 && quotient.high > 3 && quotient.low > 1.999999999 && quotient.high < 3.000000001) {
		t.Errorf("6 / [2, 3] gave %v, want about [2, 3]", quotient)
	}
}
