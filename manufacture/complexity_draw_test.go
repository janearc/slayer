//go:build draw

package manufacture

import (
	"reflect"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestComplexityKeepsTheSpecimenExtentAndQuickGeometry(t *testing.T) {
	for _, name := range []string{"spruce", "norway", "coast-redwood", "giant-sequoia", "coast-redwood-shoot", "giant-sequoia-shoot"} {
		row, err := testKind(name)
		if err != nil {
			t.Fatal(err)
		}
		for number := range uint64(6) {
			original, err := Make(row, Sow(number))
			if err != nil {
				t.Fatal(err)
			}
			for _, detail := range Complexities() {
				made, err := MakeWithComplexity(row, Sow(number), detail)
				if err != nil {
					t.Fatal(err)
				}
				if made.Bounds() != original.Bounds() {
					t.Errorf("%s seed %d %s changed extent", name, number, detail)
				}
				if detail == "quick" && (!reflect.DeepEqual(made.skins(), original.skins()) || !reflect.DeepEqual(made.groups, original.groups)) {
					t.Error("quick changed the existing mesh")
				}
				if _, err := made.Things(maths.Subspace{}); err != nil {
					t.Fatalf("%s seed %d %s: %v", name, number, detail, err)
				}
			}
		}
	}
}
