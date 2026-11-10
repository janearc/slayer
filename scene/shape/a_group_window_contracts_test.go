package shape_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

func TestGroupPreservesFiniteRayWindowsAtEveryScale(t *testing.T) {
	// Two nearly touching faces have different normals. A ray meets both;
	// a narrow window selects exactly one, so a fresh unrestricted normal
	// search cannot accidentally select the neighbour.
	points := []maths.Vec{
		maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0),
		maths.V(0, 0, .01), maths.V(0, 1, .01), maths.V(1, 0, .01),
	}
	solid, err := mesh.New(points, [][3]int{{0, 1, 2}, {3, 4, 5}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, scale := range []float64{0, .25, 1, 3} {
		factor := scale
		if factor == 0 {
			factor = 1
		}
		space := maths.Subspace{Origin: maths.V(3, -2, 5), Orientation: maths.Rotation(maths.V(1, 2, 3), .6), Scale: scale}
		group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: solid, Space: space}})
		if err != nil {
			t.Fatal(err)
		}
		origin := space.Out(maths.V(.2, .2, 2))
		direction := space.Orientation.Out(maths.ZAxis.Neg())
		ray := maths.Ray{At: origin, Dir: direction}
		for _, face := range []struct{ z, sign float64 }{{.01, -1}, {0, 1}} {
			distance := (2 - face.z) * factor
			epsilon := .001 * factor
			near, far := distance-epsilon, distance+epsilon
			got, normal, hit := group.HitNormal(ray, near, far)
			wantNormal := space.Orientation.Out(maths.ZAxis.Scale(face.sign))
			if !hit || !(math.Abs(got-distance) <= 1e-11) || !(normal.Sub(wantNormal).Len() <= 1e-12) {
				t.Fatalf("scale %g z %g hit %t distance %g normal %v", scale, face.z, hit, got, normal)
			}
			if d, ok := group.Hit(ray, near, far); !ok || !(math.Abs(d-distance) <= 1e-11) {
				t.Fatalf("scale %g hit window lost distance: %g/%t", scale, d, ok)
			}
		}
		if _, hit := group.Hit(ray, .1*factor, .2*factor); hit {
			t.Fatal("surface before window")
		}
		if _, hit := group.Hit(ray, 2.1*factor, 3*factor); hit {
			t.Fatal("surface after window")
		}
	}
}

func TestGroupDistanceKeepsScaledNearestSurfaceAwayFromHits(t *testing.T) {
	triangle := groupTriangle(t)
	for _, scale := range []float64{.25, 2, 5} {
		group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: triangle, Space: maths.Subspace{Scale: scale}}})
		if err != nil {
			t.Fatal(err)
		}
		point := maths.V(.2*scale, .2*scale, 3)
		if d := group.Distance(point); !(math.Abs(d-3) <= 1e-12) {
			t.Fatalf("scale %g distance %g", scale, d)
		}
	}
}

func TestGroupBoundsAgreeWithRotatedBoxCentreAndHalfWidths(t *testing.T) {
	points := []maths.Vec{maths.V(1, -2, 3), maths.V(5, -2, 3), maths.V(1, 4, 3), maths.V(1, -2, 13)}
	solid, err := mesh.New(points, [][3]int{{0, 2, 1}, {0, 1, 3}, {0, 3, 2}, {1, 2, 3}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, axis := range []maths.Vec{maths.XAxis, maths.YAxis, maths.ZAxis, maths.V(1, 2, 3)} {
		for _, angle := range []float64{-.7, .4, 1.2} {
			space := maths.Subspace{Origin: maths.V(7, -11, 3), Orientation: maths.Rotation(axis, angle), Scale: 2.5}
			group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: solid, Space: space}})
			if err != nil {
				t.Fatal(err)
			}
			centre := space.Out(maths.V(3, 1, 8))
			turn := space.Orientation
			half := maths.V(
				2.5*(2*math.Abs(turn.X.X)+3*math.Abs(turn.Y.X)+5*math.Abs(turn.Z.X)),
				2.5*(2*math.Abs(turn.X.Y)+3*math.Abs(turn.Y.Y)+5*math.Abs(turn.Z.Y)),
				2.5*(2*math.Abs(turn.X.Z)+3*math.Abs(turn.Y.Z)+5*math.Abs(turn.Z.Z)))
			want := maths.Box{Min: centre.Sub(half), Max: centre.Add(half)}
			got := group.Bounds()
			if !(got.Min.Sub(want.Min).Len() <= 1e-12) || !(got.Max.Sub(want.Max).Len() <= 1e-12) {
				t.Fatalf("axis %v angle %g bounds %v, want %v", axis, angle, got, want)
			}
		}
	}
}

func TestGroupsAcceptFlatNormalsAndRefuseEveryNonemptySmoothNormalSet(t *testing.T) {
	points := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}
	faces := [][3]int{{0, 1, 2}}
	normals := []maths.Vec{maths.ZAxis, maths.ZAxis, maths.ZAxis}
	for _, row := range []struct {
		normals []maths.Vec
		smooth  bool
	}{{normals, false}, {nil, true}} {
		reference, err := shape.NewMesh(points, faces, row.normals, row.smooth)
		if err != nil {
			t.Fatal(err)
		}
		group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: reference}})
		if err != nil {
			t.Fatalf("flat reference mesh refused: %v", err)
		}
		_, normal, hit := group.HitNormal(maths.Ray{At: maths.V(.2, .2, 1), Dir: maths.ZAxis.Neg()}, 0, 2)
		if !hit || normal != maths.ZAxis {
			t.Fatal("flat reference normal changed")
		}
	}
	// Mesh exposes its fields. A malformed one-normal smooth input must be
	// refused by the group's flat-only boundary before any normal is indexed.
	malformed := &shape.Mesh{Vertices: points, Faces: faces, Normals: []maths.Vec{maths.ZAxis}, Smooth: true}
	if _, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: malformed}}); err == nil {
		t.Fatal("accepted nonempty smooth normals")
	}
}
