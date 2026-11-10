package manufacture

import "testing"

func TestRecursiveTreeLevelsChangeTheActualFoliageGeometry(t *testing.T) {
	for _, name := range []string{"coast-redwood", "giant-sequoia"} {
		row, err := KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		counts := map[string]int{}
		for _, detail := range Complexities() {
			made, err := MakeWithComplexity(row, Sow(5), detail)
			if err != nil {
				t.Fatal(err)
			}
			counts[detail] = made.faces()
			if detail == "hull" {
				if len(made.groups) != 0 {
					t.Fatal("hull retained detailed foliage instances")
				}
			} else if len(made.groups) == 0 {
				t.Fatal("recursive tree lost its foliage instances")
			}
		}
		if !(counts["hull"] < counts["clumps"] && counts["clumps"] < counts["quick"] && counts["quick"] < counts["demo"]) {
			t.Fatalf("%s levels do not change geometry: %v", name, counts)
		}
	}
}
