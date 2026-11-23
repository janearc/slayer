package manufacture

import (
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

func TestPrimaComplexityReceipt(test *testing.T) {
	row := spacecraft{design: "prima", aperture: measure.Metres(1.8)}
	object, err := row.madeAt(Sow(5), "demo")
	if err != nil {
		test.Fatal(err)
	}
	things, err := object.Things(maths.Local())
	if err != nil {
		test.Fatal(err)
	}
	staged := 0
	for _, one := range things {
		polygon, ok := one.Shape.(*mesh.Mesh)
		if !ok {
			test.Fatalf("unexpected PRIMA staged shape %T", one.Shape)
		}
		staged += polygon.FaceCount()
	}
	stored := object.faces()
	test.Logf("PRIMA demo complexity: components=%d stored_faces=%d staged_mesh_faces=%d",
		len(object.parts), stored, staged)
	if len(object.parts) < 100 || stored < 1000 || staged != stored {
		test.Fatalf("PRIMA detail is too coarse or changed at staging: components=%d stored_faces=%d staged_faces=%d",
			len(object.parts), stored, staged)
	}
}
