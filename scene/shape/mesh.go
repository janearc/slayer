package shape

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// Mesh is a surface made of triangles, centred on its own zero as every
// shape is: the world places it through its motion.
//
// Each face is three corners turning counterclockwise seen from outside,
// so the turn says which side is out, the way a sphere's normal points
// out. A ray meets a face from either side, as it meets a sphere from
// inside.
//
// Faces that meet share their corners exactly, by index or by place,
// and no ray slips between them. A mesh is one material; a thing of
// several is several meshes.
//
// NewMesh makes one: it checks the mesh and builds the tree of boxes that
// finds a ray's face among many quickly. Nothing in it should change
// after: the fields are open, and a caller that changes them leaves the
// tree finding the faces as they were.
type Mesh struct {
	// Vertices are the corners, in the mesh's own space.
	Vertices []maths.Vec
	// Faces are the triangles, each three indices into Vertices.
	Faces [][3]int
	// Normals, if there are any, are which way the surface faces at
	// each vertex, one each and one unit long.
	Normals []maths.Vec
	// Smooth blends Normals across each face; without it, or without
	// Normals, each face is flat.
	Smooth bool

	tris   []tri     // the faces, in the tree's order
	nodes  []node    // the tree, its root first
	box    maths.Box // the box the faces fill
	closed bool      // every edge is shared, so there is an inside
}

// NewMesh checks a mesh and builds it. normals may be nil, for flat
// faces; given, there is one for each vertex, and smooth blends them
// across each face.
//
// A triangle that names a vertex there is not, has no area, or runs an
// edge the same way as its neighbour is an error that names it; so is a
// closed mesh wound inside out.
//
// a mesh with any open edge is not checked for that, so a closed piece
// wound inside out beside an open one passes.
func NewMesh(vertices []maths.Vec, faces [][3]int, normals []maths.Vec,
	smooth bool) (*Mesh, error) {
	if len(faces) == 0 {
		return nil, errors.New("a mesh needs at least one triangle")
	}
	if normals != nil && len(normals) != len(vertices) {
		return nil, fmt.Errorf("%d normals for %d vertices: give one "+
			"for each, or none", len(normals), len(vertices))
	}
	m := &Mesh{
		Vertices: append([]maths.Vec(nil), vertices...),
		Faces:    append([][3]int(nil), faces...),
		Smooth:   smooth,
	}
	if normals != nil {
		m.Normals = make([]maths.Vec, len(normals))
		for i, n := range normals {
			m.Normals[i] = n
			if finite(n) && n != (maths.Vec{}) {
				m.Normals[i] = n.Unit()
			}
		}
	}
	for i := range m.Faces {
		if err := m.check(i); err != nil {
			return nil, err
		}
	}
	if err := m.wind(); err != nil {
		return nil, err
	}
	m.build()
	return m, nil
}

// farthest is as far out as a corner may be: far past any scene, and near
// enough that the tree's sums of corners cannot overflow.
const farthest = 1e150

// near is whether every part of v is within farthest of the middle.
func near(v maths.Vec) bool {
	return math.Abs(v.X) <= farthest && math.Abs(v.Y) <= farthest &&
		math.Abs(v.Z) <= farthest
}

// check is what is wrong with face i, if anything: a vertex it names
// that is not there, is not a point or is too far out, a normal that
// points nowhere, or no area, its corners in a line but for rounding.
func (m *Mesh) check(i int) error {
	f, ns := m.Faces[i], m.Normals
	for _, v := range f {
		switch {
		case v < 0 || v >= len(m.Vertices):
			return fmt.Errorf("triangle %d names vertex %d, "+
				"and there are %d", i, v, len(m.Vertices))
		case !finite(m.Vertices[v]):
			return fmt.Errorf("triangle %d: vertex %d "+
				"is not a finite point", i, v)
		case !near(m.Vertices[v]):
			return fmt.Errorf("triangle %d: vertex %d is too far "+
				"out, past %g", i, v, farthest)
		case ns != nil && (!finite(ns[v]) || ns[v] == maths.Vec{}):
			return fmt.Errorf("triangle %d: the normal at "+
				"vertex %d points nowhere", i, v)
		}
	}
	a, b, c := m.Vertices[f[0]], m.Vertices[f[1]], m.Vertices[f[2]]
	ab, ac := b.Sub(a), c.Sub(a)
	// measured at the edges' own size, so that no square below overflows
	// or vanishes, from the nearest corners to the farthest
	s := math.Max(math.Max(math.Abs(ab.X), math.Abs(ab.Y)),
		math.Max(math.Max(math.Abs(ab.Z), math.Abs(ac.X)),
			math.Max(math.Abs(ac.Y), math.Abs(ac.Z))))
	ab = maths.V(ab.X/s, ab.Y/s, ab.Z/s)
	ac = maths.V(ac.X/s, ac.Y/s, ac.Z/s)
	// the cross's length is the sine of the angle between the edges,
	// times their lengths. below a sine of 1e-12, a cutoff chosen and not
	// rounding itself, rounding can turn the face's normal by as much as
	// a few ten-thousandths of a radian, and the face is refused.
	if area := ab.Cross(ac); s == 0 ||
		!(area.Dot(area) > 1e-24*ab.Dot(ab)*ac.Dot(ac)) {
		return fmt.Errorf("triangle %d has no area to face a way by: "+
			"its corners are in a line, or within 1e-12 of one", i)
	}
	return nil
}

// finite is whether v is somewhere: not nowhere, nor infinitely far.
func finite(v maths.Vec) bool {
	for _, c := range [3]float64{v.X, v.Y, v.Z} {
		if math.IsNaN(c) || math.IsInf(c, 0) {
			return false
		}
	}
	return true
}

// wind checks that neighbours turn the same way, each running the edge
// they share in opposite directions, and learns whether the mesh is
// closed: every edge shared, so that it has an inside. Corners at the
// same place are the same corner, whatever their index.
//
// Each closed piece must turn counterclockwise seen from outside, which
// is the same as enclosing a positive volume.
func (m *Mesh) wind() error {
	at := make(map[maths.Vec]int32, len(m.Vertices))
	id := make([]int32, len(m.Vertices))
	for i, v := range m.Vertices {
		j, ok := at[v]
		if !ok {
			j = int32(len(at))
			at[v] = j
		}
		id[i] = j
	}
	type edge struct{ from, to int32 }
	runs := make(map[edge]int, 3*len(m.Faces))
	for i, f := range m.Faces {
		for k := range 3 {
			from, to := f[k], f[(k+1)%3]
			e := edge{id[from], id[to]}
			if j, ok := runs[e]; ok {
				return fmt.Errorf("triangles %d and %d both "+
					"run from vertex %d to vertex %d: "+
					"neighbours run the edge they "+
					"share in opposite directions",
					j, i, from, to)
			}
			runs[e] = i
		}
	}
	for e := range runs {
		if _, ok := runs[edge{e.to, e.from}]; !ok {
			return nil // an edge on its own: open, with no inside
		}
	}
	m.closed = true

	// a piece is the faces joined by their corners; its volume is the sum
	// of the cones from a point to each face, taken from the middle of
	// the mesh so that the sum loses little to rounding
	piece := make([]int32, len(at))
	for i := range piece {
		piece[i] = int32(i)
	}
	root := func(i int32) int32 {
		for piece[i] != i {
			piece[i] = piece[piece[i]]
			i = piece[i]
		}
		return i
	}
	lo, hi := m.Vertices[m.Faces[0][0]], m.Vertices[m.Faces[0][0]]
	for _, f := range m.Faces {
		for _, v := range f {
			p := m.Vertices[v]
			lo, hi = least(lo, p), most(hi, p)
		}
		piece[root(id[f[1]])] = root(id[f[0]])
		piece[root(id[f[2]])] = root(id[f[0]])
	}
	mid := lo.Add(hi).Scale(0.5)
	volume := make(map[int32]float64)
	for _, f := range m.Faces {
		a := m.Vertices[f[0]].Sub(mid)
		b := m.Vertices[f[1]].Sub(mid)
		c := m.Vertices[f[2]].Sub(mid)
		volume[root(id[f[0]])] += a.Dot(b.Cross(c))
	}
	for i, f := range m.Faces {
		if volume[root(id[f[0]])] < 0 {
			return fmt.Errorf("the closed piece with "+
				"triangle %d in it is inside out: faces "+
				"turn counterclockwise seen from outside", i)
		}
	}
	return nil
}

// Bounds is the box the faces fill.
func (m *Mesh) Bounds() maths.Box {
	m.built()
	return m.box
}

// Closed is whether every edge of the mesh is shared by two faces, so
// that it has an inside.
func (m *Mesh) Closed() bool { return m.closed }

// Hit is the nearest face the ray meets between near and far, by the tree:
// the boxes the ray passes through, nearest first, until no box left could
// hold a nearer face.
func (m *Mesh) Hit(r maths.Ray, near, far float64) (float64, bool) {
	t, _, ok := m.first(r, near, far)
	return t, ok
}

// Normal is which way the face nearest p faces there: the face's own way,
// by the turn of its corners, when it is flat, and the vertices' normals
// blended by where p is on it when it is smooth.
func (m *Mesh) Normal(p maths.Vec) maths.Vec {
	i, w, _ := m.nearest(p)
	t := &m.tris[i]
	a, b, c := vec(t.p[0]), vec(t.p[1]), vec(t.p[2])
	flat := b.Sub(a).Cross(c.Sub(a)).Unit()
	if !m.Smooth || m.Normals == nil {
		return flat
	}
	f := m.Faces[t.face]
	n := m.Normals[f[0]].Scale(w[0]).
		Add(m.Normals[f[1]].Scale(w[1])).
		Add(m.Normals[f[2]].Scale(w[2]))
	if n == (maths.Vec{}) {
		return flat // the corners' normals cancel here
	}
	return n.Unit()
}

// Distance is how far p is from the nearest face, and negative inside.
func (m *Mesh) Distance(p maths.Vec) float64 {
	_, _, d2 := m.nearest(p)
	if m.Inside(p) {
		return -math.Sqrt(d2)
	}
	return math.Sqrt(d2)
}

// look is the way Inside looks from a point: any way would do, since the
// faces are met watertight, and this one lines up with no axis.
var look = maths.V(0.48, 0.6, 0.64)

// Inside is whether p is inside the mesh: whether a ray from p out to
// forever leaves through more faces than it enters by. Pieces that
// overlap are inside each other. A mesh that is not closed encloses
// nothing, and nothing is inside it. A point on a face may be either.
func (m *Mesh) Inside(p maths.Vec) bool {
	if !m.closed {
		return false
	}
	return m.winding(maths.Ray{At: p, Dir: look}) > 0
}

// least is the corner-by-corner smaller of two points.
func least(a, b maths.Vec) maths.Vec {
	return maths.V(math.Min(a.X, b.X), math.Min(a.Y, b.Y),
		math.Min(a.Z, b.Z))
}

// most is the corner-by-corner larger of two points.
func most(a, b maths.Vec) maths.Vec {
	return maths.V(math.Max(a.X, b.X), math.Max(a.Y, b.Y),
		math.Max(a.Z, b.Z))
}
