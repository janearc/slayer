package shape

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// pointsNoMemberIsNearest are points every member's distance fails to be a
// number or is infinite for: not a number, infinite, and so far out that
// squaring them overflows.
var pointsNoMemberIsNearest = groupPointsStrangers()

// groupPointsStrangers are those points, the far one found from the
// numbers book, which a missing name panics.
func groupPointsStrangers() []maths.Vec {
	far := triangleChosen("group-points-far")
	return []maths.Vec{
		maths.V(math.NaN(), nought, nought),
		maths.V(nought, math.NaN(), nought),
		maths.V(math.Inf(unity), nought, nought),
		maths.V(nought, nought, math.Inf(-unity)),
		maths.V(far, far, far),
	}
}

// groupPointsAsk is a group asked for the normal at a point, the answer
// kept for the test to read.
type groupPointsAsk struct {
	group  *MeshGroup
	point  maths.Vec
	normal maths.Vec
}

// find asks the group for the normal at the point, and keeps it.
func (ask *groupPointsAsk) find() { ask.normal = ask.group.Normal(ask.point) }

// TestAGroupHasNoNormalWhereNoMemberIsNearest says that Normal, for a point
// that is not a number, is infinite or is far enough out that every
// distance overflows, does not panic, and is the zero vector; and that
// Distance there is positive infinity.
func TestAGroupHasNoNormalWhereNoMemberIsNearest(t *testing.T) {
	group := plateRow(t)
	zero := maths.V(nought, nought, nought)
	for _, point := range pointsNoMemberIsNearest {
		ask := &groupPointsAsk{group: group, point: point}
		if got := refusal(ask.find); got != "" {
			t.Errorf("%v: Normal panicked with %q", point, got)
		}
		if ask.normal != zero {
			t.Errorf("%v: Normal %v, want the zero vector", point,
				ask.normal)
		}
		if got := group.Distance(point); !math.IsInf(got, unity) {
			t.Errorf("%v: Distance %v, want positive infinity", point, got)
		}
	}
}

// TestAGroupHasANormalWhereAMemberIsNearest is the anti-test of the one
// above: a point over a plate has a normal one long and a distance of a
// metre, so the zero vector above is the answer for those points and not for
// every point.
func TestAGroupHasANormalWhereAMemberIsNearest(t *testing.T) {
	group := plateRow(t)
	tolerance := picked(t, "group-points-tolerance")
	if got := group.Normal(aboveThePlate).Len(); math.Abs(got-unity) >
		tolerance {
		t.Errorf("Normal %v long, want one", got)
	}
	if got := group.Distance(aboveThePlate); math.Abs(got-unity) >
		tolerance {
		t.Errorf("Distance %v, want a metre", got)
	}
}
