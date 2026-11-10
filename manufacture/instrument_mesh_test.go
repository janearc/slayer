package manufacture_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
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
		polygon, ok := object.Shape.(*mesh.Mesh)
		if !ok || !polygon.Closed() ||
			polygon.FaceCount() > contract.MaximumFaces {
			test.Fatal("unbounded, open or unexpected instrument mesh",
				object.Material, object.Shape)
		}
		for _, point := range polygon.Points() {
			contract.assertPoint(test, point, bounds)
		}
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
