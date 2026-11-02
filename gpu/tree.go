package gpu

import (
	"math"
	"sort"

	"github.com/janearc/slayer/maths"
)

// triangle is one face in its mesh's own space, as the kernel reads it.
// what it is made of, and whether it is clear, is the standing's. a face
// of a mesh shaded smooth has its corners' normals, in the same order.
type triangle struct {
	corners [3]maths.Vec
	normals [3]maths.Vec
	smooth  bool
}

// box is the smallest box round some corners.
type box struct{ low, high maths.Vec }

// empty is a box that grows to fit whatever is added to it.
func empty() box {
	inf := math.Inf(1)
	return box{maths.V(inf, inf, inf), maths.V(-inf, -inf, -inf)}
}

// fit grows the box to take in a point.
func (bounds box) fit(point maths.Vec) box {
	return box{lower(bounds.low, point), higher(bounds.high, point)}
}

// lower is the smaller of two points along each axis.
func lower(one, other maths.Vec) maths.Vec {
	return maths.V(math.Min(one.X, other.X), math.Min(one.Y, other.Y),
		math.Min(one.Z, other.Z))
}

// higher is the larger of two points along each axis.
func higher(one, other maths.Vec) maths.Vec {
	return maths.V(math.Max(one.X, other.X), math.Max(one.Y, other.Y),
		math.Max(one.Z, other.Z))
}

// holds is whether another box lies wholly within this one.
func (bounds box) holds(inner box) bool {
	return lower(bounds.low, inner.low) == bounds.low &&
		higher(bounds.high, inner.high) == bounds.high
}

// node is one box of the tree. A leaf holds count triangles from first; a
// branch has count zero, and its two children are at first and first+1.
type node struct {
	bounds       box
	first, count int
}

// insideOut is a box no ray enters: its low corner is past its high one
// on every axis, by finite numbers, since the kernel is compiled with
// fast math and cannot be promised an infinity
var insideOut = box{low: maths.V(1e30, 1e30, 1e30),
	high: maths.V(-1e30, -1e30, -1e30)}

// leafSize is the most triangles a leaf holds: few enough that a ray
// tests little in one, many enough that the tree stays shallow.
const leafSize = 4

// tree is the triangles, put in the order its leaves hold them, and
// the boxes over them, the root first.
//
// With no triangles the root is a box inside out, its low corner past
// its high one on every axis, so no ray can enter it. the kernel walks
// a mesh's tree only from a standing of it, and with no standings walks
// none.
type tree struct {
	triangles []triangle
	nodes     []node
}

// leaf is the triangles a leaf holds.
func (grown tree) leaf(branch node) []triangle {
	return grown.triangles[branch.first : branch.first+branch.count]
}

// grow builds the tree, splitting each box across its longest side at
// the middle triangle, until a box holds a leaf's worth.
func grow(triangles []triangle) tree {
	grown := tree{triangles: triangles, nodes: []node{{bounds: insideOut}}}
	if len(triangles) > 0 {
		grown.split(0, 0, len(triangles))
	}
	return grown
}

// split makes the node at index cover triangles from first to last,
// and below it the boxes that divide them.
func (grown *tree) split(index, first, last int) {
	bounds := empty()
	for _, face := range grown.triangles[first:last] {
		for _, corner := range face.corners {
			bounds = bounds.fit(corner)
		}
	}
	grown.nodes[index] = node{bounds: bounds, first: first,
		count: last - first}
	if last-first <= leafSize {
		return
	}
	axis := longest(bounds)
	faces := grown.triangles[first:last]
	sort.Slice(faces, func(one, other int) bool {
		return centre(faces[one], axis) < centre(faces[other], axis)
	})
	middle := first + (last-first)/2
	children := len(grown.nodes)
	grown.nodes = append(grown.nodes, node{}, node{})
	grown.nodes[index] = node{bounds: bounds, first: children}
	grown.split(children, first, middle)
	grown.split(children+1, middle, last)
}

// longest is the axis the box is longest along: 0, 1 or 2.
func longest(bounds box) int {
	size := bounds.high.Sub(bounds.low)
	switch {
	case size.X >= size.Y && size.X >= size.Z:
		return 0
	case size.Y >= size.Z:
		return 1
	}
	return 2
}

// centre is where a triangle's middle is along an axis.
func centre(face triangle, axis int) float64 {
	sum := 0.0
	for _, corner := range face.corners {
		sum += [3]float64{corner.X, corner.Y, corner.Z}[axis]
	}
	return sum / 3
}
