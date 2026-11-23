package trace

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

func TestMeshPaintSamplesTheRestPointOfARealRayHit(t *testing.T) {
	rest := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}
	posed := []maths.Vec{maths.V(2, 3, 1), maths.V(4, 3, 1), maths.V(2, 7, 1)}
	faces := [][3]int{{0, 1, 2}}
	skin, err := mesh.NewWithRestPoints(posed, rest, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	surface := thing.Thing{Shape: skin}
	placed := placedThing{thing: &surface, placement: placement{at: maths.NewSubspace(maths.V(10, 0, 0), maths.Identity, 1), scale: 1}}
	ray := maths.Ray{At: maths.V(12.5, 5, 4), Dir: maths.V(0, 0, -1)}
	distance, on, found := placed.reach(ray, 0, 8)
	if !found {
		t.Fatal("ray missed the mesh")
	}
	met := hit{thing: placed, on: on, dist: distance}
	local := placed.at.In(ray.Point(distance))
	if got := met.paintPoint(local); got.Sub(maths.V(.25, .5, 0)).Len() > 1e-12 {
		t.Fatalf("paint used %v, want the corresponding rest point", got)
	}
	surface.Shape, err = mesh.New(posed, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := met.paintPoint(local); got != local {
		t.Fatalf("ordinary mesh paint changed from %v to %v", local, got)
	}
}
