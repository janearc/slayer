package mesh

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/janearc/slayer/maths"
)

// barycentric is the weights of a point over a triangle's corners in
// the plane z=0, by the areas of the three triangles it makes with
// them: each corner's weight is the area opposite it, over the whole.
func barycentric(corners [3]maths.Vec, x, y float64) weights {
	area := func(one, two maths.Vec, x, y float64) float64 {
		return (two.X-one.X)*(y-one.Y) - (two.Y-one.Y)*(x-one.X)
	}
	whole := area(corners[0], corners[1], corners[2].X, corners[2].Y)
	return weights{
		area(corners[1], corners[2], x, y) / whole,
		area(corners[2], corners[0], x, y) / whole,
		area(corners[0], corners[1], x, y) / whole}
}

// a ray down onto a face, or up from under it, meets it where it is
// over it, in its middle, on a side exactly, and on a corner exactly,
// with the corners given in each turn and the face given both ways
// round, and not where it is beside it, and the weights are the point's
// own over the corners. the face is (0,0), (2,0), (0,2) in the plane z=0.
func TestARayMeetsAFaceWhereItIsOverIt(t *testing.T) {
	corners := [3]maths.Vec{maths.V(0, 0, 0), maths.V(2, 0, 0),
		maths.V(0, 2, 0)}
	points := []struct {
		x, y float64
		met  bool
	}{
		{0.5, 0.5, true}, {0.25, 1.5, true}, {1, 0, true}, {0, 1, true},
		{1, 1, true}, {0, 0, true}, {2, 0, true}, {0, 2, true},
		{-0.5, 0.5, false}, {1.5, 1.5, false}, {3, 0, false},
		{1, -0.25, false}, {1.0000001, 1.0000001, false}, {0.5, 2.5, false},
	}
	for turn := range 3 {
		for _, backward := range []bool{false, true} {
			var turned [3]maths.Vec
			for i := range turned {
				turned[i] = corners[(i+turn)%3]
			}
			face := [3]int{0, 1, 2}
			if backward {
				face = [3]int{0, 2, 1}
			}
			made := mustMesh(t, turned[:], [][3]int{face}, nil, false)
			for _, from := range []float64{1, -1} {
				for _, each := range points {
					checkMeets(t, made, from, each.x, each.y, each.met,
						barycentric([3]maths.Vec{turned[face[0]],
							turned[face[1]], turned[face[2]]}, each.x,
							each.y), face)
				}
			}
		}
	}
}

// checkMeets holds one ray to meeting a face or not, one away, with the
// weights over the corners in the order of the mesh's face.
func checkMeets(t *testing.T, made *Mesh, height, x, y float64, met bool,
	over weights, face [3]int) {
	t.Helper()
	ray := maths.Ray{At: maths.V(x, y, height), Dir: maths.V(0, 0, -height)}
	look := aimed(ray)
	along, got, ok := look.meets(made, 0, 0, math.Inf(1))
	if ok != met {
		t.Errorf("from %v at (%v, %v) face %v: met %v", height, x, y, face,
			ok)
		return
	}
	if ok && (along != 1/math.Abs(height) || !sameWeights(got, over)) {
		t.Errorf("from %v at (%v, %v): along %v weights %v, not %v",
			height, x, y, along, got, over)
	}
}

// a ray that misses the mesh's box does no work; one that meets it does
// little: a thousand rays from outside toward points within a unit
// and a fifth of the middle of a ball of 1280 faces and 1469 nodes, and
// a thousand from inside it, in any direction. measured, the most for
// one ray is 42 nodes and 21 faces from outside, and 40 and 8 from
// inside, and the thousand cost 15269 nodes and 3582 faces from outside
// and 20048 and 3370 from inside. the bounds are those and a twentieth.
func TestTheWalkOfARayEntersFewNodesAndTriesFewFaces(t *testing.T) {
	made, _, _ := ball(t, 3)
	random := rand.New(rand.NewPCG(11, 12))
	unit := func(scale float64) maths.Vec {
		return maths.V(random.NormFloat64(), random.NormFloat64(),
			random.NormFloat64()).Unit().Scale(scale)
	}
	var most, tried, total, all int
	count := func(ray maths.Ray) {
		nodes, faces := work(func() { made.Meet(ray, 0, math.Inf(1)) })
		most, tried = max(most, nodes), max(tried, faces)
		total += nodes
		all += faces
	}
	for range 1000 {
		from := unit(4)
		count(maths.Ray{At: from, Dir: unit(math.Sqrt(random.Float64()) *
			1.2).Sub(from).Unit()})
	}
	if most > 44 || tried > 22 || total > 16033 || all > 3761 {
		t.Errorf("from outside: %d, %d for one, %d, %d in all", most,
			tried, total, all)
	}
	most, tried, total, all = 0, 0, 0, 0
	for range 1000 {
		count(maths.Ray{At: unit(math.Sqrt(random.Float64()) * 0.9),
			Dir: unit(1)})
	}
	if most > 42 || tried > 9 || total > 21050 || all > 3539 {
		t.Errorf("from inside: %d, %d for one, %d, %d in all", most, tried,
			total, all)
	}
}

// a ray that does not reach the mesh's box, because it is beside it,
// behind it, or stopped short, enters no node and tries no face.
func TestARayThatMissesTheBoxDoesNoWork(t *testing.T) {
	made, _, _ := ball(t, 3)
	hit := maths.Ray{At: maths.V(0, 0, 4), Dir: maths.V(0, 0, -1)}
	nodes, faces := work(func() { made.Meet(hit, 0, math.Inf(1)) })
	if nodes == 0 || faces == 0 {
		t.Errorf("a ray that hits entered %d nodes and tried %d faces",
			nodes, faces)
	}
	for name, each := range map[string]struct {
		ray       maths.Ray
		near, far float64
	}{
		"beside": {maths.Ray{At: maths.V(2, 2, 4), Dir: maths.V(0, 0, -1)},
			0, math.Inf(1)},
		"behind": {maths.Ray{At: maths.V(0, 0, 4), Dir: maths.V(0, 0, 1)},
			0, math.Inf(1)},
		"short": {maths.Ray{At: maths.V(0, 0, 4), Dir: maths.V(0, 0, -1)},
			0, 2.5},
		"after": {maths.Ray{At: maths.V(0, 0, 4), Dir: maths.V(0, 0, -1)},
			6, math.Inf(1)},
	} {
		nodes, faces = work(func() {
			if _, ok := made.Meet(each.ray, each.near, each.far); ok {
				t.Errorf("%s: met", name)
			}
		})
		if nodes != 0 || faces != 0 {
			t.Errorf("%s: %d nodes and %d faces", name, nodes, faces)
		}
	}
}
