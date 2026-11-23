package manufacture_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

type meshContract struct {
	Tolerance    float64
	MaximumFaces int
}

func (contract meshContract) assertMeshes(test *testing.T,
	things []thing.Thing, bounds maths.Box,
) {
	test.Helper()
	for _, object := range things {
		group, grouped := object.Shape.(*shape.MeshGroup)
		if !grouped {
			contract.assertMesh(test, object, maths.Local(), bounds)
			continue
		}
		if group.MemberCount() == 0 {
			test.Fatal("an instrument group holds no member", object.Material.Name)
		}
		for _, member := range group.Members() {
			contract.assertMesh(test, thing.Thing{Shape: member.Mesh,
				Material: object.Material}, member.Space, bounds)
		}
	}
}

// assertMesh holds one mesh to the contract: a closed mesh of no more faces
// than the limit, every point of which, placed by the space it sits in, is
// inside the bounds. A mesh that is a member of a group is placed by the
// member's space, and a mesh of its own is placed by the local one.
func (contract meshContract) assertMesh(test *testing.T, object thing.Thing,
	space maths.Subspace, bounds maths.Box,
) {
	test.Helper()
	polygon, ok := object.Shape.(*mesh.Mesh)
	if !ok {
		test.Fatal("unexpected instrument mesh type", object.Material.Name)
	}
	if !polygon.Closed() || polygon.FaceCount() > contract.MaximumFaces {
		test.Fatalf("unbounded or open instrument mesh: material %q faces %d closed %t (limit %d)",
			object.Material.Name, polygon.FaceCount(), polygon.Closed(),
			contract.MaximumFaces)
	}
	for _, point := range polygon.Points() {
		contract.assertPoint(test, space.Out(point), bounds)
	}
}

func (contract meshContract) assertPoint(test *testing.T,
	point maths.Vec, bounds maths.Box,
) {
	test.Helper()
	if math.IsNaN(point.Len()) ||
		math.IsInf(point.Len(), spacecraftPositiveSign) {
		test.Fatal("non-finite instrument surface", point)
	}
	slack := contract.Tolerance
	outside := point.X < bounds.Min.X-slack || point.X > bounds.Max.X+slack ||
		point.Y < bounds.Min.Y-slack || point.Y > bounds.Max.Y+slack ||
		point.Z < bounds.Min.Z-slack || point.Z > bounds.Max.Z+slack
	if outside {
		test.Fatal("instrument surface outside its physical support",
			point, bounds)
	}
}
