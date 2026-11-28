package workingview

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
)

func TestStableBatchesRejectCameraAndStructureChanges(t *testing.T) {
	a, b := &placement.DisplayMesh{}, &placement.DisplayMesh{}
	scene := &placement.DisplayScene{Width: 640, Height: 400, Batches: []placement.StagedBatch{
		{Mesh: a, Places: []maths.Subspace{{Origin: maths.V(1, 0, 0)}}},
		{Mesh: b, Places: []maths.Subspace{{Origin: maths.V(2, 0, 0)}}},
	}}
	last := rememberBatches(scene, 640, 400, nil)
	changed, ok := last.changes(scene, 640, 400)
	if !ok || len(changed) != 0 {
		t.Fatal("unchanged batches were not reusable")
	}
	copy := *scene
	copy.Batches = append([]placement.StagedBatch(nil), scene.Batches...)
	copy.Batches[0].Places = []maths.Subspace{{Origin: maths.V(3, 0, 0)}}
	changed, ok = last.changes(&copy, 640, 400)
	if !ok || !changed[a] || changed[b] {
		t.Fatal("replaced placement did not invalidate only its mesh")
	}
	copy.Eye.Origin.X++
	if _, ok := last.changes(&copy, 640, 400); ok {
		t.Fatal("camera change reused old visibility")
	}
	copy.Eye = scene.Eye
	copy.Batches[1].Mesh = a
	if _, ok := last.changes(&copy, 640, 400); ok {
		t.Fatal("changed batch structure reused old buffers")
	}
	if _, ok := last.changes(scene, 320, 200); !ok {
		t.Fatal("same field of view did not reuse visibility at a new size")
	}
	wider := *scene
	wider.SetViewport(400, 400)
	if _, ok := last.changes(&wider, 400, 400); ok {
		t.Fatal("expanded field of view reused incomplete visibility")
	}
}
