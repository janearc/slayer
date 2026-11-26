//go:build draw && darwin && arm64

package manufacture

import (
	"crypto/sha256"
	"fmt"
	"github.com/janearc/slayer/manufacture/geometry"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// These records were captured before the two field meshers were unified.
// Mesh points, faces, normals, shared prototypes, placements, material names
// and non-mesh bounds all contribute to the fingerprint.
func bioformMesherDigest(things []thing.Thing) string {
	hash := sha256.New()
	referenceCount(hash, len(things))
	seen := map[shape.Shape]int{}
	for _, part := range things {
		fmt.Fprintf(hash, "%T\x00", part.Shape)
		if part.Material != nil {
			fmt.Fprintf(hash, "%s\x00", part.Material.Name)
		} else {
			fmt.Fprint(hash, "<nil>\x00")
		}
		referenceSpace(hash, part.At(0))
		switch part.Shape.(type) {
		case *mesh.Mesh, *shape.Mesh, *shape.MeshGroup:
			referenceGeometry(hash, part.Shape, seen)
		default:
			box := part.Shape.Bounds()
			referenceVector(hash, box.Min)
			referenceVector(hash, box.Max)
		}
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func TestBioformMesherKeepsPreconsolidationGeometry(t *testing.T) {
	for _, model := range []struct {
		name  string
		build func(float64) ([]thing.Thing, error)
		parts int
		want  [2]string
	}{
		{"purussaurus", func(phase float64) ([]thing.Thing, error) { return PurussaurusStudy(phase, .15, .2) }, 15, [2]string{"c41c5140441f7ee8109e6cf6834a210a361bafb6b2d08c996b774d2e30934918", "512b55dc0ad219a5b91654dbcd64d99e4d103f30ecba3c95c6de2461dffeeaa0"}},
		{"hatzegopteryx", func(phase float64) ([]thing.Thing, error) { return HatzDensityStudy(phase, maths.Local()) }, 1, [2]string{"f3025031ecc36384365f0e5de82830f5d9dc68b4dda8a223077187d7f3cc6b39", "30f927f834043eebf13c6cac6cb143fa42c1da6e10b45edf543ec563484044ce"}},
		{"humboldt-squid", NewHumboldtSquid, 292, [2]string{"7b439f611fadc0d85e8dbd8b556c71a9e169f896f21300e6d913e7c9f11677ef", "23fb58c0824c4eea4b9057acc981b4575b9d1b14ff789d82ee49b5ad7bf8d63b"}},
		{"pliosaur", PliosaurContinuousSkin, 19, [2]string{"38817309fb37f1bc064af7a7a99c15330131ec5dd2542c46d6257b31da5134ac", "e914c40c3a2d07fcde0b97c455ccba7a466fec2e4ad6458dce9bcd39872f96f9"}},
	} {
		for i, phase := range []float64{0, 1.1} {
			t.Run(fmt.Sprintf("%s_%g", model.name, phase), func(t *testing.T) {
				parts, err := model.build(phase)
				if err != nil {
					t.Fatal(err)
				}
				if digest := bioformMesherDigest(parts); len(parts) != model.parts || digest != model.want[i] {
					t.Fatalf("got %d parts and %s, want %d and %s", len(parts), digest, model.parts, model.want[i])
				}
			})
		}
	}
	field := func(p maths.Vec) float64 { return p.Len() - .32 }
	warp := func(p maths.Vec) maths.Vec { return p.Add(maths.V(.03*p.Y, .04*p.X, -.02*p.Z)) }
	for _, test := range []struct {
		name            string
		build           func(func(maths.Vec) float64, func(maths.Vec) maths.Vec) (*shape.Mesh, error)
		vertices, faces int
		want            string
	}{
		{"coarse", geometry.FieldMesh, 2728, 5452, "482fb7e7a60bba39382bc52b1fdeb60d657e5ca6f7cbacf8ade484f66efd2fa1"},
		{"fine", geometry.FieldMeshFine, 7984, 15964, "9f8b0125e0b6fe20abbbb23b6490cf42ea48b1f4291296fabb6178fc155d8b2e"},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.build(field, warp)
			if err != nil {
				t.Fatal(err)
			}
			digest := bioformMesherDigest([]thing.Thing{{Shape: result}})
			if len(result.Vertices) != test.vertices || len(result.Faces) != test.faces || digest != test.want {
				t.Fatalf("%d vertices, %d faces, %s", len(result.Vertices), len(result.Faces), digest)
			}
		})
	}
}
