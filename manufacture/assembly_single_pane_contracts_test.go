package manufacture

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// Seed 22 has one lit pane and five dark panes in the authored house recipe.
func TestHomeKeepsASinglePaneMaterialAlongsideItsOtherWindows(t *testing.T) {
	row := homeRow()
	made, err := row.made(Sow(22))
	if err != nil {
		t.Fatal(err)
	}
	things, err := made.Things(maths.Subspace{})
	if err != nil {
		t.Fatal(err)
	}
	faces := map[string]int{}
	for _, one := range things {
		solid, ok := one.Shape.(*mesh.Mesh)
		if !ok {
			t.Fatalf("unexpected house surface %T", one.Shape)
		}
		faces[one.Material.Name] += solid.FaceCount()
	}
	// Each pane is a closed six-sided box, two triangles per side.
	if faces[litGlass.Name] != 12 || faces[darkGlass.Name] != 60 {
		t.Fatalf("lit and dark pane faces: %v", faces)
	}
}
