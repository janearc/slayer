package gpu

import (
	"slices"

	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

// pendingMesh is a mesh's tree waiting to go into the scene's pool, and
// its key where a keeper took one.
type pendingMesh struct {
	key   meshKey
	keyed bool
	tree  tree
}

// keptPool is a frame's packed pool of faces: the triangles, bends and
// boxes of every mesh it drew, each once, as the kernel reads them, and
// the keys of those meshes in the order they went in.
//
// The pool is made from the trees alone, in their order, so a frame whose
// meshes have the keys of the last frame's, in the same order, has the
// last frame's pool to the byte. Such a frame takes it whole, and packs
// nothing: only where the meshes stand is worked again.
//
// A kept pool is never written to: its bytes are the device's inputs of
// two frames at once.
type keptPool struct {
	keys                     []meshKey
	triangles, bends, nodes  []byte
	faceNodes                []node
	triangleCount, nodeCount int
}

// shapeTree is a shape.Mesh's tree for the pool: kept by its bytes where
// the scene has a keeper, and grown otherwise.
func (encoded *scene) shapeTree(made *shape.Mesh) pendingMesh {
	if encoded.keeper != nil {
		return encoded.keeper.shapeTree(made)
	}
	return pendingMesh{tree: grownOver(meshFaces(made))}
}

// slayersTree is one of slayer's meshes' tree for the pool: kept by its
// faces where the scene has a keeper, and grown otherwise.
func (encoded *scene) slayersTree(skin *mesh.Mesh) pendingMesh {
	if encoded.keeper != nil {
		return encoded.keeper.slayersTree(skin)
	}
	return pendingMesh{tree: grownOver(slayersFaces(skin))}
}

// pend puts a mesh's tree in line for the pool, and says where it will
// be there: its root, after the boxes of every tree in line before it,
// as pooled would place it, and the box round it in its own space.
func (encoded *scene) pend(waiting pendingMesh) pooled {
	at := pooled{root: encoded.pendingNodes,
		bounds: waiting.tree.nodes[0].bounds}
	encoded.pending = append(encoded.pending, waiting)
	encoded.pendingNodes += len(waiting.tree.nodes)
	return at
}

// packPool packs the trees in line into the scene's triangles, bends and
// boxes: taken whole from the keeper's last pool where it has the same
// meshes in the same order, and packed and kept otherwise.
func (encoded *scene) packPool() error {
	waiting := encoded.pending
	encoded.pending = nil
	if encoded.keeper.serves(encoded, waiting) {
		return nil
	}
	for _, each := range waiting {
		encoded.pooled(each.tree)
	}
	if err := encoded.tree(encoded.faces); err != nil {
		return err
	}
	encoded.keeper.keep(encoded, waiting)
	return nil
}

// keysOf is the keys of the meshes in line, and whether every one has
// one: a pool with an unkeyed mesh in it is never kept.
func keysOf(waiting []pendingMesh) ([]meshKey, bool) {
	keys := make([]meshKey, len(waiting))
	for index, each := range waiting {
		if !each.keyed {
			return nil, false
		}
		keys[index] = each.key
	}
	return keys, true
}

// serves gives the scene the last frame's pool where its meshes are the
// ones in line, in their order, and says whether it did; a scene with no
// keeper is never served.
func (keeper *keptTrees) serves(encoded *scene, waiting []pendingMesh) bool {
	if keeper == nil || keeper.pool == nil {
		return false
	}
	keys, keyed := keysOf(waiting)
	if !keyed || !slices.Equal(keys, keeper.pool.keys) {
		return false
	}
	last := keeper.pool
	encoded.triangles.bytes = last.triangles
	encoded.bends.bytes = last.bends
	encoded.nodes.bytes = last.nodes
	encoded.faces.nodes = last.faceNodes
	encoded.counts.triangles = last.triangleCount
	encoded.counts.nodes = last.nodeCount
	keeper.pooling = last
	keeper.reused++
	return true
}

// keep holds the scene's packed pool for the next frame, where every
// mesh in it has a key; a scene with no keeper keeps nothing.
func (keeper *keptTrees) keep(encoded *scene, waiting []pendingMesh) {
	if keeper == nil {
		return
	}
	keys, keyed := keysOf(waiting)
	keeper.pooling = nil
	if !keyed {
		return
	}
	// clipped, so nothing appended to a frame's buffers can reach them
	keeper.pooling = &keptPool{keys: keys,
		triangles:     slices.Clip(encoded.triangles.bytes),
		bends:         slices.Clip(encoded.bends.bytes),
		nodes:         slices.Clip(encoded.nodes.bytes),
		faceNodes:     slices.Clip(encoded.faces.nodes),
		triangleCount: encoded.counts.triangles,
		nodeCount:     encoded.counts.nodes}
}
