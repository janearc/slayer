package manufacture

import (
	"github.com/janearc/slayer/maths"
	"math"
	"testing"
)

func TestPreparedPlacementKeepsGeometryAndMaterials(t *testing.T) {
	kind, err := KindNamed("spitzer")
	if err != nil {
		t.Fatal(err)
	}
	made, err := MakeWithComplexity(kind, Sow(1), "quick")
	if err != nil {
		t.Fatal(err)
	}
	ready, err := made.Prepare()
	if err != nil {
		t.Fatal(err)
	}
	at := maths.NewSubspace(maths.V(10, 20, 30), maths.About(maths.YAxis, .7), 3)
	got, err := ready.Things(at)
	if err != nil {
		t.Fatal(err)
	}
	want, err := made.Things(at)
	if err != nil {
		t.Fatal(err)
	}
	first, err := ready.Things(maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) || len(got) != len(first) {
		t.Fatal("lost material groups")
	}
	for i := range got {
		if got[i].Shape != first[i].Shape || got[i].Material != want[i].Material || got[i].At(0) != want[i].At(0) {
			t.Fatal("placement rebuilt geometry or changed appearance", i)
		}
		if got[i].Shape.Bounds() != want[i].Shape.Bounds() {
			t.Fatal("geometry changed", i)
		}
	}
	profile := ready.Resources()
	if !profile.RetainLocalGeometry || !profile.IndependentPlacement || !profile.NeedsNativeMeasurement || !profile.SharedMaterials || !profile.NeedsFrameStateSnapshot || profile.Geometry.StoredFaces == 0 {
		t.Fatal("incorrect pipeline facts", profile)
	}
	got[0].Shape = nil
	again, _ := ready.Things(at)
	if again[0].Shape == nil {
		t.Fatal("caller changed retained placement array")
	}
	if _, err := ready.Things(maths.NewSubspace(maths.Vec{}, maths.Identity, math.NaN())); err == nil {
		t.Fatal("invalid placement admitted")
	}
}
