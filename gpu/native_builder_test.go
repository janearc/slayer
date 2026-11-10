//go:build darwin

package gpu

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/thing"
)

type damagedNativeGroup struct {
	*metal.Acceleration
	calls int
}

// Instances simulates a damaged group submission at the builder boundary.
// The real builder refuses the singular matrix; no error is manufactured.
func (plan *damagedNativeGroup) Instances(places ...metal.Placement) (int, error) {
	plan.calls++
	return plan.Acceleration.Instances(metal.Placement{})
}

func TestNativeGroupBuilderRefusalStopsTranslationAndPreservesItsCause(t *testing.T) {
	group, _ := groupFixture(t)
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: group})
	encoded, err := encode(world, maths.Subspace{}, film.Pinhole{}, 0, 4, 3)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := metal.NewAcceleration(encoded.triangles.bytes, 16)
	if err != nil {
		t.Fatal(err)
	}
	plan := &damagedNativeGroup{Acceleration: actual}
	_, err = encoded.populateAcceleration(plan)
	if err == nil || !strings.Contains(err.Error(), "instance needs an earlier child and valid transform") {
		t.Fatalf("group builder cause lost: %v", err)
	}
	if plan.calls != 1 {
		t.Fatalf("world translation continued after group refusal: %d calls", plan.calls)
	}
	// A refused group did not poison the actual builder's earlier mesh nodes.
	place, err := metal.Placed(0, nativeMatrix(maths.Subspace{}), 3)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := actual.Instances(place); err != nil {
		t.Fatalf("group refusal damaged earlier nodes: %v", err)
	}
}
