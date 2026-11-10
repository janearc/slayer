package shape_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

func groupTriangle(t *testing.T) *mesh.Mesh {
	t.Helper()
	triangle, err := mesh.New([]maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}, [][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return triangle
}

func TestGroupMatchesExplicitlyTransformedFaces(t *testing.T) {
	triangle := groupTriangle(t)
	var members []shape.MeshMember
	var points []maths.Vec
	var faces [][3]int
	for i := range 16 {
		space := maths.Subspace{Origin: maths.V(float64(i%4)*3, float64(i/4)*3, float64(i%3)),
			Orientation: maths.About(maths.YAxis, float64(i)*0.1), Scale: 1 + float64(i%2)*0.5}
		members = append(members, shape.MeshMember{Mesh: triangle, Space: space})
		a, b, c := triangle.Corners(0)
		faces = append(faces, [3]int{len(points), len(points) + 1, len(points) + 2})
		points = append(points, space.Out(a), space.Out(b), space.Out(c))
	}
	group, err := shape.NewMeshGroup(members)
	if err != nil {
		t.Fatal(err)
	}
	flat, err := mesh.New(points, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	hits := 0
	// Avoid mathematical edge rays: rotating a ray versus rotating vertices
	// can round to opposite sides of the same zero-area silhouette boundary.
	for x := -0.983; x < 12; x += 0.2 {
		for y := -0.971; y < 12; y += 0.2 {
			ray := maths.Ray{At: maths.V(x, y, 10), Dir: maths.V(0, 0, -1)}
			a, hitA := group.Hit(ray, 0, 20)
			shaded, normal, hitNormal := group.HitNormal(ray, 0, 20)
			if hitNormal != hitA || hitA && math.Abs(shaded-a) > 1e-10 {
				t.Fatal("shading lookup changed the intersection")
			}
			b, hitB := flat.Hit(ray, 0, 20)
			if hitA != hitB || hitA && math.Abs(a-b) > 1e-10 {
				t.Fatalf("ray %v: group %g/%v, flat %g/%v", ray, a, hitA, b, hitB)
			}
			if hitA {
				hits++
				at := ray.Point(a)
				if normal.Dot(flat.Normal(at)) < 1-1e-9 {
					t.Fatal("hit normal differs from the transformed face")
				}
				if group.Normal(at).Dot(flat.Normal(at)) < 1-1e-9 {
					t.Fatal("transformed normal differs")
				}
				if math.Abs(group.Distance(at)) > 1e-9 {
					t.Fatal("hit point is not on group surface")
				}
			}
		}
	}
	if hits < 100 {
		t.Fatalf("only %d rays exercised group surfaces", hits)
	}
	if group.StoredFaces() != 1 || group.FaceCount() != 16 {
		t.Fatal("group expanded shared geometry")
	}
	copy := group.Members()
	copy[0].Mesh = nil
	if group.Members()[0].Mesh == nil {
		t.Fatal("caller changed group members")
	}
}

func TestGroupRejectsEmptyNestedAndInvalidTransforms(t *testing.T) {
	triangle := groupTriangle(t)
	group, _ := shape.NewMeshGroup([]shape.MeshMember{{Mesh: triangle}})
	for _, members := range [][]shape.MeshMember{nil, {{Mesh: shape.Sphere{Radius: 1}}},
		{{Mesh: group}}, {{Mesh: triangle, Space: maths.Subspace{Scale: -1}}},
		{{Mesh: triangle, Space: maths.Subspace{Origin: maths.V(math.NaN(), 0, 0)}}}} {
		if _, err := shape.NewMeshGroup(members); err == nil {
			t.Fatal("accepted invalid group")
		}
	}
}
