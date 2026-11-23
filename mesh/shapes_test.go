package mesh

import (
	"math"
	"strconv"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The csv files the shapes are read from.
const (
	shapesCubeFacesFile       = "testdata/shapes-cube-faces.csv"
	shapesIcosahedronCorners  = "testdata/shapes-icosahedron-corners.csv"
	shapesIcosahedronFacesCSV = "testdata/shapes-icosahedron-faces.csv"
)

// cubeCornerCount is how many corners a cube has.
const cubeCornerCount = 8 // defined: a cube has eight corners

// goldenRadicand is the number under the root in the golden ratio.
const goldenRadicand = 5 // defined: the golden ratio is (1 + root of 5) / 2

// shapeRows is a csv file of a shape, read at once. A fixture that is
// called with no test to stop panics if its file cannot be read.
func shapeRows(file string) rows {
	got, err := loading{file}.load()
	if err != nil {
		panic(err)
	}
	return got
}

// shapeNumber is the number in a column of a row of a shape's file.
func shapeNumber(row map[string]string, column string) float64 {
	got, err := strconv.ParseFloat(row[column], floatBits)
	if err != nil {
		panic(err)
	}
	return got
}

// shapePlace is the whole number in a column of a row of a shape's
// file.
func shapePlace(row map[string]string, column string) int {
	return int(shapeNumber(row, column))
}

// shapeFaces is the faces a csv file holds, a row each: the corners
// first, second and third.
func shapeFaces(file string) [][triangleCorners]int {
	var faces [][triangleCorners]int
	for _, row := range shapeRows(file) {
		faces = append(faces, [triangleCorners]int{
			shapePlace(row, "first"), shapePlace(row, "second"),
			shapePlace(row, "third")})
	}
	return faces
}

// mustMesh is a mesh, or the test fails.
func mustMesh(t *testing.T, corners []maths.Vec,
	faces [][triangleCorners]int, normals []maths.Vec,
	smooth bool) *Mesh {
	t.Helper()
	made, err := New(corners, faces, normals, smooth)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// cubeCorners is a cube of side one about the origin: corner i stands
// at its bits, x the first, each less a half.
func cubeCorners() []maths.Vec {
	corners := make([]maths.Vec, cubeCornerCount)
	for i := range corners {
		corners[i] = maths.V(float64(i&unity)-boxHalf,
			float64(i>>unity&unity)-boxHalf,
			float64(i>>pair&unity)-boxHalf)
	}
	return corners
}

// cubeFaces is the cube's twelve faces, each turning counterclockwise
// seen from outside, by -x, +x, -y, +y, -z and +z.
func cubeFaces() [][triangleCorners]int {
	return shapeFaces(shapesCubeFacesFile)
}

// cube is the cube as a mesh.
func cube(t *testing.T) *Mesh {
	return mustMesh(t, cubeCorners(), cubeFaces(), nil, false)
}

// oneLong is the vector scaled to one unit long, with each square
// rounded before the squares are summed. Left to itself the compiler
// may fold one of the multiplies into an add, and which one it folds
// depends on what the sum is inlined into, so the last bit of a corner
// would differ from one chip to the next, and with it the tree the
// corners are sorted into and every count taken on that tree. A shape
// the walks are counted on is the same shape everywhere.
func oneLong(vector maths.Vec) maths.Vec {
	length := math.Sqrt(float64(vector.X*vector.X) +
		float64(vector.Y*vector.Y) + float64(vector.Z*vector.Z))
	return vector.Scale(unity / length)
}

// goldenRatio is the ratio of the long side of an icosahedron's
// rectangles to the short.
func goldenRatio() float64 {
	return (unity + math.Sqrt(goldenRadicand)) / pair
}

// icosahedronCorners is the twelve corners of an icosahedron on the
// unit sphere: in each row of its file, a sign for each part, and the
// part that is the golden ratio times its sign instead.
func icosahedronCorners() []maths.Vec {
	gold := goldenRatio()
	var corners []maths.Vec
	for _, row := range shapeRows(shapesIcosahedronCorners) {
		part := [axes]float64{shapeNumber(row, "x"),
			shapeNumber(row, "y"), shapeNumber(row, "z")}
		long := shapePlace(row, "long")
		part[long] *= gold
		corner := oneLong(vectorOf(part))
		corners = append(corners, corner)
	}
	return corners
}

// icosahedronFaces is its twenty faces; outward turns them out.
func icosahedronFaces() [][triangleCorners]int {
	return shapeFaces(shapesIcosahedronFacesCSV)
}

// outward turns each face of a shape round its middle to face away
// from the middle.
func outward(corners []maths.Vec, faces [][triangleCorners]int) {
	for i, at := range faces {
		one, two, three := corners[at[first]], corners[at[second]],
			corners[at[third]]
		turn := two.Sub(one).Cross(three.Sub(one))
		if turn.Dot(one.Add(two).Add(three)) < nought {
			faces[i][second], faces[i][third] = at[third], at[second]
		}
	}
}

// halver cuts the edges of faces at their middles: it holds the
// corners so far and the middle already made for each edge.
type halver struct {
	corners []maths.Vec
	middles map[[pair]int]int
}

// middle is the corner at the middle of an edge, pushed out to the
// unit sphere and made once for the two faces that share the edge.
func (cutter *halver) middle(one, other int) int {
	key := [pair]int{min(one, other), max(one, other)}
	if at, seen := cutter.middles[key]; seen {
		return at
	}
	between := cutter.corners[one].Add(cutter.corners[other])
	middle := oneLong(between.Scale(boxHalf))
	cutter.corners = append(cutter.corners, middle)
	cutter.middles[key] = len(cutter.corners) - unity
	return len(cutter.corners) - unity
}

// halved is each face cut in four at its edges' middles, pushed out to
// the unit sphere, each middle shared by the two faces at its edge.
func halved(corners []maths.Vec, faces [][triangleCorners]int) ([]maths.Vec,
	[][triangleCorners]int) {
	cutter := &halver{corners: corners, middles: make(map[[pair]int]int,
		len(faces)*triangleCorners/pair)}
	var cut [][triangleCorners]int
	for _, at := range faces {
		ab, bc, ca := cutter.middle(at[first], at[second]),
			cutter.middle(at[second], at[third]),
			cutter.middle(at[third], at[first])
		cut = append(cut, [triangleCorners]int{at[first], ab, ca},
			[triangleCorners]int{at[second], bc, ab},
			[triangleCorners]int{at[third], ca, bc},
			[triangleCorners]int{ab, bc, ca})
	}
	return cutter.corners, cut
}

// ball is an icosahedron halved so many times: a closed ball of faces
// round the unit sphere.
func ball(t *testing.T, times int) (*Mesh, []maths.Vec,
	[][triangleCorners]int) {
	corners, faces := icosahedronCorners(), icosahedronFaces()
	outward(corners, faces)
	for range times {
		corners, faces = halved(corners, faces)
	}
	return mustMesh(t, corners, faces, nil, false), corners, faces
}
