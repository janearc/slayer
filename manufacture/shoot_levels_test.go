package manufacture

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestLooseShootsKeepTheirBoundsAndProvideEveryDetailLevel(t *testing.T) {
	for _, name := range []string{"coast-redwood-shoot", "giant-sequoia-shoot"} {
		row, err := KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		counts := map[string]int{}
		for _, detail := range Complexities() {
			made, err := MakeWithComplexity(row, Sow(3), detail)
			if err != nil {
				t.Fatal(err)
			}
			things, err := made.Things(maths.Subspace{})
			if err != nil || len(things) == 0 {
				t.Fatal("empty shoot", name, detail, err)
			}
			counts[detail] = made.faces()
			if detail == "hull" {
				if len(made.groups) != 0 || len(made.parts) != 1 {
					t.Fatal("hull is not a coarse crown")
				}
			} else if len(made.groups) != 2 {
				t.Fatal("shoot lost wood or foliage", name, detail)
			}
		}
		if counts["hull"] >= counts["clumps"] || counts["clumps"] >= counts["quick"] ||
			counts["quick"] >= counts["demo"] {
			t.Fatal("shoot detail did not increase", name, counts)
		}
	}
}
