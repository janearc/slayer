package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// sealTailRing is one section of the tail: how far back of the last
// trunk section it lies, and its share of that section's outline.
type sealTailRing struct{ back, scale float64 }

// sealTailRings make the short tail, about eight centimetres, that tapers
// out of the end of the trunk between the hindflippers. The numbers are
// chosen.
var sealTailRings = []sealTailRing{
	{-.026, .84}, {-.052, .62}, {-.070, .36}, {-.080, .12},
}

// tail grows the short tail out of the rear trunk ring, as more of the
// same indexed skin, and closes it. middle is the centre of that ring.
func (rig *SealRig) tail(middle maths.Vec) {
	previous := sealBodyLoop(0)
	for _, section := range sealTailRings {
		ring := make([]int, len(previous))
		for column := range ring {
			offset := rig.vertices[sealIndex(0, column)].point.Sub(middle)
			point := middle.Add(maths.V(section.back, 0, 0)).
				Add(offset.Scale(section.scale))
			ring[column] = rig.add(point, point.X, 0, 0)
		}
		for column, first := range ring {
			next := (column + 1) % len(ring)
			rig.joinAround(column, first, ring[next], previous[next],
				previous[column])
		}
		previous = ring
	}
	end := sealTailRings[len(sealTailRings)-1].back
	rig.cap(previous, middle.Add(maths.V(end, 0, 0)), true, 0, 0)
}

// settle works out what depends on the finished rest skin: its vertex
// normals, the way the fur lies at each vertex, and which vertices are
// bare skin, the digit tips, where claws grow instead of fur.
func (rig *SealRig) settle() {
	rig.normals = make([]maths.Vec, len(rig.rest))
	for index, normal := range sealSmoothNormals(rig.rest, rig.faces) {
		rig.normals[index] = normal.Unit()
	}
	field := rig.flowField()
	rig.flows = make([]maths.Vec, len(rig.rest))
	for index, point := range rig.rest {
		rig.flows[index] = field.at(point)
	}
	rig.bare = make([]bool, len(rig.rest))
	for _, limb := range rig.limbs {
		for _, index := range limb.outer {
			rig.bare[index] = true
		}
		rig.bare[limb.pole] = true
	}
}

// sealFlow is the direction the coat lies at a point, in rest space.
type sealFlow struct{ roots [len(sealOpenings)]maths.Vec }

// flowField is the coat's direction field: from the muzzle back toward
// the tail, fanned out from the midline on the head, and away from each
// flipper root over the flipper.
func (rig *SealRig) flowField() sealFlow {
	var flow sealFlow
	for index, limb := range rig.limbs {
		flow.roots[index] = limb.root
	}
	return flow
}

const (
	// sealFlowLean is the distance from the midline, in metres, at which
	// the head's coat has leaned half way out. Chosen.
	sealFlowLean = .04
	// sealFlowFan is how far the head's coat fans out against how far it
	// runs back. Chosen.
	sealFlowFan = .8
	// sealFlowRoot is how strongly a flipper root pulls the coat outward
	// against the run toward the tail. Chosen.
	sealFlowRoot = 1.6
	// sealFlowNear and sealFlowFar are the distances from a flipper root,
	// in metres, between which its pull falls away.
	sealFlowNear, sealFlowFar = .04, .26
	// sealFlowHeadStart and sealFlowHeadEnd are the X, in metres, over
	// which the coat's fan comes in toward the muzzle.
	sealFlowHeadStart, sealFlowHeadEnd = .58, .72
)

// at is the unit direction the coat lies at a rest point.
func (flow sealFlow) at(point maths.Vec) maths.Vec {
	lean := point.Z / (math.Abs(point.Z) + sealFlowLean)
	head := maths.Smooth(sealFlowHeadStart, sealFlowHeadEnd, point.X)
	direction := maths.V(-1, 0, sealFlowFan*head*lean)
	for _, root := range flow.roots {
		away := point.Sub(root)
		if away.Len() == 0 {
			continue
		}
		pull := 1 - maths.Smooth(sealFlowNear, sealFlowFar, away.Len())
		direction = direction.Add(away.Unit().Scale(sealFlowRoot * pull))
	}
	return direction.Unit()
}

// restHit is where a ray first meets the rest skin, with the skin's unit
// normal there. It is how every landmark is set on the surface, not near
// it.
func (rig *SealRig) restHit(origin, direction maths.Vec) (maths.Vec, maths.Vec, bool) {
	best := math.Inf(1)
	var point, normal maths.Vec
	for _, face := range rig.faces {
		corners := [3]maths.Vec{rig.rest[face[0]], rig.rest[face[1]], rig.rest[face[2]]}
		distance, first, second, hit := sealRayTriangle(origin, direction, corners)
		if !hit || distance >= best {
			continue
		}
		best = distance
		point = origin.Add(direction.Scale(distance))
		normal = rig.normals[face[0]].Scale(1 - first - second).
			Add(rig.normals[face[1]].Scale(first)).
			Add(rig.normals[face[2]].Scale(second)).Unit()
	}
	return point, normal, !math.IsInf(best, 1)
}

// sealRayTriangle is the distance along a ray to a triangle and the
// triangle's barycentric weights for its second and third corners.
// Möller and Trumbore, "Fast, minimum storage ray-triangle intersection",
// Journal of Graphics Tools 2(1), 1997. The triangle is hit from either
// side, and a miss behind the origin is a miss.
func sealRayTriangle(origin, direction maths.Vec, corners [3]maths.Vec) (float64, float64, float64, bool) {
	const parallel = 1e-14
	first, second := corners[1].Sub(corners[0]), corners[2].Sub(corners[0])
	pivot := direction.Cross(second)
	determinant := first.Dot(pivot)
	if math.Abs(determinant) < parallel {
		return 0, 0, 0, false
	}
	from := origin.Sub(corners[0])
	u := from.Dot(pivot) / determinant
	across := from.Cross(first)
	v := direction.Dot(across) / determinant
	distance := second.Dot(across) / determinant
	return distance, u, v, u >= 0 && v >= 0 && u+v <= 1 && distance > 0
}
