package hillside

import (
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"strings"
	"testing"
	"time"
)

func spinSelection(t *testing.T, material string) set.Thing {
	t.Helper()
	text := positionedSetText("space", "        east: 0\n        north: 0\n        part-spin: {material: '"+material+"', pivot-metres: [1, 2, 3], axis: [0, 0, 1], rpm: 60}\n", "        east: 0\n        north: 0\n")
	return readPositionSet(t, text).Things()[0]
}
func TestPartSpinSelectionOnlyTouchesNewNamedPartsAndSharesGeometry(t *testing.T) {
	stage := testStaging()
	material := &thing.Material{Name: "blade"}
	geometry := &shape.Sphere{Radius: 1}
	parent := thing.Still(maths.NewSubspace(maths.V(12, 7, -30), maths.Rotation(maths.YAxis, .4), 2))
	old := stage.world.Add(thing.Thing{Shape: geometry, Material: material, Motion: parent})
	start := len(stage.world.Things)
	selected := stage.world.Add(thing.Thing{Shape: geometry, Material: material, Motion: parent})
	other := stage.world.Add(thing.Thing{Shape: geometry, Material: &thing.Material{Name: "hull"}, Motion: parent})
	light, err := emit.NewDistant(maths.Up, maths.Grey(1))
	if err != nil {
		t.Fatal(err)
	}
	stage.world.Lamps = []thing.Lamp{light}
	if err = stage.partSpin(start, spinSelection(t, "blade")); err != nil {
		t.Fatal(err)
	}
	if old.Motion != parent || other.Motion != parent || stage.world.Lamps[0] != light {
		t.Fatal("other object/material/light changed")
	}
	if selected.Shape != geometry || selected.Material != material {
		t.Fatal("shape/material replaced")
	}
	if selected.At(0) != parent.At(0) {
		t.Fatal("zero phase changed exact pose")
	}
	if selected.At(250*time.Millisecond) == parent.At(250*time.Millisecond) {
		t.Fatal("selected part did not spin")
	}
	pivot := maths.V(1, 2, 3)
	if got, want := selected.At(250*time.Millisecond).Out(pivot), parent.At(250*time.Millisecond).Out(pivot); got.Sub(want).Len() > 1e-12 {
		t.Fatal("native pivot or parent scale changed")
	}
}
func TestPartSpinRefusesNoMatchBeforeMutation(t *testing.T) {
	stage := testStaging()
	parent := thing.Still(maths.Local())
	one := stage.world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: &thing.Material{Name: "hull"}, Motion: parent})
	err := stage.partSpin(0, spinSelection(t, "missing"))
	if err == nil || !strings.Contains(err.Error(), "matched no newly placed parts") {
		t.Fatalf("no-match result%v", err)
	}
	if one.Motion != parent {
		t.Fatal("no-match mutated existing motion")
	}
}
func TestPartSpinConstructorFailureDoesNotMutateEarlierMatch(t *testing.T) {
	stage := testStaging()
	parent := thing.Still(maths.Local())
	material := &thing.Material{Name: "blade"}
	one := stage.world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: material, Motion: parent})
	stage.world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: material})
	if err := stage.partSpin(0, spinSelection(t, "blade")); err == nil {
		t.Fatal("nil parent accepted")
	}
	if one.Motion != parent {
		t.Fatal("earlier match mutated before complete validation")
	}
}
