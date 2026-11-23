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
var corner = []maths.Vec{maths.V(nought, nought, nought),
	maths.V(unity, nought, nought), maths.V(nought, unity, nought),
	maths.V(nought, nought, unity)}

// cornerFaces are the four faces of the unit corner, each turning
// counterclockwise seen from outside.
var cornerFaces = []meshFace{
	{meshFirst, meshThird, meshSecond}, {meshFirst, meshSecond, meshFourth},
	{meshFirst, meshFourth, meshThird}, {meshSecond, meshThird, meshFourth},
}

// windFloor is the one face of the corner that lies on the floor.
func windFloor() []meshFace {
	return []meshFace{{meshFirst, meshSecond, meshThird}}
}

// a normal given at any length is kept one unit long, as Normals says.
func TestGivenNormalsAreMadeOneUnitLong(t *testing.T) {
	up := maths.V(nought, nought, pair)
	built, err := NewMesh(corner[:meshCorners], windFloor(),
		[]maths.Vec{up, up, up}, true)
	if err != nil {
		t.Fatal(err)
	}
	for place, normal := range built.Normals {
		if normal != maths.V(nought, nought, unity) {
			t.Errorf("normal %d is %v", place, normal)
		}
	}
}

// a corner at minus infinity is not a point, and is told so.
func TestACornerAtMinusInfinityIsNoPoint(t *testing.T) {
	vertices := []maths.Vec{maths.V(math.Inf(-unity), nought, nought),
		corner[meshSecond], corner[meshThird]}
	_, err := NewMesh(vertices, windFloor(), nil, false)
	if err == nil || !strings.Contains(err.Error(), "not a finite point") {
		t.Errorf("a corner at minus infinity: %v", err)
	}
}

// a triangle and its back, two faces over one another, is closed and encloses
// nothing: its volume is exactly zero, and zero is not inside out. every
// product here is exact, so the zero is too.
func TestATwoSidedTriangleIsClosedAndNotInsideOut(t *testing.T) {
	faces := []meshFace{{meshFirst, meshSecond, meshThird},
		{meshFirst, meshThird, meshSecond}}
	built, err := NewMesh(corner[:meshCorners], faces, nil, false)
	if err != nil || !built.Closed() {
		t.Fatalf("a two-sided triangle: %v", err)
	}
}

// a small corner, an eighth of the unit one, is wound outward, though its six
// volumes come to an eighth: any volume greater than zero is outward.
func TestASmallClosedPieceIsOutward(t *testing.T) {
	small := make([]maths.Vec, len(corner))
	for place, vertex := range corner {
		small[place] = vertex.Scale(half)
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
	reach := picked(t, "wind-far-reach")
	far := maths.V(reach, reach, reach)
	vertices := append([]maths.Vec(nil), corner...)
	for _, vertex := range corner {
		vertices = append(vertices, vertex.Add(far))
	}
	faces := []meshFace{{meshFirst, meshThird, meshSecond},
		{meshSecond, meshFourth, meshFirst},
		{meshThird, meshFirst, meshFourth},
		{meshFourth, meshSecond, meshThird}}
	near := len(corner)
	for _, face := range cornerFaces {
		faces = append(faces, meshFace{face[meshFirst] + near,
			face[meshSecond] + near, face[meshThird] + near})
	}
	built, err := NewMesh(vertices, faces, nil, false)
	if err != nil || !built.Closed() {
		t.Fatalf("two outward corners: %v", err)
	}
}

// where the corners' normals cancel, smooth has no way to face, and the
// face's own way is the answer: the middle of an edge whose two ends
// face opposite ways.
func TestWhereTheNormalsCancelTheFaceIsFlat(t *testing.T) {
	up := maths.V(nought, nought, unity)
	down := maths.V(nought, nought, -unity)
	vertices := []maths.Vec{corner[meshFirst], maths.V(pair, nought, nought),
		maths.V(nought, pair, nought)}
	built, err := NewMesh(vertices, windFloor(), []maths.Vec{up, down, up},
		true)
	if err != nil {
		t.Fatal(err)
	}
	if got := built.Normal(maths.V(unity, nought, nought)); got != up {
		t.Errorf("at the edge's middle the normal is %v", got)
	}
}

// an open mesh has no inside, whichever way it is wound, from below it,
// where Inside's ray passes up through it.
func TestAnOpenMeshHasNoInsideWhereItsRayCrosses(t *testing.T) {
	reach := picked(t, "wind-flat-reach")
	flat := []maths.Vec{maths.V(-reach, -reach, nought),
		maths.V(reach, -reach, nought), maths.V(nought, reach, nought)}
	for _, face := range []meshFace{{meshFirst, meshSecond, meshThird},
		{meshFirst, meshThird, meshSecond}} {
		built, err := NewMesh(flat, []meshFace{face}, nil, false)
		if err != nil {
			t.Fatal(err)
		}
		if built.Inside(maths.V(nought, nought, -unity)) {
			t.Errorf("wound %v, a point below it is inside", face)
		}
	}
}

// a small piece far out is weighed from its own middle, so its volume is
// not lost to rounding: a corner a thousandth across at a million out.
// from a point as far again, each face's cone is near 1e18 and the piece
// near 1e-9, past what a float carries.
func TestASmallPieceFarOutIsWeighedFromItsMiddle(t *testing.T) {
	reach := picked(t, "wind-far-out")
	out := maths.V(reach, reach, reach)
	scale := picked(t, "wind-small-scale")
	small := make([]maths.Vec, len(corner))
	for place, vertex := range corner {
		small[place] = vertex.Scale(scale).Add(out)
	}
	if _, err := NewMesh(small, cornerFaces, nil, false); err != nil {
		t.Error(err)
	}
}
