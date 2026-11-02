package shape

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// three corners in a line but for rounding are no face: their cross is
// about a part in 1e16 of their edges, and the mesh is refused, by name,
// rather than given a face whose normal is any way at all.
func TestCornersInALineButForRoundingAreNoFace(t *testing.T) {
	corners := []maths.Vec{maths.V(0.1, 0.2, 0.3), maths.V(0.3, 0.6, 0.9),
		maths.V(0.7, 1.4, 2.1)}
	_, err := NewMesh(corners, [][3]int{{0, 1, 2}}, nil, false)
	if err == nil || !strings.Contains(err.Error(), "no area") {
		t.Errorf("a face in a line but for rounding: %v", err)
	}
	slim := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0),
		maths.V(0.5, 1e-9, 0)}
	if _, err := NewMesh(slim, [][3]int{{0, 1, 2}}, nil, false); err != nil {
		t.Errorf("a slim face with area was refused: %v", err)
	}
}

// a corner further out than any scene reaches is refused, by name, so the
// tree's sums cannot overflow and index past its bins.
func TestACornerPastAnySceneIsRefused(t *testing.T) {
	far := []maths.Vec{maths.V(0, 0, 0), maths.V(1e300, 0, 0),
		maths.V(0, 1e300, 0)}
	_, err := NewMesh(far, [][3]int{{0, 1, 2}}, nil, false)
	if err == nil || !strings.Contains(err.Error(), "too far") {
		t.Errorf("a corner at 1e300: %v", err)
	}
}

// the cutoff is a sine of 1e-12, chosen: a face of a sine under half
// that is refused, though its corners are exact; one of 1.8e-12 is a face.
func TestTheThinnestFaceIsASineOf1e12(t *testing.T) {
	for h, ok := range map[float64]bool{5e-13: false, 2e-12: true} {
		thin := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0),
			maths.V(0.5, h, 0)}
		_, err := NewMesh(thin, [][3]int{{0, 1, 2}}, nil, false)
		if (err == nil) != ok {
			t.Errorf("a face %v thin: %v", h, err)
		}
	}
}

// a face is measured at its own size: from legs of 1e-150 to 1e149, the
// right triangle is a face, and a closed mesh wound inside out at 1e100
// is told so, not refused for its area.
func TestAFaceIsMeasuredAtItsOwnSize(t *testing.T) {
	for _, leg := range []float64{1e-150, 1e-90, 1, 1e100, 1e140, 1e149} {
		right := []maths.Vec{maths.V(0, 0, 0), maths.V(leg, 0, 0),
			maths.V(0, leg, 0)}
		if _, err := NewMesh(right, [][3]int{{0, 1, 2}}, nil, false); err != nil {
			t.Errorf("legs of %v: %v", leg, err)
		}
	}
	at := func(x, y, z float64) maths.Vec { return maths.V(x, y, z).Scale(1e100) }
	tet := []maths.Vec{at(0, 0, 0), at(1, 0, 0), at(0, 1, 0), at(0, 0, 1)}
	inside := [][3]int{{0, 1, 2}, {0, 3, 1}, {0, 2, 3}, {1, 3, 2}}
	_, err := NewMesh(tet, inside, nil, false)
	if err == nil || strings.Contains(err.Error(), "no area") {
		t.Errorf("an inside-out tetrahedron at 1e100: %v", err)
	}
}

// a corner at farthest on any axis is in; past it, out. a face that is a
// needle, one leg 1e149 and the other 1e-10 at right angles, whichever
// way it lies, is a face: measured at its long leg's size, nothing
// overflows. and a face that names a vertex past the last is refused.
func TestTheEdgesOfWhatAMeshTakes(t *testing.T) {
	one := [][3]int{{0, 1, 2}}
	for _, corner := range []maths.Vec{maths.V(farthest, 0, 0),
		maths.V(0, farthest, 0), maths.V(0, 0, farthest)} {
		in := []maths.Vec{maths.V(0, 0, 0), corner, corner.Cross(maths.V(1, 1, 1)).Unit()}
		if _, err := NewMesh(in, one, nil, false); err != nil {
			t.Errorf("a corner at %v: %v", corner, err)
		}
	}
	past := []maths.Vec{maths.V(0, 0, 0), maths.V(1.5e150, 0, 0), maths.V(0, 1, 0)}
	if _, err := NewMesh(past, one, nil, false); err == nil ||
		!strings.Contains(err.Error(), "too far") {
		t.Errorf("a corner at 1.5e150: %v", err)
	}
	long, short := 1e149, 1e-10
	for _, legs := range [][2]maths.Vec{
		{maths.V(long, 0, 0), maths.V(0, short, 0)},
		{maths.V(0, 0, long), maths.V(short, 0, 0)},
		{maths.V(0, 0, short), maths.V(0, long, 0)},
	} {
		needle := []maths.Vec{maths.V(0, 0, 0), legs[0], legs[1]}
		if _, err := NewMesh(needle, one, nil, false); err != nil {
			t.Errorf("a needle %v, %v: %v", legs[0], legs[1], err)
		}
	}
	// thin, its long edge the second: a sine of 2e-12, a face
	thin := []maths.Vec{maths.V(0, 0, 0), maths.V(0.5, 1e-12, 0), maths.V(1, 0, 0)}
	if _, err := NewMesh(thin, one, nil, false); err != nil {
		t.Errorf("a thin face, long edge second: %v", err)
	}
	three := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}
	if _, err := NewMesh(three, [][3]int{{0, 1, 3}}, nil, false); err == nil {
		t.Error("a face naming vertex 3 of 3 was taken")
	}
}
