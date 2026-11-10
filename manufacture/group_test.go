package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

func TestPlacementAndGatherRetainGroupedGeometry(t *testing.T) {
	triangle, err := (surface{{maths.V(-1, -1, 0), maths.V(1, -1, 0), maths.V(0, 1, 0)}}).meshed()
	if err != nil {
		t.Fatal(err)
	}
	group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: triangle}})
	if err != nil {
		t.Fatal(err)
	}
	original := good{groups: map[slot]*shape.MeshGroup{body: group}, dressed: looks{body: bark}}
	space := maths.Subspace{Origin: maths.V(8, 3, -2), Orientation: maths.About(maths.YAxis, .4), Scale: 2}
	moved, err := original.placed(space)
	if err != nil {
		t.Fatal(err)
	}
	together, err := gather(original, moved)
	if err != nil {
		t.Fatal(err)
	}
	things, err := together.Things(maths.Subspace{})
	if err != nil {
		t.Fatal(err)
	}
	if len(things) != 2 {
		t.Fatalf("lost grouped material: %d things", len(things))
	}
	for i, placement := range []maths.Subspace{{}, space} {
		solid, ok := things[i].Shape.(*shape.MeshGroup)
		if !ok || solid.StoredFaces() != 1 {
			t.Fatal("group was expanded or dropped")
		}
		point := placement.Out(maths.V(0, 0, 0))
		normal := placement.Orientation.Out(maths.ZAxis)
		ray := maths.Ray{At: point.Add(normal.Scale(2)), Dir: normal.Neg()}
		distance, hit := solid.Hit(ray, 0, 10)
		if !hit || math.Abs(distance-2) > 1e-10 {
			t.Fatalf("placement %d: hit %v, distance %g", i, hit, distance)
		}
	}
}
