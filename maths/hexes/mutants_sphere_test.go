package hexes

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// the icosahedron's own geometry, which the tiling is built on: twelve
// corners on the unit sphere, each with five corners one edge away, the
// published edge of 4/sqrt(10+2*sqrt(5)) = 1.0514622, five two edges away
// at the edge times the golden ratio, and one opposite.
//
// twenty faces, each three corners an edge apart, with a unit middle
// 0.7946545 from each of its corners (the inradius over the circumradius).
func TestTheIcosahedronIsTheIcosahedron(t *testing.T) {
	const edgeLen, twoEdges, inradius = 1.0514622242382672, 1.7013016167040798, 0.7946544722917661
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-12 }
	for a, va := range verts {
		if !near(va.Len(), 1) {
			t.Fatalf("corner %d is %v long", a, va.Len())
		}
		one, two, opposite := 0, 0, 0
		for b, vb := range verts {
			d := va.Sub(vb).Len()
			switch {
			case a == b:
			case near(d, edgeLen):
				one++
			case near(d, twoEdges):
				two++
			case near(d, 2):
				opposite++
			default:
				t.Fatalf("corners %d and %d are %v apart", a, b, d)
			}
		}
		if one != 5 || two != 5 || opposite != 1 {
			t.Fatalf("corner %d has %d, %d and %d", a, one, two, opposite)
		}
	}
	for f, tri := range faces {
		for k := range 3 {
			a, b := verts[tri[k]], verts[tri[(k+1)%3]]
			if !near(a.Sub(b).Len(), edgeLen) {
				t.Fatalf("face %d has a side of %v", f, a.Sub(b).Len())
			}
			if !near(normals[f].Dot(a), inradius) {
				t.Fatalf("face %d's middle is %v from a corner", f, normals[f].Dot(a))
			}
		}
		if !near(normals[f].Len(), 1) {
			t.Fatalf("face %d's middle is %v long", f, normals[f].Len())
		}
	}
}

// weights is a lattice corner of a face as whole shares of the icosahedron's
// corners, which name it forever: the corner (i, j) of face f is n-i-j parts of
// the face's first corner, i of its second and j of its third, and a part of
// zero is not there.
func weights(n, f, i, j int) map[int]int {
	w := map[int]int{}
	for k, c := range []int{n - i - j, i, j} {
		if c > 0 {
			w[faces[f][k]] = c
		}
	}
	return w
}

// lowest is the one name of a corner: the lowest numbered face that holds
// every corner it is made of, and the shares on that face's second and
// third corners.
func lowest(n, f, i, j int) Tile {
	w := weights(n, f, i, j)
	for g := range faces {
		all := true
		for v := range w {
			has := false
			for _, c := range faces[g] {
				has = has || c == v
			}
			all = all && has
		}
		if all {
			return Tile{g, w[faces[g][1]], w[faces[g][2]]}
		}
	}
	panic("a corner on no face")
}

// a corner shared by faces is named on the lowest numbered of them, found here
// from the whole shares it is made of, for every corner of every face at
// several splits and for a sample at the finest.
func TestEveryCornerHasOneNameTheLowestFace(t *testing.T) {
	for _, n := range []int{1, 2, 3, 4, 6, 9} {
		tl := Sphere{N: n}
		for f := range faces {
			for i := 0; i <= n; i++ {
				for j := 0; i+j <= n; j++ {
					if got, want := tl.canonical(f, i, j), lowest(n, f, i, j); got != want {
						t.Fatalf("n=%d corner (%d, %d) of face %d is %v, want %v",
							n, i, j, f, got, want)
					}
				}
			}
		}
	}
	tl := Sphere{N: finest}
	n := finest
	for f := range faces {
		for _, c := range [][2]int{{1, 1}, {1, 2}, {2, 1}, {1, n - 2}, {n - 2, 1},
			{n / 2, n/2 - 1}, {0, 0}, {n, 0}, {0, n}, {0, 1}, {1, 0}, {0, 12345},
			{12345, 0}, {0, n / 2}, {n / 2, 0}, {n - 1, 1}, {1, n - 1},
			{n / 2, n / 2}, {0, n - 1}, {n - 1, 0}} {
			if got, want := tl.canonical(f, c[0], c[1]), lowest(n, f, c[0], c[1]); got != want {
				t.Fatalf("finest corner %v of face %d is %v, want %v", c, f, got, want)
			}
		}
	}
}

// ids number every corner of every face in the order face, then i, then
// j: the k-th corner of that order has id k, so there are 20 faces of
// (n+1)(n+2)/2 corners and not one id is missing or shared.
//
// worked by hand for n=3: face 2's corner (1, 1) comes after 20 corners
// of faces 0 and 1 and the four of its row 0, then (1, 0), then itself:
// 25.
func TestIDsCountTheCornersInOrder(t *testing.T) {
	for _, n := range []int{1, 2, 3, 5} {
		tl := Sphere{N: n}
		k := int64(0)
		for f := range faces {
			for i := 0; i <= n; i++ {
				for j := 0; i+j <= n; j++ {
					if got := tl.ID(Tile{f, i, j}); got != k {
						t.Fatalf("n=%d corner (%d, %d) of face %d has id %d, want %d",
							n, i, j, f, got, k)
					}
					k++
				}
			}
		}
	}
	if got := (Sphere{N: 3}).ID(Tile{2, 1, 1}); got != 25 {
		t.Errorf("the id is %d, want 25", got)
	}
	tl := Sphere{N: 5}
	last := int64(-1)
	for _, tile := range tl.Tiles() {
		if id := tl.ID(tile); id <= last {
			t.Fatalf("tile %v has id %d after %d", tile, id, last)
		} else {
			last = id
		}
	}
}

// the finest sphere is five hundred million splits, which counts
// 10*(5e8)^2+2 tiles, 2.5e18 and two, inside an int64; the rung above
// 250 million is the finest, and the rung of any other is twice it.
//
// a spacing is undone by rounding: the icosahedron's own edge is one
// split and a third of it is three.
func TestTheFinestAndTheRungsToIt(t *testing.T) {
	if SphereFor(0).N != 500_000_000 {
		t.Errorf("the finest is %d splits", SphereFor(0).N)
	}
	if got := (Sphere{N: 500_000_000}).Count(); got != 2_500_000_000_000_000_002 {
		t.Errorf("the finest counts %d tiles", got)
	}
	for _, c := range []struct{ from, to int }{{7, 14}, {0, 2}, {1, 2},
		{249_999_999, 499_999_998}, {250_000_000, 500_000_000},
		{250_000_001, 500_000_000}, {300_000_000, 500_000_000},
		{500_000_000, 500_000_000}} {
		if got := (Sphere{N: c.from}).Finer(); got != (Sphere{N: c.to}) {
			t.Errorf("below %d is %v, want %d", c.from, got, c.to)
		}
	}
	for _, c := range []struct {
		radians float64
		want    int
	}{{edge, 1}, {10, 1}, {edge / 2, 2}, {edge / 3.4, 3}, {edge / 3.6, 4},
		{edge / 500_000_000 * 1.0000001, 499_999_950}} {
		if got := SphereFor(c.radians).N; got != c.want {
			t.Errorf("for %v radians, %d splits, want %d", c.radians, got, c.want)
		}
	}
	for _, n := range []int{-5, 0, 1} {
		if (Sphere{N: n}).Count() != 12 {
			t.Errorf("%d splits count %d tiles", n, (Sphere{N: n}).Count())
		}
	}
	if (Sphere{N: 2}).Count() != 42 {
		t.Errorf("two splits count %d tiles", (Sphere{N: 2}).Count())
	}
}

// clamp keeps a corner on its face, i and j at least zero and i+j at most n,
// and moves one that is off to the nearest place on the edge it crossed.
//
// worked by hand for n=4: -1 along is zero; 6 along and zero across is the
// corner (4, 0); (3, 3) is over the long edge and comes to (3, 1); (5, 5) to
// (4, 0); (0, 5) to (0, 4); a corner already on the face, edge and all, stays.
func TestClampKeepsACornerOnItsFace(t *testing.T) {
	for _, c := range []struct{ i, j, n, ci, cj int }{
		{-1, 2, 4, 0, 2}, {2, -3, 4, 2, 0}, {-1, -1, 4, 0, 0},
		{6, 0, 4, 4, 0}, {3, 3, 4, 3, 1}, {5, 5, 4, 4, 0}, {0, 5, 4, 0, 4},
		{0, 2, 4, 0, 2}, {2, 0, 4, 2, 0}, {1, 3, 4, 1, 3}, {4, 0, 4, 4, 0},
		{0, 0, 4, 0, 0}, {2, 2, 4, 2, 2}, {2, 3, 4, 2, 2},
	} {
		if i, j := clamp(c.i, c.j, c.n); i != c.ci || j != c.cj {
			t.Errorf("clamp(%d, %d, %d) = (%d, %d), want (%d, %d)",
				c.i, c.j, c.n, i, j, c.ci, c.cj)
		}
	}
}

// walk takes up to eight steps along a lattice line toward the corner a
// direction is, in any of the six lattice directions, and no more: eight
// steps from a corner nine away leaves one to go.
//
// worked by hand on a face of 40 splits from (20, 10), where the corner
// eight away and nine away along any of the six directions is still on
// the face.
func TestWalkTakesEightStepsAndNoMore(t *testing.T) {
	tl := Sphere{N: 40}
	for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, -1}, {-1, 1}} {
		for _, away := range []int{8, 9} {
			p := tl.corner(0, 20+away*d[0], 10+away*d[1])
			got := tl.walk(p, 0, 20, 10, apart(p, tl.corner(0, 20, 10)))
			want := lowest(40, 0, 20+8*d[0], 10+8*d[1])
			if got != want {
				t.Errorf("direction %v, %d away: stopped at %v, want %v", d, away, got, want)
			}
		}
	}
}

// walk stays on its face. a direction across an edge of the face is
// nearest an extension of the lattice beyond it, and walk starting at the
// edge does not follow it off; one starting just inside walks onto the
// edge itself, on each of the three edges.
func TestWalkStaysOnItsFace(t *testing.T) {
	const n = 40
	tl := Sphere{N: n}
	onFace := func(tile Tile) bool {
		return tile.I >= 0 && tile.J >= 0 && tile.I+tile.J <= n
	}
	for _, f := range []int{0, 13} {
		for _, c := range []struct {
			start, beyond [2]int
		}{
			{[2]int{0, 10}, [2]int{-1, 10}},
			{[2]int{10, 0}, [2]int{10, -1}},
			{[2]int{10, 30}, [2]int{11, 30}},
			{[2]int{0, 10}, [2]int{-3, 12}},
			{[2]int{10, 0}, [2]int{12, -3}},
			{[2]int{10, 30}, [2]int{14, 30}},
		} {
			p := tl.corner(f, c.beyond[0], c.beyond[1])
			got := tl.walk(p, f, c.start[0], c.start[1],
				apart(p, tl.corner(f, c.start[0], c.start[1])))
			if !onFace(got) {
				t.Errorf("face %d: walked from %v toward %v to %v, off the face",
					f, c.start, c.beyond, got)
			}
		}
		for _, c := range []struct{ start, onto [2]int }{
			{[2]int{1, 10}, [2]int{0, 10}},
			{[2]int{10, 1}, [2]int{10, 0}},
			{[2]int{10, 29}, [2]int{10, 30}},
		} {
			p := tl.corner(f, c.onto[0], c.onto[1])
			got := tl.walk(p, f, c.start[0], c.start[1],
				apart(p, tl.corner(f, c.start[0], c.start[1])))
			if want := lowest(n, f, c.onto[0], c.onto[1]); got != want {
				t.Errorf("face %d: walked from %v to %v, want %v", f, c.start, got, want)
			}
		}
	}
}

// the corner a direction is nearest on its face is found by walk from any
// start a few steps away, so that where the first guess falls, a corner or
// two off, the answer is the same.
//
// the nearest is found by asking every corner of the face; the starts are
// every corner within three lattice steps of it, a step being one of the
// six directions.
func TestWalkFindsTheNearestFromAFewStepsAway(t *testing.T) {
	for _, n := range []int{3, 12, 40} {
		tl := Sphere{N: n}
		for k := range int64(300) {
			p := random(k + 77)
			if k%3 == 0 {
				p = verts[int(k/3)%12].Add(p.Scale(0.4 / float64(n))).Unit()
			}
			f := face(p)
			best, ti, tj := math.Inf(1), 0, 0
			for i := 0; i <= n; i++ {
				for j := 0; i+j <= n; j++ {
					if d := apart(p, tl.corner(f, i, j)); d < best {
						best, ti, tj = d, i, j
					}
				}
			}
			want := lowest(n, f, ti, tj)
			for di := -3; di <= 3; di++ {
				for dj := -3; dj <= 3; dj++ {
					if max(abs(di), abs(dj), abs(di+dj)) > 3 {
						continue
					}
					si, sj := ti+di, tj+dj
					if si < 0 || sj < 0 || si+sj > n {
						continue
					}
					got := tl.walk(p, f, si, sj, apart(p, tl.corner(f, si, sj)))
					if got != want {
						t.Fatalf("n=%d point %d: from (%d, %d) to %v, want %v",
							n, k, si, sj, got, want)
					}
				}
			}
		}
	}
}

// abs is the size of an int.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// settle looks across the edge of a face: from a tile on an edge, a direction
// that is the middle of any of its neighbours, across the edge or not, is in
// that neighbour, which is zero away.
func TestSettleMovesAnEdgeTileToANearerNeighbour(t *testing.T) {
	for _, n := range []int{1, 2, 4} {
		tl := Sphere{N: n}
		for _, tile := range tl.Tiles() {
			if tl.inside(tile) {
				continue
			}
			for _, nb := range tl.Neighbours(tile) {
				if got := tl.settle(tl.Centre(nb), tile); got != nb {
					t.Fatalf("n=%d: from %v to the middle of %v: %v", n, tile, nb, got)
				}
			}
		}
	}
}

// on says which lattice corner of a face a direction is, if it is one.
// every corner of the face is itself, and a corner of the lattice carried
// past an edge of the face, along or across, is not on it.
//
// nor is a direction between corners, at a fraction in either of the two,
// or just off a corner by more than a millionth of a split.
func TestOnIsTheCornersOfTheFaceAndNothingElse(t *testing.T) {
	for _, n := range []int{1, 2, 3, 5} {
		tl := Sphere{N: n}
		for g := range faces {
			for i := -2; i <= n+2; i++ {
				for j := -2; j <= n+2; j++ {
					q := tl.corner(g, i, j)
					gi, gj, ok := tl.on(g, q)
					want := i >= 0 && j >= 0 && i+j <= n
					if ok != want || ok && (gi != i || gj != j) {
						t.Fatalf("n=%d face %d corner (%d, %d): (%d, %d, %v), want on %v",
							n, g, i, j, gi, gj, ok, want)
					}
				}
			}
		}
	}
	tl := Sphere{N: 4}
	a, b, c := verts[faces[3][0]], verts[faces[3][1]], verts[faces[3][2]]
	between := func(fi, fj float64) maths.Vec {
		return a.Scale(4 - fi - fj).Add(b.Scale(fi)).Add(c.Scale(fj)).Unit()
	}
	for _, c := range []struct {
		fi, fj float64
		want   bool
	}{{1, 1, true}, {1.5, 1, false}, {1, 1.5, false}, {2.3, 1, false},
		{1, 2.3, false}, {1.5, 1.5, false}, {1 + 1e-7, 1, true},
		{1, 1 - 1e-7, true}, {1 + 1e-5, 1, false}, {1, 1 + 1e-5, false}} {
		if _, _, ok := tl.on(3, between(c.fi, c.fj)); ok != c.want {
			t.Errorf("(%v, %v) is on: %v, want %v", c.fi, c.fj, ok, c.want)
		}
	}
}

// the edge of a tile is half the chord to a neighbour at its middle, and
// the other side of it a direction at that neighbour's middle is as far
// outside.
//
// on the bare icosahedron every corner has five neighbours an edge away,
// 1.0514622 on the unit sphere, so a corner's middle is 0.5257311 from
// its edge and a neighbour's middle is as far beyond it.
func TestSphereEdgeIsHalfTheChord(t *testing.T) {
	const half = 1.0514622242382672 / 2
	tl := Sphere{N: 1}
	for _, tile := range tl.Tiles() {
		if e := tl.Edge(tl.Centre(tile), tile); math.Abs(e-half) > 1e-12 {
			t.Fatalf("tile %v: its middle is %v from its edge", tile, e)
		}
		for _, nb := range tl.Neighbours(tile) {
			if e := tl.Edge(tl.Centre(nb), tile); math.Abs(e+half) > 1e-12 {
				t.Fatalf("tile %v: the middle of %v is %v from its edge", tile, nb, e)
			}
		}
	}
}

// the angle between two directions is the arc cosine of their dot, held
// to between minus one and one: a dot past either is the half turn or
// none, not no number.
//
// worked by hand: 2 times a unit along x against 1 along x is a dot of
// 2, and against minus 1 along x is minus 2. the checks are written so
// that a result which is no number fails them.
func TestSphereDistanceHoldsADotToOne(t *testing.T) {
	tl := Sphere{N: 4}
	x := maths.V(1, 0, 0)
	if d := tl.Distance(x.Scale(2), x); !(d == 0) {
		t.Errorf("a direction past one is %v from itself", d)
	}
	if d := tl.Distance(x.Scale(2), x.Scale(-1)); !(math.Abs(d-math.Pi) <= 1e-12) {
		t.Errorf("a direction past minus one is %v from its opposite", d)
	}
	if d := tl.Distance(x.Scale(-3), x); !(math.Abs(d-math.Pi) <= 1e-12) {
		t.Errorf("a dot of minus three is %v", d)
	}
}

// somewhere on a sphere is the direction (2a-1, 2b-1, 2c-1): worked by
// hand, (0.9, 0.25, 0.75) is (0.8, -0.5, 0.5) over sqrt(1.14), a corner of
// the cube is a corner's direction, and the middle of the cube, which has
// no direction, is up.
func TestSphereSomewhereIsTheCubesDirection(t *testing.T) {
	tl := Sphere{N: 4}
	l := math.Sqrt(1.14)
	if p := tl.Somewhere(0.9, 0.25, 0.75); !p.Near(maths.V(0.8/l, -0.5/l, 0.5/l), 1e-12) {
		t.Errorf("somewhere (0.9, 0.25, 0.75) is %v", p)
	}
	r3 := math.Sqrt(3)
	if p := tl.Somewhere(1, 0, 1); !p.Near(maths.V(1/r3, -1/r3, 1/r3), 1e-12) {
		t.Errorf("somewhere (1, 0, 1) is %v", p)
	}
	if p := tl.Somewhere(0.5, 0.5, 0.5); p != maths.Up {
		t.Errorf("the middle of the cube is %v, not up", p)
	}
}

// Nearest finds the same tile as asking every tile, at more splits than
// the first test and for directions near the corners and along the edges
// of the icosahedron, where the faces meet.
func TestNearestIsTheTrueNearestAtTheFacesEdges(t *testing.T) {
	for _, n := range []int{2, 4, 7, 25} {
		tl := Sphere{N: n}
		tiles := tl.Tiles()
		for k := range int64(4000) {
			p := random(k + 5000)
			f := faces[int(k)%20]
			a, b, c := verts[f[0]], verts[f[1]], verts[f[2]]
			x, y := random(k+1).X*0.5+0.5, random(k+2).Y*0.5+0.5
			switch k % 4 {
			case 0:
				p = verts[int(k/4)%12].Add(p.Scale(0.5 / float64(n))).Unit()
			case 1:
				p = a.Scale(x).Add(b.Scale(1 - x)).Add(c.Scale(0.01 / float64(n))).Unit()
			case 2:
				p = b.Scale(y).Add(c.Scale(1 - y)).Add(a.Scale(0.01 / float64(n))).Unit()
			}
			got := tl.Nearest(p)
			best, want := math.Inf(1), Tile{}
			for _, tile := range tiles {
				if d := p.Sub(tl.Centre(tile)).Len(); d < best {
					best, want = d, tile
				}
			}
			if got != want &&
				math.Abs(p.Sub(tl.Centre(got)).Len()-best) > 1e-12 {
				t.Fatalf("n=%d point %d: tile %v, want %v", n, k, got, want)
			}
		}
	}
}

// neighbours come in order of id, lowest first, five at the corners of
// the icosahedron and six elsewhere.
func TestSphereNeighboursAreInOrderOfID(t *testing.T) {
	tl := Sphere{N: 4}
	for _, tile := range tl.Tiles() {
		ns := tl.Neighbours(tile)
		for k := 1; k < len(ns); k++ {
			if tl.ID(ns[k-1]) >= tl.ID(ns[k]) {
				t.Fatalf("tile %v: %v comes before %v", tile, ns[k-1], ns[k])
			}
		}
	}
}
