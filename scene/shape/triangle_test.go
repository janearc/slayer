package shape

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/janearc/slayer/maths"
)

var forever = math.Inf(1)

// made is a mesh that must be right.
func made(t testing.TB, vs []maths.Vec, fs [][3]int, ns []maths.Vec,
	smooth bool) *Mesh {
	t.Helper()
	m, err := NewMesh(vs, fs, ns, smooth)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// lone is the triangle on the floor of x and y, from the zero out one
// unit along each, turning counterclockwise seen from above.
func lone(t testing.TB) *Mesh {
	return made(t, []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0),
		maths.V(0, 1, 0)}, [][3]int{{0, 1, 2}}, nil, false)
}

// down is a ray from above (x, y), straight down.
func down(x, y float64) maths.Ray {
	return maths.Ray{At: maths.V(x, y, 1), Dir: maths.V(0, 0, -1)}
}

// met is how many of a mesh's faces the ray meets, after near.
func met(m *Mesh, r maths.Ray, near float64) int {
	q, n := ready(r), 0
	for i := range m.tris {
		if _, _, ok := q.meet(&m.tris[i], near, forever); ok {
			n++
		}
	}
	return n
}

func TestATriangleIsMetInsideAndMissedOutside(t *testing.T) {
	m := lone(t)
	if got, ok := m.Hit(down(0.25, 0.25), 0, forever); !ok || got != 1 {
		t.Errorf("inside: hit at %v, %v; want 1", got, ok)
	}
	for _, at := range [][2]float64{{0.75, 0.75}, {-0.1, 0.5},
		{0.5, -0.1}, {2, 2}} {
		if got, ok := m.Hit(down(at[0], at[1]), 0, forever); ok {
			t.Errorf("outside at %v: hit at %v", at, got)
		}
	}
	if got, ok := m.Hit(down(0.25, 0.25), 1, forever); ok {
		t.Errorf("a hit no further than near counts: %v", got)
	}
	if got, ok := m.Hit(down(0.25, 0.25), 0, 1); ok {
		t.Errorf("a hit no nearer than far counts: %v", got)
	}
}

// Both faces count: a ray from beneath meets the triangle as one from
// above does, and the test says which side it met.
func TestATriangleIsMetFromBehind(t *testing.T) {
	m := lone(t)
	under := maths.Ray{At: maths.V(0.25, 0.25, -1), Dir: maths.V(0, 0, 1)}
	if got, ok := m.Hit(under, 0, forever); !ok || got != 1 {
		t.Fatalf("from behind: hit at %v, %v; want 1", got, ok)
	}
	q, p := ready(under), ready(down(0.25, 0.25))
	if _, front, _ := q.meet(&m.tris[0], 0, forever); front {
		t.Error("from beneath, the ray met the outside")
	}
	if _, front, _ := p.meet(&m.tris[0], 0, forever); !front {
		t.Error("from above, the ray met the inside")
	}
}

// A ray parallel to the face never meets it: above it, or in its own
// plane, where it sees the face edge on, with no area to meet.
func TestARayParallelToATriangleMissesIt(t *testing.T) {
	m := lone(t)
	for _, r := range []maths.Ray{
		{At: maths.V(-1, 0.25, 0.5), Dir: maths.V(1, 0, 0)},
		{At: maths.V(-1, 0.25, 0), Dir: maths.V(1, 0, 0)},
		{At: maths.V(-1, -1, 0), Dir: maths.V(1, 1, 0).Unit()},
	} {
		if got, ok := m.Hit(r, 0, forever); ok {
			t.Errorf("%v met it at %v", r, got)
		}
	}
}

// A ray grazing the long edge a hair inside meets the triangle, and a
// hair outside does not; a ray that runs along the edge, in the plane,
// sees it edge on and misses.
func TestARayGrazingAnEdge(t *testing.T) {
	m := lone(t)
	const hair = 1e-12
	for s := 0.05; s < 1; s += 0.05 {
		in, out := down(s-hair, 1-s-hair), down(s+hair, 1-s+hair)
		if _, ok := m.Hit(in, 0, forever); !ok {
			t.Errorf("a hair inside at %v missed", s)
		}
		if _, ok := m.Hit(out, 0, forever); ok {
			t.Errorf("a hair outside at %v met it", s)
		}
	}
	along := maths.Ray{At: maths.V(-1, 2, 0),
		Dir: maths.V(1, -1, 0).Unit()}
	if got, ok := m.Hit(along, 0, forever); ok {
		t.Errorf("along the edge, met at %v", got)
	}
}

// slant is a way down at any slant up to about 35 degrees, or straight
// down for the first of many.
func slant(rng *rand.Rand, i int) maths.Vec {
	if i == 0 {
		return maths.V(0, 0, -1)
	}
	return maths.V(rng.Float64()-0.5, rng.Float64()-0.5, -1).Unit()
}

// The two halves of a square share its diagonal. A ray through any point
// of the diagonal, from either side and at any slant, meets one of them:
// never both and never neither.
func TestARayThroughASharedEdgeMeetsOneFace(t *testing.T) {
	m := made(t, []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0),
		maths.V(1, 1, 0), maths.V(0, 1, 0)},
		[][3]int{{0, 1, 2}, {0, 2, 3}}, nil, false)
	rng := rand.New(rand.NewPCG(1, 2))
	for i := range 4000 {
		s := float64(i%64+1) / 65 // on the diagonal exactly, mostly
		if i >= 2000 {
			s = rng.Float64()
		}
		on, way := maths.V(s, s, 0), slant(rng, i%3)
		if i%2 == 1 {
			way = way.Neg()
		}
		r := maths.Ray{At: on.Sub(way.Scale(2)), Dir: way}
		if n := met(m, r, 0); n != 1 {
			t.Fatalf("a ray through %v going %v met %d faces",
				on, way, n)
		}
	}
}

// fan is six faces about a corner at the zero, raised by rise above its
// ring, turning counterclockwise seen from above; spokes are the ring.
func fan(t testing.TB, rise float64) (*Mesh, []maths.Vec) {
	vs := []maths.Vec{maths.V(0, 0, rise)}
	var fs [][3]int
	for k := range 6 {
		a := float64(k) * math.Pi / 3
		vs = append(vs, maths.V(math.Cos(a), math.Sin(a), 0))
		fs = append(fs, [3]int{0, 1 + k, 1 + (k+1)%6})
	}
	return made(t, vs, fs, nil, false), vs[1:]
}

// A ray through the corner six faces share meets exactly one of them, as
// does a ray through one of the edges between them.
func TestARayThroughASharedCornerMeetsOneFace(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	for _, rise := range []float64{0, 0.2} {
		m, spokes := fan(t, rise)
		corner := maths.V(0, 0, rise)
		for i := range 4000 {
			at := corner
			if i%2 == 1 { // a way along a spoke
				end := spokes[rng.IntN(6)].Sub(corner)
				at = corner.Add(end.Scale(rng.Float64()))
			}
			way := slant(rng, i)
			r := maths.Ray{At: at.Sub(way.Scale(3)), Dir: way}
			if n := met(m, r, 0); n != 1 {
				t.Fatalf("rise %v: a ray through %v going %v "+
					"met %d faces", rise, at, way, n)
			}
		}
	}
}
