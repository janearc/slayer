package workingview

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
)

// stableBatches records identities only for caller-owned immutable placements.
// A changed placement must have its own slice; in-place mutation is not safe.
type stableBatches struct {
	visibility frustum
	batches    []stableBatch
	counts     map[*placement.DisplayMesh]uint32
}

type stableBatch struct {
	mesh   *placement.DisplayMesh
	places []maths.Subspace
	bounds maths.Box
	has    bool
}

func rememberBatches(scene *placement.DisplayScene, width, height int, counts map[*placement.DisplayMesh]uint32) *stableBatches {
	if len(scene.Objects) != 0 || len(scene.Batches) == 0 {
		return nil
	}
	last := &stableBatches{visibility: viewFrustum(scene, width, height), counts: counts}
	last.batches = make([]stableBatch, len(scene.Batches))
	for i, batch := range scene.Batches {
		last.batches[i] = stableBatch{mesh: batch.Mesh, places: batch.Places, bounds: batch.Bounds, has: batch.HasBounds}
	}
	return last
}

// changes permits reuse only if the new frustum fits inside the visibility
// already uploaded, with the same camera and batch structure.
func (last *stableBatches) changes(scene *placement.DisplayScene, width, height int) (map[*placement.DisplayMesh]bool, bool) {
	if last == nil || len(scene.Objects) != 0 || len(scene.Batches) != len(last.batches) {
		return nil, false
	}
	now := viewFrustum(scene, width, height)
	if now.eye != last.visibility.eye || now.horizontal > last.visibility.horizontal || now.vertical > last.visibility.vertical {
		return nil, false
	}
	changed := make(map[*placement.DisplayMesh]bool)
	for i, batch := range scene.Batches {
		old := last.batches[i]
		if batch.Mesh != old.mesh {
			return nil, false
		}
		same := len(batch.Places) == len(old.places)
		if same && len(batch.Places) > 0 {
			same = &batch.Places[0] == &old.places[0]
		}
		if !same || batch.HasBounds != old.has || batch.Bounds != old.bounds {
			changed[batch.Mesh] = true
		}
	}
	return changed, true
}
