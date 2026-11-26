package placement

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestDisplayEdgesSharePositionsAcrossDifferentFaceNormals(t *testing.T) {
	builder := newDisplayBuilder()
	a, b := maths.V(0, 0, 0), maths.V(1, 0, 0)
	c, d := maths.V(0, 1, 0), maths.V(0, 0, 1)
	builder.triangle([3]maths.Vec{a, b, c}, [3]maths.Vec{}, false, maths.Local(), maths.Light{})
	builder.triangle([3]maths.Vec{b, a, d}, [3]maths.Vec{}, false, maths.Local(), maths.Light{})
	mesh, err := builder.finish()
	if err != nil {
		t.Fatal(err)
	}
	if len(mesh.Vertices) != 6 || len(mesh.Indices) != 6 || len(mesh.Edges) != 10 {
		t.Fatalf("shared edge with distinct normals: %d vertices, %d triangle indices, %d edge indices", len(mesh.Vertices), len(mesh.Indices), len(mesh.Edges))
	}
}
