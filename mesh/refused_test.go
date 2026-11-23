package mesh

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// refused is what New is given that it must refuse, and the reason it
// must give.
type refused struct {
	body    body
	normals []maths.Vec
	smooth  bool
	want    error
}

// reasons are every reason New refuses a mesh for, each its own error.
func reasons() []error {
	return []error{errNoFaces, errNormalCount, errNoCorner, errNotPoint,
		errNormalNowhere, errNoArea, errSameWay, errCrowded, errInsideOut,
		errTooMany, errRestCount, errRestPoint}
}

// refusedCases is each thing New refuses, by what is wrong with it.
func refusedCases() map[string]refused {
	return map[string]refused{
		"no faces":             withoutFaces(),
		"too few normals":      withFewNormals(),
		"a corner not there":   namingACornerPastTheEnd(),
		"a corner before it":   namingACornerBeforeTheStart(),
		"a corner not a point": withACornerNowhere(),
		"a normal nowhere":     withANormalNowhere(),
		"no area":              withAFaceInALine(),
		"two ways the same":    runningAnEdgeTheSameWay(),
		"a shared edge":        sharingAnEdgeBetweenSolids(),
		"inside out":           turnedInside(),
	}
}

// withoutFaces is corners and no faces.
func withoutFaces() refused {
	return refused{body: body{corners: cubeCorners()}, want: errNoFaces}
}

// withFewNormals is a normal at only some corners.
func withFewNormals() refused {
	corners := cubeCorners()
	return refused{body: body{corners, cubeFaces()},
		normals: corners[:triangleCorners], smooth: true,
		want: errNormalCount}
}

// namingACornerPastTheEnd is a face that names the corner after the last.
func namingACornerPastTheEnd() refused {
	corners := cubeCorners()
	faces := append(cubeFaces(),
		[triangleCorners]int{nought, unity, len(corners)})
	return refused{body: body{corners, faces}, want: errNoCorner}
}

// namingACornerBeforeTheStart is a face that names the corner before the
// first.
func namingACornerBeforeTheStart() refused {
	faces := append(cubeFaces(), [triangleCorners]int{nought, unity, -unity})
	return refused{body: body{cubeCorners(), faces}, want: errNoCorner}
}

// withACornerNowhere is a corner a face uses that is not a number.
func withACornerNowhere() refused {
	corners := cubeCorners()
	corners[third] = maths.V(math.NaN(), nought, nought)
	return refused{body: body{corners, cubeFaces()}, want: errNotPoint}
}

// withANormalNowhere is a smooth mesh with a normal of no length.
func withANormalNowhere() refused {
	normals := cubeNormals()
	normals[second] = maths.V(nought, nought, nought)
	return refused{body: body{cubeCorners(), cubeFaces()},
		normals: normals, smooth: true, want: errNormalNowhere}
}

// withAFaceInALine is a face whose three corners are one.
func withAFaceInALine() refused {
	faces := [][triangleCorners]int{{nought, nought, nought}}
	return refused{body: body{cubeCorners(), faces}, want: errNoArea}
}

// runningAnEdgeTheSameWay is one face given twice, so each of its edges is
// run twice the same way, and by no more than two faces.
func runningAnEdgeTheSameWay() refused {
	one := cubeFaces()[nought]
	return refused{body: body{cubeCorners(),
		[][triangleCorners]int{one, one}}, want: errSameWay}
}

// sharingAnEdgeBetweenSolids is two cubes, one stood on the other, so that
// each edge of the rim they meet at has four faces on it, two each way,
// every face of both cubes wound right.
func sharingAnEdgeBetweenSolids() refused {
	above := cubeAt(maths.V(nought, unity, nought))
	return refused{body: together(unitCube(), above), want: errCrowded}
}

// turnedInside is a cube with every face turned in.
func turnedInside() refused {
	return refused{body: turnedIn(unitCube()), want: errInsideOut}
}

// Each thing that cannot be a mesh is refused with its own reason, which
// wraps ErrMesh, and with none of the other reasons: a refusal for one
// reason is never taken for another.
func TestEachRefusalIsToldApartByItsOwnReason(t *testing.T) {
	for name, each := range refusedCases() {
		_, err := New(each.body.corners, each.body.faces, each.normals,
			each.smooth)
		if !errors.Is(err, each.want) {
			t.Errorf("%s: %v, not %v", name, err, each.want)
		}
		if !errors.Is(err, ErrMesh) {
			t.Errorf("%s: %v is not a refusal of a mesh", name, err)
		}
		for _, other := range reasons() {
			if other != each.want && errors.Is(err, other) {
				t.Errorf("%s: %v is also %v", name, err, other)
			}
		}
	}
}

// Every reason a mesh is refused for wraps ErrMesh, so a caller that
// asks only whether a mesh was refused need not know the reasons.
func TestEveryReasonWrapsErrMesh(t *testing.T) {
	for _, each := range reasons() {
		if !errors.Is(each, ErrMesh) {
			t.Errorf("%v does not wrap ErrMesh", each)
		}
	}
}

// Two solids that share an edge are refused for the edge, which more than
// two faces run, and not for winding: stacked, side by side along a wall,
// and meeting along one edge only, each with four faces on an edge. Two
// that meet at no edge, at a corner only or not at all, are two pieces of
// one mesh and are accepted, closed.
func TestSolidsThatShareAnEdgeAreCrowdedAndOthersAreNot(t *testing.T) {
	origin := maths.V(nought, nought, nought)
	for name, shift := range map[string]maths.Vec{
		"stacked":       maths.V(nought, unity, nought),
		"side by side":  maths.V(unity, nought, nought),
		"along an edge": maths.V(unity, nought, unity),
	} {
		_, err := together(cubeAt(origin), cubeAt(shift)).made()
		if !errors.Is(err, errCrowded) || errors.Is(err, errSameWay) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, shift := range map[string]maths.Vec{
		"at a corner": maths.V(unity, unity, unity),
		"apart":       maths.V(pair, pair, pair),
	} {
		made, err := together(cubeAt(origin), cubeAt(shift)).made()
		if err != nil || !made.Closed() {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// New refuses more corners than a face can name, and accepts as many as
// one can.
func TestMoreCornersThanAFaceCanNameAreRefused(t *testing.T) {
	if err := fits(int(mostCorners)); err != nil {
		t.Errorf("the most a face can name: %v", err)
	}
	past := int64(mostCorners) + int64(unity)
	err := fits(int(past))
	if !errors.Is(err, errTooMany) || !errors.Is(err, ErrMesh) {
		t.Errorf("one past it: %v", err)
	}
}

// NewWithRestPoints refuses a count of rest points that is not the
// corners', and a rest point that is not a point, and whatever New
// refuses; a mesh with a rest point to each corner is made.
func TestRestPointsAreOneACornerAndEachAPoint(t *testing.T) {
	corners := cubeCorners()
	_, err := NewWithRestPoints(corners, corners[:triangleCorners],
		cubeFaces(), nil, false)
	if !errors.Is(err, errRestCount) {
		t.Errorf("too few rest points: %v", err)
	}
	for _, bad := range []float64{math.NaN(), math.Inf(plusInfinity),
		-math.Inf(plusInfinity)} {
		rest := cubeCorners()
		rest[second] = maths.V(nought, bad, nought)
		_, err = NewWithRestPoints(corners, rest, cubeFaces(), nil, false)
		if !errors.Is(err, errRestPoint) {
			t.Errorf("a rest point of %v: %v", bad, err)
		}
	}
	_, err = NewWithRestPoints(corners, cubeCorners(), nil, nil, false)
	if !errors.Is(err, errNoFaces) {
		t.Errorf("no faces: %v", err)
	}
	made, err := NewWithRestPoints(corners, cubeCorners(), cubeFaces(), nil,
		false)
	if err != nil || made == nil {
		t.Errorf("rest points to each corner: %v", err)
	}
}

// farPoints are points the mesh has no nearest face to, by what is wrong
// with them: a part that is not a number, endless parts, and a part so
// large that its square is endless.
func farPoints(test testing.TB) map[string]maths.Vec {
	far := picked(test, "overflowing-reach")
	return map[string]maths.Vec{
		"not a number":   maths.V(math.NaN(), nought, nought),
		"plus infinity":  maths.V(nought, math.Inf(plusInfinity), nought),
		"minus infinity": maths.V(nought, nought, -math.Inf(plusInfinity)),
		"overflowing":    maths.V(far, nought, nought),
	}
}

// A point the mesh has no nearest face to is faced the way the mesh's
// first face is, flat, for a mesh shaded flat and for one shaded smooth,
// and does not stop the film. A point that has a nearest face is faced as
// it was: the cube's side nearest it.
func TestNormalOfAPointWithNoNearestFaceIsTheFirstFacesWay(t *testing.T) {
	flat := cube(t)
	smooth := mustMesh(t, cubeCorners(), cubeFaces(), cubeNormals(), true)
	want := maths.V(-unity, nought, nought)
	for name, point := range farPoints(t) {
		for shading, made := range map[string]*Mesh{"flat": flat,
			"smooth": smooth} {
			if got := made.Normal(point); got != want {
				t.Errorf("%s, %s: faces %v, not %v", name, shading, got, want)
			}
		}
	}
	if got := flat.Normal(maths.V(unity, nought, nought)); got !=
		maths.V(unity, nought, nought) {
		t.Errorf("a point beside the +x side faces %v", got)
	}
}

// A point the mesh has no nearest face to is endlessly far from it and is
// not inside it, for a closed mesh; a point inside the cube is inside it,
// and a point just beside it is not.
func TestAPointWithNoNearestFaceIsEndlesslyFarAndNotInside(t *testing.T) {
	made := cube(t)
	for name, point := range farPoints(t) {
		if got := made.Distance(point); !math.IsInf(got, plusInfinity) {
			t.Errorf("%s: distance %v", name, got)
		}
		if made.inside(point) {
			t.Errorf("%s: inside", name)
		}
	}
	middle := maths.V(nought, nought, nought)
	if !made.inside(middle) ||
		math.Abs(made.Distance(middle)+picked(t, "cube-half")) >
			picked(t, "slack") {
		t.Errorf("the middle of the cube: %v", made.Distance(middle))
	}
	if made.inside(maths.V(unity, nought, nought)) {
		t.Error("a point beside the cube is inside it")
	}
}

// A mesh is closed only if every piece of it is: a closed cube beside an
// open sheet is not, and the distance to the cube is not below nought
// anywhere, its middle included.
func TestAMeshIsClosedOnlyIfEveryPieceIs(t *testing.T) {
	sheet := body{corners: cubeAt(maths.V(pair, nought, nought)).corners,
		faces: cubeFaces()[:pair]}
	made, err := together(unitCube(), sheet).made()
	if err != nil {
		t.Fatal(err)
	}
	if made.Closed() {
		t.Error("a cube and an open sheet are closed")
	}
	if got := made.Distance(middleOfCube()); !(got > nought) {
		t.Errorf("the middle of the cube, beside an open sheet: %v", got)
	}
}

// Storage counts the tree only once it is grown: no nodes before the
// first nearest point, and more bytes after.
func TestStorageCountsTheTreeOnceItIsGrown(t *testing.T) {
	made := cube(t)
	before := made.Storage()
	made.Distance(maths.V(nought, nought, nought))
	after := made.Storage()
	if before.Nodes != nought || after.Nodes == nought ||
		!(after.Bytes > before.Bytes) {
		t.Errorf("nodes %d then %d, bytes %d then %d", before.Nodes,
			after.Nodes, before.Bytes, after.Bytes)
	}
}
