package mesh

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// loosened is how far past a box's far side a ray is still let in: one
// and two gammas of three, ize's "robust bvh ray traversal" (2013), so
// the rounding of a slab never turns a ray away from a face at the
// box's edge.
const loosened = 1 + 2*(3*0x1p-53/(1-3*0x1p-53))

// weights is how much of a point on a face is each of its corners.
type weights [3]float64

// Met is where a ray meets a mesh: how far along it, which face, and
// where on the face.
type Met struct {
	along   float64
	face    int
	weights weights
}

// Along is how far along the ray the mesh is met.
func (met Met) Along() float64 { return met.along }

// Face is which face the ray met.
func (met Met) Face() int { return met.face }

// sight is a ray made ready to meet many boxes and faces: where it is
// from, one over each part of its way, and the watertight test's own
// frame, its largest part last, with the shear that makes it straight.
type sight struct {
	from, inverse          [3]float64
	entry                  [3]int
	kx, ky, kz             int
	shearX, shearY, shearZ float64
}

// aimed is a ray made ready. a part of its way that is zero, either sign, has
// an endless inverse of one sign, so a ray along a box's side is inside it
// either way. entry is which of a box's bounds on each axis the ray crosses
// first: the high one when it runs downward.
func aimed(ray maths.Ray) sight {
	way := parts(ray.Dir)
	look := sight{from: parts(ray.At)}
	for axis, part := range way {
		look.inverse[axis] = math.Inf(1)
		if part != 0 {
			look.inverse[axis] = 1 / part
		}
		if look.inverse[axis] < 0 {
			look.entry[axis] = 1
		}
	}
	look.kz = 0
	for axis := 1; axis < 3; axis++ {
		if math.Abs(way[axis]) > math.Abs(way[look.kz]) {
			look.kz = axis
		}
	}
	look.kx, look.ky = (look.kz+1)%3, (look.kz+2)%3
	if way[look.kz] < 0 {
		look.kx, look.ky = look.ky, look.kx
	}
	look.shearZ = look.inverse[look.kz]
	look.shearX = way[look.kx] * look.shearZ
	look.shearY = way[look.ky] * look.shearZ
	return look
}

// enters is where the sight enters a node's box, no nearer than near,
// and whether it does before far: each axis's slab narrows the way in,
// the bound crossed first by entry, written out an axis at a time.
func (look *sight) enters(at *node, near, far float64) (float64, bool) {
	box, from, inverse := &at.bounds, &look.from, &look.inverse
	ex, ey, ez := look.entry[0], look.entry[1], look.entry[2]
	enter, leave := near, far
	if slab := (box[ex][0] - from[0]) * inverse[0]; slab > enter {
		enter = slab
	}
	if slab := (box[1-ex][0] - from[0]) * inverse[0]; slab < leave {
		leave = slab
	}
	if slab := (box[ey][1] - from[1]) * inverse[1]; slab > enter {
		enter = slab
	}
	if slab := (box[1-ey][1] - from[1]) * inverse[1]; slab < leave {
		leave = slab
	}
	if slab := (box[ez][2] - from[2]) * inverse[2]; slab > enter {
		enter = slab
	}
	if slab := (box[1-ez][2] - from[2]) * inverse[2]; slab < leave {
		leave = slab
	}
	return enter, enter <= leave*loosened && !math.IsInf(enter, 1)
}

// meets is how far along the sight a face is met, strictly between
// near and far, and its corners' weights there: woop, benthin and
// wald's "watertight ray/triangle intersection" (2013).
//
// each corner is carried into the ray's frame alone, so two faces
// sharing an edge see it alike, and a ray through an edge meets one
// face or both, never neither.
func (look *sight) meets(made *Mesh, i int, near, far float64) (float64,
	weights, bool) {
	one, two, three := made.Corners(i)
	ax, ay, az := look.sheared(one)
	bx, by, bz := look.sheared(two)
	cx, cy, cz := look.sheared(three)
	// each corner's share of the point met, before the whole: the
	// signed area the ray makes with the edge across from it
	forOne := determinant(cx, cy, bx, by)
	forTwo := determinant(ax, ay, cx, cy)
	forThree := determinant(bx, by, ax, ay)
	if (forOne < 0 || forTwo < 0 || forThree < 0) &&
		(forOne > 0 || forTwo > 0 || forThree > 0) {
		return 0, weights{}, false
	}
	whole := forOne + forTwo + forThree
	if whole == 0 {
		return 0, weights{}, false
	}
	along := (forOne*az + forTwo*bz + forThree*cz) / whole
	if !(along > near && along < far) {
		return 0, weights{}, false
	}
	return along, weights{forOne / whole, forTwo / whole,
		forThree / whole}, true
}

// sheared is a corner in the sight's frame: from the ray's start, its
// axes turned so the ray runs along the last, and sheared so the ray
// is that axis. every face sharing a corner shears it by this one
// function, so it lands in the same place for all of them.
func (look *sight) sheared(corner maths.Vec) (float64, float64, float64) {
	at := parts(corner)
	x := at[look.kx] - look.from[look.kx]
	y := at[look.ky] - look.from[look.ky]
	z := at[look.kz] - look.from[look.kz]
	return x - look.shearX*z, y - look.shearY*z, look.shearZ * z
}

// determinant is the determinant of a two by two matrix given row by row: top
// left times bottom right, less top right times bottom left, with its sign
// always right, and zero exactly when it is zero.
//
// it is kahan's algorithm as jeannerod, louvet and muller set it out in
// "further analysis of kahan's algorithm for the accurate computation of 2x2
// determinants" (2013), and they show it within two roundings.
//
// round the second product; find exactly, by a fused multiply and add, what its
// rounding took off; take the rounded second product from the first in one more
// fused step; and add back what was taken off.
func determinant(topLeft, topRight, bottomLeft,
	bottomRight float64) float64 {
	rounded := float64(topRight * bottomLeft)
	slip := math.FMA(-topRight, bottomLeft, rounded)
	rest := math.FMA(topLeft, bottomRight, -rounded)
	return rest + slip
}

// waiting is a node still to look in, and how near the ray enters it.
type waiting struct {
	node  int
	enter float64
}

// Meet is the nearest face a ray meets strictly between near and far,
// by the tree, nearer boxes first; of two met at the same distance, the
// first face.
func (made *Mesh) Meet(ray maths.Ray, near, far float64) (Met, bool) {
	look := aimed(ray)
	best := Met{along: far, face: -1}
	enter, in := look.enters(&made.tree.nodes[0], near, far)
	if !in {
		return best, false
	}
	var stack [deepest + 2]waiting
	stack[0] = waiting{node: 0, enter: enter}
	for top := 1; top > 0; {
		top--
		next := stack[top]
		// passed over only past the best as loosely as a box is let
		// in, so a face as near as the best, in a box whose edge it
		// lies on, is still asked and the tie goes to the first face
		if next.enter > best.along*loosened {
			continue
		}
		count(entered)
		at := &made.tree.nodes[next.node]
		if at.count > 0 {
			best = made.leaf(&look, at, near, far, best)
			continue
		}
		top = look.descend(made.tree.nodes, at.first, near, best.along,
			&stack, top)
	}
	return best, best.face >= 0
}

// leaf is the best of a leaf's faces and the best so far.
func (made *Mesh) leaf(look *sight, at *node, near, far float64,
	best Met) Met {
	for _, i := range made.tree.run(at) {
		count(tried)
		along, weighed, ok := look.meets(made, i, near, far)
		if ok && best.beaten(along, i) {
			best = Met{along: along, face: i, weights: weighed}
		}
	}
	return best
}

// beaten is whether face i met so far along beats the best: it is
// nearer, or as near and an earlier face.
func (best Met) beaten(along float64, i int) bool {
	return along < best.along || along == best.along && i < best.face
}

// descend puts a node's children the ray enters on the stack, the
// nearer on top, and is the stack's new height.
func (look *sight) descend(nodes []node, first int, near, far float64,
	stack *[deepest + 2]waiting, top int) int {
	left, inLeft := look.enters(&nodes[first], near, far)
	right, inRight := look.enters(&nodes[first+1], near, far)
	if inLeft && inRight && right < left {
		stack[top] = waiting{node: first, enter: left}
		stack[top+1] = waiting{node: first + 1, enter: right}
		return top + 2
	}
	if inRight {
		stack[top] = waiting{node: first + 1, enter: right}
		top++
	}
	if inLeft {
		stack[top] = waiting{node: first, enter: left}
		top++
	}
	return top
}

// Hit is how far along a ray the mesh is first met, strictly between
// near and far, and whether it is.
func (made *Mesh) Hit(ray maths.Ray, near, far float64) (float64, bool) {
	met, ok := made.Meet(ray, near, far)
	return met.along, ok
}

// NormalAt is the way the mesh faces where a ray met it: the face's own
// way, or with smooth normals, its corners' blended by where on it.
func (made *Mesh) NormalAt(met Met) maths.Vec {
	return made.facing(met.face, met.weights)
}

// Lifted is a point where a ray met the mesh, raised to where the curved
// surface its smooth normals draw would be, for shadow rays to leave
// from: each corner's tangent plane lifts the point to it if the point
// is under it, by that corner's weight there.
//
// on a flat face, or a mesh shaded flat, it is the point as it is.
//
// a coarse mesh shaded smooth lights a point by its blended normal, and
// a shadow ray leaving its flat face meets the faces beside it, so the
// edge of a light grazing it follows the faces as teeth. hanika, "hacking
// the shadow terminator", ray tracing gems ii, chapter 4 (2021).
func (made *Mesh) Lifted(met Met, point maths.Vec) maths.Vec {
	if !made.smooth {
		return point
	}
	lifted := point
	for k, corner := range made.faces[met.face] {
		normal := made.normals[corner]
		under := math.Min(0, point.Sub(made.points[corner]).Dot(normal))
		lifted = lifted.Sub(normal.Scale(met.weights[k] * under))
	}
	return lifted
}

// facing is the way face i faces at a point given by its corners'
// weights.
func (made *Mesh) facing(i int, at weights) maths.Vec {
	if made.smooth {
		corners := made.faces[i]
		blend := made.normals[corners[0]].Scale(at[0]).
			Add(made.normals[corners[1]].Scale(at[1])).
			Add(made.normals[corners[2]].Scale(at[2]))
		if somewhere(blend) {
			return blend.Unit()
		}
	}
	return made.across(i).Unit()
}
