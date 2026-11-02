package hexes

import (
	"math"
	"sort"

	"github.com/janearc/slayer/maths"
)

// Sphere is the unit sphere with each icosahedron face split N times:
// 10N^2+2 tiles, about Spacing apart. The zero Sphere is split once, the
// bare icosahedron's twelve pentagons. It holds nothing but N, so a
// tiling of any resolution costs nothing to make.
//
// A tile is named by the lattice corner at its middle: on face Face, I
// steps toward the face's second corner and J toward its third, with I+J
// at most N. A corner shared by several faces is named on the lowest
// numbered of them, so every tile has one name.
//
// Points are directions: Nearest takes any vector and reads its direction,
// Centre is a unit vector, and Spacing and Edge are chords of the unit
// sphere.
type Sphere struct{ N int }

// n is the split, with zero read as one.
func (t Sphere) n() int {
	if t.N < 1 {
		return 1
	}
	return t.N
}

// edge is the angle an icosahedron's edge subtends at its middle.
var edge = math.Atan(2)

// Spacing is about how far apart neighbouring middles are, in radians:
// the icosahedron's edge over N. Measured, neighbours are between 0.8 and
// 1.2 of it, closest near the pentagons and furthest mid-face.
func (t Sphere) Spacing() float64 { return edge / float64(t.n()) }

// finest is the most a sphere is split: its count of tiles, ten N squared
// and two, is the most an int64 holds with room to spare.
const finest = 500_000_000

// SphereFor is the sphere tiling whose tiles are about radians apart. a spacing
// that is no number is the coarsest sphere, and one of zero or less, or finer
// than finest splits, the finest.
func SphereFor(radians float64) Sphere {
	if math.IsNaN(radians) {
		return Sphere{N: 1}
	}
	if radians <= edge/finest {
		return Sphere{N: finest}
	}
	return Sphere{N: max(1, int(math.Round(edge/radians)))}
}

// Finer is the next rung down: split twice as often. Every middle of this
// tiling is a middle of the finer one, so a thing placed on a tile stays
// put when the rung changes.
//
// past half of finest it stops at finest, and a middle is no longer kept.
// a Sphere made by hand past finest does not count its tiles, which
// overflow.
func (t Sphere) Finer() Tiling { return Sphere{N: min(finest, 2*t.n())} }

// Count is how many tiles there are.
func (t Sphere) Count() int {
	n := t.n()
	return 10*n*n + 2
}

// ID is a number for the tile, different for every tile of the tiling and the
// same forever, for hashing what is on it.
func (t Sphere) ID(tile Tile) int64 {
	n := int64(t.n())
	i, j := int64(tile.I), int64(tile.J)
	perFace := (n + 1) * (n + 2) / 2
	return int64(tile.Face)*perFace + i*(n+1) - i*(i-1)/2 + j
}

// Centre is the middle of a tile: a unit direction.
func (t Sphere) Centre(tile Tile) maths.Vec {
	return t.corner(tile.Face, tile.I, tile.J)
}

// Nearest is the tile a direction is in.
func (t Sphere) Nearest(p maths.Vec) Tile {
	p = p.Unit()
	n := t.n()
	f := face(p)
	_, v, w := bary(f, p)
	fi, fj := v*float64(n), w*float64(n)
	i0, j0 := int(math.Floor(fi)), int(math.Floor(fj))
	// the small triangle the direction crosses points up or down; its
	// three corners are the candidates
	cands := [3][2]int{{i0, j0}, {i0 + 1, j0}, {i0, j0 + 1}}
	if fi-float64(i0)+fj-float64(j0) > 1 {
		cands = [3][2]int{{i0 + 1, j0}, {i0, j0 + 1}, {i0 + 1, j0 + 1}}
	}
	best, bi, bj := math.Inf(1), 0, 0
	for _, c := range cands {
		i, j := clamp(c[0], c[1], n)
		if d := apart(p, t.corner(f, i, j)); d < best {
			best, bi, bj = d, i, j
		}
	}
	tile := t.walk(p, f, bi, bj, best)
	return t.settle(p, tile)
}

// apart is how far two unit directions are, as the square of the chord. A dot
// product is one less a hair for every neighbour at a fine rung and cannot tell
// them apart; the chord keeps its digits.
func apart(a, b maths.Vec) float64 {
	d := a.Sub(b)
	return d.Dot(d)
}

// walk steps to a neighbouring corner of the same face while one is nearer,
// and stops at the nearest. In practice it takes no step at all: none in
// 320,000 points tried; the bound is for safety.
func (t Sphere) walk(p maths.Vec, f, i, j int, best float64) Tile {
	n := t.n()
	for range 8 {
		ni, nj, moved := i, j, false
		for _, s := range steps {
			ci, cj := i+s[0], j+s[1]
			if ci < 0 || cj < 0 || ci+cj > n {
				continue
			}
			if d := apart(p, t.corner(f, ci, cj)); d < best {
				best, ni, nj, moved = d, ci, cj, true
			}
		}
		if !moved {
			break
		}
		i, j = ni, nj
	}
	return t.canonical(f, i, j)
}

// settle is one more look, across faces: a corner on the edge of a face
// has neighbours on the next face, and one of them may be nearer still.
func (t Sphere) settle(p maths.Vec, tile Tile) Tile {
	if t.inside(tile) {
		return tile
	}
	best := apart(p, t.Centre(tile))
	var buf [6]Tile
	for _, nb := range buf[:t.near(tile, &buf)] {
		if d := apart(p, t.Centre(nb)); d < best {
			best, tile = d, nb
		}
	}
	return tile
}

// steps are the six lattice directions from a corner.
var steps = [6][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, -1}, {-1, 1}}

// clamp keeps a corner on its face: i and j at least zero, i+j at most n.
func clamp(i, j, n int) (int, int) {
	i, j = max(i, 0), max(j, 0)
	if i+j > n {
		j = max(n-i, 0)
		i = n - j
	}
	return i, j
}

// inside is whether a tile's corner is strictly inside its face, where it
// belongs to that face alone.
func (t Sphere) inside(tile Tile) bool {
	return tile.I > 0 && tile.J > 0 && tile.I+tile.J < t.n()
}

// corner is lattice corner (i, j) of face f, pushed out to the sphere.
func (t Sphere) corner(f, i, j int) maths.Vec {
	n := t.n()
	a, b, c := verts[faces[f][0]], verts[faces[f][1]], verts[faces[f][2]]
	return a.Scale(float64(n - i - j)).
		Add(b.Scale(float64(i))).
		Add(c.Scale(float64(j))).Unit()
}

// canonical is the one name of a corner: itself when it is inside its
// face, and otherwise the same corner on the lowest numbered face it is on.
func (t Sphere) canonical(f, i, j int) Tile {
	tile := Tile{f, i, j}
	if t.inside(tile) {
		return tile
	}
	q := t.corner(f, i, j)
	for g := 0; g < f; g++ {
		if gi, gj, ok := t.on(g, q); ok {
			return Tile{g, gi, gj}
		}
	}
	return tile
}

// on is the lattice corner of face g that a direction is, if it is one.
func (t Sphere) on(g int, q maths.Vec) (i, j int, ok bool) {
	if q.Dot(normals[g]) < 0.7 {
		return 0, 0, false // not near that face at all
	}
	n := float64(t.n())
	u, v, w := bary(g, q)
	const slack = 1e-9
	if u < -slack || v < -slack || w < -slack {
		return 0, 0, false
	}
	fi, fj := v*n, w*n
	i, j = int(math.Round(fi)), int(math.Round(fj))
	if math.Abs(fi-float64(i)) > 1e-6 || math.Abs(fj-float64(j)) > 1e-6 {
		return 0, 0, false
	}
	return i, j, true
}

// near fills buf with the tile's neighbours, five or six, and says how
// many: the lattice neighbours on its own face, and on every other face
// the corner is on.
func (t Sphere) near(tile Tile, buf *[6]Tile) int {
	k := 0
	add := func(f, i, j int) {
		if i < 0 || j < 0 || i+j > t.n() {
			return
		}
		c := t.canonical(f, i, j)
		for _, have := range buf[:k] {
			if have == c {
				return
			}
		}
		if k < len(buf) {
			buf[k] = c
			k++
		}
	}
	for _, s := range steps {
		add(tile.Face, tile.I+s[0], tile.J+s[1])
	}
	if t.inside(tile) {
		return k
	}
	q := t.Centre(tile)
	for g := range faces {
		if g == tile.Face {
			continue
		}
		if gi, gj, ok := t.on(g, q); ok {
			for _, s := range steps {
				add(g, gi+s[0], gj+s[1])
			}
		}
	}
	return k
}

// Neighbours is the tiles sharing an edge with this one: five for a
// pentagon, six for a hexagon, in order of ID.
func (t Sphere) Neighbours(tile Tile) []Tile {
	var buf [6]Tile
	out := append([]Tile(nil), buf[:t.near(tile, &buf)]...)
	sort.Slice(out, func(a, b int) bool {
		return t.ID(out[a]) < t.ID(out[b])
	})
	return out
}

// Edge is how far a direction is from the edge of a tile, as a chord of the
// unit sphere: zero on the edge, growing toward the middle, and less than zero
// outside the tile. The edge with a neighbour is the plane square to the chord
// between the two middles, halfway along it.
func (t Sphere) Edge(p maths.Vec, tile Tile) float64 {
	p = p.Unit()
	c := t.Centre(tile)
	var buf [6]Tile
	best := math.Inf(1)
	for _, nb := range buf[:t.near(tile, &buf)] {
		cn := t.Centre(nb)
		d := c.Sub(cn)
		mid := c.Add(cn).Scale(0.5)
		best = math.Min(best, p.Sub(mid).Dot(d)/d.Len())
	}
	return best
}

// Tiles is every tile, in order of ID. It is 10N^2+2 long, so it is for
// coarse tilings: a world's plates are decided on one, a sky's sample is
// not.
func (t Sphere) Tiles() []Tile {
	n := t.n()
	out := make([]Tile, 0, t.Count())
	for f := range faces {
		for i := 0; i <= n; i++ {
			for j := 0; i+j <= n; j++ {
				c := t.canonical(f, i, j)
				if c == (Tile{f, i, j}) {
					out = append(out, c)
				}
			}
		}
	}
	return out
}

// face is which face a direction looks through: the one whose middle is
// nearest, which for an icosahedron is the one the direction crosses.
func face(p maths.Vec) int {
	best, at := -2.0, 0
	for f, n := range normals {
		if d := p.Dot(n); d > best {
			best, at = d, f
		}
	}
	return at
}

// bary is where a direction crosses the plane of face f, as weights on its
// three corners: all three from zero to one inside the face.
func bary(f int, p maths.Vec) (u, v, w float64) {
	fc := planes[f]
	a := verts[faces[f][0]]
	q := p.Scale(normals[f].Dot(a) / normals[f].Dot(p))
	v2 := q.Sub(a)
	d20, d21 := v2.Dot(fc.v0), v2.Dot(fc.v1)
	v = (fc.d11*d20 - fc.d01*d21) / fc.denom
	w = (fc.d00*d21 - fc.d01*d20) / fc.denom
	return 1 - v - w, v, w
}

// plane is what bary needs of a face, worked out once.
type plane struct {
	v0, v1               maths.Vec
	d00, d01, d11, denom float64
}

// The icosahedron: twelve corners on the unit sphere, twenty faces, each
// face's outward middle, and what bary needs of each.
var (
	verts   [12]maths.Vec
	faces   [20][3]int
	normals [20]maths.Vec
	planes  [20]plane
)

// icosahedron fills the tables.
func init() {
	g := (1 + math.Sqrt(5)) / 2
	raw := [12]maths.Vec{
		{X: -1, Y: g}, {X: 1, Y: g}, {X: -1, Y: -g}, {X: 1, Y: -g},
		{Y: -1, Z: g}, {Y: 1, Z: g}, {Y: -1, Z: -g}, {Y: 1, Z: -g},
		{X: g, Z: -1}, {X: g, Z: 1}, {X: -g, Z: -1},
		{X: -g, Z: 1},
	}
	for i, v := range raw {
		verts[i] = v.Unit()
	}
	faces = [20][3]int{
		{0, 11, 5}, {0, 5, 1}, {0, 1, 7}, {0, 7, 10}, {0, 10, 11},
		{1, 5, 9}, {5, 11, 4}, {11, 10, 2}, {10, 7, 6}, {7, 1, 8},
		{3, 9, 4}, {3, 4, 2}, {3, 2, 6}, {3, 6, 8}, {3, 8, 9},
		{4, 9, 5}, {2, 4, 11}, {6, 2, 10}, {8, 6, 7}, {9, 8, 1},
	}
	for f, tri := range faces {
		a, b, c := verts[tri[0]], verts[tri[1]], verts[tri[2]]
		normals[f] = a.Add(b).Add(c).Unit()
		v0, v1 := b.Sub(a), c.Sub(a)
		d00, d01, d11 := v0.Dot(v0), v0.Dot(v1), v1.Dot(v1)
		planes[f] = plane{v0, v1, d00, d01, d11, d00*d11 - d01*d01}
	}
}

// Distance is the angle between two directions one unit long, in radians, by
// the arc cosine of their dot: below about 1.5e-8 it is zero, so at the finest
// rungs neighbours are no distance apart, and a direction that is not one unit
// long is not measured, its dot held to one.
func (t Sphere) Distance(a, b maths.Vec) float64 {
	return math.Acos(math.Max(-1, math.Min(1, a.Dot(b))))
}

// Latitude is how far a direction lies toward either pole along north: zero on
// the equator, one at a pole, for a direction and a north one unit long. others
// are not made so, and give more than one.
func (t Sphere) Latitude(p, north maths.Vec) float64 {
	return math.Abs(p.Dot(north))
}

// Somewhere is the direction (2a-1, 2b-1, 2c-1): the sphere's points
// from three even numbers, thicker toward the cube's corners, which is
// how a world has scattered its plates since its first picture. the
// cube's middle, which is no direction, is up.
func (t Sphere) Somewhere(a, b, c float64) maths.Vec {
	at := maths.V(a*2-1, b*2-1, c*2-1)
	if at == (maths.Vec{}) {
		return maths.Up
	}
	return at.Unit()
}
