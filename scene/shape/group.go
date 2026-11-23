package shape

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// MeshMember places a flat mesh inside a single-material group. NewMeshGroup
// accepts only nonnil *mesh.Mesh and *shape.Mesh values. Repeated members may
// share the same mesh; their geometry is not expanded.
type MeshMember struct {
	Mesh  Shape
	Space maths.Subspace
}

// MeshGroup is a bounded collection of mesh instances, with a local hierarchy.
// Groups cannot contain groups. This gives tree/branch/triangle traversal a
// fixed depth on the GPU, without recursive kernel calls.
type MeshGroup struct {
	members       []MeshMember
	nodes         []groupNode
	faces, stored int
}

type groupNode struct {
	box          maths.Box
	first, count int
}

// The ways a group is refused.
var (
	errGroupMember = errors.New("shape: group needs meshes that are nonnil " +
		"*mesh.Mesh or *shape.Mesh values with a face")
	errGroupFlat  = errors.New("shape: groups require flat meshes")
	errGroupBuilt = errors.New("shape: a *shape.Mesh in a group is made by " +
		"NewMesh, which builds its tree")
	errGroupPlacement = errors.New("shape: a group member needs a finite " +
		"placement, a right-handed orthonormal orientation and a scale of " +
		"zero or more")
)

// NewMeshGroup checks members and builds their hierarchy once. It accepts 1
// through 1,073,741,823 members (MaxInt32/2), each a nonnil *mesh.Mesh or
// *shape.Mesh with at least one face.
//
// A *shape.Mesh must be made by NewMesh,
// which builds its tree; a literal is refused. Meshes must be flat:
// *mesh.Mesh values with corner normals and *shape.Mesh values with Smooth and
// normals are refused. Other Shape implementations, including nested
// MeshGroups, are refused.
//
// Each member's placement must have finite coordinates, a finite nonnegative
// uniform scale, and a right-handed orthonormal orientation (within 1e-9).
// A zero scale means identity scale. Materials belong to the enclosing thing, as
// they do for an ordinary mesh.
func NewMeshGroup(members []MeshMember) (*MeshGroup, error) {
	if !groupMemberCount(len(members)) {
		return nil, fmt.Errorf("shape: invalid mesh group member count")
	}
	group := &MeshGroup{members: append([]MeshMember(nil), members...)}
	seen := map[Shape]bool{}
	boxes := make([]maths.Box, nought, len(members))
	for _, member := range group.members {
		count, err := memberFaces(member)
		if err != nil {
			return nil, err
		}
		if !ValidPlacement(member.Space) {
			return nil, errGroupPlacement
		}
		group.faces += count
		if !seen[member.Mesh] {
			group.stored += count
			seen[member.Mesh] = true
		}
		boxes = append(boxes, memberBox(member))
	}
	group.nodes = []groupNode{{}}
	group.split(boxes, nought, nought, len(members))
	return group, nil
}

// memberFaces is how many faces a member's mesh has, or why the mesh cannot
// be in a group: it is not one of the two kinds, has no face, is smooth, or
// is a *shape.Mesh that was not made by NewMesh.
func memberFaces(member MeshMember) (int, error) {
	switch solid := member.Mesh.(type) {
	case *mesh.Mesh:
		if solid == nil || solid.FaceCount() == nought {
			return nought, errGroupMember
		}
		if _, smooth := solid.CornerNormals(nought); smooth {
			return nought, errGroupFlat
		}
		return solid.FaceCount(), nil
	case *Mesh:
		if solid == nil || len(solid.Faces) == nought {
			return nought, errGroupMember
		}
		if len(solid.nodes) == nought {
			return nought, errGroupBuilt
		}
		if solid.Smooth && len(solid.Normals) > nought {
			return nought, errGroupFlat
		}
		return len(solid.Faces), nil
	}
	return nought, errGroupMember
}

// groupMemberCount checks the signed 32-bit hierarchy budget before
// allocating storage. Reserve at most two hierarchy slots per member.
func groupMemberCount(count int) bool {
	return count > nought && count <= math.MaxInt32/pair
}

// orthonormalTolerance is how far a placement's axes may be from unit
// length and from square to each other. It is a rounding allowance, chosen
// in picks.yaml.
var orthonormalTolerance = pick("orthonormal-tolerance")

// ValidPlacement requires finite coordinates, orthonormal axes and a
// scale of zero or more. Scale zero denotes the identity scale.
func ValidPlacement(space maths.Subspace) bool {
	if !finiteNumber(space.Origin.Len()) || !finiteNumber(space.Scale) ||
		space.Scale < nought {
		return false
	}
	across, upward, onward := space.Orientation.Out(maths.XAxis),
		space.Orientation.Out(maths.YAxis), space.Orientation.Out(maths.ZAxis)
	return unitLength(across) && unitLength(upward) && unitLength(onward) &&
		square(across, upward) && square(across, onward) &&
		square(upward, onward) && across.Cross(upward).Dot(onward) > nought
}

// unitLength is whether axis is one long, within orthonormalTolerance.
func unitLength(axis maths.Vec) bool {
	return math.Abs(axis.Len()-unity) < orthonormalTolerance
}

// square is whether two axes are at right angles, within
// orthonormalTolerance.
func square(first, second maths.Vec) bool {
	return math.Abs(first.Dot(second)) < orthonormalTolerance
}

// The bits of a box corner's number that choose the high end of an axis.
const (
	boxCorners = 8 // defined: a box has eight corners
	highX      = 1 // defined: the bit that chooses the high end of x
	highY      = 2 // defined: the bit that chooses the high end of y
	highZ      = 4 // defined: the bit that chooses the high end of z
)

// memberBox encloses all eight transformed mesh-bound corners.
func memberBox(member MeshMember) maths.Box {
	local := member.Mesh.Bounds()
	start := member.Space.Out(local.Min)
	box := maths.Box{Min: start, Max: start}
	for corner := range boxCorners {
		point := local.Min
		if corner&highX != nought {
			point.X = local.Max.X
		}
		if corner&highY != nought {
			point.Y = local.Max.Y
		}
		if corner&highZ != nought {
			point.Z = local.Max.Z
		}
		placed := member.Space.Out(point)
		box = groupUnion(box, maths.Box{Min: placed, Max: placed})
	}
	return box
}

// groupUnion is the enclosure of two member boxes.
func groupUnion(first, second maths.Box) maths.Box {
	return maths.Box{
		Min: maths.V(math.Min(first.Min.X, second.Min.X),
			math.Min(first.Min.Y, second.Min.Y),
			math.Min(first.Min.Z, second.Min.Z)),
		Max: maths.V(math.Max(first.Max.X, second.Max.X),
			math.Max(first.Max.Y, second.Max.Y),
			math.Max(first.Max.Z, second.Max.Z)),
	}
}

// leafMembers is how many members a node holds before it is cut in two. It
// is chosen, in picks.yaml.
var leafMembers = int(pick("leaf-members"))

// The axes, as along reads them.
const (
	alongX = 0 // defined: the x axis
	alongY = 1 // defined: the y axis
	alongZ = 2 // defined: the z axis
)

// widestAxis is the axis a box is widest on, x before y before z when equal.
func widestAxis(box maths.Box) int {
	size := box.Max.Sub(box.Min)
	axis := alongX
	if size.Y > size.X {
		axis = alongY
	}
	if size.Z > math.Max(size.X, size.Y) {
		axis = alongZ
	}
	return axis
}

// byCentre sorts member indices by where their boxes' middles lie on one
// axis, keeping members of equal middle in their order.
type byCentre struct {
	order []int
	boxes []maths.Box
	axis  int
}

// Len is how many indices there are to sort.
func (sorted byCentre) Len() int { return len(sorted.order) }

// Swap exchanges two indices.
func (sorted byCentre) Swap(first, second int) {
	sorted.order[first], sorted.order[second] = sorted.order[second],
		sorted.order[first]
}

// Less is whether the box at the first index has its middle before the box
// at the second.
func (sorted byCentre) Less(first, second int) bool {
	return centreAlong(sorted.boxes[sorted.order[first]], sorted.axis) <
		centreAlong(sorted.boxes[sorted.order[second]], sorted.axis)
}

// centreAlong is twice the middle of a box along an axis, which orders
// middles as well as the middles do.
func centreAlong(box maths.Box, axis int) float64 {
	return along(box.Min.Add(box.Max), axis)
}

// split partitions member centres along their widest enclosing axis. The
// boxes are each member's box, in the members' order, and are put in the new
// order beside the members.
func (group *MeshGroup) split(boxes []maths.Box, node, first, last int) {
	box := boxes[first]
	for _, next := range boxes[first+unity : last] {
		box = groupUnion(box, next)
	}
	group.nodes[node] = groupNode{box: box, first: first, count: last - first}
	if last-first <= leafMembers {
		return
	}
	group.sortRange(boxes, first, last, widestAxis(box))
	child := len(group.nodes)
	group.nodes = append(group.nodes, groupNode{}, groupNode{})
	group.nodes[node].first, group.nodes[node].count = child, nought
	middle := first + (last-first)/pair
	group.split(boxes, child, first, middle)
	group.split(boxes, child+unity, middle, last)
}

// sortRange puts the members from first up to last, and their boxes, in
// order of where the boxes' middles lie on axis.
func (group *MeshGroup) sortRange(boxes []maths.Box, first, last, axis int) {
	order := make([]int, last-first)
	for index := range order {
		order[index] = first + index
	}
	sort.Stable(byCentre{order: order, boxes: boxes, axis: axis})
	members := make([]MeshMember, len(order))
	placed := make([]maths.Box, len(order))
	for index, from := range order {
		members[index], placed[index] = group.members[from], boxes[from]
	}
	copy(group.members[first:last], members)
	copy(boxes[first:last], placed)
}

// Members returns placements without exposing the group's mutable slice.
func (group *MeshGroup) Members() []MeshMember { return append([]MeshMember(nil), group.members...) }

// MemberCount is the number of placements, without copying their array.
func (group *MeshGroup) MemberCount() int { return len(group.members) }

// Member reads one placement by value without exposing the group's array.
func (group *MeshGroup) Member(index int) MeshMember { return group.members[index] }

// Bounds encloses all instances in the group's own coordinates.
func (group *MeshGroup) Bounds() maths.Box { return group.nodes[nought].box }

// FaceCount includes every repeated mesh, for logical scene cost accounting.
func (group *MeshGroup) FaceCount() int { return group.faces }

// StoredFaces counts each distinct mesh once within this group.
func (group *MeshGroup) StoredFaces() int { return group.stored }

// scaleOf is the scale a placement measures by: its own, or one when it is
// zero.
func scaleOf(space maths.Subspace) float64 {
	if space.Scale == nought {
		return unity
	}
	return space.Scale
}

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
		scale := scaleOf(member.Space)
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

// hitRoom is how many boxes waiting to be walked a walk of the group's tree
// keeps on the goroutine's stack before it grows onto the heap: one a level
// and one over. A tree that halves its members has at most as many levels as
// there are bits in the count of them, and a group has at most MaxInt32/2, so
// twice the bits of an int32 is room to spare: a walk allocates nothing, and a
// deeper one would still be walked.
const hitRoom = int32Bits * pair

// int32Bits is how many bits a member count of up to MaxInt32 takes.
const int32Bits = 32 // defined: the bits in an int32

// hit walks the hierarchy and keeps the winning member as well as distance.
func (group *MeshGroup) hit(ray maths.Ray, near, far float64) (float64, MeshMember, bool) {
	var room [hitRoom]int
	stack, found := append(room[:nought], nought), false
	var met MeshMember
	for len(stack) > nought {
		node := group.nodes[stack[len(stack)-unity]]
		stack = stack[:len(stack)-unity]
		if _, _, hit := node.box.Hit(ray, near, far); !hit {
			continue
		}
		if node.count == nought {
			stack = append(stack, node.first, node.first+unity)
			continue
		}
		for _, member := range group.members[node.first : node.first+node.count] {
			scale := scaleOf(member.Space)
			distance, hit := member.Mesh.(Hitter).Hit(member.Space.RayIn(ray), near/scale, far/scale)
			if hit {
				far, found = distance*scale, true
				met = member
			}
		}
	}
	return far, met, found
}

// nearest finds the closest member surface, pruning by box distance, and
// says whether any member was close enough to count: none is for a point
// that is not a number, or so far out that every distance overflows. Like
// hit, it keeps its waiting boxes on the stack and allocates nothing.
func (group *MeshGroup) nearest(point maths.Vec) (MeshMember, float64, bool) {
	best, signed := math.Inf(unity), math.Inf(unity)
	var found MeshMember
	var room [hitRoom]int
	stack, counted := append(room[:nought], nought), false
	for len(stack) > nought {
		node := group.nodes[stack[len(stack)-unity]]
		stack = stack[:len(stack)-unity]
		if groupBoxDistance(node.box, point) > best {
			continue
		}
		if node.count == nought {
			stack = append(stack, node.first, node.first+unity)
			continue
		}
		for _, member := range group.members[node.first : node.first+node.count] {
			scale := scaleOf(member.Space)
			distance := member.Mesh.Distance(member.Space.In(point)) * scale
			if math.Abs(distance) < best {
				best, signed, found = math.Abs(distance), distance, member
				counted = true
			}
		}
	}
	return found, signed, counted
}

// groupBoxDistance is a lower bound for distance to any enclosed surface.
func groupBoxDistance(box maths.Box, point maths.Vec) float64 {
	return maths.V(boxGap(box.Min.X, box.Max.X, point.X),
		boxGap(box.Min.Y, box.Max.Y, point.Y),
		boxGap(box.Min.Z, box.Max.Z, point.Z)).Len()
}

// boxGap is how far at lies outside the span from low to high, or nought
// inside it.
func boxGap(low, high, at float64) float64 {
	return math.Max(nought, math.Max(low-at, at-high))
}

// Distance measures the nearest component surface, signed by that component.
// A mesh group is a collection of surfaces, not a Boolean union. For a point
// no member is nearest to, which is one that is not a number or is so far
// out every distance overflows, it is positive infinity.
func (group *MeshGroup) Distance(point maths.Vec) float64 {
	_, distance, _ := group.nearest(point)
	return distance
}

// Normal returns the nearest member's normal in group coordinates. For a
// point no member is nearest to, which is one that is not a number or is so
// far out every distance overflows, no way is out and it is the zero vector.
func (group *MeshGroup) Normal(point maths.Vec) maths.Vec {
	member, _, found := group.nearest(point)
	if !found {
		return maths.Vec{}
	}
	local := Normal(member.Mesh, member.Space.In(point))
	return member.Space.Orientation.Out(local).Unit()
}
