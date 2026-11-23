package mesh

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The csv files of this file's fixtures, each row named in its first
// column.
const (
	solidCornersFile   = "testdata/solid-corners.csv"
	solidFacesFile     = "testdata/solid-faces.csv"
	solidHeightsFile   = "testdata/solid-heights.csv"
	solidPointsFile    = "testdata/solid-points.csv"
	solidDistancesFile = "testdata/solid-distances.csv"
)

// The solids of the csv files, by the name in their solid column.
const (
	solidTriangle    = "triangle"
	solidTetrahedron = "tetrahedron"
	solidOctahedron  = "octahedron"
	solidSkew        = "skew"
)

const (
	// rectangleQuarters is how many of a corner's quarters a rectangle fills.
	rectangleQuarters = 4 // defined: a rectangle has four corners
	// wholeSphere is the solid angle of the whole sphere.
	wholeSphere = 4 * math.Pi // defined: the area of a unit sphere
	// cubeFaceAngle is what one face of a cube fills from its middle.
	cubeFaceAngle = math.Pi / 3 // defined: a twelfth of the sphere
	// openFaces is how many of a cube's twelve faces an open one keeps.
	openFaces = 10 // defined: twelve faces, less the two taken away
)

// solidRowsOf is the rows of a csv file whose solid column is the name.
func solidRowsOf(test testing.TB, file, solid string) rows {
	test.Helper()
	var chosen rows
	for _, row := range read(test, file) {
		if row["solid"] == solid {
			chosen = append(chosen, row)
		}
	}
	return chosen
}

// solidPointOf is the point in the x, y and z columns of a row.
func solidPointOf(test testing.TB, row map[string]string) maths.Vec {
	test.Helper()
	along := float(test, row, "x")
	up := float(test, row, "y")
	deep := float(test, row, "z")
	return maths.V(along, up, deep)
}

// solidCorners is the corners of a solid, in the order of the file.
func solidCorners(test testing.TB, solid string) []maths.Vec {
	test.Helper()
	var corners []maths.Vec
	for _, row := range solidRowsOf(test, solidCornersFile, solid) {
		corners = append(corners, solidPointOf(test, row))
	}
	return corners
}

// solidFaces is the faces of a solid, in the order of the file.
func solidFaces(test testing.TB, solid string) [][triangleCorners]int {
	test.Helper()
	var faces [][triangleCorners]int
	for _, row := range solidRowsOf(test, solidFacesFile, solid) {
		var face [triangleCorners]int
		face[first] = whole(test, row, "first")
		face[second] = whole(test, row, "second")
		face[third] = whole(test, row, "third")
		faces = append(faces, face)
	}
	return faces
}

// solidMesh is the named solid with its corners and faces turned out.
func solidMesh(test *testing.T, solid string) *Mesh {
	test.Helper()
	return regular(test, solidCorners(test, solid), solidFaces(test, solid))
}

// rectangle is the solid angle a rectangle fills seen from a height
// straight above its middle: four of the corner's quarters, each the
// angle of a rectangle a half as long and a half as wide, seen from
// above a corner of it, which is atan(ab / (h * sqrt(a2 + b2 + h2))).
func rectangle(long, wide, height float64) float64 {
	halfLong, halfWide := long/pair, wide/pair
	return rectangleQuarters * math.Atan(halfLong*halfWide/
		(height*math.Sqrt(halfLong*halfLong+halfWide*halfWide+height*height)))
}

// a triangle that is half a rectangle, cut along the diagonal through
// its middle, fills half the rectangle's angle seen from above the
// middle; seen from behind it is that angle, and from in front, less
// that: a point is inside the turn of the face it is behind.
func TestAFaceFillsTheAngleOfHalfItsRectangle(t *testing.T) {
	made := mustMesh(t, solidCorners(t, solidTriangle),
		solidFaces(t, solidTriangle), nil, false)
	long := picked(t, "solid-rectangle-long")
	wide := picked(t, "solid-rectangle-wide")
	tolerance := picked(t, "solid-tolerance")
	for _, each := range read(t, solidHeightsFile) {
		height := float(t, each, "height")
		sign := float(t, each, "sign")
		want := sign * rectangle(long, wide, math.Abs(height)) / pair
		got := made.solidAngle(nought, maths.V(nought, nought, height))
		if !(math.Abs(got-want) <= tolerance) {
			t.Errorf("at %v the face fills %v, not %v", height, got, want)
		}
	}
}

// regular is a solid with its corners and faces turned out.
func regular(t *testing.T, corners []maths.Vec,
	faces [][triangleCorners]int) *Mesh {
	outward(corners, faces)
	return mustMesh(t, corners, faces, nil, false)
}

// tetrahedron is the regular one, its corners at alternate corners of
// the cube from minus one to one.
func tetrahedron(t *testing.T) *Mesh {
	return solidMesh(t, solidTetrahedron)
}

// octahedron is the regular one, its corners a unit out on each axis.
func octahedron(t *testing.T) *Mesh {
	return solidMesh(t, solidOctahedron)
}

// regularShare is a regular solid and the share of the whole sphere each
// of its faces fills seen from its middle.
type regularShare struct {
	name  string
	made  *Mesh
	share float64
}

// regularShares is the tetrahedron and the octahedron with their shares:
// a quarter of four pi for each of the tetrahedron's four faces, an
// eighth for each of the octahedron's eight.
func regularShares(t *testing.T) []regularShare {
	return []regularShare{
		{solidTetrahedron, tetrahedron(t), math.Pi},
		{solidOctahedron, octahedron(t), math.Pi / pair},
	}
}

// seen from its middle, each of a regular solid's faces fills an equal
// share of the whole sphere's four pi: a quarter for the tetrahedron's
// four, an eighth for the octahedron's eight.
func TestRegularFacesFillEqualSharesFromTheMiddle(t *testing.T) {
	tolerance := picked(t, "solid-tolerance")
	middle := maths.V(nought, nought, nought)
	for _, each := range regularShares(t) {
		for i := range each.made.faces {
			got := each.made.solidAngle(i, middle)
			if !(math.Abs(got-each.share) <= tolerance) {
				t.Errorf("%s face %d fills %v, not %v", each.name, i, got,
					each.share)
			}
		}
	}
}

// the faces of a closed solid fill the whole sphere, four pi, from any
// point inside it, and nothing from any point outside, where the near
// faces' angles are taken from the far ones': whatever the shape and
// wherever the point, asymmetric as the tetrahedron with corners at
// the origin and 4, 3 and 2 out on the axes.
func TestTheFacesOfASolidFillTheSphereOnlyFromInside(t *testing.T) {
	tolerance := picked(t, "solid-tolerance")
	for _, each := range read(t, solidPointsFile) {
		made := solidMesh(t, each["solid"])
		point := solidPointOf(t, each)
		inside := whole(t, each, "inside") == unity
		want := float64(nought)
		if inside {
			want = wholeSphere
		}
		got := float64(nought)
		for i := range made.faces {
			got += made.solidAngle(i, point)
		}
		if !(math.Abs(got-want) <= tolerance) {
			t.Errorf("from %v the faces fill %v, not %v", point, got, want)
		}
		if made.inside(point) != inside {
			t.Errorf("%v is inside: %v", point, !inside)
		}
	}
}

// a solid with a face missing has no inside, though its other faces
// fill more than half the sphere from the middle: ten of the cube's
// twelve fill ten thirds of pi.
func TestAnOpenSolidHasNoInside(t *testing.T) {
	open := mustMesh(t, cubeCorners(), cubeFaces()[pair:], nil, false)
	middle := maths.V(nought, nought, nought)
	sum := float64(nought)
	for i := range open.faces {
		sum += open.solidAngle(i, middle)
	}
	gap := math.Abs(sum - openFaces*cubeFaceAngle)
	if !(gap <= picked(t, "solid-tolerance")) {
		t.Fatalf("ten faces fill %v", sum)
	}
	if open.inside(middle) {
		t.Error("an open mesh has an inside")
	}
}

// a point's distance less than zero is how deep it is: in the skew tetrahedron,
// the nearest side to a point is the one it is least far from, signed by which
// side of the surface it is on.
func TestDistanceIsBelowNoughtDeepInsideASolid(t *testing.T) {
	skew := solidMesh(t, solidSkew)
	tolerance := picked(t, "solid-tolerance")
	for _, each := range read(t, solidDistancesFile) {
		point := solidPointOf(t, each)
		want := float(t, each, "want")
		got := skew.Distance(point)
		if !(math.Abs(got-want) <= tolerance) {
			t.Errorf("at %v the distance is %v, not %v", point, got, want)
		}
	}
}
