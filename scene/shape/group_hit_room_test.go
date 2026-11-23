package shape

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// groupHitRoomWalk is a group, with the ray and the point that are put to
// it, so that each question is a method a count of allocations can run.
type groupHitRoomWalk struct {
	group *MeshGroup
	ray   maths.Ray
	point maths.Vec
}

// hit puts the ray to the group's tree.
func (walk groupHitRoomWalk) hit() {
	walk.group.hit(walk.ray, nought, math.Inf(unity))
}

// distance asks the group how far the point is.
func (walk groupHitRoomWalk) distance() { walk.group.Distance(walk.point) }

// normal asks the group for the normal at the point.
func (walk groupHitRoomWalk) normal() { walk.group.Normal(walk.point) }

// members asks the group for a copy of its members.
func (walk groupHitRoomWalk) members() { walk.group.Members() }

// plateRow is a group of flat plates in a row along x, each a metre on,
// facing up: enough of them that its tree has branches to walk.
func plateRow(t *testing.T) *MeshGroup {
	t.Helper()
	corners := []maths.Vec{maths.V(nought, nought, nought),
		maths.V(nought, nought, unity), maths.V(unity, nought, nought)}
	faces := triangleFaces(t, triangleFacesFile, "lone")
	plate, err := NewMesh(corners, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	members := make([]MeshMember, pickedCount(t, "group-hit-room-members"))
	for index := range members {
		where := maths.Local()
		where.Origin = maths.V(float64(index), nought, nought)
		members[index] = MeshMember{Mesh: plate, Space: where}
	}
	group, err := NewMeshGroup(members)
	if err != nil {
		t.Fatal(err)
	}
	return group
}

// aboveThePlate is a point a metre over the middle of a plate in the row.
var aboveThePlate = groupHitRoomAbove()

// groupHitRoomAbove is the point over the middle of the plate at the
// row's centre, found from the numbers book, which a missing name panics.
func groupHitRoomAbove() maths.Vec {
	members := int(triangleChosen("group-hit-room-members"))
	offset := triangleChosen("group-hit-room-offset")
	return maths.V(float64(members/pair)+offset, unity, offset)
}

// TestAWalkThroughAGroupAllocatesNothing says that a ray walked through
// a group's tree keeps its waiting boxes on the stack: it allocates
// nothing, and still meets the plate under it.
func TestAWalkThroughAGroupAllocatesNothing(t *testing.T) {
	walk := groupHitRoomWalk{group: plateRow(t), ray: triangleRay(
		aboveThePlate, maths.V(nought, -unity, nought))}
	tolerance := picked(t, "group-hit-room-tolerance")
	far, _, found := walk.group.hit(walk.ray, nought, math.Inf(unity))
	if !found || math.Abs(far-unity) > tolerance {
		t.Fatalf("%v %v down to the row, want a metre", far, found)
	}
	walks := testing.AllocsPerRun(pickedCount(t, "group-hit-room-walks"),
		walk.hit)
	if walks != nought {
		t.Errorf("%v allocations a walk, want none", walks)
	}
}

// TestANearestWalkThroughAGroupAllocatesNothing says that the walks marching
// and shading make, Distance and Normal, keep their waiting boxes on the
// stack as a hit's walk does, and still answer: a metre from the plate under
// the point, with a normal one unit long.
func TestANearestWalkThroughAGroupAllocatesNothing(t *testing.T) {
	group := plateRow(t)
	walk := groupHitRoomWalk{group: group, point: aboveThePlate}
	tolerance := picked(t, "group-hit-room-tolerance")
	if got := group.Distance(aboveThePlate); math.Abs(got-unity) > tolerance {
		t.Fatalf("%v from the row, want a metre", got)
	}
	if got := group.Normal(aboveThePlate).Len(); math.Abs(got-unity) >
		tolerance {
		t.Fatalf("a normal %v long, want one", got)
	}
	runs := pickedCount(t, "group-hit-room-walks")
	distances := testing.AllocsPerRun(runs, walk.distance)
	normals := testing.AllocsPerRun(runs, walk.normal)
	if distances != nought || normals != nought {
		t.Errorf("%v allocations a Distance and %v a Normal, want none",
			distances, normals)
	}
}

// TestACopyOfAGroupsMembersAllocates is the anti-test of the two above: a
// call on the same group that does allocate is counted as allocating, so
// the count is one that can see an allocation.
func TestACopyOfAGroupsMembersAllocates(t *testing.T) {
	walk := groupHitRoomWalk{group: plateRow(t)}
	copies := testing.AllocsPerRun(pickedCount(t, "group-hit-room-walks"),
		walk.members)
	if copies == nought {
		t.Error("a copy of the members allocated nothing")
	}
}
