package placement

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

func TestBatchHitFindsNearestAuthoredThingAndHonoursBlockers(t *testing.T) {
	mesh := &DisplayMesh{Vertices: []DisplayVertex{{Position: maths.V(-1, -1, 0)}, {Position: maths.V(1, -1, 0)}, {Position: maths.V(0, 1, 0)}}, Indices: []uint32{0, 1, 2}}
	near := StagedBatch{Mesh: mesh, Places: []maths.Subspace{{Origin: maths.V(0, 0, -4)}}, Author: thing.AuthorRef{Collection: "scene.set.things", Name: "fish"}, Authored: true}
	far := StagedBatch{Mesh: mesh, Places: []maths.Subspace{{Origin: maths.V(0, 0, -8)}}, Author: thing.AuthorRef{Collection: "scene.set.things", Name: "city"}, Authored: true}
	scene := DisplayScene{Batches: []StagedBatch{far, near}}
	ray := maths.NewRay(maths.V(0, 0, 0), maths.V(0, 0, -2))
	hit, ok := scene.batchHit(ray, math.Inf(1))
	if !ok || hit.ID != "scene.set.things/fish" || math.Abs(hit.Distance-4) > 1e-9 {
		t.Fatalf("nearest hit = %+v %v", hit, ok)
	}
	scene.Batches[1].Authored = false
	hit, ok = scene.batchHit(ray, math.Inf(1))
	if !ok || hit.ID != "scene.world" {
		t.Fatalf("unowned blocker = %+v %v", hit, ok)
	}
	scene.Batches[1].Authored = true
	scene.Batches[1].Author.Collection = "scene.set.stands"
	hit, ok = scene.batchHit(ray, math.Inf(1))
	if !ok || hit.ID != "scene.set.stands/fish" {
		t.Fatalf("generated member did not identify its authored group: %+v %v", hit, ok)
	}
	scene.Batches[1].Author.Collection = "scene.set.things"
	mesh.Notice = "approximation"
	hit, ok = scene.batchHit(ray, math.Inf(1))
	if !ok || hit.ID != "scene.world" {
		t.Fatalf("proxy was editable: %+v %v", hit, ok)
	}
	if _, ok := scene.batchHit(maths.NewRay(maths.V(0, 0, 0), maths.V(0, 1, 0)), math.Inf(1)); ok {
		t.Fatal("missed ray selected a batch")
	}
}

func TestIndexedBatchPickingMatchesUnindexedAndInvalidatesAfterEdit(t *testing.T) {
	mesh := &DisplayMesh{Vertices: []DisplayVertex{{Position: maths.V(-1, -1, 0)}, {Position: maths.V(1, -1, 0)}, {Position: maths.V(0, 1, 0)}}, Indices: []uint32{0, 1, 2}}
	batch := StagedBatch{Mesh: mesh, Author: thing.AuthorRef{Collection: "scene.set.things", Name: "row"}, Authored: true}
	for i := 0; i < 3*PlacementBlockSize; i++ {
		batch.Places = append(batch.Places, maths.Subspace{Origin: maths.V(float64(i-PlacementBlockSize), 0, -10), Scale: 2})
	}
	scene := DisplayScene{Batches: []StagedBatch{batch}}
	stagedBounds(scene.Batches)
	rays := []maths.Ray{
		maths.NewRay(maths.V(0, 0, 0), maths.V(0, 0, -1)),
		maths.NewRay(maths.V(0, 0, 0), maths.V(100, 0, -10)),
		maths.NewRay(maths.V(0, 0, 0), maths.V(0, 1, 0)),
	}
	want := make([]DisplayHit, len(rays))
	wantOK := make([]bool, len(rays))
	for i, ray := range rays {
		want[i], wantOK[i] = scene.batchHit(ray, math.Inf(1))
	}
	scene.IndexPicking()
	if len(scene.Batches[0].pickBlocks) != 3 {
		t.Fatal("large batch did not receive three picking blocks")
	}
	for i, ray := range rays {
		got, ok := scene.batchHit(ray, math.Inf(1))
		if ok != wantOK[i] || got != want[i] {
			t.Fatalf("ray %d: indexed %v %v, unindexed %v %v", i, got, ok, want[i], wantOK[i])
		}
	}
	// An interactive edit marks old bounds invalid; stale index boxes must not
	// hide the moved geometry.
	scene.Batches[0].HasBounds = false
	scene.Batches[0].Places[0].Origin = maths.V(0, 0, -2)
	got, ok := scene.batchHit(rays[0], math.Inf(1))
	if !ok || math.Abs(got.Distance-2) > 1e-9 {
		t.Fatalf("moved instance missed after index invalidation: %+v %v", got, ok)
	}
}

func TestPlacementBlocksContainEveryTransformedVertex(t *testing.T) {
	mesh := &DisplayMesh{Vertices: []DisplayVertex{{Position: maths.V(-3, -1, -2)}, {Position: maths.V(5, 4, 1)}, {Position: maths.V(0, -2, 6)}}, Indices: []uint32{0, 1, 2}}
	batch := StagedBatch{Mesh: mesh}
	for i := 0; i < 2*PlacementBlockSize+3; i++ {
		batch.Places = append(batch.Places, maths.Subspace{Origin: maths.V(float64(i), float64(i%11), -float64(i%17)), Orientation: maths.About(maths.YAxis, float64(i)*.13), Scale: .2 + float64(i%9)})
	}
	scene := DisplayScene{Batches: []StagedBatch{batch}}
	stagedBounds(scene.Batches)
	scene.IndexPicking()
	for i, place := range scene.Batches[0].Places {
		box, ok := scene.Batches[0].BlockBounds(i / PlacementBlockSize)
		if !ok {
			t.Fatalf("missing block for placement %d", i)
		}
		for _, vertex := range mesh.Vertices {
			point := place.Out(vertex.Position)
			if point.X < box.Min.X || point.Y < box.Min.Y || point.Z < box.Min.Z || point.X > box.Max.X || point.Y > box.Max.Y || point.Z > box.Max.Z {
				t.Fatalf("placement %d: block %v does not contain %v", i, box, point)
			}
		}
	}
	scene.Batches[0].HasBounds = false
	if _, ok := scene.Batches[0].BlockBounds(0); ok {
		t.Fatal("invalidated batch retained visible block bounds")
	}
}
