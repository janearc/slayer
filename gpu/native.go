//go:build darwin

package gpu

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
)

// accelerationPlan preserves the shared mesh/group/world hierarchy. Native
// queries use sorted instance IDs shared with the existing shading buffers.
func (encoded *scene) accelerationPlan() (*metal.Acceleration, int, error) {
	plan, err := metal.NewAcceleration(encoded.triangles.bytes, 16)
	if err != nil {
		return nil, 0, err
	}
	root, err := encoded.populateAcceleration(plan)
	if err != nil {
		return nil, 0, err
	}
	return plan, root, nil
}

// nativePlan is the checked builder the geometry translator writes into.
// Keeping its refusals at this boundary preserves the builder's causes.
type nativePlan interface {
	Triangles(int, int) (int, error)
	Instances(...metal.Placement) (int, error)
}

// populateAcceleration translates local groups and world placements in order.
func (encoded *scene) populateAcceleration(plan nativePlan) (int, error) {
	var err error
	meshes := map[int]int{}
	addMesh := func(root int) error {
		if _, known := meshes[root]; known {
			return nil
		}
		first, count := encoded.triangleRange(root)
		index, err := plan.Triangles(first, count)
		if err == nil {
			meshes[root] = index
		}
		return err
	}
	for _, standing := range encoded.standings {
		if standing.group == 0 {
			if err := addMesh(standing.root); err != nil {
				return 0, err
			}
		}
	}
	for _, group := range encoded.groups {
		for _, member := range group.members {
			if err := addMesh(member.root); err != nil {
				return 0, err
			}
		}
	}
	groups := make([]int, len(encoded.groups))
	for i, group := range encoded.groups {
		var places []metal.Placement
		order, _ := topped(group.members)
		for _, index := range order {
			member := group.members[index]
			placed, err := metal.Placed(meshes[member.root], nativeMatrix(member.space), 3)
			if err != nil {
				return 0, err
			}
			places = append(places, placed)
		}
		groups[i], err = plan.Instances(places...)
		if err != nil {
			return 0, err
		}
	}
	var places []metal.Placement
	order, _ := topped(encoded.standings)
	for _, index := range order {
		standing := encoded.standings[index]
		child := meshes[standing.root]
		if standing.group != 0 {
			if standing.group > len(groups) {
				return 0, fmt.Errorf("gpu: invalid shared group %d", standing.group)
			}
			child = groups[standing.group-1]
		}
		mask := uint32(3)
		if standing.clear {
			mask = 1
		}
		placed, err := metal.Placed(child, nativeMatrix(standing.space), mask)
		if err != nil {
			return 0, err
		}
		places = append(places, placed)
	}
	return plan.Instances(places...)
}

// triangleRange recovers a pooled mesh's contiguous faces from its tree.
func (encoded *scene) triangleRange(root int) (int, int) {
	node := encoded.faces.nodes[root]
	if node.bounds.low.X > node.bounds.high.X {
		return 0, 0
	}
	if node.count != 0 {
		return node.first, node.count
	}
	first, count := encoded.triangleRange(node.first)
	other, more := encoded.triangleRange(node.first + 1)
	start := min(first, other)
	return start, max(first+count, other+more) - start
}

// nativeMatrix is the forward transform in Metal's packed column-major layout.
func nativeMatrix(space maths.Subspace) [12]float32 {
	scale := space.Scale
	if scale == 0 {
		scale = 1
	}
	var matrix [12]float32
	for i, axis := range axes() {
		way := space.Orientation.Out(axis).Scale(scale)
		matrix[i*3], matrix[i*3+1], matrix[i*3+2] =
			float32(way.X), float32(way.Y), float32(way.Z)
	}
	matrix[9], matrix[10], matrix[11] = float32(space.Origin.X),
		float32(space.Origin.Y), float32(space.Origin.Z)
	return matrix
}

// nativeFirsts maps sorted native IDs to pooled faces or local member ranges.
// Its order is the same as instancesOf: global standings, then each group.
func (encoded *scene) nativeFirsts() []byte {
	firstMember := make([]int, len(encoded.groups))
	next := len(encoded.standings)
	for i, group := range encoded.groups {
		firstMember[i] = next
		next += len(group.members)
	}
	// Repeated shoots share a pooled face range. Recover it once per mesh,
	// rather than walking the same face tree for every placed shoot.
	firsts := map[int]int{}
	firstOf := func(root int) int {
		if first, known := firsts[root]; known {
			return first
		}
		first, _ := encoded.triangleRange(root)
		firsts[root] = first
		return first
	}
	var packed packer
	order, _ := topped(encoded.standings)
	for _, index := range order {
		standing := encoded.standings[index]
		first := 0
		if standing.group == 0 {
			first = firstOf(standing.root)
		} else {
			first = firstMember[standing.group-1]
		}
		packed.word(uint32(first))
	}
	for _, group := range encoded.groups {
		order, _ := topped(group.members)
		for _, index := range order {
			first := firstOf(group.members[index].root)
			packed.word(uint32(first))
		}
	}
	return packed.bytes
}
