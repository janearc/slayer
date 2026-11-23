package gpu

import (
	"math"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
)

// The analytic tree: one box tree over every sphere's and every conifer's
// box in the world, which the kernel walks front to back for the nearest
// of them, and for any of them on the way to a lamp, where it once tried
// each in a loop. Its nodes are written after the meshes' in the tops
// buffer, as the kernel's Node, so no buffer is added to the kernel's
// inputs; the header's analytic field says where they begin.
//
// A leaf holds one sphere or one conifer: its info is the index in the
// spheres or the trees, a count of one, and the kind. A plane has no box,
// nor has anything whose box is not finite, so those stand as leaves
// before the tree's nodes, which the kernel tries one by one, and which
// are few.

// The kinds of thing a leaf holds, as the kernel names them in
// analytic_sphere and analytic_conifer.
const (
	analyticSphere  = 0
	analyticConifer = 1
)

// analyticMargin is how far a box is grown past what it holds, as a share
// of its radius, and analyticReach as a share of its place's furthest
// coordinate. The kernel works in float32, whose rounding at a distant
// place is wider than a thin shape's edge, and a box that cut off a hit
// the loop would have found would change the picture.
const (
	analyticMargin = 1e-4
	analyticReach  = 1e-6
)

// coniferLean and coniferWiden are the kernel's allowance for how far a
// branch's children lean past its top, and the share a reach is widened
// by, in conifer_hit. The box of a conifer is the sphere its walk culls by.
const (
	coniferLean  = 1.12
	coniferWiden = 1.001
)

// coniferHeightScale and coniferRadiusScale are the shares the kernel
// takes of the depth scale and the taper for the step from one level to the
// next, as conifer_hit has them.
const (
	coniferHeightScale = 0.6
	coniferRadiusScale = 0.5
)

// analyticItem is one sphere or conifer that the tree is to hold: its kind
// and its index among its kind, its box, and whether it has none.
type analyticItem struct {
	kind, index int
	box         strategies.TreeBox
	loose       bool
}

// analyticPlace is where the analytic nodes stand in the tops buffer: the
// first, how many loose leaves come first, and how many nodes follow them.
type analyticPlace struct {
	first, loose, nodes int
}

// boundSphere records the sphere about to be appended, of a radius in the
// world, with a centre.
func (encoded *scene) boundSphere(centre maths.Vec, radius float64) {
	box, ok := boxAround(centre, math.Abs(radius))
	encoded.analytic = append(encoded.analytic, analyticItem{
		kind: analyticSphere, index: encoded.counts.spheres, box: box,
		loose: !ok})
}

// boundPlane records the plane about to be appended, which has no box.
func (encoded *scene) boundPlane() {
	encoded.analytic = append(encoded.analytic, analyticItem{
		kind: analyticSphere, index: encoded.counts.spheres, loose: true})
}

// boundConifer records the conifer about to be appended, standing at a
// place with a scale.
func (encoded *scene) boundConifer(tree grown, scale float64) {
	box, ok := boxAround(tree.place.Origin,
		coniferReach(tree.params)*scale)
	encoded.analytic = append(encoded.analytic, analyticItem{
		kind: analyticConifer, index: encoded.counts.trees, box: box,
		loose: !ok})
}

// boxAround is the box of a ball, grown by the margins, and whether it is
// finite, which is whether it is a box at all.
func boxAround(centre maths.Vec, radius float64) (strategies.TreeBox, bool) {
	at := [3]float64{centre.X, centre.Y, centre.Z}
	farthest := 0.0
	for _, number := range at {
		farthest = max(farthest, math.Abs(number))
	}
	grown := radius + radius*analyticMargin + farthest*analyticReach
	var box strategies.TreeBox
	for axis, number := range at {
		box.Low[axis], box.High[axis] = number-grown, number+grown
	}
	for axis := range at {
		if !fits(box.Low[axis]) || !fits(box.High[axis]) {
			return strategies.TreeBox{}, false
		}
	}
	return box, true
}

// coniferReach is how far from a conifer's foot, in the tree's own unit,
// anything of it can be: reaches[0] of the kernel's conifer_hit, level by
// level from the tips, a level's own cone or its children's, the greater.
func coniferReach(params manufacture.ConiferParams) float64 {
	deepest := min(params.Depth, mostConiferDepth)
	grow := coniferHeightScale * params.DepthScale
	thin := coniferRadiusScale * params.TaperRate
	reaches := make([]float64, deepest+2)
	for level := deepest; level >= 0; level-- {
		height := params.Height * math.Pow(grow, float64(level))
		radius := params.BaseRadius * math.Pow(thin, float64(level))
		own := math.Hypot(height, radius)
		grown := 0.0
		if level != deepest {
			grown = height*coniferLean + reaches[level+1]
		}
		reaches[level] = max(own, grown) * coniferWiden
	}
	return reaches[0]
}

// analyticTree writes the analytic nodes after the meshes' in the tops
// buffer and says where they went. It is called once the standings' tree
// is written and every sphere and conifer is recorded. A world with
// neither writes nothing and leaves the place empty, which the kernel
// reads as no work at all.
func (encoded *scene) analyticTree() error {
	var loose, tight []analyticItem
	var boxes []strategies.TreeBox
	for _, item := range encoded.analytic {
		if item.loose {
			loose = append(loose, item)
			continue
		}
		tight = append(tight, item)
		boxes = append(boxes, item.box)
	}
	if len(loose)+len(tight) == 0 {
		return nil
	}
	var tree strategies.BoxTree
	if len(boxes) > 0 {
		var err error
		if tree, err = strategies.BuildBoxTree(boxes, treeStack); err != nil {
			return err
		}
	}
	first := encoded.counts.tops
	encoded.tops.reserveWords((len(loose) + len(tree.Nodes)) *
		nodeStruct.size())
	for _, item := range loose {
		encoded.analyticLeaf(item, strategies.TreeBox{})
	}
	base := first + len(loose)
	for _, branch := range tree.Nodes {
		if branch.IsLeaf() {
			encoded.analyticLeaf(tight[branch.First], strategies.TreeBox{
				Low: branch.Low, High: branch.High})
			continue
		}
		encoded.tops.row(nodeStruct, values{
			"low": branch.Low[:], "high": branch.High[:],
			"info": wholes(base+int(branch.First), 0)})
	}
	encoded.analyticAt = analyticPlace{first: first, loose: len(loose),
		nodes: len(tree.Nodes)}
	return nil
}

// analyticLeaf writes one leaf: its box, the index of what it holds, a
// count of one, and the kind.
func (encoded *scene) analyticLeaf(item analyticItem, box strategies.TreeBox) {
	encoded.tops.row(nodeStruct, values{
		"low": box.Low[:], "high": box.High[:],
		"info": wholes(item.index, 1, item.kind)})
}
