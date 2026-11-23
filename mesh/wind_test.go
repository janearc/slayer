package mesh

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// everyFace is the places of faces nought up to so many, in order.
func everyFace(faces int) []int {
	every := make([]int, faces)
	for i := range every {
		every[i] = i
	}
	return every
}

// skewCorners are the corners of the skew tetrahedron: the origin, and one
// corner along each axis, as far as the picked reach along it.
func skewCorners(t *testing.T) []maths.Vec {
	return []maths.Vec{maths.V(nought, nought, nought),
		maths.V(picked(t, "wind-skew-x"), nought, nought),
		maths.V(nought, picked(t, "wind-skew-y"), nought),
		maths.V(nought, nought, picked(t, "wind-skew-z"))}
}

// skewFaces are the skew tetrahedron's four faces, each turning out, read
// from the table of them.
func skewFaces(t *testing.T) [][triangleCorners]int {
	var faces [][triangleCorners]int
	for _, row := range read(t, "testdata/wind-skew-faces.csv") {
		faces = append(faces, [triangleCorners]int{whole(t, row, "first"),
			whole(t, row, "second"), whole(t, row, "third")})
	}
	return faces
}

// a closed piece's volume, six times, is the same from any corner of
// it: the unit cube is six, and the skew tetrahedron is its three reaches
// multiplied together, from whichever of its faces is first.
func TestAClosedPiecesVolumeIsSixTimesTheSpaceItHolds(t *testing.T) {
	cubed := cube(t)
	every := everyFace(cubed.FaceCount())
	if got := volume(cubed.points, cubed.faces, every); got !=
		picked(t, "wind-cube-volume") {
		t.Errorf("the cube holds %v", got)
	}
	skew := regular(t, skewCorners(t), skewFaces(t))
	want := picked(t, "wind-skew-volume")
	forward := everyFace(skew.FaceCount())
	if got := volume(skew.points, skew.faces, forward); got != want {
		t.Errorf("the tetrahedron holds %v, not %v", got, want)
	}
	backward := everyFace(skew.FaceCount())
	for i := range backward {
		backward[i] = len(backward) - unity - i
	}
	if got := volume(skew.points, skew.faces, backward); got != want {
		t.Errorf("the same faces the other way round hold %v", got)
	}
}

// the error for a closed piece inside out names the first face of that
// piece: a cube turned out and one turned in beside it, whose faces
// are those after the first cube's.
func TestAnInsideOutPieceIsNamedByItsFirstFace(t *testing.T) {
	reach := picked(t, "wind-apart")
	apart := cubeAt(maths.V(reach, nought, nought))
	both := together(unitCube(), turnedIn(apart))
	_, err := both.made()
	afterTheCube := len(cubeFaces())
	want := fmt.Sprintf("face %d", afterTheCube)
	if !errors.Is(err, errInsideOut) || !strings.Contains(err.Error(), want) {
		t.Errorf("the error is %v", err)
	}
	if _, err := unitCube().made(); err != nil {
		t.Errorf("the cube turned out alone: %v", err)
	}
}
