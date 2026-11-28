package placement

import (
	"reflect"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

func TestReservedStagedBuilderKeepsMeshGeometry(t *testing.T) {
	made := &shape.Mesh{}
	for i := 0; i < 1200; i++ {
		x := float64(i)
		base := len(made.Vertices)
		made.Vertices = append(made.Vertices, maths.V(x, 0, 0), maths.V(x+1, 0, 0), maths.V(x, 1, 0))
		made.Faces = append(made.Faces, [3]int{base, base + 1, base + 2})
	}
	plain, reserved := newDisplayBuilder(), newStagedDisplayBuilder(made)
	for _, builder := range []*displayBuilder{plain, reserved} {
		if err := builder.surface(made, maths.Local(), maths.Light{}); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(plain.skin.Vertices, reserved.skin.Vertices) || !reflect.DeepEqual(plain.skin.Indices, reserved.skin.Indices) || !reflect.DeepEqual(plain.skin.Edges, reserved.skin.Edges) {
		t.Fatal("reserving tables changed the display mesh")
	}
}

// BenchmarkDisplayMeshGrid measures conversion of a shared staged mesh.
func BenchmarkDisplayMeshGrid(b *testing.B) {
	const side = 150
	made := &shape.Mesh{}
	for y := 0; y <= side; y++ {
		for x := 0; x <= side; x++ {
			made.Vertices = append(made.Vertices, maths.V(float64(x), 0, float64(y)))
		}
	}
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			a := y*(side+1) + x
			made.Faces = append(made.Faces, [3]int{a, a + 1, a + side + 1}, [3]int{a + 1, a + side + 2, a + side + 1})
		}
	}
	for _, test := range []struct {
		name string
		make func() *displayBuilder
	}{{"unreserved", newDisplayBuilder}, {"reserved", func() *displayBuilder { return newStagedDisplayBuilder(made) }}} {
		b.Run(test.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				builder := test.make()
				if err := builder.surface(made, maths.Local(), maths.Light{}); err != nil {
					b.Fatal(err)
				}
				if _, err := builder.finish(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
