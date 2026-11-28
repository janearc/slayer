package assembly

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
	"testing"
)

func TestPartsCanSharePoseOrSupplyAnAttachmentPose(t *testing.T) {
	rest := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}
	part := Part{Rest: rest, Faces: [][3]int{{0, 1, 2}}, Material: &thing.Material{Name: "scales"}}
	carry := func(p maths.Vec) maths.Vec { return p.Add(maths.V(0, 0, 3)) }
	fin := part
	fin.Carry = func(p maths.Vec) maths.Vec { return p.Add(maths.V(0, 0, 4)) }
	built, err := Build([]Part{part, fin}, carry)
	if err != nil || len(built) != 2 {
		t.Fatal(built, err)
	}
	for i, item := range built {
		m := item.Shape.(*mesh.Mesh)
		r := maths.Ray{At: maths.V(.25, .25, 5), Dir: maths.ZAxis.Neg()}
		hit, ok := m.Meet(r, 0, 8)
		if !ok || hit.Along() != float64(2-i) {
			t.Fatal(i, hit, ok)
		}
		q, _ := m.RestPointAt(hit)
		if q != maths.V(.25, .25, 0) || item.Material != part.Material {
			t.Fatal(q, item)
		}
	}
	if _, err := Build([]Part{{}}, carry); err == nil {
		t.Fatal("accepted empty part")
	}
	part.Faces = [][3]int{{0, 1, 3}}
	if _, err := Build([]Part{part}, nil); err == nil {
		t.Fatal("accepted out-of-range index")
	}
}
