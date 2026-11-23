package shape

import (
	"github.com/janearc/slayer/maths"
	"testing"
)

// sparseBinsBox is the box from low to high along x, one across in y and z.
func sparseBinsBox(low, high float64) maths.Box {
	return maths.Box{Min: maths.V(low, nought, nought),
		Max: maths.V(high, unity, unity)}
}

// TestSplitChoiceIgnoresEmptyBinsInALooseCentroidBound says that two
// clusters far apart are cut apart, though the box the bins are laid over
// is far looser than the clusters' middles.
func TestSplitChoiceIgnoresEmptyBinsInALooseCentroidBound(t *testing.T) {
	left := sparseBinsBox(picked(t, "sparse-bins-left-low"),
		picked(t, "sparse-bins-left-high"))
	right := sparseBinsBox(picked(t, "sparse-bins-right-low"),
		picked(t, "sparse-bins-right-high"))
	leftMiddle := picked(t, "sparse-bins-left-middle")
	rightMiddle := picked(t, "sparse-bins-right-middle")
	tree := builder{order: []int32{nought, unity},
		mids: []maths.Vec{maths.V(leftMiddle, nought, nought),
			maths.V(rightMiddle, nought, nought)},
		boxes: []maths.Box{left, right}}
	loose := sparseBinsBox(nought, picked(t, "sparse-bins-loose-reach"))
	cut := tree.cheapest(nought, len(tree.order), nought, loose, loose)
	nearer := tree.mids[tree.order[meshFirst]].X
	farther := tree.mids[tree.order[meshSecond]].X
	if cut != pickedCount(t, "sparse-bins-expected-cut") ||
		nearer != leftMiddle || farther != rightMiddle {
		t.Fatalf("split put both clusters on one side: cut %d, order %v",
			cut, tree.order)
	}
}
