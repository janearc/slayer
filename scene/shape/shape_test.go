package shape

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// shapeVectorFile is the table of points and rays the tests of this file
// use. A row called name has its place in x, y, z and, for a ray, its
// direction in dirx, diry, dirz.
const shapeVectorFile = "testdata/shape-vectors.csv"

// shapeVectors are the points and rays of one csv table, asked for by name.
type shapeVectors struct {
	test  *testing.T
	file  string
	table sheet
}

// shapeVectorsIn reads the csv table of vectors in file.
func shapeVectorsIn(t *testing.T, file string) shapeVectors {
	t.Helper()
	return shapeVectors{test: t, file: file, table: readSheet(t, file)}
}

// point is the place of the row called name.
func (vectors shapeVectors) point(name string) maths.Vec {
	vectors.test.Helper()
	row := vectors.table.row(vectors.test, vectors.file, name)
	return maths.V(cell(vectors.test, row, "x"),
		cell(vectors.test, row, "y"), cell(vectors.test, row, "z"))
}

// ray is the ray of the row called name: from its place, along its
// direction.
func (vectors shapeVectors) ray(name string) maths.Ray {
	vectors.test.Helper()
	row := vectors.table.row(vectors.test, vectors.file, name)
	direction := maths.V(cell(vectors.test, row, "dirx"),
		cell(vectors.test, row, "diry"), cell(vectors.test, row, "dirz"))
	return maths.Ray{At: vectors.point(name), Dir: direction}
}

// box is the box whose corners are the rows name-least and name-most.
func (vectors shapeVectors) box(name string) maths.Box {
	vectors.test.Helper()
	return maths.Box{Min: vectors.point(name + "-least"),
		Max: vectors.point(name + "-most")}
}

// the exact answer and the marched answer have to agree, or one of them is
// lying about the shape.
func TestSphereHitAgreesWithItsDistance(t *testing.T) {
	vectors := shapeVectorsIn(t, shapeVectorFile)
	ball := Sphere{Radius: picked(t, "shape-ball-radius")}
	ray := vectors.ray("ball-from-above")
	tolerance := picked(t, "shape-tolerance")
	want := picked(t, "shape-near-side-distance")
	got, ok := ball.Hit(ray, nought, math.Inf(unity))
	if !ok || math.Abs(got-want) > tolerance {
		t.Fatalf("hit at %v, %v; want %v", got, ok, want)
	}
	if gap := ball.Distance(ray.Point(got)); math.Abs(gap) > tolerance {
		t.Fatalf("the point it hit is %v from the surface", gap)
	}
}

// TestARayFromInsideMeetsTheFarSide says that a ray from the centre of a
// unit sphere meets it one out, when the near end of its range is a hair
// past nought.
func TestARayFromInsideMeetsTheFarSide(t *testing.T) {
	vectors := shapeVectorsIn(t, shapeVectorFile)
	ball := Sphere{Radius: unity}
	nearest := picked(t, "shape-nearest")
	ray := vectors.ray("from-inside")
	got, ok := ball.Hit(ray, nearest, math.Inf(unity))
	if !ok || math.Abs(got-unity) > picked(t, "shape-tolerance") {
		t.Fatalf("hit at %v, %v; want %v", got, ok, unity)
	}
}

// TestAMissIsAMiss says that a ray that passes over a unit sphere does not
// hit it.
func TestAMissIsAMiss(t *testing.T) {
	vectors := shapeVectorsIn(t, shapeVectorFile)
	ball := Sphere{Radius: unity}
	over := vectors.ray("miss-above")
	if _, ok := ball.Hit(over, nought, math.Inf(unity)); ok {
		t.Fatal("a ray passing four units above the sphere hit it")
	}
}

// TestThePlaneFacesUpAndHasNoEdge says that a ray from far out to the side
// meets the plane at its height, and that the plane's box does not miss it.
func TestThePlaneFacesUpAndHasNoEdge(t *testing.T) {
	vectors := shapeVectorsIn(t, shapeVectorFile)
	ground := Plane{}
	ray := vectors.ray("plane-from-far")
	want := picked(t, "shape-plane-hit-distance")
	got, ok := ground.Hit(ray, nought, math.Inf(unity))
	if !ok || got != want {
		t.Fatalf("hit at %v, %v; want %v", got, ok, want)
	}
	if _, _, ok := ground.Bounds().Hit(ray, nought, math.Inf(unity)); !ok {
		t.Fatal("the plane's box missed a ray that meets the plane")
	}
}

// TestTheMarchedNormalMatchesTheExactOne says that the normal measured from
// the distance of a shape that hides its own is the exact one.
func TestTheMarchedNormalMatchesTheExactOne(t *testing.T) {
	vectors := shapeVectorsIn(t, shapeVectorFile)
	ball := Sphere{Radius: unity}
	point := vectors.point("diagonal").Unit()
	marched := Normal(struct{ Shape }{ball}, point) // hides normaler
	tolerance := picked(t, "shape-marched-tolerance")
	if !marched.Near(ball.Normal(point), tolerance) {
		t.Fatalf("marched %v, exact %v", marched, ball.Normal(point))
	}
}

// A sphere spans a ray from where it enters to where it leaves, and a
// ray that misses spans nothing.
func TestSphereSpan(t *testing.T) {
	vectors := shapeVectorsIn(t, shapeVectorFile)
	ball := Sphere{Radius: picked(t, "shape-ball-radius")}
	tolerance := picked(t, "shape-tolerance")
	wantEnter := picked(t, "shape-near-side-distance")
	wantLeave := picked(t, "shape-far-side-distance")
	ray := vectors.ray("ball-from-above")
	enter, leave, ok := ball.Span(ray, nought, math.Inf(unity))
	if !ok || math.Abs(enter-wantEnter) > tolerance ||
		math.Abs(leave-wantLeave) > tolerance {
		t.Fatalf("spans %v to %v, %v; want %v to %v", enter, leave, ok,
			wantEnter, wantLeave)
	}
	over := vectors.ray("miss-above")
	if _, _, ok := ball.Span(over, nought, math.Inf(unity)); ok {
		t.Fatal("a miss spans the sphere")
	}
}
