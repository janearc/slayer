package manufacture

import (
	"math"
	"reflect"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestLegacyHullEnclosesTheFoliageRatherThanTheWoodyScaffold(t *testing.T) {
	for _, name := range []string{"spruce", "norway"} {
		row, err := KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		quick, err := MakeWithComplexity(row, Sow(5), "quick")
		if err != nil {
			t.Fatal(err)
		}
		hull, err := MakeWithComplexity(row, Sow(5), "hull")
		if err != nil {
			t.Fatal(err)
		}
		// The foliage is the quick tree's leaf group: needled shoots as
		// shared mesh instances, not parts.
		leaves := quick.groups[leaf]
		if leaves == nil || len(leaves.Members()) == 0 {
			t.Fatal("crown reference contains no foliage")
		}
		if len(hull.parts) != 2 || hull.parts[0].of != body || hull.parts[1].of != leaf {
			t.Fatal("legacy hull lost its trunk/crown assembly")
		}
		crown, ok := hull.parts[1].shape.(crownShape)
		if !ok {
			t.Fatalf("%s hull crown is %T, want a crown box", name, hull.parts[1].shape)
		}
		// The crown is the shoots' physical bounds: it encloses every
		// foliage mesh, and stays within the whole tree's extent.
		foliage, whole := leaves.Bounds(), *quick.extent
		if crown.box.Min.X > foliage.Min.X || crown.box.Min.Y > foliage.Min.Y ||
			crown.box.Min.Z > foliage.Min.Z || crown.box.Max.X < foliage.Max.X ||
			crown.box.Max.Y < foliage.Max.Y || crown.box.Max.Z < foliage.Max.Z {
			t.Fatalf("%s hull crown %v leaves foliage %v outside", name, crown.box, foliage)
		}
		if crown.box.Min.X < whole.Min.X || crown.box.Min.Y < whole.Min.Y ||
			crown.box.Min.Z < whole.Min.Z || crown.box.Max.X > whole.Max.X ||
			crown.box.Max.Y > whole.Max.Y || crown.box.Max.Z > whole.Max.Z {
			t.Fatalf("%s hull crown %v reaches past the tree %v", name, crown.box, whole)
		}
		if !reflect.DeepEqual(hull.parts[0], quick.parts[0]) {
			t.Fatal("hull replaced its trunk with a branch")
		}
		physical := maths.Box{Min: maths.V(math.Inf(1), math.Inf(1), math.Inf(1)), Max: maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))}
		for _, face := range hull.parts[1].skin() {
			for _, p := range face {
				physical.Min = maths.V(math.Min(physical.Min.X, p.X), math.Min(physical.Min.Y, p.Y), math.Min(physical.Min.Z, p.Z))
				physical.Max = maths.V(math.Max(physical.Max.X, p.X), math.Max(physical.Max.Y, p.Y), math.Max(physical.Max.Z, p.Z))
			}
		}
		if physical != crown.box {
			t.Fatalf("%s placed hull crown %v, want %v", name, physical, crown.box)
		}

	}
}
