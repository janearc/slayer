package placement

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestStagedMeshIdentifiesNearestAuthoredSurface(t *testing.T) {
	world := thing.NewWorld()
	near := world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 0, -4)})})
	far := world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 0, -8)})})
	first := thing.AuthorRef{Collection: "scene.world.spheres", Name: "near/with slash"}
	world.RecordAuthor(near, first)
	world.RecordAuthor(far, thing.AuthorRef{Collection: "scene.world.spheres", Name: "far"})
	got, err := WorkingWorld(world, 0)
	if err != nil {
		t.Fatal(err)
	}
	ray := maths.NewRay(maths.V(0, 0, 0), maths.V(0, 0, -2))
	if author, ok := got.Mesh.AuthorAt(ray); !ok || author != first {
		t.Fatalf("hit = %+v %v", author, ok)
	}
	for _, bad := range []maths.Ray{
		maths.NewRay(maths.V(0, 0, 0), maths.V(0, 0, 1)),
		maths.NewRay(maths.V(0, 0, 0), maths.V(0, 0, 0)),
		maths.NewRay(maths.V(math.NaN(), 0, 0), maths.V(0, 0, -1)),
	} {
		if author, ok := got.Mesh.AuthorAt(bad); ok {
			t.Fatalf("invalid or missed ray named %+v", author)
		}
	}
}

func TestUnannotatedSurfaceBlocksAuthoredSurfaceBehind(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 0, -4)})})
	far := world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 0, -8)})})
	world.RecordAuthor(far, thing.AuthorRef{Collection: "scene.world.spheres", Name: "far"})
	got, err := WorkingWorld(world, 0)
	if err != nil {
		t.Fatal(err)
	}
	if author, ok := got.Mesh.AuthorAt(maths.NewRay(maths.V(0, 0, 0), maths.V(0, 0, -1))); ok {
		t.Fatalf("unknown front surface selected %+v behind it", author)
	}
}
