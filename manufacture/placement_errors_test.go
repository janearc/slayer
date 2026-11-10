package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

func TestPublicPlacementRefusesInvalidTransformsWithoutPanicking(t *testing.T) {
	made := mustMake(t, "spotlight", 2)
	for _, space := range []maths.Subspace{
		{Scale: -1}, {Scale: math.Inf(1)}, {Scale: math.NaN()},
		{Origin: maths.V(math.NaN(), 0, 0)},
		{Origin: maths.V(0, math.Inf(1), 0)},
		{Orientation: maths.Orientation{X: maths.XAxis.Scale(2), Y: maths.YAxis, Z: maths.ZAxis}},
		{Orientation: maths.Orientation{X: maths.XAxis, Y: maths.YAxis, Z: maths.ZAxis.Neg()}},
	} {
		if _, err := made.Lights(space); err == nil {
			t.Fatal("invalid light placement accepted", space)
		}
		if _, err := made.Things(space); err == nil {
			t.Fatal("invalid object placement accepted", space)
		}
		if _, err := made.placed(space); err == nil {
			t.Fatal("invalid composition accepted", space)
		}
	}
	plain := good{}
	if lights, err := plain.Lights(maths.Subspace{}); err != nil || len(lights) != 0 {
		t.Fatal("unlit valid placement changed", lights, err)
	}
}

func TestGroupPlacementReportsOverflowInsteadOfPanicking(t *testing.T) {
	triangle, err := (surface{{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}}).meshed()
	if err != nil {
		t.Fatal(err)
	}
	group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: triangle, Space: maths.Subspace{Scale: 2}}})
	if err != nil {
		t.Fatal(err)
	}
	made := good{groups: map[slot]*shape.MeshGroup{body: group}}
	if _, err = made.placed(maths.Subspace{Scale: math.MaxFloat64}); err == nil {
		t.Fatal("overflowing composed placement accepted")
	}
}
