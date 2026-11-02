package mesh

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// a face flat in y zero, its corners' normals leaning out from its middle as a
// dome's would, so the curved surface they draw stands above the face
// everywhere inside it.
func dome(t *testing.T, smooth bool) *Mesh {
	t.Helper()
	corners := []maths.Vec{maths.V(0, 0, 0), maths.V(4, 0, 0),
		maths.V(0, 0, 4)}
	normals := []maths.Vec{maths.V(-1, 3, -1), maths.V(1, 3, -0.2),
		maths.V(-0.2, 3, 1)}
	return mustMesh(t, corners, [][3]int{{0, 1, 2}}, normals, smooth)
}

// a ray straight down onto the dome's face, met where it lands.
func landed(t *testing.T, made *Mesh, x, z float64) (Met, maths.Vec) {
	t.Helper()
	ray := maths.Ray{At: maths.V(x, 5, z), Dir: maths.V(0, -1, 0)}
	met, ok := made.Meet(ray, 0, 10)
	if !ok {
		t.Fatalf("no face under %g, %g", x, z)
	}
	return met, ray.Point(met.Along())
}

// a point inside a face shaded smooth, under each corner's tangent
// plane, is lifted above the face, toward the surface its normals draw;
// at a corner it stays where it is; shaded flat, nothing moves.
func TestAPointIsLiftedToItsCurve(t *testing.T) {
	smooth := dome(t, true)
	met, point := landed(t, smooth, 1.2, 1.3)
	lifted := smooth.Lifted(met, point)
	if !(lifted.Y > 1e-6) {
		t.Errorf("inside the face, lifted to %v from %v", lifted, point)
	}
	if moved := lifted.Sub(point).Len(); moved > 1 {
		t.Errorf("lifted %g, further than the face is wide allows", moved)
	}
	corner, at := landed(t, smooth, 1e-6, 1e-6)
	if moved := smooth.Lifted(corner, at).Sub(at).Len(); moved > 1e-5 {
		t.Errorf("at a corner, moved %g", moved)
	}
	flat := dome(t, false)
	met, point = landed(t, flat, 1.2, 1.3)
	if got := flat.Lifted(met, point); got != point {
		t.Errorf("shaded flat, moved to %v from %v", got, point)
	}
}

// a mesh shaded smooth gives its corners' normals, made one long, in
// the order of its corners; shaded flat, none.
func TestCornerNormalsFollowTheCorners(t *testing.T) {
	smooth := dome(t, true)
	normals, ok := smooth.CornerNormals(0)
	if !ok {
		t.Fatal("a smooth mesh gave no normals")
	}
	want := []maths.Vec{maths.V(-1, 3, -1).Unit(),
		maths.V(1, 3, -0.2).Unit(), maths.V(-0.2, 3, 1).Unit()}
	for k, normal := range normals {
		if normal.Sub(want[k]).Len() > 1e-12 {
			t.Errorf("corner %d's normal is %v, not %v", k, normal,
				want[k])
		}
	}
	if _, ok := dome(t, false).CornerNormals(0); ok {
		t.Error("a mesh shaded flat gave normals")
	}
}

// a point above every corner's tangent plane is not pulled down: the
// lift only raises.
func TestALiftOnlyRaises(t *testing.T) {
	corners := []maths.Vec{maths.V(0, 0, 0), maths.V(4, 0, 0),
		maths.V(0, 0, 4)}
	bowl := []maths.Vec{maths.V(1, 3, 1), maths.V(-1, 3, 0.2),
		maths.V(0.2, 3, -1)}
	made := mustMesh(t, corners, [][3]int{{0, 1, 2}}, bowl, true)
	met, point := landed(t, made, 1.2, 1.3)
	if got := made.Lifted(met, point); math.Abs(got.Sub(point).Len()) >
		1e-12 {
		t.Errorf("in a bowl, moved to %v from %v", got, point)
	}
}
