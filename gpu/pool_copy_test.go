package gpu

import (
	"bytes"
	"slices"
	"testing"

	"github.com/janearc/slayer/maths"
)

// smoothRow is rowOfFaces with every face shaded smooth, its normals
// leaning by height so no two rows' bends are the same.
func smoothRow(height float64) tree {
	grown := rowOfFaces(height)
	for index := range grown.triangles {
		lean := maths.V(height, 1, 0).Unit()
		grown.triangles[index].smooth = true
		grown.triangles[index].normals = [3]maths.Vec{lean, lean, lean}
	}
	return grown
}

// mixedTrees is trees in line, flat and smooth by turns, so that a
// bend's place has to be counted across trees.
func mixedTrees() []tree {
	trees := make([]tree, rowTrees)
	for index := range trees {
		if index%2 == 0 {
			trees[index] = rowOfFaces(float64(index))
		} else {
			trees[index] = smoothRow(float64(index))
		}
	}
	return trees
}

// pooledBy packs trees through packPool, as a draw does, or, copied,
// through the low-retention trial, which copies every face into one list
// first and packs that: the way every draw packed before.
func pooledBy(t *testing.T, trees []tree, copied bool) *scene {
	t.Helper()
	encoded := newScene()
	encoded.lowRetention = copied
	for _, grown := range trees {
		encoded.pend(pendingMesh{tree: grown})
	}
	if err := encoded.packPool(); err != nil {
		t.Fatal(err)
	}
	return encoded
}

// samePool is whether two scenes packed the same bytes and counts.
func samePool(one, other *scene) bool {
	return bytes.Equal(one.triangles.bytes, other.triangles.bytes) &&
		bytes.Equal(one.bends.bytes, other.bends.bytes) &&
		bytes.Equal(one.nodes.bytes, other.nodes.bytes) &&
		one.counts == other.counts
}

// TestPackingFromTheTreesIsPackingTheCopy says that packing each tree's
// faces where the tree holds them gives the pool, to the byte, that
// copying them into one list and packing that gave, and that it leaves
// no float64 copy of the faces behind.
func TestPackingFromTheTreesIsPackingTheCopy(t *testing.T) {
	trees := mixedTrees()
	direct, copied := pooledBy(t, trees, false), pooledBy(t, trees, true)
	if !samePool(direct, copied) {
		t.Error("the pool packed from the trees differs from the copy's")
	}
	if len(direct.faces.triangles) != 0 {
		t.Errorf("%d faces copied into the pool, want none",
			len(direct.faces.triangles))
	}
}

// TestPoolsOfTreesInAnotherOrderDiffer is the anti-test of the one
// above: the same trees in another order pack another pool, so the
// comparison above can see a difference when there is one.
func TestPoolsOfTreesInAnotherOrderDiffer(t *testing.T) {
	trees := mixedTrees()
	turned := slices.Clone(trees)
	slices.Reverse(turned)
	if samePool(pooledBy(t, trees, false), pooledBy(t, turned, true)) {
		t.Error("trees in two orders packed the same pool")
	}
}
