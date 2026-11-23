package gpu

import (
	"math"
	"slices"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// gridMesh is a mesh of a grid of cells by cells squares, two faces to
// each, raised by a fixed relief so no two cuts of its tree are alike.
func gridMesh(t *testing.T, cells int) *mesh.Mesh {
	t.Helper()
	var points []maths.Vec
	var faces [][3]int
	stride := cells + 1
	for row := range stride {
		for column := range stride {
			across, along := float64(column), float64(row)
			height := math.Sin(across*.7) * math.Cos(along*1.3)
			points = append(points, maths.V(across, along,
				height))
		}
	}
	for row := range cells {
		for column := range cells {
			low := row*stride + column
			faces = append(faces, [3]int{low, low + 1, low + stride},
				[3]int{low + 1, low + stride + 1, low + stride})
		}
	}
	made, err := mesh.New(points, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// sameTree is whether two meshes hold the same tree: as many boxes, each
// with the same corners and faces, and the same face in every place.
func sameTree(made, other *mesh.Mesh) bool {
	if made.TreeSize() != other.TreeSize() ||
		made.FaceCount() != other.FaceCount() {
		return false
	}
	for box := range made.TreeSize() {
		wantLow, wantHigh, wantFirst, wantCount := other.TreeBox(box)
		low, high, first, count := made.TreeBox(box)
		if low != wantLow || high != wantHigh || first != wantFirst ||
			count != wantCount {
			return false
		}
	}
	for place := range made.FaceCount() {
		if made.TreeFace(place) != other.TreeFace(place) {
			return false
		}
	}
	return true
}

// growMeshes sorts the slice it is given, the most faces first, so the
// meshes grown together are handed over as a copy of the slice: each is
// then read at its own place and compared with the one of its size grown
// apart. A gridMesh of 3 cells has 17 boxes and one of 90 has 16199, so a
// slice read after the sort puts the 90 where the 3 was made.
func TestMeshesGrownTogetherHoldTheTreesGrownApart(t *testing.T) {
	sizes := []int{3, 90, 10, 60}
	var together, apart []*mesh.Mesh
	for _, cells := range sizes {
		together = append(together, gridMesh(t, cells))
		apart = append(apart, gridMesh(t, cells))
	}
	growMeshes(slices.Clone(together), 4)
	for at, made := range together {
		if !sameTree(made, apart[at]) {
			t.Fatalf("mesh %d (%d cells): its tree differs from the one "+
				"grown apart", at, sizes[at])
		}
	}
}

// Anti-test: the comparison above sees a difference when there is one:
// trees of meshes of two sizes are not the same, and growMeshes does
// reorder the slice it is given, which is what a comparison by place in
// that slice would have taken for a difference in the trees.
func TestTheComparisonOfTreesSeesAnotherMesh(t *testing.T) {
	if !sameTree(gridMesh(t, 10), gridMesh(t, 10)) {
		t.Error("two meshes of one size held different trees")
	}
	if sameTree(gridMesh(t, 3), gridMesh(t, 90)) {
		t.Error("meshes of 3 and of 90 cells held the same tree")
	}
	meshes := []*mesh.Mesh{gridMesh(t, 3), gridMesh(t, 90), gridMesh(t, 10)}
	first := meshes[0]
	growMeshes(meshes, 2)
	if meshes[0] == first || meshes[0].FaceCount() < meshes[1].FaceCount() ||
		meshes[1].FaceCount() < meshes[2].FaceCount() {
		t.Error("growMeshes left the slice in the order it was given")
	}
}

func TestGrowingNoMeshesReturnsAtOnce(t *testing.T) {
	growMeshes(nil, 4)
}

func TestAKeeperOfNilHoldsNoTree(t *testing.T) {
	var keeper *keptTrees
	if keeper.holds(gridMesh(t, 2)) {
		t.Fatal("a nil keeper held a tree")
	}
}

func TestAKeeperHoldsTheTreeOfAMeshItKnows(t *testing.T) {
	skin := gridMesh(t, 2)
	keeper := newKeptTrees()
	if keeper.holds(skin) {
		t.Fatal("a new keeper held a tree")
	}
	keeper.slayersTree(skin)
	if !keeper.holds(skin) {
		t.Fatal("the keeper lost the tree it was given")
	}
}
