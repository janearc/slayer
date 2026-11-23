package shape_test

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// What the tests of this file count in, each true by definition.
const (
	cornersOfAFace  = 3 // defined: the corners of a face
	cornersFourth   = 3 // defined: the place of the fourth vertex of a square
	cornersThirdOne = 2 // defined: the place of the third corner of a face
)

// The tables the tests of this file read: points, and the places the rays
// come down at, each a row with an x and a y.
const (
	cornersVectorFile = "testdata/corners-vectors.csv"
	cornersDownFile   = "testdata/corners-downs.csv"
)

// cornersFace is the three corners of a face.
type cornersFace = [cornersOfAFace]maths.Vec

// aSquare is this package's mesh of a unit square in the z = 0 plane,
// as two faces: the lower right, then the upper left.
func aSquare(t *testing.T) *shape.Mesh {
	t.Helper()
	vectors := groupVectorsIn(t, cornersVectorFile)
	square, err := shape.NewMesh([]maths.Vec{vectors.point("square-first"),
		vectors.point("square-second"), vectors.point("square-third"),
		vectors.point("square-fourth")},
		[][cornersOfAFace]int{{groupNought, groupUnity, groupPair},
			{groupNought, groupPair, cornersFourth}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return square
}

// down is a ray from in front of the z = 0 plane at x, y, toward it.
func down(t *testing.T, x, y float64) maths.Ray {
	t.Helper()
	vectors := groupVectorsIn(t, cornersVectorFile)
	height := groupPicked(t, "corners-ray-height")
	return maths.Ray{At: maths.V(x, y, height), Dir: vectors.point("downward")}
}

// cornersDown is the ray of the row called name in the table of the places
// the rays come down at.
func cornersDown(t *testing.T, name string) maths.Ray {
	t.Helper()
	var row map[string]string
	for _, each := range groupRows(t, cornersDownFile) {
		if each["name"] == name {
			row = each
		}
	}
	if row == nil {
		t.Fatalf("%s: no row called %q", cornersDownFile, name)
	}
	return down(t, groupCell(t, row, "x"), groupCell(t, row, "y"))
}

// cornersWanted is the face whose corners are the rows name-first,
// name-second and name-third of the table of points.
func cornersWanted(t *testing.T, name string) cornersFace {
	t.Helper()
	vectors := groupVectorsIn(t, cornersVectorFile)
	return cornersFace{vectors.point(name + "-first"),
		vectors.point(name + "-second"), vectors.point(name + "-third")}
}

// sameCorners is whether two faces have the same corners, in any order.
func sameCorners(t *testing.T, got, want [cornersOfAFace]maths.Vec) bool {
	t.Helper()
	tolerance := groupPicked(t, "corners-corner-tolerance")
	for _, corner := range want {
		found := false
		for _, each := range got {
			found = found || each.Sub(corner).Len() < tolerance
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
	end := groupPicked(t, "corners-range-end")
	for _, name := range []string{"lower-right", "upper-left"} {
		ray, want := cornersDown(t, name), cornersWanted(t, name)
		along, corners, ok := square.HitCorners(ray, groupNought, end)
		hit, _ := square.Hit(ray, groupNought, end)
		if !ok || along != hit || !sameCorners(t, corners, want) {
			t.Errorf("at %v: %v %v %v, want %v at %v", ray.At, ok,
				along, corners, want, hit)
		}
	}
}

// anti-test of TestMeshHitCornersAreTheFaceHitMeets
// TestMeshHitCornersMissesBesideTheMesh holds that a ray beside the
// mesh meets no face and gives no corners.
func TestMeshHitCornersMissesBesideTheMesh(t *testing.T) {
	end := groupPicked(t, "corners-range-end")
	ray := cornersDown(t, "beside-the-square")
	_, corners, ok := aSquare(t).HitCorners(ray, groupNought, end)
	if ok || corners != (cornersFace{}) {
		t.Errorf("a ray beside the square met %v", corners)
	}
}

// TestGroupHitCornersArePutOutOfTheMembersPlace holds that a group's
// corners are its member's face, of either kind of mesh, carried out
// of the member's place into the group's space.
func TestGroupHitCornersArePutOutOfTheMembersPlace(t *testing.T) {
	vectors := groupVectorsIn(t, cornersVectorFile)
	placed := maths.NewSubspace(vectors.point("placement"), maths.Identity,
		groupPicked(t, "corners-placement-scale"))
	end := groupPicked(t, "corners-range-end")
	ray := cornersDown(t, "over-the-member")
	for _, member := range []shape.Shape{groupTriangle(t), aSquare(t)} {
		group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: member,
			Space: placed}})
		if err != nil {
			t.Fatal(err)
		}
		along, corners, ok := group.HitCorners(ray, groupNought, end)
		want := cornersWanted(t, "placed")
		if _, square := member.(*shape.Mesh); !square {
			want[cornersThirdOne] = vectors.point("placed-third-of-triangle")
		}
		if !ok || along != groupPicked(t, "corners-placed-distance") ||
			!sameCorners(t, corners, want) {
			t.Errorf("%T: %v %v %v, want %v", member, ok, along, corners,
				want)
		}
	}
}

// anti-test of TestGroupHitCornersArePutOutOfTheMembersPlace
// TestGroupHitCornersMissesBesideTheGroup holds that a ray beside every
// member gives no corners.
func TestGroupHitCornersMissesBesideTheGroup(t *testing.T) {
	space := maths.Local()
	space.Orientation = maths.Identity
	group, err := shape.NewMeshGroup([]shape.MeshMember{{
		Mesh: groupTriangle(t), Space: space}})
	if err != nil {
		t.Fatal(err)
	}
	end := groupPicked(t, "corners-range-end")
	ray := cornersDown(t, "beside-the-group")
	if _, corners, ok := group.HitCorners(ray, groupNought, end); ok {
		t.Errorf("a ray beside the group met %v", corners)
	}
}
