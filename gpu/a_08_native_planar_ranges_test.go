//go:build darwin

package gpu

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// A nondegenerate planar mesh can have equal bounds on any one axis.
// Give it enough faces for branches and a preceding mesh for a nonzero
// pooled offset, then recover the authored interval from the actual tree.
func TestNativeFaceRangesKeepPlanarMeshesAndTheirPooledOffsets(t *testing.T) {
	for _, axes := range [][2]maths.Vec{
		{maths.YAxis, maths.ZAxis},
		{maths.XAxis, maths.ZAxis},
		{maths.XAxis, maths.YAxis},
	} {
		prefix, err := shape.NewMesh(
			[]maths.Vec{{}, maths.XAxis, maths.YAxis},
			[][3]int{{0, 1, 2}}, nil, false)
		if err != nil {
			t.Fatal(err)
		}
		var corners []maths.Vec
		var faces [][3]int
		for i := range 6 {
			start := axes[0].Scale(float64(i) * 3)
			at := len(corners)
			corners = append(corners, start,
				start.Add(axes[0]), start.Add(axes[1].Scale(2)))
			faces = append(faces, [3]int{at, at + 1, at + 2})
		}
		planar, err := shape.NewMesh(corners, faces, nil, false)
		if err != nil {
			t.Fatal(err)
		}
		world := thing.NewWorld()
		world.Add(thing.Thing{Shape: prefix})
		world.Add(thing.Thing{Shape: planar})
		encoded, err := encode(world, maths.Subspace{}, nil, 0, 4, 3)
		if err != nil {
			t.Fatal(err)
		}
		if len(encoded.standings) != 2 || len(encoded.faces.triangles) != 7 {
			t.Fatal("fixture lost the prefix face or six planar faces")
		}
		for i, want := range [][2]int{{0, 1}, {1, 6}} {
			root := encoded.standings[i].root
			if i == 1 && encoded.faces.nodes[root].count != 0 {
				t.Fatal("six-face fixture needs a branch, not just one leaf")
			}
			first, count := encoded.triangleRange(root)
			if first != want[0] || count != want[1] {
				t.Fatalf("axes %v: mesh %d covers %d+%d, want %d+%d",
					axes, i, first, count, want[0], want[1])
			}
		}
		// This builds the real checked host plan without dispatching a GPU.
		plan, _, err := encoded.accelerationPlan()
		if err != nil || plan == nil {
			t.Fatalf("planar mesh did not make a native plan: %v", err)
		}
	}
}
