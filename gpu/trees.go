package gpu

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// the most levels of branching a conifer may have in the kernel, and the
// branches its walk may hold waiting: a branch at level l has 3+l children, and
// the walk takes one off before it puts its children on, so a tree 6 deep holds
// at most 2+3+4+5+6+8 = 28 at once (stack_test.go works it out).
//
// the go side refuses a deeper tree, rather than drawing it without its tips;
// the kernel does not check, and a stack too small writes past its end on the
// gpu.
const (
	mostConiferDepth = 6
	coniferStack     = 40
)

// mostConiferSides is the most sides a conifer's cones may have. the
// kernel tries every side of every cone a ray meets, so the sides bound
// how long one ray can take, and a tree of very many would hold the gpu
// past the time the system allows a command.
const mostConiferSides = 64

// treeStruct is a conifer as the kernel reads it: where it stands, and
// manufacture's parameters. the cones are not stored: the kernel makes each
// one, as manufacture.Conifer does, in the tree's own space with up along z.
//
// the axes are the world's directions of the tree's own x, y and z, and the
// scale in origin's w is how large the tree's unit is.
var treeStruct = structure{"Tree", []field{
	floatField("origin", "the trunk's foot in the world; scale in w"),
	floatField("x", "the tree's own axes in the world"),
	floatField("y", ""),
	floatField("z", ""),
	floatField("shape", "height, radius, taper, depth scale"),
	floatField("lean", "where the first branches lean, in x"),
	intField("info", "depth, sides, material"),
}}

// grown is one conifer to be grown in the kernel: where it stands, what
// manufacture says it is, and what it is made of. made only by newGrown,
// which refuses what is not a tree.
type grown struct {
	place    maths.Subspace
	params   manufacture.ConiferParams
	material *thing.Material
}

// newGrown is a conifer of these parameters standing at a place, made of a
// material, or why there can be none.
func newGrown(place maths.Subspace, params manufacture.ConiferParams,
	material *thing.Material) (grown, error) {
	if params.Depth < 0 || params.Depth > mostConiferDepth {
		return grown{}, unsupported{fmt.Sprintf(
			"a conifer %d levels deep, past %d", params.Depth,
			mostConiferDepth)}
	}
	if params.Sides < 3 || params.Sides > mostConiferSides ||
		!above(params.Height) || !above(params.BaseRadius) ||
		!above(params.TaperRate) || !above(params.DepthScale) ||
		!fits(params.BranchStart) {
		return grown{}, errNotATree
	}
	if place.Scale != 0 && !above(place.Scale) {
		return grown{}, errNotAPlace
	}
	return grown{place: place, params: params, material: material}, nil
}

// what a conifer is refused with: parameters that are not a tree, or a
// place that cannot hold one
var (
	errNotATree = errors.New("gpu: a conifer needs from three to " +
		fmt.Sprint(mostConiferSides) + " sides, a height, radius, " +
		"taper and depth scale above nought, and a finite lean, each " +
		"within a float32")
	errNotAPlace = errors.New("gpu: a conifer's scale must be nought " +
		"(read as one) or above nought, within a float32")
)

// fits is whether a number is finite and within what the kernel's
// float32 holds. the kernel is built with fast maths, where an infinity
// or a nan gives no defined answer, so neither may reach it.
func fits(number float64) bool {
	return math.Abs(number) <= math.MaxFloat32
}

// above is whether a number fits the kernel and is greater than zero.
func above(number float64) bool {
	return number > 0 && fits(number)
}

// conifers appends every conifer of a grove, as the kernel reads them.
func (encoded *scene) conifers(trees []grown) {
	for _, tree := range trees {
		encoded.conifer(tree, encoded.surface(tree.material))
	}
}

// conifer appends one conifer.
func (encoded *scene) conifer(tree grown, material int) {
	scale, axes := tree.place.Scale, tree.place.Orientation
	if scale == 0 {
		scale = 1
	}
	params := tree.params
	encoded.trees.row(treeStruct, values{
		"origin": append(point(tree.place.Origin), scale),
		"x":      point(axes.Out(maths.XAxis)),
		"y":      point(axes.Out(maths.YAxis)),
		"z":      point(axes.Out(maths.ZAxis)),
		"shape": {params.Height, params.BaseRadius, params.TaperRate,
			params.DepthScale},
		"lean": {params.BranchStart},
		"info": wholes(params.Depth, params.Sides, material)})
	encoded.counts.trees++
}
