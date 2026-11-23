package cities

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

func TestADesignsReachHoldsEveryPartPlacedWhereItStands(t *testing.T) {
	part := put("box", Box{2, 3, 4}, &thing.Material{Name: "box"},
		maths.V(10, 20, 30))
	design := Design{Name: "one box", Elements: []placedElement{part}}
	got, err := design.Reach("hull")
	if err != nil {
		t.Fatal(err)
	}
	want := maths.Box{Min: maths.V(10, 20, 30), Max: maths.V(12, 23, 34)}
	if got != want {
		t.Errorf("a reach of %+v, not %+v", got, want)
	}
}

func TestADesignOfNothingHasAnEmptyReachAtNought(t *testing.T) {
	got, err := Design{Name: "nothing"}.Reach("hull")
	if err != nil || got != (maths.Box{}) {
		t.Errorf("a reach of %+v: %v", got, err)
	}
}

func TestEveryPartOfAPiersDesignIsInsideItsReach(t *testing.T) {
	design, err := Pier(Seed(3))
	if err != nil {
		t.Fatal(err)
	}
	reach, err := design.Reach("hull")
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := design.Rendered("hull")
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range drawn {
		for _, triangle := range one.Mesh.Triangles {
			for _, corner := range triangle {
				at := one.Place.Out(corner)
				if at.X < reach.Min.X || at.X > reach.Max.X ||
					at.Y < reach.Min.Y || at.Y > reach.Max.Y ||
					at.Z < reach.Min.Z || at.Z > reach.Max.Z {
					t.Fatalf("%v is outside the reach %+v", at, reach)
				}
			}
		}
	}
}
