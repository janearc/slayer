//go:build draw

package gpu

import (
	"errors"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// A reference mesh is flat unless both Smooth and vertex normals are present.
// Keeping unused normals, or an unused Smooth flag, must not remove a scene.
//
// From 2026-11-12 a reference mesh that is smooth, standing by itself, is
// drawn: its face carries its corners' normals and is bent by them and not
// lifted, as slayer's tracer shades it. In a group it is still refused. And
// a smooth mesh whose normals are no longer one to a vertex is refused
// either way, by a name of its own: malformed data does not become a
// picture by the door that smoothing opened.
func TestReferenceMeshFlatCasesRemainEncodableInsideAndOutsideGroups(t *testing.T) {
	points := []maths.Vec{{}, maths.XAxis, maths.YAxis}
	faces := [][3]int{{0, 1, 2}}
	normals := []maths.Vec{maths.XAxis, maths.XAxis, maths.XAxis}
	const (
		drawn   = ""
		partial = "a smooth mesh without a normal for every vertex"
	)
	for _, test := range []struct {
		name    string
		smooth  bool
		normal  []maths.Vec
		refused bool
		// direct is what the mesh standing by itself is refused as, or
		// drawn when it is not refused
		direct string
		later  []maths.Vec
	}{
		{"flat", false, nil, false, drawn, nil},
		{"flat with retained normals", false, normals, false, drawn, nil},
		{"smooth flag without normals", true, nil, false, drawn, nil},
		{"smooth with normals", true, normals, true, drawn, nil},
		{"smooth with a late partial normal list", true, nil, true, partial, normals[:1]},
	} {
		for _, grouped := range []bool{false, true} {
			name := test.name + "/direct"
			if grouped {
				name = test.name + "/grouped"
			}
			t.Run(name, func(t *testing.T) {
				initialSmooth := test.smooth
				if grouped && test.refused && test.later == nil {
					initialSmooth = false
				}
				reference, err := shape.NewMesh(points, faces, test.normal, initialSmooth)
				if err != nil {
					t.Fatal(err)
				}
				var object shape.Shape = reference
				if grouped {
					object, err = shape.NewMeshGroup([]shape.MeshMember{{Mesh: reference}})
					if err != nil {
						t.Fatal(err)
					}
				}
				world := thing.NewWorld()
				world.Add(thing.Thing{Shape: object})
				if initialSmooth != test.smooth {
					if _, err := encode(world, maths.Subspace{}, nil, 0, 4, 3); err != nil {
						t.Fatalf("flat group before enabling smoothing: %v", err)
					}
					reference.Smooth = test.smooth
				}
				if test.later != nil {
					if _, err := encode(world, maths.Subspace{}, nil, 0, 4, 3); err != nil {
						t.Fatalf("valid mesh before late change: %v", err)
					}
					// Public reference fields can change after the group is made.
					// Refuse unsupported smoothing before flattening even a partial
					// normal list; do not let malformed data bypass that refusal.
					reference.Normals = append([]maths.Vec(nil), test.later...)
				}
				encoded, err := encode(world, maths.Subspace{}, nil, 0, 4, 3)
				want := test.direct
				if grouped && test.refused {
					want = "a smooth reference mesh in a group"
				}
				if want != drawn {
					var cause unsupported
					if !errors.As(err, &cause) || cause.what != want {
						t.Fatalf("smooth reference lost its GPU refusal: %v", err)
					}
					return
				}
				if err != nil {
					t.Fatalf("valid reference refused: %v", err)
				}
				if len(encoded.faces.triangles) != 1 {
					t.Fatal("reference lost its face")
				}
				face := encoded.faces.triangles[0]
				if test.refused {
					// smooth, standing by itself: bent by its corners'
					// normals, in their order, and not lifted
					if !face.smooth || !face.unlifted ||
						face.normals != [3]maths.Vec{maths.XAxis, maths.XAxis, maths.XAxis} {
						t.Fatalf("smooth reference is encoded as %+v", face)
					}
					return
				}
				if face.smooth || face.corners != [3]maths.Vec{{}, maths.XAxis, maths.YAxis} ||
					reference.Normal(maths.V(0.25, 0.25, 0)) != maths.ZAxis {
					t.Fatal("unused reference normals changed geometric flat shading")
				}
			})
		}
	}
}
