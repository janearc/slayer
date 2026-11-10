package gpu

import (
	"fmt"
	"math"
	"sort"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

// instanceStruct is one mesh standing in the world as the kernel reads
// it: three rows that carry a point of the world into the mesh's own
// space, each with its shift in w, and the mesh's own axes in the
// world, which carry its normals back out.
//
// its info is its tree's root among the nodes, its material, and
// whether it is clear, and a local group root plus one (zero for a mesh).
var instanceStruct = structure{"Instance", []field{
	floatField("into_x", "the world into the mesh, a row each, shift in w"),
	floatField("into_y", ""),
	floatField("into_z", ""),
	floatField("out_x", "the mesh's own axes in the world"),
	floatField("out_y", ""),
	floatField("out_z", ""),
	intField("info", "mesh root, material, clear, group root plus one"),
}}

// pooled is a mesh's faces as they went into the pool, once: its tree's
// root among the nodes, and the box round it in its own space.
type pooled struct {
	root   int
	bounds box
}

// standing is a mesh standing somewhere: its tree's root, where it
// stands, what it is made of, whether it is clear, and its box in the
// world.
type standing struct {
	root     int
	space    maths.Subspace
	material int
	clear    bool
	bounds   box
	group    int
}

// meshGroup keeps local placements once, shared by all world instances.
type meshGroup struct{ members []standing }

// group pools member meshes and records one shared local hierarchy.
func (encoded *scene) group(made *shape.MeshGroup, space maths.Subspace,
	material int, clear bool) error {
	index, known := encoded.groupOf[made]
	if !known {
		local := meshGroup{}
		for _, member := range made.Members() {
			pooled, seen := encoded.meshes[member.Mesh]
			if !seen {
				var faces []triangle
				switch solid := member.Mesh.(type) {
				case *mesh.Mesh:
					faces = slayersFaces(solid)
				case *shape.Mesh:
					if solid.Smooth && len(solid.Normals) > 0 {
						return unsupported{"a smooth reference mesh in a group"}
					}
					faces = meshFaces(solid)
				}
				pooled = encoded.pool(faces)
				encoded.meshes[member.Mesh] = pooled
			}
			local.members = append(local.members, standing{root: pooled.root,
				space: member.Space, bounds: worldBox(pooled.bounds, member.Space)})
		}
		index = len(encoded.groups)
		encoded.groups = append(encoded.groups, local)
		encoded.groupOf[made] = index
	}
	bounds := made.Bounds()
	encoded.standings = append(encoded.standings, standing{space: space,
		material: material, clear: clear, group: index + 1,
		bounds: worldBox(box{low: bounds.Min, high: bounds.Max}, space)})
	return nil
}

// stand puts a mesh in the world at a space: its faces into the pool
// the first time the mesh is seen, by its key, and a standing of it
// every time. local makes its faces in its own space, when they are
// needed.
func (encoded *scene) stand(key any, local func() []triangle,
	space maths.Subspace, material int, clear bool) {
	at, seen := encoded.meshes[key]
	if !seen {
		at = encoded.pool(local())
		encoded.meshes[key] = at
	}
	encoded.standings = append(encoded.standings, standing{root: at.root,
		space: space, material: material, clear: clear,
		bounds: worldBox(at.bounds, space)})
}

// pool grows a tree over a mesh's faces in its own space and adds the
// faces and the tree's boxes to the scene's, each tree's indices moved
// past the ones already there.
func (encoded *scene) pool(faces []triangle) pooled {
	grown := grow(faces)
	firstFace := len(encoded.faces.triangles)
	firstNode := len(encoded.faces.nodes)
	encoded.faces.triangles = append(encoded.faces.triangles,
		grown.triangles...)
	for _, each := range grown.nodes {
		if each.count == 0 {
			each.first += firstNode
		} else {
			each.first += firstFace
		}
		encoded.faces.nodes = append(encoded.faces.nodes, each)
	}
	return pooled{root: firstNode, bounds: grown.nodes[0].bounds}
}

// worldBox is the box round a mesh's own box standing at a space.
func worldBox(local box, space maths.Subspace) box {
	made := empty()
	for corner := range 8 {
		at := maths.V(pick(corner&1, local.low.X, local.high.X),
			pick(corner&2, local.low.Y, local.high.Y),
			pick(corner&4, local.low.Z, local.high.Z))
		made = made.fit(space.Out(at))
	}
	return made
}

// pick is low when a bit is clear, high when it is set.
func pick(bit int, low, high float64) float64 {
	if bit == 0 {
		return low
	}
	return high
}

// instancesOf writes every standing, in the order the leaves of their
// own tree hold them, and that tree's boxes, which the kernel walks
// before any mesh's own.
func (encoded *scene) instancesOf() error {
	return encoded.instancesWithin(math.MaxInt32)
}

// instanceBudget bounds every packed index and both hierarchy buffers before
// materializing local members. The byte cap is Metal.Grid's input limit.
//
// Each nonempty hierarchy has at most 2*n-1 nodes; two nodes occupy less
// than one instance. Empty scenes still need one root. Groups are nonempty
// and are encoded only when a world standing references them.
func instanceBudget(bytes int, counts ...int) error {
	if bytes < nodeStruct.size() {
		return fmt.Errorf("gpu: instance hierarchy exceeds %d bytes", bytes)
	}
	total := 0
	limit := bytes / instanceStruct.size()
	for _, count := range counts {
		if count > limit-total {
			return fmt.Errorf("gpu: instance buffer exceeds %d bytes", bytes)
		}
		total += count
	}
	return nil
}

// instancesWithin makes the same layout under an explicit byte cap. Small
// caps exercise refusal before allocation; production uses Metal.Grid's cap.
func (encoded *scene) instancesWithin(bytes int) error {
	counts := []int{len(encoded.standings)}
	for _, group := range encoded.groups {
		counts = append(counts, len(group.members))
	}
	if err := instanceBudget(bytes, counts...); err != nil {
		return err
	}

	order, nodes := topped(encoded.standings)
	placed := make([]standing, 0, len(order))
	for _, index := range order {
		placed = append(placed, encoded.standings[index])
	}
	// Global root stays at zero. Local roots and member ranges follow it.
	roots := make([]int, len(encoded.groups))
	for index, group := range encoded.groups {
		localOrder, localNodes := topped(group.members)
		firstNode, firstInstance := len(nodes), len(placed)
		roots[index] = firstNode
		for _, member := range localOrder {
			placed = append(placed, group.members[member])
		}
		for _, node := range localNodes {
			if node.count == 0 {
				node.first += firstNode
			} else {
				node.first += firstInstance
			}
			nodes = append(nodes, node)
		}
	}
	encoded.instances.reserveWords(len(placed) * instanceStruct.size())
	encoded.tops.reserveWords(len(nodes) * nodeStruct.size())
	for _, one := range placed {
		if one.group > 0 {
			one.group = roots[one.group-1] + 1
		}
		encoded.instance(one)
	}
	for _, each := range nodes {
		encoded.tops.row(nodeStruct, values{
			"low":  point(each.bounds.low),
			"high": point(each.bounds.high),
			"info": wholes(each.first, each.count)})
	}
	encoded.counts.instances = len(placed)
	encoded.counts.tops = len(nodes)
	return nil
}

// instance writes one standing: the rows into its own space, a mesh's axes in
// the world over its scale, each shifted so the space's origin goes to zero,
// and its axes as they are, for normals.
func (encoded *scene) instance(placed standing) {
	encoded.packedInstance(placed)
}

// axes is the three axes, x, y and z.
func axes() []maths.Vec {
	return []maths.Vec{maths.XAxis, maths.YAxis, maths.ZAxis}
}

// xyzName is an axis by its letter.
func xyzName(index int) string { return string(rune('x' + index)) }

// topLeaf is the most standings a leaf of their tree holds.
const topLeaf = 2

// topped is the tree over the standings' boxes, split across the
// longest side at the middle one, and the order its leaves hold them
// in. with none, the root is a box no ray enters.
func topped(standings []standing) ([]int, []node) {
	order := make([]int, len(standings))
	for index := range order {
		order[index] = index
	}
	nodes := []node{{bounds: insideOut}}
	if len(order) > 0 {
		nodes = splitTops(standings, order, nodes, 0, 0, len(order))
	}
	return order, nodes
}

// splitTops makes node at cover the standings from first to last of the
// order, and below it the boxes that divide them.
func splitTops(standings []standing, order []int, nodes []node, at,
	first, last int) []node {
	bounds := empty()
	for _, index := range order[first:last] {
		bounds = bounds.fit(standings[index].bounds.low).
			fit(standings[index].bounds.high)
	}
	nodes[at] = node{bounds: bounds, first: first, count: last - first}
	if last-first <= topLeaf {
		return nodes
	}
	axis := longest(bounds)
	run := order[first:last]
	sort.SliceStable(run, func(one, other int) bool {
		return middleOf(standings[run[one]].bounds, axis) <
			middleOf(standings[run[other]].bounds, axis)
	})
	middle := first + (last-first)/2
	children := len(nodes)
	nodes = append(nodes, node{}, node{})
	nodes[at] = node{bounds: bounds, first: children}
	nodes = splitTops(standings, order, nodes, children, first, middle)
	return splitTops(standings, order, nodes, children+1, middle, last)
}

// middleOf is where a box's middle is along an axis.
func middleOf(bounds box, axis int) float64 {
	low, high := partOf(bounds.low, axis), partOf(bounds.high, axis)
	return (low + high) / 2
}

// partOf is a point's part along an axis.
func partOf(at maths.Vec, axis int) float64 {
	switch axis {
	case 0:
		return at.X
	case 1:
		return at.Y
	}
	return at.Z
}
