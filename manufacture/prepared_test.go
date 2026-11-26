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
	if again, _ := ready.Things(at); again[0].Shape == nil {
		t.Fatal("caller changed retained placement array")
	}
	misplaced := maths.NewSubspace(maths.Vec{}, maths.Identity, math.NaN())
	if _, err := ready.Things(misplaced); err == nil {
		t.Fatal("invalid placement admitted")
	}
}

func TestPreparedPlacementPreservesNativeAttachments(t *testing.T) {
	for _, name := range []string{"bay-seal", "walrus-fresh-5", "orca-clean-05"} {
		t.Run(name, func(t *testing.T) {
			row, err := referenceNativeKind(name)
			if err != nil {
				t.Fatal(err)
			}
			made, err := MakeWithComplexity(row, Sow(17), "quick")
			if err != nil {
				t.Fatal(err)
			}
			ready, err := made.Prepare()
			if err != nil {
				t.Fatal(err)
			}
			first, err := ready.Things(maths.Local())
			if err != nil {
				t.Fatal(err)
			}
			for _, at := range []maths.Subspace{
				maths.NewSubspace(maths.V(4, 5, 6), maths.About(maths.YAxis, .7), 1.3),
				maths.NewSubspace(maths.V(-2, 3, 4), maths.About(maths.XAxis, -.4), .8),
				maths.Local(),
			} {
				got, err := ready.Things(at)
				if err != nil {
					t.Fatal(err)
				}
				want, err := made.Things(at)
				if err != nil {
					t.Fatal(err)
				}
				if len(got) != len(want) || len(got) != len(first) {
					t.Fatal("part count changed")
				}
				for i := range got {
					if got[i].Shape != first[i].Shape || got[i].Material != want[i].Material || got[i].At(0) != want[i].At(0) {
						t.Fatalf("part %d: prepared placement or shared geometry differs", i)
					}
				}
			}
			if len(first) > 1 && first[1].At(0).Origin == (maths.Vec{}) {
				t.Fatal("test did not exercise offset attachments")
			}
		})
	}
}
