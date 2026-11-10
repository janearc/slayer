package gpu

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// These are valid float64 meshes made by the public constructor. The
// larger one exceeds the GPU coordinate format without a large allocation.
func TestGeometryPackingRefusalsReachThingsAndTheSceneEncoder(t *testing.T) {
	for _, test := range []struct {
		name    string
		x       float64
		refused bool
	}{
		{"ordinary planar mesh", 2, false},
		{"outside float32", 2 * float64(math.MaxFloat32), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			mesh, err := shape.NewMesh([]maths.Vec{
				maths.V(test.x, 0, 0),
				maths.V(test.x, 1, 0),
				maths.V(test.x, 0, 1),
			}, [][3]int{{0, 1, 2}}, nil, false)
			if err != nil {
				t.Fatalf("public mesh fixture is invalid: %v", err)
			}
			world := thing.NewWorld()
			world.Add(thing.Thing{Shape: mesh})
			encoded := newScene()
			err = encoded.things(world, 0)
			if test.refused {
				if err == nil || !strings.Contains(err.Error(), "finite float32") {
					t.Fatalf("things lost the geometry refusal: %v", err)
				}
				if len(encoded.triangles.bytes) != 0 ||
					len(encoded.bends.bytes) != 0 || len(encoded.nodes.bytes) != 0 {
					t.Fatal("refused geometry left partial packed faces")
				}
			} else if err != nil || encoded.counts.triangles != 1 {
				t.Fatalf("ordinary mesh refused or lost its face: %v", err)
			}
			made, err := encode(world, maths.Subspace{}, nil, 0, 4, 3)
			if test.refused {
				if made != nil || err == nil ||
					!strings.Contains(err.Error(), "finite float32") {
					t.Fatalf("scene encoder lost the geometry refusal: %v", err)
				}
			} else if err != nil || made == nil || made.counts.triangles != 1 {
				t.Fatalf("ordinary scene refused or lost its face: %v", err)
			}
		})
	}
}
