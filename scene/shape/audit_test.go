package shape

import (
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The files of rows the audit tests read.
const (
	auditCornersFile = "testdata/audit-corners.csv"
	auditFacesFile   = "testdata/audit-faces.csv"
	auditThinFile    = "testdata/audit-thin-heights.csv"
	auditLegsFile    = "testdata/audit-legs.csv"
)

// auditNeedle is a face with its two long legs, one from the zero to each.
type auditNeedle struct {
	first, second maths.Vec
}

// auditCorner is the corner (x, y, z) of a unit solid, scaled by size.
func auditCorner(size, x, y, z float64) maths.Vec {
	return maths.V(x, y, z).Scale(size)
}

// three corners in a line but for rounding are no face: their cross is
// about a part in 1e16 of their edges, and the mesh is refused, by name,
// rather than given a face whose normal is any way at all.
func TestCornersInALineButForRoundingAreNoFace(t *testing.T) {
	corners := triangleMeshCorners(t, auditCornersFile, "line")
	one := triangleFaces(t, auditFacesFile, "one")
	_, err := NewMesh(corners, one, nil, false)
	if err == nil || !strings.Contains(err.Error(), "no area") {
		t.Errorf("a face in a line but for rounding: %v", err)
	}
	slim := []maths.Vec{maths.V(nought, nought, nought),
		maths.V(unity, nought, nought),
		maths.V(half, picked(t, "audit-slim-height"), nought)}
	if _, err := NewMesh(slim, one, nil, false); err != nil {
		t.Errorf("a slim face with area was refused: %v", err)
	}
}

// a corner further out than any scene reaches is refused, by name, so the
// tree's sums cannot overflow and index past its bins.
func TestACornerPastAnySceneIsRefused(t *testing.T) {
	out := picked(t, "audit-far-corner")
	far := []maths.Vec{maths.V(nought, nought, nought),
		maths.V(out, nought, nought), maths.V(nought, out, nought)}
	one := triangleFaces(t, auditFacesFile, "one")
	_, err := NewMesh(far, one, nil, false)
	if err == nil || !strings.Contains(err.Error(), "too far") {
		t.Errorf("a corner at 1e300: %v", err)
	}
}

// the cutoff is a sine of 1e-12, chosen: a face of a sine under half
// that is refused, though its corners are exact; one of 1.8e-12 is a face.
func TestTheThinnestFaceIsASineOf1e12(t *testing.T) {
	one := triangleFaces(t, auditFacesFile, "one")
	for _, row := range triangleRows(t, auditThinFile) {
		height := triangleCell(t, row, "height")
		accepted, err := strconv.ParseBool(row["accepted"])
		if err != nil {
			t.Fatalf("accepted of %q: %v", row["name"], err)
		}
		thin := []maths.Vec{maths.V(nought, nought, nought),
			maths.V(unity, nought, nought), maths.V(half, height, nought)}
		_, err = NewMesh(thin, one, nil, false)
		if (err == nil) != accepted {
			t.Errorf("a face %v thin: %v", height, err)
		}
	}
}

// a face is measured at its own size: from legs of 1e-150 to 1e149, the
// right triangle is a face, and a closed mesh wound inside out at 1e100
// is told so, not refused for its area.
func TestAFaceIsMeasuredAtItsOwnSize(t *testing.T) {
	one := triangleFaces(t, auditFacesFile, "one")
	for _, row := range triangleRows(t, auditLegsFile) {
		leg := triangleCell(t, row, "leg")
		right := []maths.Vec{maths.V(nought, nought, nought),
			maths.V(leg, nought, nought), maths.V(nought, leg, nought)}
		if _, err := NewMesh(right, one, nil, false); err != nil {
			t.Errorf("legs of %v: %v", leg, err)
		}
	}
	size := picked(t, "audit-tetrahedron-size")
	tet := []maths.Vec{auditCorner(size, nought, nought, nought),
		auditCorner(size, unity, nought, nought),
		auditCorner(size, nought, unity, nought),
		auditCorner(size, nought, nought, unity)}
	inside := triangleFaces(t, auditFacesFile, "tetrahedron")
	_, err := NewMesh(tet, inside, nil, false)
	if err == nil || strings.Contains(err.Error(), "no area") {
		t.Errorf("an inside-out tetrahedron at 1e100: %v", err)
	}
}

// auditEdgeCorners is a corner at farthest on each axis, whichever way it
// lies, in a face with the zero and a unit that is not along it: each is
// taken.
func auditEdgeCorners(t *testing.T, one []triangleCorners) {
	t.Helper()
	for _, corner := range []maths.Vec{maths.V(farthest, nought, nought),
		maths.V(nought, farthest, nought),
		maths.V(nought, nought, farthest)} {
		across := corner.Cross(maths.V(unity, unity, unity)).Unit()
		in := []maths.Vec{maths.V(nought, nought, nought), corner, across}
		if _, err := NewMesh(in, one, nil, false); err != nil {
			t.Errorf("a corner at %v: %v", corner, err)
		}
	}
}

// auditNeedles are faces of one long leg and one short, whichever way
// they lie: each is a face, measured at the long leg's size.
func auditNeedles(t *testing.T, one []triangleCorners) {
	t.Helper()
	long := picked(t, "audit-needle-long")
	short := picked(t, "audit-needle-short")
	for _, legs := range []auditNeedle{
		{maths.V(long, nought, nought), maths.V(nought, short, nought)},
		{maths.V(nought, nought, long), maths.V(short, nought, nought)},
		{maths.V(nought, nought, short), maths.V(nought, long, nought)},
	} {
		needle := []maths.Vec{maths.V(nought, nought, nought), legs.first,
			legs.second}
		if _, err := NewMesh(needle, one, nil, false); err != nil {
			t.Errorf("a needle %v, %v: %v", legs.first, legs.second, err)
		}
	}
}

// a corner at farthest on any axis is in; past it, out. a face that is a
// needle, one leg 1e149 and the other 1e-10 at right angles, whichever
// way it lies, is a face: measured at its long leg's size, nothing
// overflows. and a face that names a vertex past the last is refused.
func TestTheEdgesOfWhatAMeshTakes(t *testing.T) {
	one := triangleFaces(t, auditFacesFile, "one")
	auditEdgeCorners(t, one)
	out := picked(t, "audit-past-corner")
	past := []maths.Vec{maths.V(nought, nought, nought),
		maths.V(out, nought, nought), maths.V(nought, unity, nought)}
	if _, err := NewMesh(past, one, nil, false); err == nil ||
		!strings.Contains(err.Error(), "too far") {
		t.Errorf("a corner at 1.5e150: %v", err)
	}
	auditNeedles(t, one)
	// thin, its long edge the second: a sine of 2e-12, a face
	thin := []maths.Vec{maths.V(nought, nought, nought),
		maths.V(half, picked(t, "audit-thin-height"), nought),
		maths.V(unity, nought, nought)}
	if _, err := NewMesh(thin, one, nil, false); err != nil {
		t.Errorf("a thin face, long edge second: %v", err)
	}
	three := []maths.Vec{maths.V(nought, nought, nought),
		maths.V(unity, nought, nought), maths.V(nought, unity, nought)}
	named := triangleFaces(t, auditFacesFile, "past")
	if _, err := NewMesh(three, named, nil, false); err == nil {
		t.Error("a face naming vertex 3 of 3 was taken")
	}
}
