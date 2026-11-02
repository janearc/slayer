package shape

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// held is the faces under tree node i, by their numbers in the mesh,
// smallest first.
func held(m *Mesh, i int32) []int {
	var out []int
	var under func(i int32)
	under = func(i int32) {
		nd := m.nodes[i]
		if nd.count > 0 {
			for k := nd.next; k < nd.next+nd.count; k++ {
				out = append(out, int(m.tris[k].face))
			}
			return
		}
		under(i + 1)
		under(nd.next)
	}
	under(i)
	slices.Sort(out)
	return out
}

// walk calls f for every node of the tree with its depth, the root at 0.
func walk(m *Mesh, f func(i int32, depth int)) {
	var down func(i int32, depth int)
	down = func(i int32, depth int) {
		f(i, depth)
		if nd := m.nodes[i]; nd.count == 0 {
			down(i+1, depth+1)
			down(nd.next, depth+1)
		}
	}
	down(0, 0)
}

// shapeOf writes the tree as its leaves' sizes: a leaf is its count, and a
// cut is its two halves in brackets.
func shapeOf(m *Mesh, i int32) string {
	nd := m.nodes[i]
	if nd.count > 0 {
		return fmt.Sprint(nd.count)
	}
	return "(" + shapeOf(m, i+1) + " " + shapeOf(m, nd.next) + ")"
}

// nest adds a triangle about the point c for each size s: its corners are
// c+(s,0,0), c+(0,s,0) and c+(-s,-s,0), which turn counterclockwise seen
// from above and add up to 3c. With c and s whole numbers each middle is c
// exactly, and each box is flat at c's height, 2s across and 2s up.
func nest(vs *[]maths.Vec, fs *[][3]int, c maths.Vec, sizes ...float64) {
	for _, s := range sizes {
		n := len(*vs)
		*vs = append(*vs, c.Add(maths.V(s, 0, 0)), c.Add(maths.V(0, s, 0)),
			c.Add(maths.V(-s, -s, 0)))
		*fs = append(*fs, [3]int{n, n + 1, n + 2})
	}
}

// grown is nest at each of the places in at, with the sizes for each, made
// into one mesh.
func grown(t testing.TB, at []maths.Vec, sizes ...[]float64) *Mesh {
	var vs []maths.Vec
	var fs [][3]int
	for i, c := range at {
		nest(&vs, &fs, c, sizes[i]...)
	}
	return made(t, vs, fs, nil, false)
}

// bounded holds a tree to what the code relies on: no leaf past eight
// faces, every face in a leaf, and nothing deeper than the 128 places a
// walk has to keep its way back.
func bounded(t *testing.T, why string, m *Mesh) {
	t.Helper()
	deep, total := 0, 0
	walk(m, func(i int32, depth int) {
		deep = max(deep, depth)
		if c := m.nodes[i].count; c > 8 {
			t.Errorf("%s: a leaf holds %d faces", why, c)
		} else {
			total += int(c)
		}
	})
	if deep >= 128 {
		t.Errorf("%s: the tree is %d deep", why, deep)
	}
	if total != len(m.Faces) {
		t.Errorf("%s: the leaves hold %d of %d faces", why, total,
			len(m.Faces))
	}
}

// Faces with one middle cannot be told apart by where they are, so they
// are cut by count: up to eight make a leaf, and more are cut in halves,
// the smaller first.
//
// Worked by hand: nine cut at 9/2, four and five; ten, five and five;
// seventeen, eight and nine, and the nine again four and five.
func TestFacesWithOneMiddleAreHalvedByCount(t *testing.T) {
	for _, c := range []struct {
		n    int
		want string
	}{
		{1, "1"}, {2, "2"}, {3, "3"}, {5, "5"}, {8, "8"}, {9, "(4 5)"},
		{10, "(5 5)"}, {16, "(8 8)"}, {17, "(8 (4 5))"},
		{18, "((4 5) (4 5))"},
	} {
		sizes := make([]float64, c.n)
		for i := range sizes {
			sizes[i] = float64(i + 1)
		}
		m := grown(t, []maths.Vec{{}}, sizes)
		if got := shapeOf(m, 0); got != c.want {
			t.Errorf("%d faces: %s, want %s", c.n, got, c.want)
		}
		bounded(t, fmt.Sprint(c.n, " faces"), m)
	}
}

// Eight faces that lie almost on one another are searched as well by
// testing all eight as by cutting them, so they stay one leaf.
//
// Worked by hand: eight copies of one triangle, 10 across and 15 up, each
// an eighth along from the last, have boxes of area 300 and, all together,
// 326.25. The cheapest cut, four and four, costs 311.25*4 + 311.25*4 =
// 2490, and 1+2490/326.25 is 8.63, past the 8 a leaf costs.
func TestFacesThatOverlapStayOneLeaf(t *testing.T) {
	var vs []maths.Vec
	var fs [][3]int
	for i := range 8 {
		d := maths.V(float64(i)/8, 0, 0)
		vs = append(vs, maths.V(-5, -5, 0).Add(d), maths.V(5, -5, 0).Add(d),
			maths.V(0, 10, 0).Add(d))
		fs = append(fs, [3]int{3 * i, 3*i + 1, 3*i + 2})
	}
	if got := shapeOf(made(t, vs, fs, nil, false), 0); got != "8" {
		t.Errorf("eight overlapping faces are %s", got)
	}
}

// More than eight faces are cut whatever a leaf would cost, for a leaf
// of more than eight is not allowed.
//
// Worked by hand: eight nested squares, 200 to 214 across, and a ninth 200
// across and one to the side. The eight are in a box of area 91592 at most,
// and the ninth in 80000, so the cut costs 8*91592 + 80000 = 812736 and
// 1+812736/91592 is 9.87, past the 9 a leaf of them would cost. Cut, they
// are eight and one.
func TestMoreThanEightFacesAreCutWhateverALeafCosts(t *testing.T) {
	var vs []maths.Vec
	var fs [][3]int
	nest(&vs, &fs, maths.V(0, 0, 0), 100, 101, 102, 103, 104, 105, 106, 107)
	nest(&vs, &fs, maths.V(1, 0, 0), 100)
	m := made(t, vs, fs, nil, false)
	if got := shapeOf(m, 0); got != "(8 1)" {
		t.Errorf("nine faces, one off to the side: %s, want (8 1)", got)
	}
	if got := held(m, 1); !slices.Equal(got, []int{0, 1, 2, 3, 4, 5, 6, 7}) {
		t.Errorf("the first half holds %v", got)
	}
}

// Faces far enough apart that cutting costs less than testing them are
// cut, even when there are few: two are a leaf, as any two are, and three
// are cut in a one and a two.
//
// Worked by hand: three squares 2 across, at 0, 100 and 200 along, cost
// 8 + 2*408 = 824 for either cut, over 808 for the box. 1+824/808 is 2.02,
// under the 3 a leaf of them costs.
func TestThreeFacesFarApartAreCut(t *testing.T) {
	two := grown(t, []maths.Vec{{}, maths.V(100, 0, 0)}, []float64{1},
		[]float64{1})
	if got := shapeOf(two, 0); got != "2" {
		t.Errorf("two faces far apart are %s, want 2", got)
	}
	three := grown(t, []maths.Vec{{}, maths.V(100, 0, 0), maths.V(200, 0, 0)},
		[]float64{1}, []float64{1}, []float64{1})
	if got := shapeOf(three, 0); got != "(1 2)" && got != "(2 1)" {
		t.Errorf("three faces far apart are %s, want a cut", got)
	}
}

// Two clusters a long way apart are split between them at the root, each
// whole in a leaf of its own, the lower one first.
//
// Worked by hand: the five are squares up to 10 across, area 200 at most,
// and the three up to 6, area 72, so the cut costs 5*200 + 3*72 = 1216,
// over 20160 for the box: 1.06 against the 8 a leaf of eight costs.
func TestTwoFarClustersAreSplitAtTheRoot(t *testing.T) {
	m := grown(t, []maths.Vec{{}, maths.V(1000, 0, 0)},
		[]float64{1, 2, 3, 4, 5}, []float64{1, 2, 3})
	if got := shapeOf(m, 0); got != "(5 3)" {
		t.Fatalf("two clusters are %s, want (5 3)", got)
	}
	if got := held(m, 1); !slices.Equal(got, []int{0, 1, 2, 3, 4}) {
		t.Errorf("the first cluster is %v", got)
	}
	if got := held(m, m.nodes[0].next); !slices.Equal(got, []int{5, 6, 7}) {
		t.Errorf("the second cluster is %v", got)
	}
}

// cutAt is the faces on each side of the root of a mesh of squares 2
// across, one at each of the places, which the root must have cut in two.
func cutAt(t *testing.T, at ...maths.Vec) (low, high []int) {
	t.Helper()
	sizes := make([][]float64, len(at))
	for i := range sizes {
		sizes[i] = []float64{1}
	}
	m := grown(t, at, sizes...)
	if m.nodes[0].count > 0 {
		t.Fatalf("the root of %d faces is a leaf", len(at))
	}
	return held(m, 1), held(m, m.nodes[0].next)
}

// A cut that leaves one face alone is a cut like any other, at whichever
// end the one is.
//
// Worked by hand, with a square of area 8 at each place, and the middles'
// bins by sixteenths of the 100 along x. Faces at x = 0 (50 up), 7 and 100
// are in bins 0, 1 and 15: the first alone costs 8 + 2*380 = 768, and the
// first two, 2*936 + 8 = 1880. Faces at x = 0, 93 and 100 (50 up) are in
// bins 0, 14 and 15: the last alone costs 2*380 + 8 = 768, and the first
// alone, 8 + 2*936 = 1880.
func TestAFaceAloneAtEitherEndIsCutOff(t *testing.T) {
	low, high := cutAt(t, maths.V(0, 50, 0), maths.V(7, 0, 0),
		maths.V(100, 0, 0))
	if !slices.Equal(low, []int{0}) || !slices.Equal(high, []int{1, 2}) {
		t.Errorf("the first alone: %v and %v", low, high)
	}
	low, high = cutAt(t, maths.V(0, 0, 0), maths.V(93, 0, 0),
		maths.V(100, 50, 0))
	if !slices.Equal(low, []int{0, 1}) || !slices.Equal(high, []int{2}) {
		t.Errorf("the last alone: %v and %v", low, high)
	}
}

// The cut taken is the one where the two halves cost least, after
// whichever bin that is.
//
// Worked by hand: squares 2 across at x = 0, 10, 20 and 100 are in bins
// 0, 1, 3 and 15 of the 100. The first alone costs 8 + 3*368 = 1112; the
// first two, 2*48 + 2*328 = 752; the first three, 3*88 + 8 = 272.
func TestTheCheapestCutIsTheOneTaken(t *testing.T) {
	low, high := cutAt(t, maths.V(0, 0, 0), maths.V(10, 0, 0),
		maths.V(20, 0, 0), maths.V(100, 0, 0))
	if !slices.Equal(low, []int{0, 1, 2}) || !slices.Equal(high, []int{3}) {
		t.Errorf("four in a row: %v and %v", low, high)
	}
}

// chain is n triangles in a row along x, each ten times the size of the
// last and ten times as far out: the one of rank k has corners
// (x-s,-s), (x+s,-s) and (x,2s) with x = sign * 10^k and s = x/10. The
// faces are numbered out of order, and the rank of face j is rank[j].
func chain(t *testing.T, n int, sign float64) (*Mesh, []int) {
	var vs []maths.Vec
	var fs [][3]int
	rank := make([]int, n)
	for j := range n {
		k := (j * 37) % n
		rank[j] = k
		x, s := sign*math.Pow10(k), math.Pow10(k)/10
		vs = append(vs, maths.V(x-s, -s, 0), maths.V(x+s, -s, 0),
			maths.V(x, 2*s, 0))
		fs = append(fs, [3]int{3 * j, 3*j + 1, 3*j + 2})
	}
	return made(t, vs, fs, nil, false), rank
}

// ranks is the ranks of the faces under node i, smallest first.
func ranks(m *Mesh, rank []int, i int32) []int {
	var out []int
	for _, f := range held(m, i) {
		out = append(out, rank[f])
	}
	slices.Sort(out)
	return out
}

// upTo is the ranks from lo to hi, both included.
func upTo(lo, hi int) []int {
	var out []int
	for k := lo; k <= hi; k++ {
		out = append(out, k)
	}
	return out
}

// Faces that each stand ten times as far out as the one before are cut
// one at a time, the outermost alone, down to depth 64. From there a node
// is cut in half by count, at the median of the middles, the lower half
// first.
//
// Worked by hand: of m faces at 10^0 to 10^(m-1), the farthest, at p, is
// in bin 15, the next in bin 1 and the rest in bin 0. Cutting off the
// farthest costs about (0.0066m + 0.12) p squared, and cutting off the
// two about 1.2 p squared, so the farthest goes alone. The node at depth
// d holds ranks 0 to 79-d; at depth 64 it holds 16, and cuts them 8 and 8.
//
// Mirrored, it is the lowest face, which is the farthest out, that goes
// alone, and the halves are the other way about.
func TestACutPerFaceThenHalvesAtTheDepthLimit(t *testing.T) {
	const n = 80
	for _, sign := range []float64{1, -1} {
		m, rank := chain(t, n, sign)
		bounded(t, fmt.Sprint("chain ", sign), m)
		at := int32(0)
		for d := 0; d < 64; d++ {
			if want := upTo(0, n-1-d); !slices.Equal(ranks(m, rank, at), want) {
				t.Fatalf("chain %v depth %d: ranks %v", sign, d,
					ranks(m, rank, at))
			}
			rest, alone := at+1, m.nodes[at].next
			if sign < 0 {
				rest, alone = alone, rest
			}
			if m.nodes[alone].count != 1 ||
				!slices.Equal(ranks(m, rank, alone), []int{n - 1 - d}) {
				t.Fatalf("chain %v depth %d: the one alone is %v", sign, d,
					ranks(m, rank, alone))
			}
			at = rest
		}
		first, second := at+1, m.nodes[at].next
		low, high := upTo(0, 7), upTo(8, 15)
		if sign < 0 {
			low, high = high, low
		}
		if m.nodes[first].count != 8 || m.nodes[second].count != 8 ||
			!slices.Equal(ranks(m, rank, first), low) ||
			!slices.Equal(ranks(m, rank, second), high) {
			t.Errorf("chain %v depth 64: halves %v and %v, want %v and %v",
				sign, ranks(m, rank, first), ranks(m, rank, second), low, high)
		}
	}
}

// Every node's box is the box of its faces let out by a hair on every
// side, so that a ray along a face's edge is not turned away by it. The
// hair is a billionth of the longest side of the whole mesh.
//
// Worked by hand for the first mesh, a cube from -1 to 1 stretched four
// times up and twice deep: its sides are 2, 8 and 4, so the hair is 8e-9.
// The faces' boxes are found here from the corners, face by face.
func TestEveryBoxIsItsFacesLetOutByAHair(t *testing.T) {
	rng := rand.New(rand.NewPCG(20, 21))
	sv, sf := soup(rng, 600)
	cv, cf := cube(3, false)
	for i := range cv {
		cv[i] = maths.V(cv[i].X, 4*cv[i].Y, 2*cv[i].Z)
	}
	ch, _ := chain(t, 80, 1)
	for _, c := range []struct {
		why  string
		mesh *Mesh
		hair float64 // by hand, or 0 where it is left to the formula
	}{
		{"a stretched cube", made(t, cv, cf, nil, false), 8e-9},
		{"a soup", made(t, sv, sf, nil, false), 0},
		{"a chain", ch, 0},
	} {
		m := c.mesh
		b := m.Bounds()
		size := b.Max.Sub(b.Min)
		hair := 1e-9 * math.Max(size.X, math.Max(size.Y, size.Z))
		if c.hair != 0 && hair != c.hair {
			t.Fatalf("%s: the sides are %v, so the hair is %g, not %g",
				c.why, size, hair, c.hair)
		}
		walk(m, func(i int32, depth int) {
			s, got := spanOf(m, held(m, i)), m.nodes[i].box
			want := maths.Box{
				Min: maths.V(s.lo[0]-hair, s.lo[1]-hair, s.lo[2]-hair),
				Max: maths.V(s.hi[0]+hair, s.hi[1]+hair, s.hi[2]+hair)}
			if got != want {
				t.Errorf("%s: node %d at depth %d is %v, want %v", c.why, i,
					depth, got, want)
			}
		})
	}
}

// boxAt is where a ray from at, going along way, enters the box from lo to
// hi, and whether it does before far, from near.
func boxAt(at, way, lo, hi maths.Vec, near, far float64) (float64, bool) {
	q := ready(maths.Ray{At: at, Dir: way})
	return q.box(&maths.Box{Min: lo, Max: hi}, near, far)
}

// The box test says where a ray enters a box, and whether it does.
//
// Worked by hand against the box from 0 to 1 on each axis: along each
// axis the ray is in the slab from (lo-at)/way to (hi-at)/way, and it
// enters the box at the latest of the slabs' starts and no sooner than
// near, if that is no later than the earliest of their ends and far.
func TestARayEntersABoxWhereItsSlabsSay(t *testing.T) {
	zero, one := maths.V(0, 0, 0), maths.V(1, 1, 1)
	inf := math.Inf(1)
	for _, c := range []struct {
		why       string
		at, way   maths.Vec
		near, far float64
		enter     float64
		ok        bool
	}{
		{"along x", maths.V(-1, .5, .5), maths.V(1, 0, 0), 0, inf, 1, true},
		{"back along x", maths.V(2, .5, .5), maths.V(-1, 0, 0), 0, inf, 1,
			true},
		{"a way of 2 halves the distance", maths.V(-1, .5, .5),
			maths.V(2, 0, 0), 0, inf, .5, true},
		{"a near past the entry", maths.V(-1, .5, .5), maths.V(1, 0, 0),
			1.5, inf, 1.5, true},
		{"a near past the leaving", maths.V(-1, .5, .5), maths.V(1, 0, 0),
			2.5, inf, 2.5, false},
		{"a far short of the entry", maths.V(-1, .5, .5), maths.V(1, 0, 0),
			0, .5, 1, false},
		{"started past the box", maths.V(3, .5, .5), maths.V(1, 0, 0),
			0, inf, 0, false},
		{"started on its far side", maths.V(1, .5, .5), maths.V(1, 0, 0),
			0, inf, 0, true},
		{"across the corner of its slabs", maths.V(-1, 4, .5),
			maths.V(1, -1, 0), 0, inf, 3, false},
		{"touching the corner", maths.V(0, 2, .5), maths.V(1, -1, 0), 0, inf,
			1, true},
		{"slantwise through", maths.V(-1, 1.5, .5), maths.V(1, -1, 0), 0,
			inf, 1, true},
	} {
		enter, ok := boxAt(c.at, c.way, zero, one, c.near, c.far)
		if ok != c.ok || (ok && enter != c.enter) {
			t.Errorf("%s: enters at %v, %v; want %v, %v", c.why, enter, ok,
				c.enter, c.ok)
		}
	}
}

// A slab's distances are a little uncertain, so a ray whose slabs miss
// each other by no more than a trillionth of the leaving is let through,
// and one that misses by more is not.
//
// Worked by hand: along (1,-1,0) from x = 0, the x slab is left at 1, and
// the y slab is entered at (at.y - 1), which for at.y of 2 plus 2^-41
// (4.5e-13) is 1 plus 4.5e-13, and for 2^-39 (1.8e-12) 1 plus 1.8e-12.
func TestASlabMissedByAHairIsLetThrough(t *testing.T) {
	zero, one := maths.V(0, 0, 0), maths.V(1, 1, 1)
	for _, c := range []struct {
		miss float64
		ok   bool
	}{
		{0, true}, {0x1p-41, true}, {0x1p-39, false},
	} {
		at := maths.V(0, 2+c.miss, .5)
		if enter, ok := boxAt(at, maths.V(1, -1, 0), zero, one, 0,
			math.Inf(1)); ok != c.ok {
			t.Errorf("missing by %g: %v at %v, want %v", c.miss, ok, enter,
				c.ok)
		}
	}
}

// area is a box's whole surface: for 2 by 3 by 5, twice 6+15+10.
func TestTheAreaOfABox(t *testing.T) {
	b := maths.Box{Max: maths.V(2, 3, 5)}
	if got := area(b); got != 62 {
		t.Errorf("a box 2 by 3 by 5 has area %v, want 62", got)
	}
}

// longest is the axis along which a box is longest. Boxes are given by
// their sides, each axis the longest in turn, and one where the first is
// longer than the second but not than the third.
func TestTheLongestAxisOfABox(t *testing.T) {
	for _, c := range []struct {
		d    maths.Vec
		want int
	}{
		{maths.V(3, 2, 1), 0}, {maths.V(1, 3, 2), 1}, {maths.V(1, 2, 3), 2},
		{maths.V(2, 3, 1), 1}, {maths.V(2, 1, 3), 2},
	} {
		if got := longest(maths.Box{Max: c.d}); got != c.want {
			t.Errorf("a box %v across: axis %d, want %d", c.d, got, c.want)
		}
	}
}

// along is a point's part on the axis 0, 1 or 2.
func TestAPointAlongAnAxis(t *testing.T) {
	v := maths.V(1, 2, 3)
	for axis, want := range []float64{1, 2, 3} {
		if got := along(v, axis); got != want {
			t.Errorf("axis %d of %v is %v, want %v", axis, v, got, want)
		}
	}
}

// away is how far a point is from a box, squared, on each axis and past
// either end: from the box of 0 to 1, a point 2 past a side is 4 away,
// and one 3 past on one axis and 2 and 1 on the others is 9+4+1.
func TestHowFarAPointIsFromABox(t *testing.T) {
	b := maths.Box{Max: maths.V(1, 1, 1)}
	for _, c := range []struct {
		p    maths.Vec
		want float64
	}{
		{maths.V(.5, .5, .5), 0}, {maths.V(1, 0, .5), 0},
		{maths.V(3, .5, .5), 4}, {maths.V(-2, .5, .5), 4},
		{maths.V(.5, 4, .5), 9}, {maths.V(.5, -1, .5), 1},
		{maths.V(.5, .5, 3), 4}, {maths.V(.5, .5, -3), 9},
		{maths.V(4, 3, -1), 9 + 4 + 1},
	} {
		if got := away(b, c.p); got != c.want {
			t.Errorf("%v is %v from the box, want %v", c.p, got, c.want)
		}
	}
}

// A Mesh not made by NewMesh says what is wrong with it, in words, from
// whichever question is put to it.
func TestAMeshNotBuiltSaysWhatIsWrong(t *testing.T) {
	const want = "shape: a Mesh is made by NewMesh, which builds its tree"
	for name, ask := range map[string]func(m *Mesh){
		"Bounds":   func(m *Mesh) { m.Bounds() },
		"Hit":      func(m *Mesh) { m.Hit(down(.1, .1), 0, forever) },
		"Normal":   func(m *Mesh) { m.Normal(maths.V(.1, .1, 1)) },
		"Distance": func(m *Mesh) { m.Distance(maths.V(.1, .1, 1)) },
	} {
		func() {
			defer func() {
				if got := recover(); got != want {
					t.Errorf("%s: panicked with %v, want %q", name, got,
						want)
				}
			}()
			ask(&Mesh{Vertices: []maths.Vec{{}, maths.XAxis, maths.Up},
				Faces: [][3]int{{0, 1, 2}}})
		}()
	}
}

// The face nearest a point is the one face of a mesh of one, wherever the
// point is, even so far out that every distance squared is past what a
// double holds: worked by hand, 1e200 squared is endless.
func TestTheOnlyFaceIsNearestFromAnywhere(t *testing.T) {
	m := lone(t)
	if got := m.Normal(maths.V(1e200, 0, 0)); got != maths.V(0, 0, 1) {
		t.Errorf("the normal from 1e200 out is %v, want up", got)
	}
}

// winding is how many more faces a ray leaves through than it enters by.
//
// Worked by hand on the cube from -1 to 1: from inside, out through one
// face; from outside going away, none; through the middle, in and out;
// from inside it and a cube twice the size round it, out through both;
// from between the two, out through the larger alone.
func TestWindingCountsFacesLeftLessFacesEntered(t *testing.T) {
	vs, fs := cube(1, false)
	one := made(t, vs, fs, nil, false)
	n, k := len(vs), len(fs)
	for i := range n {
		vs = append(vs, vs[i].Scale(2))
	}
	for _, f := range fs[:k] {
		fs = append(fs, [3]int{f[0] + n, f[1] + n, f[2] + n})
	}
	two := made(t, vs, fs, nil, false)
	for _, c := range []struct {
		why  string
		m    *Mesh
		r    maths.Ray
		want int
	}{
		{"from inside", one, maths.Ray{At: maths.V(.1, .2, .3),
			Dir: maths.V(.48, .6, .64)}, 1},
		{"from outside going away", one, maths.Ray{At: maths.V(3, .2, .3),
			Dir: maths.V(1, 0, 0)}, 0},
		{"through", one, maths.Ray{At: maths.V(-3, .2, .3),
			Dir: maths.V(1, 0, 0)}, 0},
		{"from inside both", two, maths.Ray{At: maths.V(.1, .2, .3),
			Dir: maths.V(.48, .6, .64)}, 2},
		{"from between", two, maths.Ray{At: maths.V(1.5, .2, .3),
			Dir: maths.V(1, 0, 0)}, 1},
	} {
		if got := c.m.winding(c.r); got != c.want {
			t.Errorf("%s: %d, want %d", c.why, got, c.want)
		}
	}
}

// span is the box of some faces, found from their corners.
type span struct{ lo, hi [3]float64 }

// spanOf is the box of the faces fs of m.
func spanOf(m *Mesh, fs []int) span {
	inf := math.Inf(1)
	s := span{[3]float64{inf, inf, inf}, [3]float64{-inf, -inf, -inf}}
	for _, f := range fs {
		for _, v := range m.Faces[f] {
			p := m.Vertices[v]
			for k, c := range [3]float64{p.X, p.Y, p.Z} {
				s.lo[k] = math.Min(s.lo[k], c)
				s.hi[k] = math.Max(s.hi[k], c)
			}
		}
	}
	return s
}

// surface is twice the sum of the three products of the sides.
func (s span) surface() float64 {
	x, y, z := s.hi[0]-s.lo[0], s.hi[1]-s.lo[1], s.hi[2]-s.lo[2]
	return 2 * (x*y + y*z + z*x)
}

// plan is what the tree may do with the faces fs of m at a depth, by brute
// force from their corners alone, trying every cut. Along the axis the
// middles spread furthest on, the faces fall into sixteen bins, and the
// cut between two bins is cheap where the faces below times the surface of
// their box, and the faces above times theirs, come to least. A leaf of
// eight or fewer is kept instead when a cut, priced as a box and the cost
// over the box's surface, is no cheaper than the faces tested one by one.
// Past depth 64, or without a spread, eight or fewer make a leaf and more
// are cut in half by their order along the axis.
//
// Prices within a billionth of each other are taken as equal, so either
// choice is allowed: leaf says a leaf is, and cuts are the first halves
// that are.
func plan(m *Mesh, fs []int, depth int) (leaf bool, cuts [][]int) {
	n := len(fs)
	if n <= 2 {
		return true, nil
	}
	mids := make([][3]float64, n)
	var lo, hi [3]float64
	for i, f := range fs {
		var sum [3]float64
		for _, v := range m.Faces[f] {
			p := m.Vertices[v]
			sum[0], sum[1], sum[2] = sum[0]+p.X, sum[1]+p.Y, sum[2]+p.Z
		}
		for k := range sum {
			mids[i][k] = sum[k] / 3
			if i == 0 {
				lo[k], hi[k] = mids[i][k], mids[i][k]
			}
			lo[k] = math.Min(lo[k], mids[i][k])
			hi[k] = math.Max(hi[k], mids[i][k])
		}
	}
	axis := 0
	for k := 1; k < 3; k++ {
		if hi[k]-lo[k] > hi[axis]-lo[axis] {
			axis = k
		}
	}
	if depth < 64 && hi[axis] > lo[axis] {
		width := hi[axis] - lo[axis]
		cost := make([]float64, 16)
		below := make([][]int, 16)
		best := math.Inf(1)
		for k := 1; k < 16; k++ {
			var under, over []int
			for i, f := range fs {
				bin := min(int(16*(mids[i][axis]-lo[axis])/width), 15)
				if bin < k {
					under = append(under, f)
				} else {
					over = append(over, f)
				}
			}
			if len(under) == 0 || len(over) == 0 {
				continue
			}
			cost[k] = spanOf(m, under).surface()*float64(len(under)) +
				spanOf(m, over).surface()*float64(len(over))
			below[k] = under
			best = math.Min(best, cost[k])
		}
		cheaper := 1 + best/spanOf(m, fs).surface()
		leaf = n <= 8 && cheaper >= float64(n)*(1-1e-9)
		if n <= 8 && cheaper >= float64(n)*(1+1e-9) {
			return leaf, nil
		}
		for k := 1; k < 16; k++ {
			if below[k] != nil && cost[k] <= best*(1+1e-9) {
				slices.Sort(below[k])
				if !slices.ContainsFunc(cuts, func(c []int) bool {
					return slices.Equal(c, below[k])
				}) {
					cuts = append(cuts, below[k])
				}
			}
		}
		return leaf, cuts
	}
	if n <= 8 {
		return true, nil
	}
	order := slices.Clone(fs)
	sort.Slice(order, func(a, b int) bool {
		return mids[slices.Index(fs, order[a])][axis] <
			mids[slices.Index(fs, order[b])][axis]
	})
	half := slices.Clone(order[:n/2])
	slices.Sort(half)
	return false, [][]int{half}
}

// scatter is n triangles of about the given size, strewn through the cube
// from -spread to spread, overlapping as they fall.
func scatter(rng *rand.Rand, n int, spread, size float64) ([]maths.Vec, [][3]int) {
	var vs []maths.Vec
	var fs [][3]int
	for range n {
		mid := within(rng, spread)
		s := size * (0.5 + rng.Float64())
		for range 3 {
			vs = append(vs, mid.Add(anyway(rng).Scale(s)))
		}
		fs = append(fs, [3]int{len(vs) - 3, len(vs) - 2, len(vs) - 1})
	}
	return vs, fs
}

// lumps is n lumps of 30 small triangles each, strewn through the cube
// from -3 to 3.
func lumps(rng *rand.Rand, n int) ([]maths.Vec, [][3]int) {
	var vs []maths.Vec
	var fs [][3]int
	for range n {
		v, f := scatter(rng, 30, 0.05, 0.1)
		c, base := within(rng, 3), len(vs)
		for _, p := range v {
			vs = append(vs, p.Add(c))
		}
		for _, f := range f {
			fs = append(fs, [3]int{f[0] + base, f[1] + base, f[2] + base})
		}
	}
	return vs, fs
}

// Every node of the tree is a leaf or a cut that brute force allows there,
// found again from the corners, for faces strewn thinly, thickly and in
// lumps, and for the chains.
func TestEveryNodeIsCutAsBruteForceAllows(t *testing.T) {
	rng := rand.New(rand.NewPCG(30, 31))
	var meshes []*Mesh
	names := []string{"a soup", "thin", "thick", "lumps", "chain", "mirror"}
	vs, fs := soup(rng, 3000)
	meshes = append(meshes, made(t, vs, fs, nil, false))
	vs, fs = scatter(rng, 1500, 1, 0.05)
	meshes = append(meshes, made(t, vs, fs, nil, false))
	vs, fs = scatter(rng, 400, 0.2, 0.5)
	meshes = append(meshes, made(t, vs, fs, nil, false))
	vs, fs = lumps(rng, 40)
	meshes = append(meshes, made(t, vs, fs, nil, false))
	up, _ := chain(t, 80, 1)
	down, _ := chain(t, 80, -1)
	meshes = append(meshes, up, down)
	for w, m := range meshes {
		leaves, cuts := 0, 0
		walk(m, func(i int32, depth int) {
			fs := held(m, i)
			leaf, halves := plan(m, fs, depth)
			if m.nodes[i].count > 0 {
				leaves++
				if !leaf {
					t.Errorf("%s: node %d at depth %d of %d faces is a "+
						"leaf, and a cut is meant", names[w], i, depth,
						len(fs))
				}
				return
			}
			cuts++
			got := held(m, i+1)
			if !slices.ContainsFunc(halves, func(h []int) bool {
				return slices.Equal(h, got)
			}) {
				t.Errorf("%s: node %d at depth %d of %d faces is cut "+
					"%d and %d, and no cut brute force allows does that",
					names[w], i, depth, len(fs), len(got), len(fs)-len(got))
			}
		})
		bounded(t, names[w], m)
		t.Logf("%s: %d faces, %d leaves, %d cuts", names[w], len(m.Faces),
			leaves, cuts)
	}
}

// plates is n big triangles one above another, a unit apart from z = 0
// up, each 20 across and 20 up, so that a ray along z through the middle
// meets every one.
func plates(t testing.TB, n int) *Mesh {
	var vs []maths.Vec
	var fs [][3]int
	for k := range n {
		z := float64(k)
		vs = append(vs, maths.V(-10, -10, z), maths.V(10, -10, z),
			maths.V(0, 10, z))
		fs = append(fs, [3]int{3 * k, 3*k + 1, 3*k + 2})
	}
	return made(t, vs, fs, nil, false)
}

// quickest is the least time, over a few tries, that doing f takes.
func quickest(f func()) time.Duration {
	best := time.Hour
	for range 7 {
		start := time.Now()
		f()
		best = min(best, time.Since(start))
	}
	return best
}

// The tree finds a ray's face among many quickly: ten rays down a tower of
// twenty thousand faces take less than asking every face once. A tower is
// the worst case for a tree that does not look in the nearer box first,
// which meets every face on the way to the nearest. Rays go down it and up
// it, so that either child may be the nearer.
//
// Worked by hand: from 5 below the first plate, at z = 0, the ray meets
// it at 5; from 20004, 5 above the last, at z = 19999, also at 5.
func TestTheNearerBoxIsLookedInFirstForARay(t *testing.T) {
	const n = 20000
	m := plates(t, n)
	for _, c := range []struct {
		why string
		r   maths.Ray
		t   float64
	}{
		{"from below", maths.Ray{At: maths.V(0, 0, -5),
			Dir: maths.V(0, 0, 1)}, 5},
		{"from above", maths.Ray{At: maths.V(0, 0, n+4),
			Dir: maths.V(0, 0, -1)}, 5},
	} {
		if got, _, ok := m.first(c.r, 0, forever); !ok || got != c.t {
			t.Fatalf("%s: met at %v, %v; want %v", c.why, got, ok, c.t)
		}
		all := quickest(func() { every(m, c.r, 0, forever) })
		few := quickest(func() {
			for range 10 {
				m.first(c.r, 0, forever)
			}
		})
		t.Logf("%s: ten rays %v, every face once %v", c.why, few, all)
		if few > all {
			t.Errorf("%s: ten rays took %v, longer than every face once, %v",
				c.why, few, all)
		}
	}
}

// The same for the nearest face to a point, from below the tower and from
// above it. Worked by hand: 5 from the first plate or the last, so 25
// squared.
func TestTheNearerBoxIsLookedInFirstForAPoint(t *testing.T) {
	const n = 20000
	m := plates(t, n)
	for _, c := range []struct {
		why string
		p   maths.Vec
		d2  float64
	}{
		{"from below", maths.V(0, 0, -5), 25},
		{"from above", maths.V(0, 0, n+4), 25},
	} {
		if _, _, d2 := m.nearest(c.p); d2 != c.d2 {
			t.Fatalf("%s: nearest is %v away squared, want %v", c.why, d2,
				c.d2)
		}
		all := quickest(func() { nearestOfAll(m, c.p) })
		few := quickest(func() {
			for range 10 {
				m.nearest(c.p)
			}
		})
		t.Logf("%s: ten points %v, every face once %v", c.why, few, all)
		if few > all {
			t.Errorf("%s: ten points took %v, longer than every face once, "+
				"%v", c.why, few, all)
		}
	}
}

// A ray that misses the whole mesh is turned away at the root by winding,
// as it is by a hit: ten of them take less than asking every face once.
func TestAWindingRayThatMissesIsTurnedAwayAtTheRoot(t *testing.T) {
	const n = 20000
	m := plates(t, n)
	r := maths.Ray{At: maths.V(100, 100, -5), Dir: maths.V(0, 0, 1)}
	if got := m.winding(r); got != 0 {
		t.Fatalf("a ray that misses winds %d", got)
	}
	all := quickest(func() { every(m, r, 0, forever) })
	few := quickest(func() {
		for range 10 {
			m.winding(r)
		}
	})
	t.Logf("ten rays %v, every face once %v", few, all)
	if few > all {
		t.Errorf("ten rays took %v, longer than every face once, %v", few, all)
	}
}
