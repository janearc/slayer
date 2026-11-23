package gpu

import (
	"bytes"
	"errors"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// beyondFloat is a coordinate no float32 holds, and a float64 does.
const beyondFloat = 1e300

// slayersSmoothFan is a fan of slayer's own mesh shaded smooth, lifted
// as slayersFan is.
func slayersSmoothFan(t *testing.T, lift float64) *mesh.Mesh {
	t.Helper()
	corners, made, normals := fanCorners(lift, fanFaces)
	fan, err := mesh.New(corners, made, normals, true)
	if err != nil {
		t.Fatal(err)
	}
	return fan
}

// packedBy is the three packed arrays of a tree packed after a place.
func packedBy(grown tree, place pageSize) *scene {
	packed := newScene()
	packed.packTree(grown, place)
	return packed
}

// TestASkinHoldsNoFacesOrBoxesOfItsOwn says that the tree of one of
// slayer's meshes is the mesh, with the counts the mesh gives, and no
// float64 copy of a face or a box.
func TestASkinHoldsNoFacesOrBoxesOfItsOwn(t *testing.T) {
	fan := slayersSmoothFan(t, 1)
	grown := skinTree(fan)
	if grown.skin != fan || grown.triangles != nil || grown.nodes != nil {
		t.Fatal("the skin's tree holds values of its own")
	}
	if grown.faceCount() != fan.FaceCount() ||
		grown.nodeCount() != fan.TreeSize() ||
		grown.bendCount() != fan.FaceCount() {
		t.Fatal("the skin's counts are not the mesh's")
	}
	if flat := skinTree(slayersFan(t, 1)); flat.bendCount() != 0 {
		t.Fatalf("a mesh shaded flat has %d bends", flat.bendCount())
	}
}

// TestASkinPacksWhatItsCopyPacks says that packing a mesh where it lies
// gives, to the byte, what packing meshTree's copy of it gives, shaded
// flat and smooth, and after faces, bends and boxes already packed.
func TestASkinPacksWhatItsCopyPacks(t *testing.T) {
	after := pageSize{triangles: 7, bends: 3, nodes: 5}
	for name, fan := range map[string]*mesh.Mesh{
		"flat": slayersFan(t, 1), "smooth": slayersSmoothFan(t, 1)} {
		for _, place := range []pageSize{{}, after} {
			direct := packedBy(skinTree(fan), place)
			copied := packedBy(meshTree(fan), place)
			if !samePool(direct, copied) {
				t.Errorf("%s after %+v: packed from the mesh, not as its "+
					"copy is", name, place)
			}
			if len(direct.triangles.bytes) == 0 {
				t.Errorf("%s: nothing was packed", name)
			}
		}
	}
}

// TestASkinsBendsCountOnFromTheOnesBefore is the anti-test of the
// one above: the same mesh packed after other bends is not packed as it
// is packed after none, so the one above can tell the places apart.
func TestASkinsBendsCountOnFromTheOnesBefore(t *testing.T) {
	fan := slayersSmoothFan(t, 1)
	first := packedBy(skinTree(fan), pageSize{})
	later := packedBy(skinTree(fan), pageSize{bends: 3})
	if bytes.Equal(first.triangles.bytes, later.triangles.bytes) {
		t.Error("bend places did not move past the bends before")
	}
	if !bytes.Equal(first.bends.bytes, later.bends.bytes) {
		t.Error("the bends themselves changed with their places")
	}
}

// TestPackingSkinsIsPackingTheirCopies says that a pool of meshes
// packed as they lie is, to the byte, the pool of the same meshes
// copied out as values and packed (the low-retention trial's way), and
// that it notes where each mesh's faces went, by its root.
func TestPackingSkinsIsPackingTheirCopies(t *testing.T) {
	trees := []tree{skinTree(slayersFan(t, 1)),
		skinTree(slayersSmoothFan(t, 2)), rowOfFaces(3),
		skinTree(slayersFan(t, 4))}
	direct, copied := pooledBy(t, trees, false), pooledBy(t, trees, true)
	if !samePool(direct, copied) {
		t.Fatal("the pool packed from the meshes differs from the copy's")
	}
	root, first := 0, 0
	for index, each := range trees {
		run, noted := direct.runs[root]
		if !noted || run != (faceRun{first: first, count: each.faceCount()}) {
			t.Errorf("mesh %d: run %+v noted %v", index, run, noted)
		}
		root += each.nodeCount()
		first += each.faceCount()
	}
	if len(direct.faces.triangles) != 0 || len(direct.faces.nodes) != 0 {
		t.Error("the pool left values of its faces or boxes behind")
	}
}

// TestAFaceBeyondFloat32IsRefusedBeforeAnythingIsPacked says that a
// mesh of finite float64s that float32 cannot hold is refused as a
// copied face is, and that no array was reserved.
func TestAFaceBeyondFloat32IsRefusedBeforeAnythingIsPacked(t *testing.T) {
	far, err := mesh.New([]maths.Vec{{}, maths.V(beyondFloat, 0, 0),
		maths.YAxis}, [][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	encoded := newScene()
	encoded.pend(pendingMesh{tree: skinTree(far)})
	err = encoded.packPool()
	var refused unsupported
	if !errors.As(err, &refused) {
		t.Fatalf("a face beyond float32 gave %v, want a refusal", err)
	}
	if cap(encoded.triangles.bytes) != 0 || cap(encoded.nodes.bytes) != 0 {
		t.Error("room was reserved for a mesh that was refused")
	}
}

// TestTwoMeshesWithOneKeyShareTheFirstsSkin says that a keeper hands
// the second mesh made apart with the same faces the first's tree, and a
// mesh with other faces its own, and that a tree it keeps for the next
// frame is still the mesh and no copy of it.
func TestTwoMeshesWithOneKeyShareTheFirstsSkin(t *testing.T) {
	keeper := newKeptTrees()
	one, again := slayersFan(t, 3), slayersFan(t, 3)
	first := keeper.slayersTree(one)
	second := keeper.slayersTree(again)
	other := keeper.slayersTree(slayersFan(t, 4))
	if second.tree.skin != one || other.tree.skin == one {
		t.Fatal("the keeper did not share a skin by its faces")
	}
	keeper.turn()
	next := keeper.slayersTree(one)
	if next.tree.skin != first.tree.skin || next.tree.triangles != nil {
		t.Error("the kept tree is not the mesh it was")
	}
}

// TestTheWalkHoldsAsDeepATreeAsAMeshIsCut says that the kernel's stack
// holds one box more than the deepest a mesh's tree goes.
func TestTheWalkHoldsAsDeepATreeAsAMeshIsCut(t *testing.T) {
	if treeStack < mesh.MostTreeDepth+1 {
		t.Fatalf("a stack of %d, a tree %d deep", treeStack,
			mesh.MostTreeDepth)
	}
}
