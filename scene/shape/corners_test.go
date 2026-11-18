package shape_test

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// aSquare is this package's mesh of a unit square in the z = 0 plane,
// as two faces: the lower right, then the upper left.
func aSquare(t *testing.T) *shape.Mesh {
	t.Helper()
	square, err := shape.NewMesh([]maths.Vec{maths.V(0, 0, 0),
		maths.V(1, 0, 0), maths.V(1, 1, 0), maths.V(0, 1, 0)},
		[][3]int{{0, 1, 2}, {0, 2, 3}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return square
}

// down is a ray from in front of the z = 0 plane at x, y, toward it.
func down(x, y float64) maths.Ray {
	return maths.Ray{At: maths.V(x, y, 5), Dir: maths.V(0, 0, -1)}
}

// sameCorners is whether two faces have the same corners, in any order.
func sameCorners(got, want [3]maths.Vec) bool {
	for _, corner := range want {
		found := false
		for _, each := range got {
			found = found || each.Sub(corner).Len() < 1e-12
		}
		if !found {
			return false
		}
	}
	return true
}

// TestMeshHitCornersAreTheFaceHitMeets holds that the corners a mesh
// gives are the face's the ray met, at the distance Hit gives.
func TestMeshHitCornersAreTheFaceHitMeets(t *testing.T) {
	square := aSquare(t)
	for _, each := range []struct {
		ray  maths.Ray
		want [3]maths.Vec
	}{
		{down(0.8, 0.2), [3]maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0),
			maths.V(1, 1, 0)}},
		{down(0.2, 0.8), [3]maths.Vec{maths.V(0, 0, 0), maths.V(1, 1, 0),
			maths.V(0, 1, 0)}},
	} {
		along, corners, ok := square.HitCorners(each.ray, 0, 100)
		hit, _ := square.Hit(each.ray, 0, 100)
		if !ok || along != hit || !sameCorners(corners, each.want) {
			t.Errorf("at %v: %v %v %v, want %v at %v", each.ray.At, ok,
				along, corners, each.want, hit)
		}
	}
}

// anti-test of TestMeshHitCornersAreTheFaceHitMeets
// TestMeshHitCornersMissesBesideTheMesh holds that a ray beside the
// mesh meets no face and gives no corners.
func TestMeshHitCornersMissesBesideTheMesh(t *testing.T) {
	_, corners, ok := aSquare(t).HitCorners(down(1.5, 0.5), 0, 100)
	if ok || corners != ([3]maths.Vec{}) {
		t.Errorf("a ray beside the square met %v", corners)
	}
}

// TestGroupHitCornersArePutOutOfTheMembersPlace holds that a group's
// corners are its member's face, of either kind of mesh, carried out
// of the member's place into the group's space.
func TestGroupHitCornersArePutOutOfTheMembersPlace(t *testing.T) {
	placed := maths.Subspace{Origin: maths.V(5, 0, 0),
		Orientation: maths.Identity, Scale: 2}
	for _, member := range []shape.Shape{groupTriangle(t), aSquare(t)} {
		group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: member,
			Space: placed}})
		if err != nil {
			t.Fatal(err)
		}
		along, corners, ok := group.HitCorners(down(5.4, 0.2), 0, 100)
		want := [3]maths.Vec{maths.V(5, 0, 0), maths.V(7, 0, 0),
			maths.V(7, 2, 0)}
		if _, square := member.(*shape.Mesh); !square {
			want[2] = maths.V(5, 2, 0)
		}
		if !ok || along != 5 || !sameCorners(corners, want) {
			t.Errorf("%T: %v %v %v, want %v", member, ok, along, corners,
				want)
		}
	}
}

// anti-test of TestGroupHitCornersArePutOutOfTheMembersPlace
// TestGroupHitCornersMissesBesideTheGroup holds that a ray beside every
// member gives no corners.
func TestGroupHitCornersMissesBesideTheGroup(t *testing.T) {
	group, err := shape.NewMeshGroup([]shape.MeshMember{{
		Mesh: groupTriangle(t), Space: maths.Subspace{
			Orientation: maths.Identity}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, corners, ok := group.HitCorners(down(3, 3), 0, 100); ok {
		t.Errorf("a ray beside the group met %v", corners)
	}
}
