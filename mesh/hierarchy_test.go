package mesh

import "testing"

// TestTheTreeAccessorsGiveTheTreeRaysAreMetBy says that the boxes and the
// order the accessors give are the tree's own, node for node and place
// for place, and that asking for them is what grows the tree.
func TestTheTreeAccessorsGiveTheTreeRaysAreMetBy(t *testing.T) {
	made := cube(t)
	if len(made.tree.nodes) != nought {
		t.Fatal("tree grown by New")
	}
	size := made.TreeSize()
	if size != len(made.tree.nodes) || size == nought {
		t.Fatalf("TreeSize %d, the tree holds %d", size, len(made.tree.nodes))
	}
	for i, at := range made.tree.nodes {
		smaller, larger, place, count := made.TreeBox(i)
		if smaller != vectorOf(at.bounds[low]) ||
			larger != vectorOf(at.bounds[high]) ||
			place != at.first || count != at.count {
			t.Errorf("box %d is not the tree's", i)
		}
	}
	for i, face := range made.tree.order {
		if made.TreeFace(i) != face {
			t.Errorf("place %d holds another face than the tree's", i)
		}
	}
}

// TestEveryFaceIsHeldByOneLeafOfTheTree says that walking the boxes from
// the root by the accessors alone reaches every face exactly once, and
// that every box holds the corners of the faces below it.
func TestEveryFaceIsHeldByOneLeafOfTheTree(t *testing.T) {
	made := cube(t)
	held := make([]int, made.FaceCount())
	waiting := []int{nought}
	for len(waiting) > nought {
		at := waiting[len(waiting)-unity]
		waiting = waiting[:len(waiting)-unity]
		smaller, larger, place, count := made.TreeBox(at)
		if count == nought {
			waiting = append(waiting, place, place+unity)
			continue
		}
		for i := place; i < place+count; i++ {
			face := made.TreeFace(i)
			held[face]++
			one, two, three := made.Corners(face)
			for _, corner := range []struct{ x, y, z float64 }{
				{one.X, one.Y, one.Z}, {two.X, two.Y, two.Z},
				{three.X, three.Y, three.Z}} {
				if corner.x < smaller.X || corner.x > larger.X ||
					corner.y < smaller.Y || corner.y > larger.Y ||
					corner.z < smaller.Z || corner.z > larger.Z {
					t.Errorf("face %d leaves its box %d", face, at)
				}
			}
		}
	}
	for face, times := range held {
		if times != unity {
			t.Errorf("face %d is held %d times", face, times)
		}
	}
}

// TestNoTreeIsDeeperThanItsStated says that the deepest leaf of a tree
// is no more than MostTreeDepth levels below the root.
func TestNoTreeIsDeeperThanItsStated(t *testing.T) {
	made := cube(t)
	type level struct{ at, depth int }
	waiting := []level{{nought, nought}}
	for len(waiting) > nought {
		next := waiting[len(waiting)-unity]
		waiting = waiting[:len(waiting)-unity]
		if next.depth > MostTreeDepth {
			t.Fatalf("box %d is %d levels down", next.at, next.depth)
		}
		_, _, place, count := made.TreeBox(next.at)
		if count == nought {
			waiting = append(waiting, level{place, next.depth + unity},
				level{place + unity, next.depth + unity})
		}
	}
}
