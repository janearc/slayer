package gpu

import (
	"runtime"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// treeGrowers is how many goroutines grow the trees of the world's
// meshes: as many as the processors the program is allowed.
func treeGrowers() int { return runtime.GOMAXPROCS(askOnly) }

// growTrees grows the tree of every one of slayer's meshes the world
// places, before the encoder asks for them one at a time and waits on
// each. The trees are the ones the encoder would have grown at its first
// ask, so nothing it packs changes; only the waiting does. A mesh whose
// tree the keeper holds from the frame before is left alone.
func (encoded *scene) growTrees(world *thing.World) {
	var wanted []*mesh.Mesh
	seen := map[*mesh.Mesh]bool{}
	want := func(made *mesh.Mesh) {
		if !seen[made] && !encoded.keeper.holds(made) {
			wanted = append(wanted, made)
		}
		seen[made] = true
	}
	for _, placed := range world.Things {
		if placed.Hidden() {
			continue
		}
		switch made := placed.Shape.(type) {
		case *mesh.Mesh:
			want(made)
		case *shape.MeshGroup:
			for _, member := range made.Members() {
				if skin, ok := member.Mesh.(*mesh.Mesh); ok {
					want(skin)
				}
			}
		}
	}
	growMeshes(wanted, treeGrowers())
}

// growMeshes grows the trees of meshes on at most workers goroutines, the
// meshes with the most faces first, so the longest builds start at once
// and the short ones fill in behind them. It leaves meshes in that order.
//
// A mesh cut by one goroutine alone, because the rest are busy, is cut
// by more as they come free.
func growMeshes(meshes []*mesh.Mesh, workers int) {
	slices.SortStableFunc(meshes, func(one, other *mesh.Mesh) int {
		return other.FaceCount() - one.FaceCount()
	})
	growers := mesh.NewGrowers(workers)
	var next atomic.Int64
	var done sync.WaitGroup
	for range min(workers, len(meshes)) {
		done.Add(1)
		go func() {
			defer done.Done()
			for {
				at := int(next.Add(1)) - 1
				if at >= len(meshes) {
					return
				}
				growers.Grow(meshes[at])
			}
		}()
	}
	done.Wait()
}

// holds is whether the keeper has the tree of a mesh from the frame
// before, or this one: the mesh, by its key, was drawn then. A keeper of
// nil holds nothing. It changes nothing the keeper keeps.
func (keeper *keptTrees) holds(skin *mesh.Mesh) bool {
	if keeper == nil {
		return false
	}
	key, known := keeper.naming[skin]
	if !known {
		key, known = keeper.named[skin]
	}
	if !known {
		key = slayersMeshKey(skin)
	}
	if _, used := keeper.used[key]; used {
		return true
	}
	_, kept := keeper.kept[key]
	return kept
}
