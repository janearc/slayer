package gpu

import "testing"

// mostWaiting is the most branches conifer_hit's walk holds at once for
// a tree depth levels deep, found by walking it as the kernel does:
// each branch taken off the stack puts its 3+level children on.
func mostWaiting(depth int) int {
	waiting, most := 1, 1
	for level := 0; level < depth; level++ {
		waiting += 3 + level - 1
		most = max(most, waiting)
	}
	return most
}

// the kernel does not check its conifer stack: one too small writes past
// its end on the gpu, which hangs it. so the go side must hold the bound.
func TestTheConiferStackHoldsTheDeepestTree(t *testing.T) {
	if got := mostWaiting(2); got != 6 {
		t.Fatalf("a tree two deep holds %d at most, want 6", got)
	}
	if need := mostWaiting(mostConiferDepth); coniferStack < need {
		t.Errorf("the stack holds %d, but a tree %d deep needs %d",
			coniferStack, mostConiferDepth, need)
	}
}
