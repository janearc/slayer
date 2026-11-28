package bodies

import (
	"fmt"
	"math"
	"sort"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/world"
)

// reliefEdgeID is independent of which side names an edge or its direction.
type reliefEdgeID struct{ a, b world.ReliefVertex }

func reliefLess(a, b world.ReliefVertex) bool {
	if a.X != b.X {
		return a.X < b.X
	}
	if a.Y != b.Y {
		return a.Y < b.Y
	}
	return a.Z < b.Z
}

func reliefID(a, b world.ReliefVertex) reliefEdgeID {
	if reliefLess(b, a) {
		a, b = b, a
	}
	return reliefEdgeID{a, b}
}

type reliefEdgeUse struct {
	patch         int
	edge          world.ReliefEdge
	level         int
	keys          world.ReliefEdgeKeys
	paired, split bool
}

// ReliefPatches makes a closed, stitched cube-sphere mesh from a caller's
// selected leaf patches. Every face of the cube must be covered exactly
// once, and neighbouring leaves may differ by at most one level. The
// function rejects overlap, gaps, unsupported depth and an insufficient
// face budget. It does not choose leaves, prove a screen-space error, or
// animate transitions. The returned displacement is the largest movement
// of a stitched vertex from its field sample, in metres before scale.
//
// Vertices and the field use metres; scale converts to the scene's units.
// sea clamps negative heights to sea level, not to a separate ocean.
func ReliefPatches(field *world.Relief, patches []world.ReliefPatch,
	maxFaces int, sea bool, scale float64) (*shape.Mesh, float64, error) {
	if field == nil || len(patches) == 0 || maxFaces < 1 ||
		len(patches) > maxFaces/4 || !(scale > 0) || math.IsInf(scale, 0) {
		return nil, 0, fmt.Errorf("bodies: invalid relief patches, budget or scale")
	}
	seen := make(map[world.ReliefPatch]bool, len(patches))
	for _, patch := range patches {
		face, level, x, y := patch.Address()
		if level >= 16 || seen[patch] {
			return nil, 0, fmt.Errorf("bodies: duplicate or too-deep relief patch")
		}
		seen[patch] = true
		for depth := level - 1; depth >= 0; depth-- {
			shift := level - depth
			parent, err := world.NewReliefPatch(face, depth, x>>shift, y>>shift)
			if err != nil {
				return nil, 0, err
			}
			if seen[parent] {
				return nil, 0, fmt.Errorf("bodies: overlapping relief patches")
			}
		}
	}
	// Check again after all leaves are known, so input order is irrelevant.
	for _, patch := range patches {
		face, level, x, y := patch.Address()
		for depth := level - 1; depth >= 0; depth-- {
			shift := level - depth
			parent, _ := world.NewReliefPatch(face, depth, x>>shift, y>>shift)
			if seen[parent] {
				return nil, 0, fmt.Errorf("bodies: overlapping relief patches")
			}
		}
	}
	uses := make([]reliefEdgeUse, 0, len(patches)*4)
	byEdge := make(map[reliefEdgeID][]int, len(patches)*4)
	for i, patch := range patches {
		_, level, _, _ := patch.Address()
		for edge := world.ReliefLowX; edge <= world.ReliefHighY; edge++ {
			keys, err := patch.EdgeKeys(edge)
			if err != nil {
				return nil, 0, err
			}
			index := len(uses)
			uses = append(uses, reliefEdgeUse{patch: i, edge: edge,
				level: level, keys: keys})
			id := reliefID(keys.First, keys.Last)
			byEdge[id] = append(byEdge[id], index)
		}
	}
	for _, group := range byEdge {
		if len(group) > 2 {
			return nil, 0, fmt.Errorf("bodies: relief edge is shared more than twice")
		}
		if len(group) == 2 {
			a, b := group[0], group[1]
			if uses[a].level != uses[b].level || uses[a].patch == uses[b].patch {
				return nil, 0, fmt.Errorf("bodies: incompatible relief edge")
			}
			uses[a].paired, uses[b].paired = true, true
		}
	}
	constraints, err := world.NewReliefConstraints(field, sea)
	if err != nil {
		return nil, 0, err
	}
	order := make([]int, len(uses))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return uses[order[a]].level < uses[order[b]].level })
	var moved float64
	for _, i := range order {
		if uses[i].paired {
			continue
		}
		u := &uses[i]
		left := byEdge[reliefID(u.keys.First, u.keys.Midpoint)]
		right := byEdge[reliefID(u.keys.Midpoint, u.keys.Last)]
		if len(left) != 1 || len(right) != 1 || left[0] == right[0] ||
			uses[left[0]].paired || uses[right[0]].paired ||
			uses[left[0]].level != u.level+1 || uses[right[0]].level != u.level+1 {
			return nil, 0, fmt.Errorf("bodies: relief patches have a gap or unbalanced edge")
		}
		shift, err := constraints.Stitch(u.keys)
		if err != nil {
			return nil, 0, err
		}
		moved = math.Max(moved, shift)
		u.paired, u.split = true, true
		uses[left[0]].paired, uses[right[0]].paired = true, true
	}
	return reliefPatchMesh(patches, uses, constraints, maxFaces, scale, moved)
}

func reliefPatchMesh(patches []world.ReliefPatch,
	uses []reliefEdgeUse, constraints *world.ReliefConstraints,
	maxFaces int, scale, moved float64) (*shape.Mesh, float64, error) {
	vertices := make([]maths.Vec, 0, len(patches)*2)
	indices := make(map[world.ReliefVertex]int, len(patches)*2)
	faces := make([][3]int, 0, min(maxFaces, len(patches)*8))
	vertex := func(key world.ReliefVertex) (int, error) {
		if i, ok := indices[key]; ok {
			return i, nil
		}
		point, err := constraints.Point(key)
		if err != nil {
			return 0, err
		}
		i := len(vertices)
		vertices = append(vertices, point.Scale(scale))
		indices[key] = i
		return i, nil
	}
	for i, patch := range patches {
		corners := patch.Corners()
		children, err := patch.Children()
		if err != nil {
			return nil, 0, err
		}
		centre := children[0].Corners()[3]
		// Walk around the boundary, including coarse edges' constrained
		// midpoints where two finer neighbours meet them.
		ring := []world.ReliefVertex{corners[0]}
		for _, step := range []struct {
			edge world.ReliefEdge
			next world.ReliefVertex
		}{
			{world.ReliefLowY, corners[1]}, {world.ReliefHighX, corners[3]},
			{world.ReliefHighY, corners[2]}, {world.ReliefLowX, corners[0]},
		} {
			use := uses[4*i+int(step.edge)]
			if use.split {
				ring = append(ring, use.keys.Midpoint)
			}
			ring = append(ring, step.next)
		}
		if len(faces)+len(ring)-1 > maxFaces {
			return nil, 0, fmt.Errorf("bodies: relief patch mesh exceeds face budget")
		}
		middle, err := vertex(centre)
		if err != nil {
			return nil, 0, err
		}
		for j := 0; j+1 < len(ring); j++ {
			a, err := vertex(ring[j])
			if err != nil {
				return nil, 0, err
			}
			b, err := vertex(ring[j+1])
			if err != nil {
				return nil, 0, err
			}
			n := vertices[a].Sub(vertices[middle]).Cross(vertices[b].Sub(vertices[middle]))
			if n.Dot(vertices[middle]) < 0 {
				a, b = b, a
			}
			faces = append(faces, [3]int{middle, a, b})
		}
	}
	normals := make([]maths.Vec, len(vertices))
	for _, tri := range faces {
		a, b, c := tri[0], tri[1], tri[2]
		n := vertices[b].Sub(vertices[a]).Cross(vertices[c].Sub(vertices[a]))
		normals[a], normals[b], normals[c] = normals[a].Add(n), normals[b].Add(n), normals[c].Add(n)
	}
	mesh, err := shape.NewMesh(vertices, faces, normals, true)
	if err != nil {
		return nil, 0, err
	}
	return mesh, moved, nil
}
