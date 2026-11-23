package mesh

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/janearc/slayer/maths"
)

// byValue is the check as it was first written: edges are pairs of
// points, matched by comparing the points. It stays here so that the
// check by places can be held to it.
func byValue(points []maths.Vec, faces []face) (bool, []piece, error) {
	runs := make(map[edge]int, triangleCorners*len(faces))
	for i := range faces {
		for _, side := range sides(points, faces[i]) {
			if other, seen := runs[side]; seen {
				return false, nil, refusal(points, faces, side, other, i)
			}
			runs[side] = i
		}
	}
	joins := newJoins(len(faces))
	open := make([]bool, len(faces))
	for i, at := range faces {
		for _, side := range sides(points, at) {
			other, shared := runs[side.back()]
			if !shared {
				open[i] = true
				continue
			}
			joins.join(i, other)
		}
	}
	which := make(map[int]int, unity)
	var made []piece
	for i := range faces {
		root := joins.root(i)
		at, seen := which[root]
		if !seen {
			at = len(made)
			which[root] = at
			made = append(made, piece{closed: true})
		}
		made[at].faces = append(made[at].faces, i)
		made[at].closed = made[at].closed && !open[i]
	}
	closed := true
	for _, each := range made {
		closed = closed && each.closed
		if each.closed && volume(points, faces, each.faces) < nought {
			return false, nil, insideOut(each.faces[first])
		}
	}
	return closed, made, nil
}

// narrowed is a body's faces as the mesh keeps them.
func narrowed(of body) []face {
	faces := make([]face, len(of.faces))
	for i, at := range of.faces {
		faces[i] = face{int32(at[first]), int32(at[second]),
			int32(at[third])}
	}
	return faces
}

// tetrahedronCorners are the origin and a corner along each axis, then
// the same four again, with a negative nought where a part is nought.
func tetrahedronCorners() []maths.Vec {
	negative := math.Copysign(nought, -unity)
	plain := []maths.Vec{maths.V(nought, nought, nought),
		maths.V(unity, nought, nought), maths.V(nought, unity, nought),
		maths.V(nought, nought, unity)}
	again := []maths.Vec{maths.V(negative, negative, negative),
		maths.V(unity, negative, negative),
		maths.V(negative, unity, negative),
		maths.V(negative, negative, unity)}
	return append(plain, again...)
}

// splitTetrahedron is a tetrahedron whose faces name its corners by
// both sets of places, so that no face meets another at the same place
// number, and some meet at a nought and a negative nought.
func splitTetrahedron() body {
	return body{corners: tetrahedronCorners(),
		faces: [][triangleCorners]int{{0, 2, 1}, {4, 5, 7}, {0, 7, 6},
			{1, 6, 7}}}
}

// plainTetrahedron is a tetrahedron by one set of places.
func plainTetrahedron() body {
	return body{corners: tetrahedronCorners()[:4],
		faces: [][triangleCorners]int{{0, 2, 1}, {0, 1, 3}, {0, 3, 2},
			{1, 2, 3}}}
}

// withoutALid is the cube without its last face.
func withoutALid() body {
	made := unitCube()
	made.faces = made.faces[:len(made.faces)-unity]
	return made
}

// twiceOver is the cube with its first face a second time.
func twiceOver() body {
	made := unitCube()
	made.faces = append(made.faces, made.faces[first])
	return made
}

// foldedOver is two faces that both run the edge from the first corner
// to the second.
func foldedOver() body {
	return body{corners: tetrahedronCorners()[:4],
		faces: [][triangleCorners]int{{0, 1, 2}, {0, 1, 3}}}
}

// byAnEdge is two cubes that touch along one edge.
func byAnEdge() body {
	corner := maths.V(unity, unity, nought)
	return together(unitCube(), cubeAt(corner))
}

// windCase is a mesh for the check, whether it is closed and how many
// pieces it has, or the reason it is refused, with no reason where it
// is not.
type windCase struct {
	made   body
	closed bool
	pieces int
	want   error
}

// windCases are the meshes for the check, by what each is.
func windCases() map[string]windCase {
	apart := cubeAt(maths.V(pair+unity, nought, nought))
	return map[string]windCase{
		"a cube":              {unitCube(), true, 1, nil},
		"two cubes apart":     {together(unitCube(), apart), true, 2, nil},
		"a cube and a sheet":  {together(withoutALid(), apart), false, 2, nil},
		"an open cube":        {withoutALid(), false, 1, nil},
		"a tetrahedron":       {plainTetrahedron(), true, 1, nil},
		"a split tetrahedron": {splitTetrahedron(), true, 1, nil},
		"a face twice":        {twiceOver(), false, 0, errCrowded},
		"two the same way":    {foldedOver(), false, 0, errSameWay},
		"cubes by an edge":    {byAnEdge(), false, 0, errCrowded},
		"a cavity": {together(unitCube(), turnedIn(apart)), false, 0,
			errInsideOut},
	}
}

// the check by places decides what the check by value decided, for each
// small mesh: closed or not, the same pieces in the same order, and the
// same error to the word.
func TestTheCheckByPlacesIsTheCheckByValue(t *testing.T) {
	for name, each := range windCases() {
		points, faces := each.made.corners, narrowed(each.made)
		wantClosed, wantPieces, wantErr := byValue(points, faces)
		closed, err := wound(points, faces)
		if closed != wantClosed || (err == nil) != (wantErr == nil) ||
			err != nil && err.Error() != wantErr.Error() {
			t.Errorf("%s: closed %v, %v; by value %v, %v", name, closed, err,
				wantClosed, wantErr)
		}
		if err != nil {
			continue
		}
		named := placesOf(points, faces)
		runs := runners(len(faces))
		for i := range faces {
			if err := running(runs, named, points, faces, i); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
		}
		if got := piecesOf(named, runs); !reflect.DeepEqual(got, wantPieces) {
			t.Errorf("%s: pieces %v, by value %v", name, got, wantPieces)
		}
	}
}

// each small mesh is closed, or refused, as it is expected to be, with
// the pieces it is expected to have.
func TestTheCheckDecidesEachSmallMesh(t *testing.T) {
	for name, each := range windCases() {
		closed, err := wound(each.made.corners, narrowed(each.made))
		if !errors.Is(err, each.want) && each.want != nil || err != nil &&
			each.want == nil {
			t.Errorf("%s: the error is %v, not %v", name, err, each.want)
			continue
		}
		if err == nil && closed != each.closed {
			t.Errorf("%s: closed is %v", name, closed)
		}
		if err == nil {
			faces := narrowed(each.made)
			named := placesOf(each.made.corners, faces)
			runs := runners(len(faces))
			for i := range faces {
				_ = running(runs, named, each.made.corners, faces, i)
			}
			if got := len(piecesOf(named, runs)); got != each.pieces {
				t.Errorf("%s: %d pieces, not %d", name, got, each.pieces)
			}
		}
	}
}

// a nought and a negative nought are one place, and two corners a
// rounding apart are two.
func TestPlacesAreByTheBitsExceptNought(t *testing.T) {
	negative := math.Copysign(nought, -unity)
	near := math.Nextafter(unity, pair)
	points := []maths.Vec{maths.V(nought, nought, nought),
		maths.V(negative, nought, negative), maths.V(unity, nought, nought),
		maths.V(near, nought, nought)}
	faces := []face{{0, 1, 2}, {1, 2, 3}}
	named := placesOf(points, faces)
	if named[first][first] != named[first][second] {
		t.Error("a nought and a negative nought are two places")
	}
	if named[first][third] == named[second][third] {
		t.Error("two corners a rounding apart are one place")
	}
}

// a point no face names is not given a place, however odd it is.
func TestAPointNoFaceNamesTakesNoPlace(t *testing.T) {
	points := []maths.Vec{maths.V(math.NaN(), nought, nought),
		maths.V(nought, nought, nought), maths.V(unity, nought, nought),
		maths.V(nought, unity, nought)}
	named := placesOf(points, []face{{1, 2, 3}})
	if named[first] != [triangleCorners]place{0, 1, 2} {
		t.Errorf("the places are %v", named[first])
	}
}
