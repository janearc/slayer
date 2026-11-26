package placement

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// PlacementBlockSize groups consecutive instances behind a conservative box.
const PlacementBlockSize = 256

// BlockBounds reports a conservative box for a block of placements. Bounds
// are unavailable after an edit invalidates the enclosing batch.
func (batch StagedBatch) BlockBounds(index int) (maths.Box, bool) {
	if !batch.HasBounds || index < 0 || index >= len(batch.pickBlocks) {
		return maths.Box{}, false
	}
	return batch.pickBlocks[index], true
}

// IndexPicking builds a compact, read-only picking index for large batches.
// It may be called after the snapshot has been validated and before sharing
// it with the UI. When an edit invalidates a batch's bounds, picking ignores
// its index until a fresh snapshot has been staged.
func (scene *DisplayScene) IndexPicking() {
	for i := range scene.Batches {
		batch := &scene.Batches[i]
		batch.pickBlocks = nil
		if !batch.HasBounds || batch.Mesh == nil || len(batch.Places) <= PlacementBlockSize {
			continue
		}
		low, high, ok := batch.Mesh.Bounds()
		if !ok {
			continue
		}
		centre := low.Add(high).Scale(.5)
		radius := high.Sub(low).Len() * .5
		for start := 0; start < len(batch.Places); start += PlacementBlockSize {
			minimum := maths.V(math.Inf(1), math.Inf(1), math.Inf(1))
			maximum := maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))
			for _, place := range batch.Places[start:min(start+PlacementBlockSize, len(batch.Places))] {
				point := place.Out(centre)
				scale := place.Scale
				if scale == 0 {
					scale = 1
				}
				reach := radius * scale
				minimum = maths.V(min(minimum.X, point.X-reach), min(minimum.Y, point.Y-reach), min(minimum.Z, point.Z-reach))
				maximum = maths.V(max(maximum.X, point.X+reach), max(maximum.Y, point.Y+reach), max(maximum.Z, point.Z+reach))
			}
			batch.pickBlocks = append(batch.pickBlocks, maths.Box{Min: minimum, Max: maximum})
		}
	}
}
