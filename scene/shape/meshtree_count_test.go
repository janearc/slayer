//go:build shapecount

package shape

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// The tests here count faces a walk asks about, which only the shapecount
// build does: run them with -tags shapecount.

// towerFaces is how many faces the tower the walks are counted on has.
const towerFaces = 20000

// plates is count big triangles one above another, a unit apart from z = 0
// up, each 20 across and 20 up, so that a ray along z through the middle
// meets every one.
func plates(test testing.TB, count int) *Mesh {
	reach := meshtreeMorePicked(test, "meshtree-more-plate-reach")
	var vertices []maths.Vec
	var faces [][cornersOfAFace]int
	for k := range count {
		z := float64(k)
		left := maths.V(-reach, -reach, z)
		right := maths.V(reach, -reach, z)
		top := maths.V(nought, reach, z)
		vertices = append(vertices, left, right, top)
		faces = append(faces, meshtreeMoreCorners(cornersOfAFace*k))
	}
	return made(test, vertices, faces, nil, false)
}

// askedOfAtMost says that a walk of a tree finds a face by asking about no
// more than one face in this many of a tower's. It was chosen: a tree that
// looks in the nearer box first asks about a few leaves, and one that does
// not asks about nearly all.
const askedOfAtMost = 100

// askedWhile is how many faces walks ask about while f runs.
func askedWhile(f func()) int {
	facesAskedAbout = 0
	f()
	return facesAskedAbout
}

// The tree finds a ray's face among many quickly: a ray down a tower of
// twenty thousand faces asks about a hundredth of them or fewer. A tower is
// the worst case for a tree that does not look in the nearer box first.
// Rays go down it and up it, so that either child may be the nearer.
//
// Worked by hand: from 5 below the first plate, at z = 0, the ray meets
// it at 5; from 20004, 5 above the last, at z = 19999, also at 5.
func TestTheNearerBoxIsLookedInFirstForARay(t *testing.T) {
	m := plates(t, towerFaces)
	for _, c := range []struct {
		why string
		r   maths.Ray
		t   float64
	}{
		{"from below", maths.Ray{At: maths.V(0, 0, -5),
			Dir: maths.V(0, 0, 1)}, 5},
		{"from above", maths.Ray{At: maths.V(0, 0, towerFaces+4),
			Dir: maths.V(0, 0, -1)}, 5},
	} {
		var got float64
		var ok bool
		asked := askedWhile(func() { got, _, ok = m.first(c.r, 0, forever) })
		if !ok || got != c.t {
			t.Fatalf("%s: met at %v, %v; want %v", c.why, got, ok, c.t)
		}
		if asked > towerFaces/askedOfAtMost {
			t.Errorf("%s: the walk asked about %d of %d faces, want at most %d",
				c.why, asked, towerFaces, towerFaces/askedOfAtMost)
		}
	}
}

// A scan of every face asks about every face, so the count in the tests
// here can tell a walk that looks in the nearer box first from one that does
// not: the scan is over the limit they hold a walk to.
func TestAScanOfEveryFaceAsksAboutEveryFace(t *testing.T) {
	m := plates(t, towerFaces)
	ray := maths.Ray{At: maths.V(0, 0, -5), Dir: maths.V(0, 0, 1)}
	asked := askedWhile(func() { every(m, ray, 0, forever) })
	if asked != towerFaces {
		t.Fatalf("a scan asked about %d faces of %d", asked, towerFaces)
	}
	if asked <= towerFaces/askedOfAtMost {
		t.Errorf("a scan of %d faces is within the limit of %d", asked,
			towerFaces/askedOfAtMost)
	}
}

// The same for the nearest face to a point, from below the tower and from
// above it. Worked by hand: 5 from the first plate or the last, so 25
// squared.
func TestTheNearerBoxIsLookedInFirstForAPoint(t *testing.T) {
	m := plates(t, towerFaces)
	for _, c := range []struct {
		why string
		p   maths.Vec
		d2  float64
	}{
		{"from below", maths.V(0, 0, -5), 25},
		{"from above", maths.V(0, 0, towerFaces+4), 25},
	} {
		var d2 float64
		asked := askedWhile(func() { _, _, d2 = m.nearest(c.p) })
		if d2 != c.d2 {
			t.Fatalf("%s: nearest is %v away squared, want %v", c.why, d2, c.d2)
		}
		if asked > towerFaces/askedOfAtMost {
			t.Errorf("%s: the walk asked about %d of %d faces, want at most %d",
				c.why, asked, towerFaces, towerFaces/askedOfAtMost)
		}
	}
}

// A ray that misses the whole mesh is turned away at the root by winding,
// as it is by a hit: the walk asks about no face.
func TestAWindingRayThatMissesIsTurnedAwayAtTheRoot(t *testing.T) {
	m := plates(t, towerFaces)
	ray := maths.Ray{At: maths.V(100, 100, -5), Dir: maths.V(0, 0, 1)}
	var wound int
	asked := askedWhile(func() { wound = m.winding(ray) })
	if wound != 0 {
		t.Fatalf("a ray that misses winds %d", wound)
	}
	if asked != 0 {
		t.Errorf("a ray that misses the root asked about %d faces", asked)
	}
}

// A ray through the middle of the tower is not turned away at the root, so
// the walk asks about faces: the count above is not zero for every ray.
func TestAWindingRayThatHitsAsksAboutFaces(t *testing.T) {
	m := plates(t, towerFaces)
	ray := maths.Ray{At: maths.V(0, 0, -5), Dir: maths.V(0, 0, 1)}
	if asked := askedWhile(func() { m.winding(ray) }); asked == 0 {
		t.Error("a ray through the tower asked about no face")
	}
}
