package skeleton

import (
	"errors"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// testCheckSkeleton is a skeleton of two bones, a root and one hung from
// it, failing the test if either is refused.
func testCheckSkeleton(t *testing.T) *Skeleton {
	t.Helper()
	bones := New()
	for _, row := range []struct{ name, parent string }{
		{"root", ""}, {"limb", "root"},
	} {
		err := bones.Add(row.name, row.parent, maths.V(0, 0, 0),
			maths.V(1, 0, 0), .1, .1)
		if err != nil {
			t.Fatal(err)
		}
	}
	return bones
}

func TestCheckPassesAPoseOfJointsTheSkeletonHas(t *testing.T) {
	bones := testCheckSkeleton(t)
	turn := NewTurn(maths.V(0, 0, 1), .3)
	for name, pose := range map[string]Pose{
		"no turn": {}, "one": Pose{}.With("limb", turn),
		"both": Pose{}.With("root", turn).Compose("limb", turn),
	} {
		if err := bones.Check(pose); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestCheckNamesEveryJointWithNoBone(t *testing.T) {
	bones := testCheckSkeleton(t)
	turn := NewTurn(maths.V(0, 0, 1), .3)
	pose := Pose{}.With("limb", turn).With("wing", turn).Compose("antler", turn)
	err := bones.Check(pose)
	if !errors.Is(err, ErrNoJoint) {
		t.Fatalf("a pose turning two joints with no bone gave %v", err)
	}
	said := err.Error()
	if !strings.Contains(said, "antler, wing") || strings.Contains(said, "limb") {
		t.Fatalf("the refusal names %q", said)
	}
}
