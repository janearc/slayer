package gpu

import "github.com/janearc/slayer/mesh"

// keptTrees is the face trees a retained drawer keeps from one frame to
// the next, one for each of slayer's meshes it drew, by the mesh.
//
// A tree is grown over a mesh's faces in the mesh's own space, so a mesh
// that only moves has the same tree in every frame: zappy's shells turn
// and tumble every frame and regrew about four seconds of trees each
// time.
//
// A *mesh.Mesh cannot change once mesh.New has made it, its fields
// being its own, so the pointer is a key that cannot name another mesh's
// faces. A mesh made again, as zappy's band and opening channels make
// one, is a new pointer and grows its own tree.
//
// Only the trees the last frame used are kept: a film whose meshes are
// made again every frame holds two frames' trees at most, and never
// grows without bound. A shape.Mesh, whose fields can be changed after
// it is made, is never kept.
type keptTrees struct {
	// kept is the trees the frame before used, and used this frame's
	kept, used map[*mesh.Mesh]tree
	// served is how many trees were taken from kept rather than grown
	served int
}

// newKeptTrees is a keeper holding no trees yet.
func newKeptTrees() *keptTrees {
	return &keptTrees{kept: map[*mesh.Mesh]tree{},
		used: map[*mesh.Mesh]tree{}}
}

// grownFor is the tree over a mesh's faces: the one kept from the last
// frame, or grown now from its faces as an encode without a keeper
// grows it. Either way this frame used it.
func (keeper *keptTrees) grownFor(skin *mesh.Mesh,
	faces func() []triangle) tree {
	grown, known := keeper.used[skin]
	if !known {
		if grown, known = keeper.kept[skin]; known {
			keeper.served++
		}
	}
	if !known {
		grown = grownOver(faces())
	}
	keeper.used[skin] = grown
	return grown
}

// turn ends a frame: the trees it used are kept for the next, and every
// other tree is let go.
func (keeper *keptTrees) turn() {
	keeper.kept, keeper.used = keeper.used, map[*mesh.Mesh]tree{}
}

// held is how many trees are kept for the next frame.
func (keeper *keptTrees) held() int { return len(keeper.kept) }
