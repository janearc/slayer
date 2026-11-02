package mesh

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// mustMesh is a mesh, or the test fails.
func mustMesh(t *testing.T, corners []maths.Vec, faces [][3]int,
	normals []maths.Vec, smooth bool) *Mesh {
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
	corners := make([]maths.Vec, 8)
	for i := range corners {
		corners[i] = maths.V(float64(i&1)-0.5, float64(i>>1&1)-0.5,
			float64(i>>2&1)-0.5)
	}
	return corners
}

// cubeFaces is the cube's twelve faces, each turning counterclockwise
// seen from outside, by -x, +x, -y, +y, -z and +z.
func cubeFaces() [][3]int {
	return [][3]int{{0, 4, 6}, {0, 6, 2}, {1, 3, 7}, {1, 7, 5},
		{0, 1, 5}, {0, 5, 4}, {2, 6, 7}, {2, 7, 3},
		{0, 2, 3}, {0, 3, 1}, {4, 5, 7}, {4, 7, 6}}
}

// cube is the cube as a mesh.
func cube(t *testing.T) *Mesh {
	return mustMesh(t, cubeCorners(), cubeFaces(), nil, false)
}

// icosahedronCorners is the twelve corners of an icosahedron on the
// unit sphere.
func icosahedronCorners() []maths.Vec {
	gold := (1 + math.Sqrt(5)) / 2
	raw := []maths.Vec{maths.V(-1, gold, 0), maths.V(1, gold, 0),
		maths.V(-1, -gold, 0), maths.V(1, -gold, 0),
		maths.V(0, -1, gold), maths.V(0, 1, gold),
		maths.V(0, -1, -gold), maths.V(0, 1, -gold),
		maths.V(gold, 0, -1), maths.V(gold, 0, 1),
		maths.V(-gold, 0, -1), maths.V(-gold, 0, 1)}
	for i, corner := range raw {
		raw[i] = corner.Unit()
	}
	return raw
}

// icosahedronFaces is its twenty faces; outward turns them out.
func icosahedronFaces() [][3]int {
	return [][3]int{{0, 11, 5}, {0, 5, 1}, {0, 1, 7}, {0, 7, 10},
		{0, 10, 11}, {1, 5, 9}, {5, 11, 4}, {11, 10, 2}, {10, 7, 6},
		{7, 1, 8}, {3, 9, 4}, {3, 4, 2}, {3, 2, 6}, {3, 6, 8},
		{3, 8, 9}, {4, 9, 5}, {2, 4, 11}, {6, 2, 10}, {8, 6, 7},
		{9, 8, 1}}
}

// outward turns each face of a shape round its middle to face away
// from the middle.
func outward(corners []maths.Vec, faces [][3]int) {
	for i, at := range faces {
		one, two, three := corners[at[0]], corners[at[1]], corners[at[2]]
		turn := two.Sub(one).Cross(three.Sub(one))
		if turn.Dot(one.Add(two).Add(three)) < 0 {
			faces[i][1], faces[i][2] = at[2], at[1]
		}
	}
}

// halved is each face cut in four at its edges' middles, pushed out to
// the unit sphere, each middle shared by the two faces at its edge.
func halved(corners []maths.Vec, faces [][3]int) ([]maths.Vec,
	[][3]int) {
	middles := make(map[[2]int]int, len(faces)*3/2)
	middle := func(one, other int) int {
		key := [2]int{min(one, other), max(one, other)}
		if at, seen := middles[key]; seen {
			return at
		}
		corners = append(corners,
			corners[one].Add(corners[other]).Scale(0.5).Unit())
		middles[key] = len(corners) - 1
		return len(corners) - 1
	}
	var cut [][3]int
	for _, at := range faces {
		ab, bc, ca := middle(at[0], at[1]), middle(at[1], at[2]),
			middle(at[2], at[0])
		cut = append(cut, [3]int{at[0], ab, ca}, [3]int{at[1], bc, ab},
			[3]int{at[2], ca, bc}, [3]int{ab, bc, ca})
	}
	return corners, cut
}

// ball is an icosahedron halved so many times: a closed ball of faces
// round the unit sphere.
func ball(t *testing.T, times int) (*Mesh, []maths.Vec, [][3]int) {
	corners, faces := icosahedronCorners(), icosahedronFaces()
	outward(corners, faces)
	for range times {
		corners, faces = halved(corners, faces)
	}
	return mustMesh(t, corners, faces, nil, false), corners, faces
}
