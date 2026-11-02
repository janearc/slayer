package mesh

import (
	"math"
	"slices"
	"testing"

	"github.com/janearc/slayer/maths"
)

// a face is met strictly between near and far: a face five along is
// not met with near at five, nor with far at five, and is just past it.
func TestAFaceIsMetStrictlyBetweenNearAndFar(t *testing.T) {
	made := mustMesh(t, aSquare(5), [][3]int{{0, 1, 2}}, nil, false)
	ray := maths.Ray{At: maths.V(0.2, 0.2, 0), Dir: maths.V(0, 0, 1)}
	if _, ok := made.Hit(ray, 5, 10); ok {
		t.Error("met at near")
	}
	if _, ok := made.Hit(ray, 0, 5); ok {
		t.Error("met at far")
	}
	if along, ok := made.Hit(ray, 0, math.Nextafter(5, 6)); !ok ||
		along != 5 {
		t.Errorf("just past far: %v at %v", ok, along)
	}
}

// aSquare is the corners of a unit square standing at a height on z.
func aSquare(height float64) []maths.Vec {
	return []maths.Vec{maths.V(0, 0, height), maths.V(1, 0, height),
		maths.V(1, 1, height), maths.V(0, 1, height)}
}

// two faces back to back are met at the same distance, and the first
// of them is the one met; it faces its own way.
func TestOfTwoAsNearTheFirstIsMet(t *testing.T) {
	made := mustMesh(t, aSquare(2), [][3]int{{0, 1, 2}, {0, 2, 1}}, nil,
		false)
	ray := maths.Ray{At: maths.V(0.6, 0.3, 0), Dir: maths.V(0, 0, 1)}
	met, ok := made.Meet(ray, 0, math.Inf(1))
	if !ok || met.Face() != 0 || made.NormalAt(met) != maths.V(0, 0, 1) {
		t.Errorf("met %v face %d facing %v", ok, met.Face(),
			made.NormalAt(met))
	}
}

// smooth normals blend by where a face is met: at a corner, that
// corner's normal; at the middle, the mean of the three made one long,
// one long.
func TestSmoothNormalsBlendWhereMet(t *testing.T) {
	normals := []maths.Vec{maths.V(1, 0, 1), maths.V(0, 1, 1),
		maths.V(0, 0, 1), maths.V(0, 0, 1)}
	made := mustMesh(t, aSquare(0), [][3]int{{0, 1, 2}}, normals, true)
	corner := made.facing(0, weights{1, 0, 0})
	middle := made.facing(0, weights{1.0 / 3, 1.0 / 3, 1.0 / 3})
	if corner.Sub(maths.V(1, 0, 1).Unit()).Len() > 1e-15 {
		t.Errorf("at a corner it faces %v", corner)
	}
	want := maths.V(1, 0, 1).Unit().Add(maths.V(0, 1, 1).Unit()).
		Add(maths.V(0, 0, 1)).Unit()
	if middle.Sub(want).Len() > 1e-14 {
		t.Errorf("at the middle it faces %v, not %v", middle, want)
	}
}

// a point's distance is to the nearest face, less than zero inside a closed
// mesh: the cube's middle is half in, a point two along x is one and a half
// out, a point beyond a corner is that far from it.
func TestDistanceIsToTheNearestFace(t *testing.T) {
	made := cube(t)
	for _, each := range []struct {
		point maths.Vec
		want  float64
	}{
		{maths.V(0, 0, 0), -0.5},
		{maths.V(2, 0, 0), 1.5},
		{maths.V(0.1, 0.2, -0.45), -0.05},
		{maths.V(1.5, 1.5, 1.5), math.Sqrt(3)},
	} {
		if got := made.Distance(each.point); math.Abs(got-each.want) >
			1e-12 {
			t.Errorf("at %v the distance is %v, not %v", each.point, got,
				each.want)
		}
	}
	open := mustMesh(t, cubeCorners(), cubeFaces()[:2], nil, false)
	if open.Inside(maths.V(0, 0, 0)) {
		t.Error("an open mesh has an inside")
	}
}

// a mesh gives back the corners and faces it was made of, as copies
// that change nothing in it when changed.
func TestAMeshGivesBackWhatItWasMadeOf(t *testing.T) {
	made := cube(t)
	points, faces := made.Points(), made.Faces()
	if !slices.Equal(points, cubeCorners()) ||
		!slices.Equal(faces, cubeFaces()) || made.FaceCount() != 12 {
		t.Fatalf("it gave back %v and %v", points, faces)
	}
	points[0], faces[0] = maths.V(9, 9, 9), [3]int{1, 1, 1}
	if first, _, _ := made.Corners(0); first != cubeCorners()[0] {
		t.Errorf("changing the copies moved its first corner to %v", first)
	}
}

// a point on the cube faces out of the side it is on, by the face
// nearest it.
func TestThePointsNormalIsItsSides(t *testing.T) {
	made := cube(t)
	for _, each := range []struct{ point, want maths.Vec }{
		{maths.V(0.5, 0.1, -0.2), maths.V(1, 0, 0)},
		{maths.V(0.2, -0.5, 0.3), maths.V(0, -1, 0)},
		{maths.V(-0.1, 0.3, 0.5), maths.V(0, 0, 1)},
	} {
		if got := made.Normal(each.point); got.Sub(each.want).Len() >
			1e-15 {
			t.Errorf("at %v the cube faces %v", each.point, got)
		}
	}
}
