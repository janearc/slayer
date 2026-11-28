package placement

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestFarClippedBatchesUseStagedCameraDepth(t *testing.T) {
	eye := maths.V(3000, 7, 10)
	box := func(depth, reach float64) StagedBatch {
		centre := eye.Add(maths.V(depth, 0, 1e6))
		edge := maths.V(reach, reach, reach)
		return StagedBatch{Bounds: maths.Box{Min: centre.Sub(edge), Max: centre.Add(edge)}, HasBounds: true}
	}
	scene := &DisplayScene{
		Eye: maths.Subspace{Origin: eye, Orientation: maths.LookAt(eye, eye.Add(maths.XAxis))},
		Batches: []StagedBatch{
			box(DisplayFar+100, 50),
			box(DisplayFar+50, 50),
			box(DisplayFar+20, 50),
			box(10, 5),
			box(-DisplayFar-100, 50),
			{Bounds: box(DisplayFar+100, 50).Bounds},
		},
	}
	if got := scene.FarClippedBatches(); got != 1 {
		t.Fatalf("far-clipped batches = %d, want just the first", got)
	}
}
