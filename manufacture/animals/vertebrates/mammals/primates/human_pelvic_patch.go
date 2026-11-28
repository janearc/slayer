package primates

import (
	"math"

	"github.com/janearc/slayer/maths"
)

type pelvicEdge struct {
	vertex, node int
	weight       float64
}

type pelvicNode struct {
	vertex int
	hold   float64
	edges  []pelvicEdge
}

// finishFemalePelvis shapes the lower trunk, pelvic rows and proximal thighs
// as one joined surface before applying individual external structures.
func (model *humanMesh) finishFemalePelvis(trunk [][]int, firstCuff int,
	bridge []int, back int, form VulvarForm,
) {
	model.shapeFemaleAdductors(bridge[0])
	nodes, lookup := model.pelvicPatch(trunk, firstCuff)
	model.fairPelvicPatch(nodes)
	chart := model.pelvicChart(trunk, firstCuff, bridge, back)
	model.fitPelvicRelief(nodes, lookup, chart, form)
}

// pelvicPatch uses existing mesh edges. In particular, the first thigh rows
// and the lower trunk are adjacent to the same patch, not independently fit.
func (model *humanMesh) pelvicPatch(trunk [][]int, firstCuff int) ([]pelvicNode, []int) {
	lookup := make([]int, len(model.vertices))
	var nodes []pelvicNode
	consider := func(index int) {
		point := model.vertices[index].point
		level := maths.Smooth(.650, .686, point.Y) * (1 - maths.Smooth(.854, .897, point.Y))
		if level == 0 {
			return
		}
		front := .44 + .56*maths.Smooth(-.051, .051, point.Z)
		mobility := level * front
		hold := .016 + .83*(1-mobility)*(1-mobility)
		nodes = append(nodes, pelvicNode{vertex: index, hold: hold})
		lookup[index] = len(nodes)
	}
	for _, row := range trunk {
		for _, index := range row {
			consider(index)
		}
	}
	for index := firstCuff; index < len(model.vertices); index++ {
		consider(index)
	}
	bind := func(from, to int) {
		if lookup[from] == 0 {
			return
		}
		node := &nodes[lookup[from]-1]
		distance := model.vertices[to].point.Sub(model.vertices[from].point).Len()
		edge := pelvicEdge{vertex: to, node: lookup[to], weight: 1 / math.Max(.0016, distance)}
		node.edges = append(node.edges, edge)
	}
	for _, face := range model.faces {
		for i, first := range face {
			next := face[(i+1)%len(face)]
			bind(first, next)
			bind(next, first)
		}
	}
	return nodes, lookup
}

// fairPelvicPatch solves the edge-weighted membrane equations with a rest
// term. Points beyond the patch remain fixed. The rest term preserves the
// wide hip and thigh contours while the shared joins settle together.
func (model *humanMesh) fairPelvicPatch(nodes []pelvicNode) {
	current := make([]maths.Vec, len(nodes))
	next := make([]maths.Vec, len(nodes))
	for i, node := range nodes {
		current[i] = model.vertices[node.vertex].point
	}
	const sweeps = 76
	for sweep := 0; sweep < sweeps; sweep++ {
		for i, node := range nodes {
			rest := model.vertices[node.vertex].point
			var sum maths.Vec
			total := 0.0
			for _, edge := range node.edges {
				point := model.vertices[edge.vertex].point
				if edge.node != 0 {
					point = current[edge.node-1]
				}
				sum = sum.Add(point.Scale(edge.weight))
				total += edge.weight
			}
			if total == 0 {
				next[i] = rest
				continue
			}
			target := mixPoint(sum.Scale(1/total), rest, node.hold)
			next[i] = mixPoint(current[i], target, .78)
			if math.Abs(rest.X) <= 1e-12 {
				next[i].X = rest.X
			}
		}
		current, next = next, current
	}
	for i, node := range nodes {
		model.movePelvicPoint(node.vertex, current[i])
	}
}

func (model *humanMesh) movePelvicPoint(index int, point maths.Vec) {
	vertex := &model.vertices[index]
	vertex.chart = vertex.chart.Add(point.Sub(vertex.point))
	vertex.point = point
}
