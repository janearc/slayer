package placement

import (
	"testing"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestWorkingInstancesShareGeometryAndKeepStagedTransforms(t *testing.T) {
	triangle, err := shape.NewMesh([]maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}, [][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	first := maths.NewSubspace(maths.V(1, 2, 3), maths.About(maths.YAxis, 0.4), 2)
	second := maths.NewSubspace(maths.V(-2, 4, 1), maths.About(maths.YAxis, -0.2), 0.5)
	group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: triangle, Space: first}, {Mesh: triangle, Space: second}})
	if err != nil {
		t.Fatal(err)
	}
	parent := maths.NewSubspace(maths.V(8, 3, -4), maths.About(maths.YAxis, 0.7), 3)
	world := thing.NewWorld()
	part := world.Add(thing.Thing{Shape: group, Motion: thing.Still(parent)})
	author := thing.AuthorRef{Collection: "scene.set.things", Name: "example"}
	world.RecordAuthor(part, author)
	world.AddVolume(thing.Volume{})
	world.Add(thing.Thing{Shape: &arid.Sea{}})
	got, err := WorkingInstances(world, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Batches) != 2 || len(got.Batches[0].Places) != 2 || got.Volumes != 1 || got.OmittedSeas != 0 {
		t.Fatalf("instances do not share a mesh: %+v", got)
	}
	sea := got.Batches[1]
	if sea.Mesh.Notice != "Sea shown as a finite mean-level proxy; waves and coastline are not shown" {
		t.Fatal("sea proxy was not labeled")
	}
	if low, high, ok := sea.Mesh.Bounds(); !ok || low.Y != 0 || high.Y != 0 || low.X >= high.X || low.Z >= high.Z {
		t.Fatalf("sea proxy bounds = %v %v %v", low, high, ok)
	}
	point := maths.V(1, 0, 0)
	for i, local := range []maths.Subspace{first, second} {
		batch := got.Batches[0]
		if !batch.Authored || batch.Author != author {
			t.Fatalf("instance lost its author: %+v", batch)
		}
		if delta := batch.Places[i].Out(point).Sub(parent.Out(local.Out(point))).Len(); delta > 1e-9 {
			t.Fatalf("instance %d transform differs by %g", i, delta)
		}
	}
}
