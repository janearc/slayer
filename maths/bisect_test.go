package maths_test

import (
	"github.com/janearc/slayer/maths"
	"testing"
)

func TestBisectKeepsTheCrossingAndEvaluationBudget(t *testing.T) {
	calls := 0
	low, high := maths.Bisect(0, 1, 6, func(x float64) bool { calls++; return x < .3 })
	if calls != 6 || low != .296875 || high != .3125 {
		t.Fatalf("bracket [%v,%v] after %d calls", low, high, calls)
	}
	low, high = maths.Bisect(-2, 3, 0, func(float64) bool { t.Fatal("evaluated a zero-step field"); return false })
	if low != -2 || high != 3 {
		t.Fatalf("zero steps changed bracket: [%v,%v]", low, high)
	}
}
