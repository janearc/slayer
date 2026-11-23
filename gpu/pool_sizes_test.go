package gpu

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// the sizes of the pools these tests make, each a choice of the tests
const (
	// rowFaces is how many faces each tree of the tests is grown over:
	// more than a leaf, so a tree has branches.
	rowFaces = 9
	// rowTrees is how many trees go in line for one pool.
	rowTrees = 5
)

// rowOfFaces is a tree over a row of flat faces, one metre apart, at a
// height, so that two rows at two heights are not the same faces.
func rowOfFaces(height float64) tree {
	faces := make([]triangle, rowFaces)
	for index := range faces {
		along := float64(index)
		faces[index] = triangle{corners: [3]maths.Vec{
			maths.V(along, height, 0), maths.V(along+1, height, 0),
			maths.V(along, height, 1)}}
	}
	return grow(faces)
}

// linedUp is a scene with no keeper and rowTrees trees in line.
func linedUp() *scene {
	encoded := newScene()
	for index := range rowTrees {
		faces := rowOfFaces(float64(index))
		encoded.pend(pendingMesh{tree: faces})
	}
	return encoded
}

// TestThePoolIsMadeItsSizeOnce says that pooling the trees in line
// leaves the pool's boxes with no room past what they hold: they were
// made their size before they were filled.
func TestThePoolIsMadeItsSizeOnce(t *testing.T) {
	encoded := linedUp()
	if err := encoded.packPool(); err != nil {
		t.Fatal(err)
	}
	nodes := encoded.nodes.bytes
	if len(nodes) == 0 || cap(nodes) != len(nodes) {
		t.Errorf("%d box bytes in room for %d, want room for exactly them",
			len(nodes), cap(nodes))
	}
	if len(encoded.faces.nodes) != 0 {
		t.Errorf("%d boxes copied into the scene, want none",
			len(encoded.faces.nodes))
	}
}

// TestAPoolGrownByAppendHasRoomToSpare is the anti-test of the one
// above: the same trees pooled with no room made first leave room past
// what they hold, so the test above can tell the two apart.
func TestAPoolGrownByAppendHasRoomToSpare(t *testing.T) {
	encoded := linedUp()
	for _, each := range encoded.pending {
		encoded.pooled(each.tree)
	}
	nodes := encoded.faces.nodes
	if cap(nodes) == len(nodes) {
		t.Errorf("%d boxes in room for %d after append, want room over",
			len(nodes), cap(nodes))
	}
}
