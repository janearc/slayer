package shape_test

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

// groupEdgesVectorFile is the table of points and rays the tests of this
// file use.
const groupEdgesVectorFile = "testdata/group-edges-vectors.csv"

func TestGroupsRejectSmoothAndTypedNilMembers(t *testing.T) {
	points := groupTrianglePoints(t)
	faces := groupTriangleFaces()
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
		_, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: member}})
		if err == nil || !strings.Contains(err.Error(), "require flat meshes") {
			t.Fatalf("smooth member: %v", err)
		}
	}
	var nilFast *mesh.Mesh
	var nilReference *shape.Mesh
	for _, member := range []shape.Shape{nilFast, nilReference, &shape.Mesh{}} {
		_, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: member}})
		if err == nil || !strings.Contains(err.Error(), "group needs meshes") {
			t.Fatalf("nil/empty member: %v", err)
		}
	}
}

// groupEdgesGroup is a group of the one triangle in its own place and the
// same triangle in a place a few metres along x, at a scale of one.
func groupEdgesGroup(t *testing.T) *shape.MeshGroup {
	t.Helper()
	vectors := groupVectorsIn(t, groupEdgesVectorFile)
	reference, err := shape.NewMesh(groupTrianglePoints(t),
		groupTriangleFaces(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	placed := maths.Local()
	placed.Origin = vectors.point("second-member")
	placed.Scale = groupUnity
	group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: reference},
		{Mesh: reference, Space: placed}})
	if err != nil {
		t.Fatal(err)
	}
	return group
}

// TestReferenceMeshGroupKeepsExactHitNormalAndIdentityScale says that a
// group of reference meshes gives the exact distance and normal of a hit,
// treats a scale of one as no scale, has the bounds of its members in
// their places, and finds no surface along a ray in the plane of a face.
func TestReferenceMeshGroupKeepsExactHitNormalAndIdentityScale(t *testing.T) {
	vectors := groupVectorsIn(t, groupEdgesVectorFile)
	group := groupEdgesGroup(t)
	ray := maths.Ray{At: vectors.point("above"), Dir: maths.ZAxis.Neg()}
	end := groupPicked(t, "group-edges-range-end")
	want := groupPicked(t, "group-edges-hit-distance")
	distance, normal, hit := group.HitNormal(ray, groupNought, end)
	if !hit || distance != want || normal != maths.ZAxis {
		t.Fatalf("reference hit: %g %v %v", distance, normal, hit)
	}
	got := group.Distance(ray.At)
	if math.Abs(got-want) > groupPicked(t, "group-edges-distance-tolerance") {
		t.Fatalf("identity member distance: %g", got)
	}
	bounds := group.Bounds()
	if bounds.Min.X != groupNought ||
		bounds.Max.X != groupPicked(t, "group-edges-bounds-width") ||
		bounds.Max.Y != groupUnity {
		t.Fatalf("placed bounds: %v", bounds)
	}
	plane := shape.Plane{}
	if plane.Distance(vectors.point("plane-query")) !=
		groupPicked(t, "group-edges-plane-height") {
		t.Fatal("plane distance is not its signed height")
	}
	coplanar := vectors.ray("coplanar")
	if _, hit := group.Hit(coplanar, groupNought, end); hit {
		t.Fatal("coplanar ray invented a surface crossing")
	}
}
