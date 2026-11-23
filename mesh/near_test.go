package mesh

import (
	"math"
	"slices"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The csv files of this file's fixtures, each row named in its first
// column.
const (
	nearBackToBackFile = "testdata/near-back-to-back.csv"
	nearNormalsFile    = "testdata/near-normals.csv"
	nearWeightsFile    = "testdata/near-weights.csv"
	nearDistancesFile  = "testdata/near-distances.csv"
	nearNormalsAtFile  = "testdata/near-normals-at.csv"
)

// nearCubeFaces is how many faces the cube has.
const nearCubeFaces = 12 // defined: six sides, each of two triangles

// nearVectorAt is the vector in the columns x, y and z of a csv row, each
// column's name beginning with the prefix.
func nearVectorAt(test testing.TB, row map[string]string,
	prefix string) maths.Vec {
	test.Helper()
	return maths.V(float(test, row, prefix+"x"), float(test, row, prefix+"y"),
		float(test, row, prefix+"z"))
}

// facesInFile is the faces a csv file holds, a row each: the columns
// first, second and third.
func facesInFile(test testing.TB, file string) [][triangleCorners]int {
	test.Helper()
	var faces [][triangleCorners]int
	for _, row := range read(test, file) {
		var face [triangleCorners]int
		face[first] = whole(test, row, "first")
		face[second] = whole(test, row, "second")
		face[third] = whole(test, row, "third")
		faces = append(faces, face)
	}
	return faces
}

// nearWeights is the weights of the row of the weights file with the
// name given.
func nearWeights(test testing.TB, name string) weights {
	test.Helper()
	for _, row := range read(test, nearWeightsFile) {
		if row["name"] == name {
			return weights{float(test, row, "first"),
				float(test, row, "second"), float(test, row, "third")}
		}
	}
	test.Fatalf("%s: no row named %s", nearWeightsFile, name)
	return weights{}
}

// a face is met strictly between near and far: a face five along is
// not met with near at five, nor with far at five, and is just past it.
func TestAFaceIsMetStrictlyBetweenNearAndFar(t *testing.T) {
	height := picked(t, "near-met-height")
	made := mustMesh(t, aSquare(height), theOneFace(), nil, false)
	offset := picked(t, "near-ray-offset")
	start := maths.V(offset, offset, nought)
	ray := maths.NewRay(start, maths.V(nought, nought, unity))
	if _, ok := made.Hit(ray, height, picked(t, "near-met-reach")); ok {
		t.Error("met at near")
	}
	if _, ok := made.Hit(ray, nought, height); ok {
		t.Error("met at far")
	}
	past := math.Nextafter(height, picked(t, "near-met-past"))
	if along, ok := made.Hit(ray, nought, past); !ok || along != height {
		t.Errorf("just past far: %v at %v", ok, along)
	}
}

// aSquare is the corners of a unit square standing at a height on z.
func aSquare(height float64) []maths.Vec {
	return []maths.Vec{maths.V(nought, nought, height),
		maths.V(unity, nought, height), maths.V(unity, unity, height),
		maths.V(nought, unity, height)}
}

// two faces back to back are met at the same distance, and the first
// of them is the one met; it faces its own way.
func TestOfTwoAsNearTheFirstIsMet(t *testing.T) {
	height := picked(t, "near-back-height")
	faces := facesInFile(t, nearBackToBackFile)
	made := mustMesh(t, aSquare(height), faces, nil, false)
	rayX, rayY := picked(t, "near-back-ray-x"), picked(t, "near-back-ray-y")
	up := maths.V(nought, nought, unity)
	ray := maths.NewRay(maths.V(rayX, rayY, nought), up)
	met, ok := made.Meet(ray, nought, math.Inf(plusInfinity))
	if !ok || met.Face() != nought || made.NormalAt(met) != up {
		t.Errorf("met %v face %d facing %v", ok, met.Face(),
			made.NormalAt(met))
	}
}

// smooth normals blend by where a face is met: at a corner, that
// corner's normal; at the middle, the mean of the three made one long,
// one long.
func TestSmoothNormalsBlendWhereMet(t *testing.T) {
	normals := vectorsIn(t, nearNormalsFile)
	made := mustMesh(t, aSquare(nought), theOneFace(), normals, true)
	corner := made.facing(nought, nearWeights(t, "corner"))
	middle := made.facing(nought, nearWeights(t, "middle"))
	expected := normals[first].Unit()
	if corner.Sub(expected).Len() > picked(t, "near-corner-tolerance") {
		t.Errorf("at a corner it faces %v", corner)
	}
	want := normals[first].Unit().Add(normals[second].Unit()).
		Add(normals[third]).Unit()
	if middle.Sub(want).Len() > picked(t, "near-middle-tolerance") {
		t.Errorf("at the middle it faces %v, not %v", middle, want)
	}
}

// a point's distance is to the nearest face, less than zero inside a closed
// mesh: the cube's middle is half in, a point two along x is one and a half
// out, a point beyond a corner is that far from it.
func TestDistanceIsToTheNearestFace(t *testing.T) {
	made := cube(t)
	tolerance := picked(t, "near-distance-tolerance")
	for _, row := range read(t, nearDistancesFile) {
		point := nearVectorAt(t, row, "")
		want := float(t, row, "want")
		if got := made.Distance(point); !(math.Abs(got-want) <= tolerance) {
			t.Errorf("at %v the distance is %v, not %v", point, got, want)
		}
	}
	open := mustMesh(t, cubeCorners(), cubeFaces()[:pair], nil, false)
	if open.inside(maths.V(nought, nought, nought)) {
		t.Error("an open mesh has an inside")
	}
}

// a mesh gives back the corners and faces it was made of, as copies
// that change nothing in it when changed.
func TestAMeshGivesBackWhatItWasMadeOf(t *testing.T) {
	made := cube(t)
	points, faces := made.Points(), made.Faces()
	if !slices.Equal(points, cubeCorners()) ||
		!slices.Equal(faces, cubeFaces()) || made.FaceCount() != nearCubeFaces {
		t.Fatalf("it gave back %v and %v", points, faces)
	}
	scribble := picked(t, "near-scribble")
	var scribbled [triangleCorners]int
	for place := range scribbled {
		scribbled[place] = unity
	}
	points[nought], faces[nought] = maths.V(scribble, scribble, scribble),
		scribbled
	if moved, _, _ := made.Corners(nought); moved != cubeCorners()[nought] {
		t.Errorf("changing the copies moved its first corner to %v", moved)
	}
}

// a point on the cube faces out of the side it is on, by the face
// nearest it.
func TestThePointsNormalIsItsSides(t *testing.T) {
	made := cube(t)
	tolerance := picked(t, "near-normal-tolerance")
	for _, row := range read(t, nearNormalsAtFile) {
		point := nearVectorAt(t, row, "")
		want := nearVectorAt(t, row, "want-")
		if got := made.Normal(point); got.Sub(want).Len() > tolerance {
			t.Errorf("at %v the cube faces %v", point, got)
		}
	}
}
