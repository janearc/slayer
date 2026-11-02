package mesh

import (
	"errors"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// a closed piece's volume, six times, is the same from any corner of
// it: the unit cube is six, and the skew tetrahedron with corners at
// the origin and 4, 3 and 2 along the axes is 4 times 3 times 2.
func TestAClosedPiecesVolumeIsSixTimesTheSpaceItHolds(t *testing.T) {
	cubed := cube(t)
	every := make([]int, cubed.FaceCount())
	for i := range every {
		every[i] = i
	}
	if got := volume(cubed.points, cubed.faces, every); got != 6 {
		t.Errorf("the cube holds %v", got)
	}
	skew := regular(t, []maths.Vec{maths.V(0, 0, 0), maths.V(4, 0, 0),
		maths.V(0, 3, 0), maths.V(0, 0, 2)},
		[][3]int{{0, 2, 1}, {0, 1, 3}, {0, 3, 2}, {1, 2, 3}})
	if got := volume(skew.points, skew.faces, []int{0, 1, 2, 3}); got != 24 {
		t.Errorf("the tetrahedron holds %v, not 24", got)
	}
	if got := volume(skew.points, skew.faces, []int{3, 2, 1, 0}); got != 24 {
		t.Errorf("the same faces the other way round hold %v", got)
	}
}

// the error for a closed piece inside out names the first face of that
// piece: a cube turned out and one turned in beside it, whose faces
// are 12 to 23.
func TestAnInsideOutPieceIsNamedByItsFirstFace(t *testing.T) {
	corners := append(cubeCorners(), cubeCorners()...)
	for i := 8; i < 16; i++ {
		corners[i] = corners[i].Add(maths.V(5, 0, 0))
	}
	faces := cubeFaces()
	for _, each := range cubeFaces() {
		faces = append(faces, [3]int{each[0] + 8, each[2] + 8, each[1] + 8})
	}
	_, err := New(corners, faces, nil, false)
	if !errors.Is(err, ErrMesh) ||
		!strings.Contains(err.Error(), "face 12 is inside out") {
		t.Errorf("the error is %v", err)
	}
	if _, err := New(corners, faces[:12], nil, false); err != nil {
		t.Errorf("the cube turned out alone: %v", err)
	}
}
