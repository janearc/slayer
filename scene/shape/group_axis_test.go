package shape_test

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

func TestGroupSearchKeepsTheNearestLayerAlongItsLongestAxis(t *testing.T) {
	triangle := groupTriangle(t)
	var members []shape.MeshMember
	for _, z := range []float64{8, 0, 6, 2, 4, 1, 7, 3, 5} {
		members = append(members, shape.MeshMember{Mesh: triangle,
			Space: maths.Subspace{Origin: maths.V(0, 0, z)}})
	}
	group, err := shape.NewMeshGroup(members)
	if err != nil {
		t.Fatal(err)
	}
	for layer := 0; layer < len(members); layer++ {
		ray := maths.Ray{At: maths.V(.2, .2, float64(layer)+.5),
			Dir: maths.ZAxis.Neg()}
		distance, normal, hit := group.HitNormal(ray, 0, 20)
		if !hit || distance != .5 || normal != maths.ZAxis {
			t.Fatalf("layer %d: distance %g normal %v hit %v",
				layer, distance, normal, hit)
		}
	}
	plane := shape.Plane{}
	if normal := plane.Normal(maths.V(10, 0, 20)); normal != maths.Up {
		t.Fatalf("plane normal: %v", normal)
	}
	if _, hit := plane.Hit(maths.Ray{At: maths.V(0, 1, 0),
		Dir: maths.XAxis}, 0, 10); hit {
		t.Fatal("parallel ray met the plane")
	}
}
