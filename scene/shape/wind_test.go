package shape

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// the unit corner of a cube, A at the middle and B, C, D a unit along x,
// y and z, wound to turn counterclockwise seen from outside: worked by
// hand, (C-A)x(B-A) is -z for the floor, and so on round.
var corner = []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0),
	maths.V(0, 0, 1)}

var cornerFaces = [][3]int{{0, 2, 1}, {0, 1, 3}, {0, 3, 2}, {1, 2, 3}}

// a normal given at any length is kept one unit long, as Normals says.
func TestGivenNormalsAreMadeOneUnitLong(t *testing.T) {
	up := maths.V(0, 0, 2)
	m, err := NewMesh([]maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0),
		maths.V(0, 1, 0)}, [][3]int{{0, 1, 2}}, []maths.Vec{up, up, up}, true)
	if err != nil {
		t.Fatal(err)
	}
	for i, n := range m.Normals {
		if n != maths.V(0, 0, 1) {
			t.Errorf("normal %d is %v", i, n)
		}
	}
}

// a corner at minus infinity is not a point, and is told so.
func TestACornerAtMinusInfinityIsNoPoint(t *testing.T) {
	_, err := NewMesh([]maths.Vec{maths.V(math.Inf(-1), 0, 0), maths.V(1, 0, 0),
		maths.V(0, 1, 0)}, [][3]int{{0, 1, 2}}, nil, false)
	if err == nil || !strings.Contains(err.Error(), "not a finite point") {
		t.Errorf("a corner at minus infinity: %v", err)
	}
}

// a triangle and its back, two faces over one another, is closed and encloses
// nothing: its volume is exactly zero, and zero is not inside out. every
// product here is exact, so the zero is too.
func TestATwoSidedTriangleIsClosedAndNotInsideOut(t *testing.T) {
	m, err := NewMesh(corner[:3], [][3]int{{0, 1, 2}, {0, 2, 1}}, nil, false)
	if err != nil || !m.Closed() {
		t.Fatalf("a two-sided triangle: %v", err)
	}
}

// a small corner, an eighth of the unit one, is wound outward, though its six
// volumes come to an eighth: any volume greater than zero is outward.
func TestASmallClosedPieceIsOutward(t *testing.T) {
	small := make([]maths.Vec, len(corner))
	for i, v := range corner {
		small[i] = v.Scale(0.5)
	}
	if _, err := NewMesh(small, cornerFaces, nil, false); err != nil {
		t.Error(err)
	}
}

// each closed piece is weighed whole. here the near corner's faces are
// turned so that two of its corners, C and D, meet the others only at a
// face's middle place; a far corner, a hundred out each way, puts the
// mesh's middle at fifty. seen from there, the near corner's faces on C
// and D alone come to minus a hundred, by hand: but the piece is whole,
// and outward, and so is the far one.
func TestAPieceIsWeighedWhole(t *testing.T) {
	far := maths.V(100, 100, 100)
	vertices := append([]maths.Vec(nil), corner...)
	for _, v := range corner {
		vertices = append(vertices, v.Add(far))
	}
	faces := [][3]int{{0, 2, 1}, {1, 3, 0}, {2, 0, 3}, {3, 1, 2}}
	for _, f := range cornerFaces {
		faces = append(faces, [3]int{f[0] + 4, f[1] + 4, f[2] + 4})
	}
	m, err := NewMesh(vertices, faces, nil, false)
	if err != nil || !m.Closed() {
		t.Fatalf("two outward corners: %v", err)
	}
}

// where the corners' normals cancel, smooth has no way to face, and the
// face's own way is the answer: the middle of an edge whose two ends
// face opposite ways.
func TestWhereTheNormalsCancelTheFaceIsFlat(t *testing.T) {
	up, down := maths.V(0, 0, 1), maths.V(0, 0, -1)
	m, err := NewMesh([]maths.Vec{maths.V(0, 0, 0), maths.V(2, 0, 0),
		maths.V(0, 2, 0)}, [][3]int{{0, 1, 2}}, []maths.Vec{up, down, up}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Normal(maths.V(1, 0, 0)); got != up {
		t.Errorf("at the edge's middle the normal is %v", got)
	}
}

// an open mesh has no inside, whichever way it is wound, from below it,
// where Inside's ray passes up through it.
func TestAnOpenMeshHasNoInsideWhereItsRayCrosses(t *testing.T) {
	flat := []maths.Vec{maths.V(-10, -10, 0), maths.V(10, -10, 0),
		maths.V(0, 10, 0)}
	for _, f := range [][3]int{{0, 1, 2}, {0, 2, 1}} {
		m, err := NewMesh(flat, [][3]int{f}, nil, false)
		if err != nil {
			t.Fatal(err)
		}
		if m.Inside(maths.V(0, 0, -1)) {
			t.Errorf("wound %v, a point below it is inside", f)
		}
	}
}

// a small piece far out is weighed from its own middle, so its volume is
// not lost to rounding: a corner a thousandth across at a million out.
// from a point as far again, each face's cone is near 1e18 and the piece
// near 1e-9, past what a float carries.
func TestASmallPieceFarOutIsWeighedFromItsMiddle(t *testing.T) {
	out := maths.V(1e6, 1e6, 1e6)
	small := make([]maths.Vec, len(corner))
	for i, v := range corner {
		small[i] = v.Scale(1e-3).Add(out)
	}
	if _, err := NewMesh(small, cornerFaces, nil, false); err != nil {
		t.Error(err)
	}
}
