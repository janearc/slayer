package mesh

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Normal is the way the mesh faces at a point on it: the way the face
// nearest the point faces there. a ray's Met says which face without
// looking, by NormalAt.
func (made *Mesh) Normal(point maths.Vec) maths.Vec {
	i, at, _ := made.nearest(point)
	return made.facing(i, at)
}

// Distance is how far a point is from the mesh's nearest face, and less than
// zero inside a closed mesh. telling inside asks every face.
func (made *Mesh) Distance(point maths.Vec) float64 {
	_, _, squared := made.nearest(point)
	away := math.Sqrt(squared)
	if made.closed && made.Inside(point) {
		return -away
	}
	return away
}

// Inside is whether a point is inside a closed mesh: whether the solid
// angles its faces fill around the point come to a whole turn, by van
// oosterom and strackee's formula (1983). an open mesh has no inside.
func (made *Mesh) Inside(point maths.Vec) bool {
	if !made.closed {
		return false
	}
	sum := 0.0
	for i := range made.faces {
		sum += made.solidAngle(i, point)
	}
	return sum > 2*math.Pi
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
	return 2 * math.Atan2(above, below)
}

// nearest is the face nearest a point, by the tree, nearer boxes
// first: the face, its corners' weights at its point nearest, and how
// far that is, squared. of two as near, the first face.
func (made *Mesh) nearest(point maths.Vec) (int, weights, float64) {
	at := parts(point)
	best := found{face: -1, squared: math.Inf(1)}
	var stack [deepest + 2]waiting
	stack[0] = waiting{node: 0, enter: awayFrom(&made.tree.nodes[0], at)}
	for top := 1; top > 0; {
		top--
		next := stack[top]
		if next.enter > best.squared {
			continue
		}
		count(entered)
		node := &made.tree.nodes[next.node]
		if node.count > 0 {
			best = made.nearestIn(node, point, best)
			continue
		}
		top = closer(made.tree.nodes, node.first, at, &stack, top)
	}
	return best.face, best.weights, best.squared
}

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

// closer puts a node's children on the stack, the nearer on top, and
// is the stack's new height.
func closer(nodes []node, first int, at [3]float64,
	stack *[deepest + 2]waiting, top int) int {
	left := awayFrom(&nodes[first], at)
	right := awayFrom(&nodes[first+1], at)
	if right < left {
		stack[top] = waiting{node: first, enter: left}
		stack[top+1] = waiting{node: first + 1, enter: right}
	} else {
		stack[top] = waiting{node: first + 1, enter: right}
		stack[top+1] = waiting{node: first, enter: left}
	}
	return top + 2
}

// awayFrom is how far a point is from a node's box, squared; zero inside it.
func awayFrom(at *node, point [3]float64) float64 {
	sum := 0.0
	for axis := range point {
		gap := math.Max(0, math.Max(at.bounds[0][axis]-point[axis],
			point[axis]-at.bounds[1][axis]))
		sum += gap * gap
	}
	return sum
}

// pointAt is the point on face i its corners' weights give.
func (made *Mesh) pointAt(i int, at weights) maths.Vec {
	one, two, three := made.Corners(i)
	return one.Scale(at[0]).Add(two.Scale(at[1])).Add(three.Scale(at[2]))
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
	if d1 <= 0 && d2 <= 0 {
		return weights{1, 0, 0}
	}
	fromTwo := point.Sub(two)
	d3, d4 := toTwo.Dot(fromTwo), toThree.Dot(fromTwo)
	if d3 >= 0 && d4 <= d3 {
		return weights{0, 1, 0}
	}
	if vc := d1*d4 - d3*d2; vc <= 0 && d1 >= 0 && d3 <= 0 {
		along := d1 / (d1 - d3)
		return weights{1 - along, along, 0}
	}
	fromThree := point.Sub(three)
	d5, d6 := toTwo.Dot(fromThree), toThree.Dot(fromThree)
	if d6 >= 0 && d5 <= d6 {
		return weights{0, 0, 1}
	}
	if vb := d5*d2 - d1*d6; vb <= 0 && d2 >= 0 && d6 <= 0 {
		along := d2 / (d2 - d6)
		return weights{1 - along, 0, along}
	}
	if va := d3*d6 - d5*d4; va <= 0 && d4-d3 >= 0 && d5-d6 >= 0 {
		along := (d4 - d3) / ((d4 - d3) + (d5 - d6))
		return weights{0, 1 - along, along}
	}
	va, vb, vc := d3*d6-d5*d4, d5*d2-d1*d6, d1*d4-d3*d2
	share := 1 / (va + vb + vc)
	return weights{1 - (vb+vc)*share, vb * share, vc * share}
}
