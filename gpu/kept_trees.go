package gpu

import (
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

// keptTrees is what a retained drawer keeps from one frame to the next of
// the meshes it drew: each mesh's face tree, by what its faces are (its
// meshKey), and the frame's whole packed pool of faces (keptPool).
//
// A tree is grown over a mesh's faces in the mesh's own space, so a mesh
// that only moves has the same tree in every frame. The heard finale's
// 39,112 meshes did not change between frames, and grew about fifteen
// seconds of trees and faces a frame before they were kept.
//
// A tree is kept by the mesh's bytes, never by its pointer: a shape.Mesh
// can be changed in place, and a mesh made again with the same faces is
// a new pointer.
//
// One of slayer's own meshes cannot change once mesh.New has made it, so
// its key is kept by its pointer, and its faces are hashed only the
// first frame it is seen.
//
// Only what the last frame used is kept: a film whose meshes change every
// frame holds two frames' trees at most, and never grows without bound.
type keptTrees struct {
	// kept is the trees the frame before used, and used this frame's
	kept, used map[meshKey]tree
	// named is the keys of slayer's meshes the frame before drew, by the
	// mesh, and naming this frame's
	named, naming map[*mesh.Mesh]meshKey
	// pool is the frame before's packed faces, and pooling this frame's;
	// see keptPool
	pool, pooling *keptPool
	// served is how many trees were taken from kept rather than grown,
	// and reused how many frames took the pool whole from the frame before
	served, reused int
}

// newKeptTrees is a keeper holding no trees yet.
func newKeptTrees() *keptTrees {
	return &keptTrees{kept: map[meshKey]tree{}, used: map[meshKey]tree{},
		named: map[*mesh.Mesh]meshKey{}, naming: map[*mesh.Mesh]meshKey{}}
}

// grownFor is the tree over the faces a key names: the one this frame or
// the last used, or grown now from its faces as an encode without a
// keeper grows it. Either way this frame used it.
func (keeper *keptTrees) grownFor(key meshKey,
	faces func() []triangle) tree {
	grown, known := keeper.used[key]
	if !known {
		if grown, known = keeper.kept[key]; known {
			keeper.served++
		}
	}
	if !known {
		grown = grownOver(faces())
	}
	keeper.used[key] = grown
	return grown
}

// shapeTree is a shape.Mesh's tree and key, its key taken of its bytes
// every frame, since its fields may have been changed since the last.
func (keeper *keptTrees) shapeTree(made *shape.Mesh) pendingMesh {
	key := shapeMeshKey(made)
	local := func() []triangle { return meshFaces(made) }
	return pendingMesh{key: key, keyed: true,
		tree: keeper.grownFor(key, local)}
}

// slayersTree is one of slayer's meshes' tree and key: the key by the
// mesh where it was drawn this frame or the last, and otherwise taken of
// its faces, which then grow its tree where no other mesh's has them.
func (keeper *keptTrees) slayersTree(skin *mesh.Mesh) pendingMesh {
	local := func() []triangle { return slayersFaces(skin) }
	key, known := keeper.naming[skin]
	if !known {
		key, known = keeper.named[skin]
	}
	if !known {
		faces := local()
		key = slayersMeshKey(faces)
		local = func() []triangle { return faces }
	}
	keeper.naming[skin] = key
	return pendingMesh{key: key, keyed: true,
		tree: keeper.grownFor(key, local)}
}

// turn ends a frame: the trees, keys and pool it used are kept for the
// next, and every other is let go.
func (keeper *keptTrees) turn() {
	keeper.kept, keeper.used = keeper.used, map[meshKey]tree{}
	keeper.named, keeper.naming = keeper.naming, map[*mesh.Mesh]meshKey{}
	keeper.pool, keeper.pooling = keeper.pooling, nil
}

// held is how many trees are kept for the next frame.
func (keeper *keptTrees) held() int { return len(keeper.kept) }
