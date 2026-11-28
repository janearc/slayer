package mesh

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// firstFace is the place of the face Normal answers with when no face
// is nearest.
const firstFace = 0 // defined: the first face of a mesh, which every mesh has

// Normal is the way the mesh faces at a point on it: the way the face
// nearest the point faces there. a ray's Met says which face without
// looking, by NormalAt.
//
// a point with a part that is not a number, or an endless part, or so
// far off that its squared distance to every face is endless, has no
// nearest face. The mesh then faces the way its first face does, flat,
// whatever the mesh's shading.
func (made *Mesh) Normal(point maths.Vec) maths.Vec {
	i, at, _ := made.nearest(point)
	if i < nought {
		return made.across(firstFace).Unit()
	}
	return made.facing(i, at)
}

// Distance is how far a point is from the mesh's nearest face, and less than
// zero inside a closed mesh. telling inside asks every face.
//
// a point that has no nearest face, as Normal says, is endlessly far, and
// never inside: its distance is plus infinity.
func (made *Mesh) Distance(point maths.Vec) float64 {
	face, _, squared := made.nearest(point)
	if face < nought {
		return math.Inf(plusInfinity)
	}
	away := math.Sqrt(squared)
	if made.closed && made.inside(point) {
		return -away
	}
	return away
}

// inside is whether a point is inside a closed mesh: whether the solid
// angles its faces fill around the point come to a whole turn, by van
// oosterom and strackee's formula (1983). an open mesh has no inside,
// and nor has a point outside the box the mesh fills, or one that is
// not a point at all.
func (made *Mesh) inside(point maths.Vec) bool {
	if !made.closed || !made.reaches(point) {
		return false
	}
	var sum float64
	for i := range made.faces {
		sum += made.solidAngle(i, point)
	}
	return sum > pair*math.Pi
}

// reaches is whether a point is in the box the mesh fills, its sides
// included. A part that is not a number is in no box.
func (made *Mesh) reaches(point maths.Vec) bool {
	least, most := parts(made.box.Min), parts(made.box.Max)
	for axis, part := range parts(point) {
		if !(part >= least[axis] && part <= most[axis]) {
			return false
		}
	}
	return true
}

// solidAngle is the solid angle face i fills seen from a point, below zero when
// the point is outside its turn.
func (made *Mesh) solidAngle(i int, point maths.Vec) float64 {
	one, two, three := made.Corners(i)
	one, two, three = one.Sub(point), two.Sub(point), three.Sub(point)
	toOne, toTwo, toThree := one.Len(), two.Len(), three.Len()
	above := one.Dot(two.Cross(three))
	below := toOne*toTwo*toThree + one.Dot(two)*toThree +
		one.Dot(three)*toTwo + two.Dot(three)*toOne
	return pair * math.Atan2(above, below)
}

// nearest is the face nearest a point, by the tree, nearer boxes
// first: the face, its corners' weights at its point nearest, and how
// far that is, squared. of two as near, the first face. it is face -1
// when none is: the point has a part that is not a number, or is so far
// off that every squared distance is endless.
func (made *Mesh) nearest(point maths.Vec) (int, weights, float64) {
	made.ensureTree()
	at := parts(point)
	best := found{face: noFace, squared: math.Inf(plusInfinity)}
	var stack [deepest + pushed]waiting
	stack[nought] = waiting{node: nought,
		enter: awayFrom(&made.tree.nodes[nought], at)}
	for top := unity; top > nought; {
		top--
		next := stack[top]
		if next.enter > best.squared {
			continue
		}
		count(entered)
		node := &made.tree.nodes[next.node]
		if node.count > nought {
			best = made.nearestIn(node, point, best)
			continue
		}
		top = closer(made.tree.nodes, node.first, at, &stack, top)
	}
	return best.face, best.weights, best.squared
}

// noFace is the face of a nearest search that has found none.
const noFace = -1 // defined: a place before the first, which no face is in

// found is the nearest face so far, its corners' weights at its point
// nearest, and how far that is, squared.
type found struct {
	face    int
	weights weights
	squared float64
}

// nearestIn is the nearer of a leaf's faces and the best so far.
func (made *Mesh) nearestIn(leaf *node, point maths.Vec, best found) found {
	for _, i := range made.tree.run(leaf) {
		count(tried)
		on := made.closest(i, point)
		gap := made.pointAt(i, on).Sub(point)
		squared := gap.Dot(gap)
		if best.beaten(squared, i) {
			best = found{face: i, weights: on, squared: squared}
		}
	}
	return best
}

// beaten is whether face i so far away, squared, beats the best: it is
// nearer, or as near and an earlier face.
func (best found) beaten(squared float64, i int) bool {
	return squared < best.squared ||
		squared == best.squared && i < best.face
}

// closer puts the two children of a node on the stack, the nearer on top,
// and is the stack's new height. The children are at child and the node
// after it.
func closer(nodes []node, child int, at [axes]float64,
	stack *[deepest + pushed]waiting, top int) int {
	left := awayFrom(&nodes[child], at)
	right := awayFrom(&nodes[child+unity], at)
	if right < left {
		stack[top] = waiting{node: child, enter: left}
		stack[top+unity] = waiting{node: child + unity, enter: right}
	} else {
		stack[top] = waiting{node: child + unity, enter: right}
		stack[top+unity] = waiting{node: child, enter: left}
	}
	return top + pushed
}

// awayFrom is how far a point is from a node's box, squared; zero inside it.
func awayFrom(at *node, point [axes]float64) float64 {
	var sum float64
	for axis := range point {
		gap := math.Max(nought, math.Max(at.bounds[low][axis]-point[axis],
			point[axis]-at.bounds[high][axis]))
		sum += gap * gap
	}
	return sum
}

// pointAt is the point on face i its corners' weights give.
func (made *Mesh) pointAt(i int, at weights) maths.Vec {
	one, two, three := made.Corners(i)
	return one.Scale(at[first]).Add(two.Scale(at[second])).
		Add(three.Scale(at[third]))
}

// atCorner is the weights of a point on one corner of a face: all of
// it that corner's, none of the others'.
func atCorner(corner int) weights {
	var at weights
	at[corner] = unity
	return at
}

// closest is the point of face i nearest a point, as its corners'
// weights: ericson, "real-time collision detection" (2005), 5.1.5,
// which finds the region of the face's plane the point is over, a
// corner, an edge or the face itself. the d's and v's are the book's.
func (made *Mesh) closest(i int, point maths.Vec) weights {
	one, two, three := made.Corners(i)
	toTwo, toThree := two.Sub(one), three.Sub(one)
	fromOne := point.Sub(one)
	d1, d2 := toTwo.Dot(fromOne), toThree.Dot(fromOne)
	if d1 <= nought && d2 <= nought {
		return atCorner(first)
	}
	fromTwo := point.Sub(two)
	d3, d4 := toTwo.Dot(fromTwo), toThree.Dot(fromTwo)
	if d3 >= nought && d4 <= d3 {
		return atCorner(second)
	}
	if vc := d1*d4 - d3*d2; vc <= nought && d1 >= nought && d3 <= nought {
		along := d1 / (d1 - d3)
		return weights{unity - along, along, nought}
	}
	fromThree := point.Sub(three)
	d5, d6 := toTwo.Dot(fromThree), toThree.Dot(fromThree)
	if d6 >= nought && d5 <= d6 {
		return atCorner(third)
	}
	if vb := d5*d2 - d1*d6; vb <= nought && d2 >= nought && d6 <= nought {
		along := d2 / (d2 - d6)
		return weights{unity - along, nought, along}
	}
	if va := d3*d6 - d5*d4; va <= nought && d4-d3 >= nought &&
		d5-d6 >= nought {
		along := (d4 - d3) / ((d4 - d3) + (d5 - d6))
		return weights{nought, unity - along, along}
	}
	va, vb, vc := d3*d6-d5*d4, d5*d2-d1*d6, d1*d4-d3*d2
	share := unity / (va + vb + vc)
	return weights{unity - (vb+vc)*share, vb * share, vc * share}
}

// SurfaceSite is the nearest point and the outward winding's normal on
// its face. Weights are ordered by that face's corners. The normal uses
// the local orientation even for an open mesh.
type SurfaceSite struct {
	Point, Normal maths.Vec
	Face          int
	Weights       [triangleCorners]float64
}

// SurfaceNear finds the nearest triangle using the mesh hierarchy.
func (made *Mesh) SurfaceNear(point maths.Vec) (SurfaceSite, bool) {
	face, weights, _ := made.nearest(point)
	if face < nought {
		return SurfaceSite{}, false
	}
	return SurfaceSite{
		Point: made.pointAt(face, weights), Normal: made.across(face).Unit(),
		Face: face, Weights: [triangleCorners]float64(weights),
	}, true
}
