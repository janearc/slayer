package shape

import (
	"fmt"
	"math"
	"sort"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// MeshMember places one immutable flat mesh inside a single-material group.
// Repeated members can share a mesh; their geometry is never expanded.
type MeshMember struct {
	Mesh  Shape
	Space maths.Subspace
}

// MeshGroup is a bounded collection of mesh instances, with a local hierarchy.
// Groups cannot contain groups. This gives tree/branch/triangle traversal a
// fixed depth on the GPU, without recursive kernel calls.
type MeshGroup struct {
	members       []MeshMember
	boxes         []maths.Box
	nodes         []groupNode
	faces, stored int
}

type groupNode struct {
	box          maths.Box
	first, count int
}

// NewMeshGroup checks transforms and builds the member hierarchy once.
// Materials belong to the enclosing thing, as they do for an ordinary mesh.
func NewMeshGroup(members []MeshMember) (*MeshGroup, error) {
	if !groupMemberCount(len(members)) {
		return nil, fmt.Errorf("shape: invalid mesh group member count")
	}
	group := &MeshGroup{members: append([]MeshMember(nil), members...)}
	seen := map[Shape]bool{}
	for _, member := range group.members {
		count := 0
		switch solid := member.Mesh.(type) {
		case *mesh.Mesh:
			if solid != nil {
				count = solid.FaceCount()
				if count > 0 {
					if _, smooth := solid.CornerNormals(0); smooth {
						return nil, fmt.Errorf("shape: groups require flat meshes")
					}
				}
			}
		case *Mesh:
			if solid != nil {
				count = len(solid.Faces)
				if solid.Smooth && len(solid.Normals) > 0 {
					return nil, fmt.Errorf("shape: groups require flat meshes")
				}
			}
		}
		if count == 0 || !ValidPlacement(member.Space) {
			return nil, fmt.Errorf("shape: group needs meshes with finite positive transforms")
		}
		group.faces += count
		if !seen[member.Mesh] {
			group.stored += count
			seen[member.Mesh] = true
		}
		group.boxes = append(group.boxes, memberBox(member))
	}
	group.nodes = []groupNode{{}}
	group.split(0, 0, len(members))
	return group, nil
}

// groupMemberCount checks the existing signed 32-bit hierarchy budget before
// allocating storage. Reserve at most two hierarchy slots per member.
func groupMemberCount(count int) bool {
	return count > 0 && count <= math.MaxInt32/2
}

// ValidPlacement requires finite coordinates, orthonormal axes and a
// positive uniform scale. Scale zero denotes the identity scale.
func ValidPlacement(space maths.Subspace) bool {
	finite := func(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
	if !finite(space.Origin.Len()) || !finite(space.Scale) || space.Scale < 0 {
		return false
	}
	x, y, z := space.Orientation.Out(maths.XAxis),
		space.Orientation.Out(maths.YAxis), space.Orientation.Out(maths.ZAxis)
	return math.Abs(x.Len()-1) < 1e-9 && math.Abs(y.Len()-1) < 1e-9 &&
		math.Abs(z.Len()-1) < 1e-9 && math.Abs(x.Dot(y)) < 1e-9 &&
		math.Abs(x.Dot(z)) < 1e-9 && math.Abs(y.Dot(z)) < 1e-9 &&
		x.Cross(y).Dot(z) > 0
}

// memberBox encloses all eight transformed mesh-bound corners.
func memberBox(member MeshMember) maths.Box {
	local := member.Mesh.Bounds()
	box := maths.Box{Min: member.Space.Out(local.Min), Max: member.Space.Out(local.Min)}
	for i := range 8 {
		point := local.Min
		if i&1 != 0 {
			point.X = local.Max.X
		}
		if i&2 != 0 {
			point.Y = local.Max.Y
		}
		if i&4 != 0 {
			point.Z = local.Max.Z
		}
		box = groupUnion(box, maths.Box{Min: member.Space.Out(point), Max: member.Space.Out(point)})
	}
	return box
}

// groupUnion is the enclosure of two member boxes.
func groupUnion(a, b maths.Box) maths.Box {
	return maths.Box{
		Min: maths.V(math.Min(a.Min.X, b.Min.X), math.Min(a.Min.Y, b.Min.Y), math.Min(a.Min.Z, b.Min.Z)),
		Max: maths.V(math.Max(a.Max.X, b.Max.X), math.Max(a.Max.Y, b.Max.Y), math.Max(a.Max.Z, b.Max.Z)),
	}
}

// split partitions member centres along their widest enclosing axis.
func (group *MeshGroup) split(node, first, last int) {
	box := group.boxes[first]
	for _, next := range group.boxes[first+1 : last] {
		box = groupUnion(box, next)
	}
	group.nodes[node] = groupNode{box: box, first: first, count: last - first}
	if last-first <= 4 {
		return
	}
	size := box.Max.Sub(box.Min)
	axis := 0
	if size.Y > size.X {
		axis = 1
	}
	if size.Z > math.Max(size.X, size.Y) {
		axis = 2
	}
	centre := func(box maths.Box) float64 {
		point := box.Min.Add(box.Max)
		if axis == 0 {
			return point.X
		}
		if axis == 1 {
			return point.Y
		}
		return point.Z
	}
	// Sort a temporary index to keep members and their boxes together.
	order := make([]int, last-first)
	for i := range order {
		order[i] = first + i
	}
	sort.SliceStable(order, func(i, j int) bool { return centre(group.boxes[order[i]]) < centre(group.boxes[order[j]]) })
	members, boxes := make([]MeshMember, len(order)), make([]maths.Box, len(order))
	for i, index := range order {
		members[i], boxes[i] = group.members[index], group.boxes[index]
	}
	copy(group.members[first:last], members)
	copy(group.boxes[first:last], boxes)
	child := len(group.nodes)
	group.nodes = append(group.nodes, groupNode{}, groupNode{})
	group.nodes[node].first, group.nodes[node].count = child, 0
	middle := first + (last-first)/2
	group.split(child, first, middle)
	group.split(child+1, middle, last)
}

// Members returns placements without exposing the group's mutable slice.
func (group *MeshGroup) Members() []MeshMember { return append([]MeshMember(nil), group.members...) }

// Bounds encloses all instances in the group's own coordinates.
func (group *MeshGroup) Bounds() maths.Box { return group.nodes[0].box }

// FaceCount includes every repeated mesh, for logical scene cost accounting.
func (group *MeshGroup) FaceCount() int { return group.faces }

// StoredFaces counts each distinct mesh once within this group.
func (group *MeshGroup) StoredFaces() int { return group.stored }

// Hit walks the local hierarchy and preserves distances through each scale.
func (group *MeshGroup) Hit(ray maths.Ray, near, far float64) (float64, bool) {
	distance, _, hit := group.hit(ray, near, far)
	return distance, hit
}

// HitNormal retains the member met by a ray. Shading can ask that member
// directly, without testing containment in every nearby closed prototype.
func (group *MeshGroup) HitNormal(ray maths.Ray, near, far float64) (float64, maths.Vec, bool) {
	distance, member, hit := group.hit(ray, near, far)
	if !hit {
		return distance, maths.Vec{}, false
	}
	point := member.Space.In(ray.Point(distance))
	if solid, ok := member.Mesh.(*mesh.Mesh); ok {
		scale := member.Space.Scale
		if scale == 0 {
			scale = 1
		}
		// Keep the actual face when thin leaves nearly touch.
		met, found := solid.Meet(member.Space.RayIn(ray), near/scale, far/scale)
		if found {
			normal := member.Space.Orientation.Out(solid.NormalAt(met)).Unit()
			return distance, normal, true
		}
	}
	normal := member.Space.Orientation.Out(Normal(member.Mesh, point)).Unit()
	return distance, normal, true
}

// hit walks the hierarchy and keeps the winning member as well as distance.
func (group *MeshGroup) hit(ray maths.Ray, near, far float64) (float64, MeshMember, bool) {
	stack, found := []int{0}, false
	var met MeshMember
	for len(stack) > 0 {
		node := group.nodes[stack[len(stack)-1]]
		stack = stack[:len(stack)-1]
		if _, _, hit := node.box.Hit(ray, near, far); !hit {
			continue
		}
		if node.count == 0 {
			stack = append(stack, node.first, node.first+1)
			continue
		}
		for _, member := range group.members[node.first : node.first+node.count] {
			scale := member.Space.Scale
			if scale == 0 {
				scale = 1
			}
			distance, hit := member.Mesh.(Hitter).Hit(member.Space.RayIn(ray), near/scale, far/scale)
			if hit {
				far, found = distance*scale, true
				met = member
			}
		}
	}
	return far, met, found
}

// nearest finds the closest member surface, pruning by box distance.
func (group *MeshGroup) nearest(point maths.Vec) (MeshMember, float64) {
	best, signed := math.Inf(1), math.Inf(1)
	var found MeshMember
	stack := []int{0}
	for len(stack) > 0 {
		node := group.nodes[stack[len(stack)-1]]
		stack = stack[:len(stack)-1]
		if groupBoxDistance(node.box, point) > best {
			continue
		}
		if node.count == 0 {
			stack = append(stack, node.first, node.first+1)
			continue
		}
		for _, member := range group.members[node.first : node.first+node.count] {
			scale := member.Space.Scale
			if scale == 0 {
				scale = 1
			}
			distance := member.Mesh.Distance(member.Space.In(point)) * scale
			if math.Abs(distance) < best {
				best, signed, found = math.Abs(distance), distance, member
			}
		}
	}
	return found, signed
}

// groupBoxDistance is a lower bound for distance to any enclosed surface.
func groupBoxDistance(box maths.Box, p maths.Vec) float64 {
	return maths.V(math.Max(0, math.Max(box.Min.X-p.X, p.X-box.Max.X)),
		math.Max(0, math.Max(box.Min.Y-p.Y, p.Y-box.Max.Y)),
		math.Max(0, math.Max(box.Min.Z-p.Z, p.Z-box.Max.Z))).Len()
}

// Distance measures the nearest component surface, signed by that component.
// A mesh group is a collection of surfaces, not a Boolean union.
func (group *MeshGroup) Distance(point maths.Vec) float64 {
	_, distance := group.nearest(point)
	return distance
}

// Normal returns the nearest member's normal in group coordinates.
func (group *MeshGroup) Normal(point maths.Vec) maths.Vec {
	member, _ := group.nearest(point)
	return member.Space.Orientation.Out(Normal(member.Mesh, member.Space.In(point))).Unit()
}
