package mesh

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// the numbers of the slab test's loosening.
const (
	// defined by IEEE 754 binary64: the most one rounding is out, half
	// a unit in the last place of one.
	roundoff = 0x1p-53
	// defined by ize: the roundings a slab's distance takes, the number
	// his gamma is of.
	slabRoundings = 3
	// defined by ize: the gammas of that number a ray is let in past a
	// box by.
	slabGammas = 2
)

// loosened is how far past a box's far side a ray is still let in: one
// and two gammas of three, ize's "robust bvh ray traversal" (2013), so
// the rounding of a slab never turns a ray away from a face at the
// box's edge.
const loosened = unity + slabGammas*(slabRoundings*roundoff/
	(unity-slabRoundings*roundoff))

// weights is how much of a point on a face is each of its corners.
type weights [triangleCorners]float64

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
	from, inverse          [axes]float64
	entry                  [axes]int
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
		look.inverse[axis] = math.Inf(plusInfinity)
		if part != nought {
			look.inverse[axis] = unity / part
		}
		if look.inverse[axis] < nought {
			look.entry[axis] = high
		}
	}
	look.kz = alongX
	for axis := alongY; axis < axes; axis++ {
		if math.Abs(way[axis]) > math.Abs(way[look.kz]) {
			look.kz = axis
		}
	}
	look.kx, look.ky = (look.kz+unity)%axes, (look.kz+pair)%axes
	if way[look.kz] < nought {
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
	ex, ey, ez := look.entry[alongX], look.entry[alongY], look.entry[alongZ]
	enter, leave := near, far
	if slab := (box[ex][alongX] - from[alongX]) *
		inverse[alongX]; slab > enter {
		enter = slab
	}
	if slab := (box[high-ex][alongX] - from[alongX]) *
		inverse[alongX]; slab < leave {
		leave = slab
	}
	if slab := (box[ey][alongY] - from[alongY]) *
		inverse[alongY]; slab > enter {
		enter = slab
	}
	if slab := (box[high-ey][alongY] - from[alongY]) *
		inverse[alongY]; slab < leave {
		leave = slab
	}
	if slab := (box[ez][alongZ] - from[alongZ]) *
		inverse[alongZ]; slab > enter {
		enter = slab
	}
	if slab := (box[high-ez][alongZ] - from[alongZ]) *
		inverse[alongZ]; slab < leave {
		leave = slab
	}
	return enter, enter <= leave*loosened &&
		!math.IsInf(enter, plusInfinity)
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
	if (forOne < nought || forTwo < nought || forThree < nought) &&
		(forOne > nought || forTwo > nought || forThree > nought) {
		return nought, weights{}, false
	}
	whole := forOne + forTwo + forThree
	if whole == nought {
		return nought, weights{}, false
	}
	along := (forOne*az + forTwo*bz + forThree*cz) / whole
	if !(along > near && along < far) {
		return nought, weights{}, false
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
	made.ensureTree()
	look := aimed(ray)
	best := Met{along: far, face: noFace}
	enter, in := look.enters(&made.tree.nodes[nought], near, far)
	if !in {
		return best, false
	}
	var stack [deepest + pushed]waiting
	stack[nought] = waiting{node: nought, enter: enter}
	for top := unity; top > nought; {
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
		if at.count > nought {
			best = made.leaf(&look, at, near, far, best)
			continue
		}
		top = look.descend(made.tree.nodes, at.first, near, best.along,
			&stack, top)
	}
	return best, best.face >= nought
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
// nearer on top, and is the stack's new height. The children are at
// child and the node after it.
func (look *sight) descend(nodes []node, child int, near, far float64,
	stack *[deepest + pushed]waiting, top int) int {
	left, inLeft := look.enters(&nodes[child], near, far)
	right, inRight := look.enters(&nodes[child+unity], near, far)
	if inLeft && inRight && right < left {
		stack[top] = waiting{node: child, enter: left}
		stack[top+unity] = waiting{node: child + unity, enter: right}
		return top + pushed
	}
	if inRight {
		stack[top] = waiting{node: child + unity, enter: right}
		top++
	}
	if inLeft {
		stack[top] = waiting{node: child, enter: left}
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

// RestPointAt is the corresponding point on the undeformed surface,
// using the same triangle and barycentric weights as the ray hit.
func (made *Mesh) RestPointAt(met Met) (maths.Vec, bool) {
	return made.VertexPointAt(met, made.rest)
}

// PosedSlope maps a scalar height gradient in the rest chart onto the hit's
// posed tangent plane. Its dot product with a posed edge equals the rest
// gradient's dot product with the corresponding rest edge.
func (made *Mesh) PosedSlope(met Met, gradient maths.Vec) maths.Vec {
	if len(made.rest) == nought {
		return gradient
	}
	at := made.faces[met.face]
	one, two, three := made.Corners(met.face)
	a, b := two.Sub(one), three.Sub(one)
	cross := a.Cross(b)
	squared := cross.Dot(cross)
	if !(squared > nought) {
		return maths.Vec{}
	}
	rest := made.rest[at[first]]
	du := gradient.Dot(made.rest[at[second]].Sub(rest))
	dv := gradient.Dot(made.rest[at[third]].Sub(rest))
	return b.Cross(cross).Scale(du / squared).Add(cross.Cross(a).Scale(dv / squared))
}

// VertexPointAt interpolates a field indexed by this mesh's vertices.
// The caller supplies one value per vertex and a hit on this mesh.
func (made *Mesh) VertexPointAt(met Met, field []maths.Vec) (maths.Vec, bool) {
	if len(field) == nought || len(field) != len(made.points) {
		return maths.V(nought, nought, nought), false
	}
	corners := made.faces[met.face]
	atFirst, atSecond, atThird := field[corners[first]],
		field[corners[second]], field[corners[third]]
	return atFirst.Scale(met.weights[first]).
		Add(atSecond.Scale(met.weights[second])).
		Add(atThird.Scale(met.weights[third])), true
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
		under := math.Min(nought, point.Sub(made.points[corner]).Dot(normal))
		lifted = lifted.Sub(normal.Scale(met.weights[k] * under))
	}
	return lifted
}

// facing is the way face i faces at a point given by its corners'
// weights.
func (made *Mesh) facing(i int, at weights) maths.Vec {
	if made.smooth {
		corners := made.faces[i]
		blend := made.normals[corners[first]].Scale(at[first]).
			Add(made.normals[corners[second]].Scale(at[second])).
			Add(made.normals[corners[third]].Scale(at[third]))
		if somewhere(blend) {
			return blend.Unit()
		}
	}
	return made.across(i).Unit()
}
