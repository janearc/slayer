package shape_test

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

func TestGroupsRejectSmoothAndTypedNilMembers(t *testing.T) {
	points := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}
	faces := [][3]int{{0, 1, 2}}
	normals := []maths.Vec{maths.ZAxis, maths.ZAxis, maths.ZAxis}
	fast, err := mesh.New(points, faces, normals, true)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := shape.NewMesh(points, faces, normals, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, member := range []shape.Shape{fast, reference} {
		if _, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: member}}); err == nil || !strings.Contains(err.Error(), "require flat meshes") {
			t.Fatalf("smooth member: %v", err)
		}
	}
	var nilFast *mesh.Mesh
	var nilReference *shape.Mesh
	for _, member := range []shape.Shape{nilFast, nilReference, &shape.Mesh{}} {
		if _, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: member}}); err == nil || !strings.Contains(err.Error(), "group needs meshes") {
			t.Fatalf("nil/empty member: %v", err)
		}
	}
}

func TestReferenceMeshGroupKeepsExactHitNormalAndIdentityScale(t *testing.T) {
	reference, err := shape.NewMesh([]maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}, [][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: reference}, {Mesh: reference, Space: maths.Subspace{Origin: maths.V(5, 0, 0), Scale: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	ray := maths.Ray{At: maths.V(.2, .2, 2), Dir: maths.ZAxis.Neg()}
	distance, normal, hit := group.HitNormal(ray, 0, 10)
	if !hit || distance != 2 || normal != maths.ZAxis {
		t.Fatalf("reference hit: %g %v %v", distance, normal, hit)
	}
	if got := group.Distance(ray.At); math.Abs(got-2) > 1e-14 {
		t.Fatalf("identity member distance: %g", got)
	}
	if bounds := group.Bounds(); bounds.Min.X != 0 || bounds.Max.X != 6 || bounds.Max.Y != 1 {
		t.Fatalf("placed bounds: %v", bounds)
	}
	if (shape.Plane{}).Distance(maths.V(3, 2, 4)) != 2 {
		t.Fatal("plane distance is not its signed height")
	}
	if _, hit := group.Hit(maths.Ray{At: maths.V(.2, .2, 0), Dir: maths.XAxis}, 0, 10); hit {
		t.Fatal("coplanar ray invented a surface crossing")
	}
}
