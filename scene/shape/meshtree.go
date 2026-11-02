package shape

import (
	"math"
	"sort"

	"github.com/janearc/slayer/maths"
)

// node is one box of the tree. An inner node's first child follows it
// and next is its second; a leaf holds count tris from next.
type node struct {
	box         maths.Box
	next, count int32
}

const (
	bins     = 16 // places a node's faces are sorted into, to cut it
	leafMost = 8  // faces a leaf may hold, when cutting saves nothing
	deepest  = 64 // past this depth a node is cut in half by count
)

// builder is the tree as it is made: the faces' boxes and middles, and
// their order, which the cuts sort.
type builder struct {
	boxes []maths.Box
	mids  []maths.Vec
	order []int32
	nodes []node
	pad   float64
}

// build makes the tree, once: each node cut in two where the surface area
// heuristic says searching the halves costs least, which is the chance a
// ray meets each half's box times the faces in it.
func (m *Mesh) build() {
	n := len(m.Faces)
	b := builder{
		boxes: make([]maths.Box, n),
		mids:  make([]maths.Vec, n),
		order: make([]int32, n),
	}
	m.box = empty()
	for i, f := range m.Faces {
		u, v, w := m.Vertices[f[0]], m.Vertices[f[1]], m.Vertices[f[2]]
		b.boxes[i] = maths.Box{Min: least(u, least(v, w)),
			Max: most(u, most(v, w))}
		b.mids[i] = u.Add(v).Add(w).Scale(1.0 / 3)
		b.order[i] = int32(i)
		m.box = join(m.box, b.boxes[i])
	}
	// every box is let out by a hair, so that a ray grazing a face's
	// edge is never turned away by the face's box
	size := m.box.Max.Sub(m.box.Min)
	b.pad = 1e-9 * math.Max(size.X, math.Max(size.Y, size.Z))
	b.split(0, n, 0)
	m.nodes = b.nodes
	m.tris = make([]tri, n)
	for i, j := range b.order {
		f := m.Faces[j]
		t := tri{face: j}
		for k, v := range f {
			p := m.Vertices[v]
			t.p[k] = [3]float64{p.X, p.Y, p.Z}
		}
		m.tris[i] = t
	}
}

// split makes the node for the faces order[lo:hi], and its children.
func (b *builder) split(lo, hi, depth int) {
	i := len(b.nodes)
	b.nodes = append(b.nodes, node{})
	box, mids := empty(), empty()
	for _, f := range b.order[lo:hi] {
		box = join(box, b.boxes[f])
		mids = join(mids, maths.Box{Min: b.mids[f], Max: b.mids[f]})
	}
	pad := maths.V(b.pad, b.pad, b.pad)
	b.nodes[i].box = maths.Box{Min: box.Min.Sub(pad), Max: box.Max.Add(pad)}
	n := hi - lo
	if n <= 2 {
		b.nodes[i].next, b.nodes[i].count = int32(lo), int32(n)
		return
	}
	axis := longest(mids)
	cut := lo
	if depth < deepest && along(mids.Max, axis) > along(mids.Min, axis) {
		cut = b.cheapest(lo, hi, axis, mids, box)
	}
	if cut == lo {
		if n <= leafMost {
			b.nodes[i].next, b.nodes[i].count = int32(lo), int32(n)
			return
		}
		part := b.order[lo:hi]
		sort.Slice(part, func(x, y int) bool {
			return along(b.mids[part[x]], axis) <
				along(b.mids[part[y]], axis)
		})
		cut = lo + n/2
	}
	b.split(lo, cut, depth+1)
	b.nodes[i].next = int32(len(b.nodes))
	b.split(cut, hi, depth+1)
}

// cheapest sorts order[lo:hi] into bins by their middles along axis, and
// cuts between the bins where the two halves are cheapest to search. It
// is the index of the cut, with the nearer half first, or lo when one
// leaf of them all is cheaper still.
func (b *builder) cheapest(lo, hi, axis int, mids, box maths.Box) int {
	from := along(mids.Min, axis)
	width := along(mids.Max, axis) - from
	bin := func(f int32) int {
		k := int(bins * (along(b.mids[f], axis) - from) / width)
		return min(k, bins-1)
	}
	var count [bins]int
	var boxes [bins]maths.Box
	for k := range boxes {
		boxes[k] = empty()
	}
	for _, f := range b.order[lo:hi] {
		k := bin(f)
		count[k]++
		boxes[k] = join(boxes[k], b.boxes[f])
	}
	var aboveArea [bins]float64
	var above [bins]int
	acc, c := empty(), 0
	for k := bins - 1; k > 0; k-- {
		acc, c = join(acc, boxes[k]), c+count[k]
		aboveArea[k], above[k] = area(acc), c
	}
	best, at := math.Inf(1), 0
	acc, c = empty(), 0
	for k := 1; k < bins; k++ {
		acc, c = join(acc, boxes[k-1]), c+count[k-1]
		if c == 0 || above[k] == 0 {
			continue
		}
		if cost := area(acc)*float64(c) +
			aboveArea[k]*float64(above[k]); cost < best {
			best, at = cost, k
		}
	}
	// a leaf costs a test for each face; a cut, a box and its halves
	n := hi - lo
	if at == 0 || (n <= leafMost && 1+best/area(box) >= float64(n)) {
		return lo
	}
	part, j := b.order[lo:hi], 0
	for i, f := range part {
		if bin(f) < at {
			part[i], part[j] = part[j], part[i]
			j++
		}
	}
	return lo + j
}

// empty is the box with nothing in it, which anything joined to it fills.
func empty() maths.Box {
	in, out := math.Inf(1), math.Inf(-1)
	return maths.Box{Min: maths.V(in, in, in), Max: maths.V(out, out, out)}
}

// join is the box holding both.
func join(a, b maths.Box) maths.Box {
	return maths.Box{Min: least(a.Min, b.Min), Max: most(a.Max, b.Max)}
}

// area is a box's surface, which is how likely a ray is to meet it.
func area(b maths.Box) float64 {
	d := b.Max.Sub(b.Min)
	return 2 * (d.X*d.Y + d.Y*d.Z + d.Z*d.X)
}

// longest is the axis along which the box is longest: 0, 1 or 2 for x, y
// or z.
func longest(b maths.Box) int {
	d := b.Max.Sub(b.Min)
	switch {
	case d.X >= d.Y && d.X >= d.Z:
		return 0
	case d.Y >= d.Z:
		return 1
	}
	return 2
}

// along is v's part along axis 0, 1 or 2.
func along(v maths.Vec, axis int) float64 {
	switch axis {
	case 0:
		return v.X
	case 1:
		return v.Y
	}
	return v.Z
}

// built is the tree, and a panic for a mesh that did not come from
// NewMesh: without its tree a mesh can answer nothing truly.
func (m *Mesh) built() []node {
	if len(m.nodes) == 0 {
		panic("shape: a Mesh is made by NewMesh, which builds its tree")
	}
	return m.nodes
}

// todo is a node still to look in, and how near it could be.
type todo struct {
	n int32
	d float64
}

// first is the nearest face the ray meets between near and far: how far
// along, and which, in the tree's order.
func (m *Mesh) first(r maths.Ray, near, far float64) (float64, int, bool) {
	nodes := m.built()
	q := ready(r)
	if _, ok := q.box(&nodes[0].box, near, far); !ok {
		return 0, -1, false
	}
	var stack [128]todo // deeper than the tree can be
	top, n, best, which := 0, int32(0), far, -1
	for {
		nd := &nodes[n]
		if nd.count > 0 {
			for i := nd.next; i < nd.next+nd.count; i++ {
				t, _, ok := q.meet(&m.tris[i], near, best)
				if ok {
					best, which = t, int(i)
				}
			}
		} else {
			a, b := n+1, nd.next
			ta, inA := q.box(&nodes[a].box, near, best)
			tb, inB := q.box(&nodes[b].box, near, best)
			if inA && inB {
				if tb < ta {
					a, b, ta, tb = b, a, tb, ta
				}
				stack[top] = todo{b, tb}
				top++
			}
			if inA || inB {
				if !inA {
					a = b
				}
				n = a
				continue
			}
		}
		for n = -1; top > 0 && n < 0; {
			top--
			if stack[top].d <= best {
				n = stack[top].n
			}
		}
		if n < 0 && which < 0 {
			return 0, -1, false
		}
		if n < 0 {
			return best, which, true
		}
	}
}

// winding is how many more faces the ray leaves through than it enters
// by, beyond its start.
func (m *Mesh) winding(r maths.Ray) int {
	nodes := m.built()
	q := ready(r)
	var stack [128]int32
	top, wind := 1, 0
	for top > 0 {
		top--
		nd := &nodes[stack[top]]
		if _, ok := q.box(&nd.box, 0, math.Inf(1)); !ok {
			continue
		}
		if nd.count == 0 {
			stack[top], stack[top+1] = stack[top]+1, nd.next
			top += 2
			continue
		}
		for i := nd.next; i < nd.next+nd.count; i++ {
			_, front, ok := q.meet(&m.tris[i], 0, math.Inf(1))
			switch {
			case ok && front:
				wind--
			case ok:
				wind++
			}
		}
	}
	return wind
}

// nearest is the face nearest p, in the tree's order, with the weights
// on its corners of its point nearest p, and how far that is, squared.
func (m *Mesh) nearest(p maths.Vec) (int, [3]float64, float64) {
	nodes := m.built()
	var stack [128]todo
	top, n, best, which := 0, int32(0), math.Inf(1), 0
	var weights [3]float64
	for {
		nd := &nodes[n]
		if nd.count > 0 {
			for i := nd.next; i < nd.next+nd.count; i++ {
				t := &m.tris[i]
				a, b, c := vec(t.p[0]), vec(t.p[1]), vec(t.p[2])
				w := closest(p, a, b, c)
				d := a.Scale(w[0]).Add(b.Scale(w[1])).
					Add(c.Scale(w[2])).Sub(p)
				if d2 := d.Dot(d); d2 < best {
					best, which, weights = d2, int(i), w
				}
			}
		} else {
			a, b := n+1, nd.next
			da, db := away(nodes[a].box, p), away(nodes[b].box, p)
			if db < da {
				a, b, da, db = b, a, db, da
			}
			if db < best {
				stack[top] = todo{b, db}
				top++
			}
			if da < best {
				n = a
				continue
			}
		}
		for n = -1; top > 0 && n < 0; {
			top--
			if stack[top].d < best {
				n = stack[top].n
			}
		}
		if n < 0 {
			return which, weights, best
		}
	}
}

// away is how far p is from box b, squared, and nothing inside it.
func away(b maths.Box, p maths.Vec) float64 {
	d := maths.V(
		math.Max(0, math.Max(b.Min.X-p.X, p.X-b.Max.X)),
		math.Max(0, math.Max(b.Min.Y-p.Y, p.Y-b.Max.Y)),
		math.Max(0, math.Max(b.Min.Z-p.Z, p.Z-b.Max.Z)),
	)
	return d.Dot(d)
}

// box is where the ray enters b, no nearer than near, and whether it
// does before far. A slab's distances are a little uncertain, so the far
// side is let out by a hair: a ray that grazes a box is never turned away
// from the face at its edge.
func (q *query) box(b *maths.Box, near, far float64) (float64, bool) {
	enter, leave := slab(b.Min.X, b.Max.X, q.at.X, q.inv.X, near, far)
	enter, leave = slab(b.Min.Y, b.Max.Y, q.at.Y, q.inv.Y, enter, leave)
	enter, leave = slab(b.Min.Z, b.Max.Z, q.at.Z, q.inv.Z, enter, leave)
	return enter, enter <= leave+math.Abs(leave)*1e-12
}

// slab narrows enter and leave to where the ray is between lo and hi on
// one axis. A ray that runs along a slab's side gives no number, which
// narrows nothing.
func slab(lo, hi, at, inv, enter, leave float64) (float64, float64) {
	if inv < 0 {
		lo, hi = hi, lo
	}
	if t := (lo - at) * inv; t > enter {
		enter = t
	}
	if t := (hi - at) * inv; t < leave {
		leave = t
	}
	return enter, leave
}
