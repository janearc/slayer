package shape

import (
	"github.com/janearc/slayer/maths"
	"testing"
)

func TestSplitChoiceIgnoresEmptyBinsInALooseCentroidBound(t *testing.T) {
	left := maths.Box{Min: maths.V(1.9, 0, 0), Max: maths.V(2.1, 1, 1)}
	right := maths.Box{Min: maths.V(2.9, 0, 0), Max: maths.V(3.1, 1, 1)}
	b := builder{order: []int32{0, 1}, mids: []maths.Vec{maths.V(2, 0, 0), maths.V(3, 0, 0)}, boxes: []maths.Box{left, right}}
	loose := maths.Box{Min: maths.V(0, 0, 0), Max: maths.V(10, 1, 1)}
	cut := b.cheapest(0, 2, 0, loose, loose)
	if cut != 1 || b.mids[b.order[0]].X != 2 || b.mids[b.order[1]].X != 3 {
		t.Fatalf("split put both clusters on one side: cut %d, order %v", cut, b.order)
	}
}
