package gpu

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// scattered is count small triangles thrown about a box ten wide
func scattered(count int) []triangle {
	faces := make([]triangle, count)
	for index := range faces {
		id := int64(index)
		at := maths.V(noise.Hash(1, id), noise.Hash(2, id),
			noise.Hash(3, id)).Scale(10)
		corners := [3]maths.Vec{at, at.Add(maths.V(0.1, 0, 0)),
			at.Add(maths.V(0, 0.1, 0.05))}
		faces[index] = triangle{corners: corners}
	}
	return faces
}

// named is each of some triangles' places in the list, by its first
// corner, which no two scattered triangles share.
func named(faces []triangle) map[maths.Vec]int {
	names := make(map[maths.Vec]int, len(faces))
	for index, face := range faces {
		names[face.corners[0]] = index
	}
	return names
}

func TestEveryTriangleIsInExactlyOneLeaf(t *testing.T) {
	names := named(scattered(1000))
	grown := grow(scattered(1000))
	seen := map[int]int{}
	for _, branch := range grown.nodes {
		if branch.count > leafSize {
			t.Errorf("a leaf holds %d, more than %d", branch.count,
				leafSize)
		}
		for _, face := range grown.leaf(branch) {
			seen[names[face.corners[0]]]++
		}
	}
	for index := range 1000 {
		if seen[index] != 1 {
			t.Fatalf("triangle %d is in %d leaves", index,
				seen[index])
		}
	}
}

// holdsCorners is whether a leaf's box holds every corner it holds
func holdsCorners(grown tree, leaf node) bool {
	for _, face := range grown.leaf(leaf) {
		for _, corner := range face.corners {
			if !leaf.bounds.holds(box{corner, corner}) {
				return false
			}
		}
	}
	return true
}

func TestEveryLeafHoldsItsCorners(t *testing.T) {
	grown := grow(scattered(1000))
	for index, leaf := range grown.nodes {
		if !holdsCorners(grown, leaf) {
			t.Fatalf("leaf %d misses a corner", index)
		}
	}
}

func TestEveryBranchHoldsItsChildren(t *testing.T) {
	grown := grow(scattered(1000))
	for index, branch := range grown.nodes {
		if branch.count > 0 {
			continue
		}
		for _, child := range grown.nodes[branch.first:][:2] {
			if !branch.bounds.holds(child.bounds) {
				t.Fatalf("branch %d misses a child", index)
			}
		}
	}
}

// depth is how many boxes deep the tree goes below a node
func depth(grown tree, index int) int {
	branch := grown.nodes[index]
	if branch.count > 0 || len(grown.nodes) == 1 {
		return 1
	}
	return 1 + max(depth(grown, branch.first), depth(grown, branch.first+1))
}

// the kernel walks the tree with a stack of 64; the split at the middle
// face keeps the depth near log2(faces / 4) even when every face is in
// one place, so the stack never overflows
func TestTheTreeStaysShallowEvenForFacesInOnePlace(t *testing.T) {
	same := scattered(1)[0]
	faces := make([]triangle, 1000)
	for index := range faces {
		faces[index] = same
	}
	if deep := depth(grow(faces), 0); deep > 10 {
		t.Errorf("a thousand faces in one place are %d deep", deep)
	}
}

// with no faces the tree is one root no ray can enter: a box inside
// out on every axis
func TestNoTrianglesIsARootNoRayEnters(t *testing.T) {
	grown := grow(nil)
	if len(grown.nodes) != 1 || grown.nodes[0].count != 0 {
		t.Fatalf("an empty tree is %+v", grown.nodes)
	}
	root := grown.nodes[0].bounds
	if root.low.X <= root.high.X || root.low.Y <= root.high.Y ||
		root.low.Z <= root.high.Z {
		t.Errorf("the empty root is %+v, not inside out", root)
	}
}
