package mesh

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// vectorsIn is the vectors a csv file holds, a row each: x, y and z.
func vectorsIn(t *testing.T, file string) []maths.Vec {
	t.Helper()
	var vectors []maths.Vec
	for _, row := range read(t, file) {
		x, y, z := float(t, row, "x"), float(t, row, "y"), float(t, row, "z")
		vectors = append(vectors, maths.V(x, y, z))
	}
	return vectors
}

// theOneFace is the faces of a mesh of one triangle: its three corners,
// in their order.
func theOneFace() [][triangleCorners]int {
	var face [triangleCorners]int
	for place := range face {
		face[place] = place
	}
	return [][triangleCorners]int{face}
}

// a face flat in y zero, its corners' normals leaning out from its middle as a
// dome's would, so the curved surface they draw stands above the face
// everywhere inside it.
func dome(t *testing.T, smooth bool) *Mesh {
	t.Helper()
	corners := vectorsIn(t, "testdata/lifted-triangle.csv")
	normals := vectorsIn(t, "testdata/lifted-dome.csv")
	return mustMesh(t, corners, theOneFace(), normals, smooth)
}

// a ray straight down onto the dome's face, met where it lands.
func landed(t *testing.T, made *Mesh, x, z float64) (Met, maths.Vec) {
	t.Helper()
	start := maths.V(x, picked(t, "lifted-start-height"), z)
	ray := maths.NewRay(start, maths.V(nought, -unity, nought))
	met, ok := made.Meet(ray, nought, picked(t, "lifted-reach"))
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
	inX, inZ := picked(t, "lifted-inside-x"), picked(t, "lifted-inside-z")
	met, point := landed(t, smooth, inX, inZ)
	lifted := smooth.Lifted(met, point)
	if !(lifted.Y > picked(t, "lifted-least")) {
		t.Errorf("inside the face, lifted to %v from %v", lifted, point)
	}
	if moved := lifted.Sub(point).Len(); moved > picked(t, "lifted-most") {
		t.Errorf("lifted %g, further than the face is wide allows", moved)
	}
	nudge := picked(t, "lifted-corner-nudge")
	corner, at := landed(t, smooth, nudge, nudge)
	moved := smooth.Lifted(corner, at).Sub(at).Len()
	if moved > picked(t, "lifted-corner-slack") {
		t.Errorf("at a corner, moved %g", moved)
	}
	flat := dome(t, false)
	met, point = landed(t, flat, inX, inZ)
	if got := flat.Lifted(met, point); got != point {
		t.Errorf("shaded flat, moved to %v from %v", got, point)
	}
}

// a mesh shaded smooth gives its corners' normals, made one long, in
// the order of its corners; shaded flat, none.
func TestCornerNormalsFollowTheCorners(t *testing.T) {
	smooth := dome(t, true)
	normals, ok := smooth.CornerNormals(nought)
	if !ok {
		t.Fatal("a smooth mesh gave no normals")
	}
	want := vectorsIn(t, "testdata/lifted-dome.csv")
	for k, normal := range normals {
		unit := want[k].Unit()
		if normal.Sub(unit).Len() > picked(t, "lifted-normal-slack") {
			t.Errorf("corner %d's normal is %v, not %v", k, normal, unit)
		}
	}
	if _, ok := dome(t, false).CornerNormals(nought); ok {
		t.Error("a mesh shaded flat gave normals")
	}
}

// a point above every corner's tangent plane is not pulled down: the
// lift only raises.
func TestALiftOnlyRaises(t *testing.T) {
	corners := vectorsIn(t, "testdata/lifted-triangle.csv")
	bowl := vectorsIn(t, "testdata/lifted-bowl.csv")
	made := mustMesh(t, corners, theOneFace(), bowl, true)
	inX, inZ := picked(t, "lifted-inside-x"), picked(t, "lifted-inside-z")
	met, point := landed(t, made, inX, inZ)
	got := made.Lifted(met, point)
	if !(math.Abs(got.Sub(point).Len()) <= picked(t, "lifted-bowl-slack")) {
		t.Errorf("in a bowl, moved to %v from %v", got, point)
	}
}
