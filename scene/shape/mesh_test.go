package shape

import (
	"math"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// cube is the cube from -1 to 1 on each axis, each side cut n by n into
// squares and each square into two faces, corner to corner. With split,
// each side has corners of its own, at the same places as its
// neighbours'.
func cube(n int, split bool) ([]maths.Vec, [][3]int) {
	x, y, z := maths.V(1, 0, 0), maths.V(0, 1, 0), maths.V(0, 0, 1)
	sides := [6][3]maths.Vec{ // out, then two ways across it, u by v out
		{x, y, z}, {x.Neg(), z, y}, {y, z, x},
		{y.Neg(), x, z}, {z, x, y}, {z.Neg(), y, x},
	}
	var vs []maths.Vec
	var fs [][3]int
	at := map[maths.Vec]int{}
	corner := func(p maths.Vec) int {
		if i, ok := at[p]; ok && !split {
			return i
		}
		vs = append(vs, p)
		at[p] = len(vs) - 1
		return len(vs) - 1
	}
	s := func(i int) float64 { return -1 + 2*float64(i)/float64(n) }
	for _, side := range sides {
		out, u, v := side[0], side[1], side[2]
		grid := make([][]int, n+1)
		for i := range grid {
			grid[i] = make([]int, n+1)
			for j := range grid[i] {
				p := out.Add(u.Scale(s(i))).Add(v.Scale(s(j)))
				grid[i][j] = corner(p)
			}
		}
		for i := range n {
			for j := range n {
				a, b := grid[i][j], grid[i+1][j]
				c, d := grid[i+1][j+1], grid[i][j+1]
				fs = append(fs, [3]int{a, b, c},
					[3]int{a, c, d})
			}
		}
	}
	return vs, fs
}

// icosphere is the icosahedron in the unit sphere, each face split in
// four level times, every new corner pushed out to the sphere.
func icosphere(level int) ([]maths.Vec, [][3]int) {
	g := (1 + math.Sqrt(5)) / 2
	vs := []maths.Vec{
		maths.V(-1, g, 0), maths.V(1, g, 0), maths.V(-1, -g, 0),
		maths.V(1, -g, 0), maths.V(0, -1, g), maths.V(0, 1, g),
		maths.V(0, -1, -g), maths.V(0, 1, -g), maths.V(g, 0, -1),
		maths.V(g, 0, 1), maths.V(-g, 0, -1), maths.V(-g, 0, 1),
	}
	for i := range vs {
		vs[i] = vs[i].Unit()
	}
	fs := [][3]int{
		{0, 11, 5}, {0, 5, 1}, {0, 1, 7}, {0, 7, 10}, {0, 10, 11},
		{1, 5, 9}, {5, 11, 4}, {11, 10, 2}, {10, 7, 6}, {7, 1, 8},
		{3, 9, 4}, {3, 4, 2}, {3, 2, 6}, {3, 6, 8}, {3, 8, 9},
		{4, 9, 5}, {2, 4, 11}, {6, 2, 10}, {8, 6, 7}, {9, 8, 1},
	}
	for range level {
		fs = quarter(&vs, fs)
	}
	return vs, fs
}

// quarter splits each face in four at its edges' middles, pushed out to
// the unit sphere, each new face turning as the old one did.
func quarter(vs *[]maths.Vec, fs [][3]int) [][3]int {
	mids := map[[2]int]int{}
	mid := func(a, b int) int {
		k := [2]int{min(a, b), max(a, b)}
		if i, ok := mids[k]; ok {
			return i
		}
		*vs = append(*vs, (*vs)[a].Add((*vs)[b]).Unit())
		mids[k] = len(*vs) - 1
		return len(*vs) - 1
	}
	var next [][3]int
	for _, f := range fs {
		a, b, c := mid(f[0], f[1]), mid(f[1], f[2]), mid(f[2], f[0])
		next = append(next, [3]int{f[0], a, c}, [3]int{f[1], b, a},
			[3]int{f[2], c, b}, [3]int{a, b, c})
	}
	return next
}

// anyway is a direction, any one alike.
func anyway(rng *rand.Rand) maths.Vec {
	return maths.V(rng.NormFloat64(), rng.NormFloat64(),
		rng.NormFloat64()).Unit()
}

// within is a point anywhere in the cube from -s to s.
func within(rng *rand.Rand, s float64) maths.Vec {
	return maths.V(s*(2*rng.Float64()-1), s*(2*rng.Float64()-1),
		s*(2*rng.Float64()-1))
}

// inBall is a point anywhere in the ball of radius s.
func inBall(rng *rand.Rand, s float64) maths.Vec {
	for {
		if p := within(rng, s); p.Len() < s {
			return p
		}
	}
}

// aimed is a ray from at toward to.
func aimed(at, to maths.Vec) maths.Ray {
	return maths.Ray{At: at, Dir: to.Sub(at).Unit()}
}

// angle is between two directions, true however small.
func angle(a, b maths.Vec) float64 {
	return math.Atan2(a.Cross(b).Len(), a.Dot(b))
}

// passes is how near the middle the ray's line comes.
func passes(r maths.Ray) float64 {
	return r.At.Sub(r.Dir.Scale(r.At.Dot(r.Dir))).Len()
}

func TestNewMeshNamesTheTriangleAtFault(t *testing.T) {
	sq := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0),
		maths.V(1, 1, 0), maths.V(0, 1, 0)}
	halves := [][3]int{{0, 1, 2}, {0, 2, 3}}
	nowhere := append([]maths.Vec(nil), sq...)
	nowhere[3] = maths.V(0, math.NaN(), 0)
	inline := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 1, 1),
		maths.V(2, 2, 2)}
	flat := []maths.Vec{maths.Up, maths.Up, {}, maths.Up}
	inside, sides := cube(1, false)
	for i := range sides {
		sides[i][1], sides[i][2] = sides[i][2], sides[i][1]
	}
	_, twisted := cube(1, false)
	twisted[5][1], twisted[5][2] = twisted[5][2], twisted[5][1]
	for _, c := range []struct {
		why  string
		vs   []maths.Vec
		fs   [][3]int
		ns   []maths.Vec
		want string
	}{
		{"no faces", sq, nil, nil, "at least one triangle"},
		{"a vertex not there", sq, [][3]int{{0, 1, 2}, {0, 2, 7}},
			nil, "triangle 1 names vertex 7, and there are 4"},
		{"a vertex before the first", sq, [][3]int{{-1, 1, 2}}, nil,
			"triangle 0 names vertex -1"},
		{"a corner twice", sq, [][3]int{{0, 1, 2}, {0, 2, 2}}, nil,
			"triangle 1 has no area"},
		{"corners in a line", inline, [][3]int{{0, 1, 2}}, nil,
			"triangle 0 has no area"},
		{"a corner nowhere", nowhere, halves, nil,
			"triangle 1: vertex 3 is not a finite point"},
		{"a normal short", sq, halves, sq[:3],
			"3 normals for 4 vertices"},
		{"a normal pointing nowhere", sq, halves, flat,
			"triangle 0: the normal at vertex 2 points nowhere"},
		{"neighbours turning apart", sq,
			[][3]int{{0, 1, 2}, {0, 3, 2}}, nil,
			"triangles 0 and 1 both run from vertex 2 to vertex 0"},
		{"one face of a cube turned", inside, twisted, nil,
			"triangles 0 and 5 both run"},
		{"a cube inside out", inside, sides, nil,
			"the closed piece with triangle 0 in it is inside out"},
	} {
		m, err := NewMesh(c.vs, c.fs, c.ns, false)
		switch {
		case err == nil:
			t.Errorf("%s: made a mesh of %d faces", c.why,
				len(m.Faces))
		case !strings.Contains(err.Error(), c.want):
			t.Errorf("%s: %q, wanting %q", c.why, err, c.want)
		}
	}
}

// Faces turn counterclockwise seen from outside, so a flat face's normal
// points out, and a ray from outside meets the outside of the face.
func TestAMeshTurnsCounterclockwiseFromOutside(t *testing.T) {
	vs, fs := icosphere(2)
	ball := made(t, vs, fs, nil, false)
	if !ball.Closed() {
		t.Fatal("the icosphere is not closed")
	}
	for _, f := range fs {
		mid := vs[f[0]].Add(vs[f[1]]).Add(vs[f[2]]).Scale(1.0 / 3)
		if n := ball.Normal(mid); n.Dot(mid) <= 0 {
			t.Fatalf("at %v the normal %v points in", mid, n)
		}
		r := aimed(mid.Scale(3), maths.Vec{})
		_, i, ok := ball.first(r, 0, forever)
		if !ok {
			t.Fatalf("a ray at %v missed", mid)
		}
		q := ready(r)
		if _, front, _ := q.meet(&ball.tris[i], 0, forever); !front {
			t.Fatalf("a ray from outside at %v met the inside", mid)
		}
	}
	cvs, cfs := cube(2, true)
	box := made(t, cvs, cfs, nil, false)
	for k := range 6 {
		out := cubeSide(k)
		at := out.Add(maths.V(out.Y, out.Z, out.X).Scale(0.3)).
			Add(maths.V(out.Z, out.X, out.Y).Scale(0.6))
		if n := box.Normal(at); !n.Near(out, 1e-12) {
			t.Errorf("the cube's side at %v faces %v", at, n)
		}
	}
}

// Inside and Distance on the cube are the box's own: inside is within it,
// and the distance is the box's distance, less than nothing inside.
func TestInsideAndDistanceOnACube(t *testing.T) {
	vs, fs := cube(3, false)
	m := made(t, vs, fs, nil, false)
	rng := rand.New(rand.NewPCG(5, 6))
	for range 5000 {
		p := within(rng, 2)
		q := maths.V(math.Abs(p.X)-1, math.Abs(p.Y)-1, math.Abs(p.Z)-1)
		gap := maths.V(math.Max(q.X, 0), math.Max(q.Y, 0),
			math.Max(q.Z, 0)).Len()
		want := gap + math.Min(math.Max(q.X, math.Max(q.Y, q.Z)), 0)
		if in := m.Inside(p); in != (want < 0) {
			t.Fatalf("at %v, inside %v; want %v", p, in, want < 0)
		}
		if got := m.Distance(p); math.Abs(got-want) > 1e-12 {
			t.Fatalf("at %v, %v from the surface; want %v",
				p, got, want)
		}
	}
}

// Two cubes in one mesh, overlapping, are inside each other: a point in
// both is inside, as a point in either is. Counting faces crossed in
// twos would say the overlap was out.
func TestOverlappingPiecesAreInsideEachOther(t *testing.T) {
	vs, fs := cube(1, false)
	n, k := len(vs), len(fs)
	for _, v := range vs[:n] {
		vs = append(vs, v.Add(maths.V(1, 0.5, 0.25)))
	}
	for _, f := range fs[:k] {
		fs = append(fs, [3]int{f[0] + n, f[1] + n, f[2] + n})
	}
	m := made(t, vs, fs, nil, false)
	for _, c := range []struct {
		at   maths.Vec
		want bool
	}{
		{maths.V(-0.5, 0, 0), true}, {maths.V(0.5, 0.5, 0.5), true},
		{maths.V(1.5, 0.5, 0.5), true}, {maths.V(0, -1.5, 0), false},
		{maths.V(2.5, 0, 0), false},
	} {
		if got := m.Inside(c.at); got != c.want {
			t.Errorf("inside at %v: %v, want %v", c.at, got, c.want)
		}
	}
}

// A mesh with an edge of its own encloses nothing.
func TestAnOpenMeshHasNoInside(t *testing.T) {
	m := lone(t)
	if m.Closed() {
		t.Error("one triangle is closed")
	}
	p := maths.V(0.25, 0.25, -0.5)
	if in, d := m.Inside(p), m.Distance(p); in ||
		math.Abs(d-0.5) > 1e-15 {
		t.Errorf("under it: inside %v, %v away", in, d)
	}
}

// inradius is the nearest any face's plane comes to the zero: the ball
// that size is inside the icosphere, which is inside the unit ball.
func inradius(vs []maths.Vec, fs [][3]int) float64 {
	in := math.Inf(1)
	for _, f := range fs {
		a, b, c := vs[f[0]], vs[f[1]], vs[f[2]]
		n := b.Sub(a).Cross(c.Sub(a)).Unit()
		in = math.Min(in, math.Abs(n.Dot(a)))
	}
	return in
}

// A tessellated sphere against the sphere, over ten thousand rays a
// level, most from outside and a fifth from within.
//
// The icosphere lies between two balls: the unit ball outside it, and
// inside it the ball of its inradius, in. A ray whose line passes nearer
// the middle than in meets both shapes, and one passing further than one
// meets neither; between in and one is the silhouette, where either may
// meet and the other not. Along a ray that meets both, the two surfaces
// are between the two balls, which are sqrt(1-b²) - sqrt(in²-b²) apart
// there, b the line's distance from the middle, so the two distances agree
// that well. The band, one less in, quarters each time the faces split in
// four, and the gap with it.
func TestATessellatedSphereAgreesWithTheSphere(t *testing.T) {
	lastBand, lastGap := math.Inf(1), math.Inf(1)
	for level := 2; level <= 5; level++ {
		band, gap := againstTheSphere(t, level)
		if band > lastBand/3 || gap > lastGap/2 {
			t.Errorf("level %d: the band %v and the gap %v did "+
				"not shrink from %v and %v", level, band, gap,
				lastBand, lastGap)
		}
		lastBand, lastGap = band, gap
	}
}

// againstTheSphere is one level of the icosphere against the sphere: the
// band at its silhouette, and how far apart the distances were for rays
// that pass within 0.9 of the middle.
func againstTheSphere(t *testing.T, level int) (float64, float64) {
	vs, fs := icosphere(level)
	m := made(t, vs, fs, nil, false)
	in, ball := inradius(vs, fs), Sphere{Radius: 1}
	rng := rand.New(rand.NewPCG(7, uint64(level)))
	worst, banded := 0.0, 0
	for i := range 10000 {
		r := aimed(anyway(rng).Scale(3), inBall(rng, 1.2))
		if i%5 == 4 {
			r = maths.Ray{At: inBall(rng, 0.5), Dir: anyway(rng)}
		}
		b := passes(r)
		s, sok := ball.Hit(r, 0, forever)
		got, ok := m.Hit(r, 0, forever)
		switch {
		case b < in && (!ok || !sok):
			t.Fatalf("level %d: %v met the sphere %v, the mesh %v",
				level, r, sok, ok)
		case b < in:
			gap := math.Sqrt(1-b*b) - math.Sqrt(in*in-b*b)
			if d := math.Abs(got - s); d > gap+1e-12 {
				t.Fatalf("level %d: %v apart along %v, past %v",
					level, d, r, gap)
			}
			if b < 0.9 {
				worst = math.Max(worst, math.Abs(got-s))
			}
		case b > 1 && (ok || sok):
			t.Fatalf("level %d: %v, past both, met the sphere %v, "+
				"the mesh %v", level, r, sok, ok)
		case b <= 1:
			banded++
			if ok && !sok {
				t.Fatalf("level %d: %v met the mesh and "+
					"not the sphere around it", level, r)
			}
		}
	}
	t.Logf("level %d, %d faces: band %.2e, %d rays in it; distances "+
		"within %.2e where b < 0.9", level, len(fs), 1-in, banded,
		worst)
	return 1 - in, worst
}

// A closed cube lets no ray through. Rays are aimed at its corners, at
// its edges, at the lines its sides are cut along, corners where six
// faces meet among them, and anywhere on its sides, from outside and
// from within, with corners shared by index and with corners only at
// the same place. Each meets the cube where it was aimed, and a ray from
// outside crosses the surface exactly twice, in and out.
func TestACubeLetsNoRayThrough(t *testing.T) {
	for _, split := range []bool{false, true} {
		vs, fs := cube(4, split)
		m := made(t, vs, fs, nil, false)
		rng := rand.New(rand.NewPCG(8, 9))
		for i := range 10000 {
			aim := cubePoint(rng, i%4)
			out := outward(rng, aim).Scale(1 + 3*rng.Float64())
			if i%5 == 4 {
				throughCube(t, m, within(rng, 0.9), aim, false)
			} else {
				throughCube(t, m, aim.Add(out), aim, true)
			}
		}
		for k := range 6 {
			straightAtASide(t, m, cubeSide(k))
		}
	}
}

// straightAtASide aims rays along an axis at the side of the cube facing
// out, at every corner and line it is cut along and between, where the
// edges' determinants are exactly zero.
func straightAtASide(t *testing.T, m *Mesh, out maths.Vec) {
	u := maths.V(out.Y, out.Z, out.X)
	v := maths.V(out.Z, out.X, out.Y)
	for i := 1; i < 8; i++ {
		for j := 1; j < 8; j++ {
			aim := out.Add(u.Scale(float64(i-4) / 4)).
				Add(v.Scale(float64(j-4) / 4))
			throughCube(t, m, aim.Add(out.Scale(3)), aim, true)
		}
	}
}

// cubeSide is the cube's kth side, by which way it faces.
func cubeSide(k int) maths.Vec {
	s := float64(1 - 2*(k%2))
	return [3]maths.Vec{maths.V(s, 0, 0), maths.V(0, s, 0),
		maths.V(0, 0, s)}[k/2]
}

// cubePoint is a point on the cube's surface: of kind 0, a corner; 1, on
// an edge; 2, on a line a side is cut along, or a square's diagonal; 3,
// anywhere on a side.
func cubePoint(rng *rand.Rand, kind int) maths.Vec {
	grid := func() float64 { return float64(rng.IntN(5)-2) / 2 }
	any := func() float64 { return 2*rng.Float64() - 1 }
	sign := func() float64 { return float64(1 - 2*rng.IntN(2)) }
	var p [3]float64
	switch kind {
	case 0:
		p = [3]float64{sign(), sign(), sign()}
	case 1:
		p = [3]float64{sign(), sign(), any()}
		if rng.IntN(3) == 0 {
			p[2] = float64(rng.IntN(3)-1) / 2
		}
	case 2:
		p = [3]float64{sign(), grid(), any()}
		switch rng.IntN(3) {
		case 0:
			p[2] = grid()
		case 1:
			s := rng.Float64() / 2
			p[1] = float64(rng.IntN(4)-2)/2 + s
			p[2] = float64(rng.IntN(4)-2)/2 + s
		}
	default:
		p = [3]float64{sign(), any(), any()}
	}
	k := rng.IntN(3) // turn it to any side
	return maths.V(p[k], p[(k+1)%3], p[(k+2)%3])
}

// outward is a way out of the cube from p on its surface: out across
// every side p is on, and any way along the others.
func outward(rng *rand.Rand, p maths.Vec) maths.Vec {
	part := func(c float64) float64 {
		if math.Abs(c) == 1 {
			return c * (0.05 + rng.Float64())
		}
		return 2*rng.Float64() - 1
	}
	return maths.V(part(p.X), part(p.Y), part(p.Z)).Unit()
}

// throughCube aims a ray from from at aim on the cube, and fails unless
// it meets the cube there; from outside it must cross twice, from inside
// once.
func throughCube(t *testing.T, m *Mesh, from, aim maths.Vec,
	outside bool) {
	t.Helper()
	r, far := aimed(from, aim), aim.Sub(from).Len()
	got, ok := m.Hit(r, 0, forever)
	if !ok || math.Abs(got-far) > 1e-9 {
		t.Fatalf("aimed at %v from %v: met at %v, %v; want %v",
			aim, from, got, ok, far)
	}
	want := 1
	if outside {
		want = 2
	}
	if n := met(m, r, 0); n != want {
		t.Fatalf("aimed at %v from %v: crossed %d times, not %d",
			aim, from, n, want)
	}
}

// Smooth normals on the icosphere, given each corner's way out: blended,
// they point out through the point exactly, as the corners do, since the
// corners are on the sphere; and the point is near the sphere, so they
// are near the sphere's own normal where the same ray meets the sphere.
//
// The two points are on the ray, as far apart as the distances are,
// which is at most (1-in²)/sqrt(1-b²); seen from the middle, at least in
// away, they are at most twice the arcsine of that over 2in apart. For
// 5,120 faces and rays that pass within 0.9 of the middle that is under
// three tenths of a degree, where flat faces are off by degrees.
func TestSmoothNormalsFollowTheSphere(t *testing.T) {
	vs, fs := icosphere(4)
	smooth := made(t, vs, fs, vs, true)
	flat := made(t, vs, fs, vs, false)
	in, ball := inradius(vs, fs), Sphere{Radius: 1}
	rng := rand.New(rand.NewPCG(12, 13))
	worst, worstFlat, most := 0.0, 0.0, 0.0
	for range 10000 {
		r := aimed(anyway(rng).Scale(3), inBall(rng, 0.9))
		got, ok := smooth.Hit(r, 0, forever)
		s, _ := ball.Hit(r, 0, forever)
		if !ok {
			t.Fatalf("%v missed", r)
		}
		p, want := r.Point(got), ball.Normal(r.Point(s))
		n := smooth.Normal(p)
		if a := angle(n, p); a > 1e-9 {
			t.Fatalf("at %v the blend is %v off the way out", p, a)
		}
		apart := (1 - in*in) / math.Sqrt(1-passes(r)*passes(r))
		bound := 2 * math.Asin(apart/(2*in))
		if a := angle(n, want); a > bound+1e-9 {
			t.Fatalf("at %v the blend is %v off the sphere's, "+
				"past %v", p, a, bound)
		}
		worst = math.Max(worst, angle(n, want))
		worstFlat = math.Max(worstFlat, angle(flat.Normal(p), want))
		most = math.Max(most, bound)
	}
	t.Logf("smooth within %.3f°, bound %.3f°; flat within %.3f°",
		worst/maths.Degree, most/maths.Degree, worstFlat/maths.Degree)
	if worst > 0.3*maths.Degree || worstFlat < 1*maths.Degree {
		t.Errorf("smooth within %v°, flat %v°", worst/maths.Degree,
			worstFlat/maths.Degree)
	}
}

// A mesh is a surface: it answers rays and normals exactly, and says
// nothing of where a ray leaves it, which is for media.
func TestAMeshIsASurfaceOnly(t *testing.T) {
	var s any = lone(t)
	if _, ok := s.(Shape); !ok {
		t.Error("a mesh is not a shape")
	}
	if _, ok := s.(Hitter); !ok {
		t.Error("a mesh is not hit exactly")
	}
	if _, ok := s.(Normaler); !ok {
		t.Error("a mesh does not know its normals")
	}
	if _, ok := s.(Spanner); ok {
		t.Error("a mesh spans, as a medium does")
	}
}

// A Mesh not made by NewMesh has nothing built, and says so rather than
// answering wrongly.
func TestAMeshNotMadeByNewMeshSaysSo(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a mesh not built answered a ray")
		}
	}()
	m := &Mesh{Vertices: []maths.Vec{{}, maths.XAxis, maths.Up},
		Faces: [][3]int{{0, 1, 2}}}
	m.Hit(down(0.1, 0.1), 0, forever)
}
