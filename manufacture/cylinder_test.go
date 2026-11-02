package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/geom"
)

func TestCylinderTriangleCount(t *testing.T) {
	sides := 8
	triangles := Cylinder(geom.Vec3{0, 0, 0}, 1, 0.5, 0.5, sides, 1)

	expectedCount := sides * 2
	if len(triangles) != expectedCount {
		t.Errorf("Cylinder with %d sides = %d triangles, want %d",
			sides, len(triangles), expectedCount)
	}
}

func TestCylinderBounds(t *testing.T) {
	base := geom.Vec3{0, 0, 0}
	height := 10.0
	radius := 2.0
	triangles := Cylinder(base, height, radius, radius, 16, 1)

	bounds := geom.BoundsOfTriangles(triangles)

	if bounds.Max[2] != height {
		t.Errorf("Cylinder height: max.Z = %v, want %v", bounds.Max[2], height)
	}

	maxRadius := math.Max(math.Abs(bounds.Max[0]), math.Abs(bounds.Max[1]))
	if maxRadius < radius*0.9 || maxRadius > radius*1.1 {
		t.Errorf("Cylinder radius: got %v, want ~%v", maxRadius, radius)
	}
}

func TestConeIsZeroTopRadius(t *testing.T) {
	cone := Cone(geom.Vec3{0, 0, 0}, 5, 1, 8, 1)

	if len(cone) == 0 {
		t.Errorf("Cone returned no triangles")
	}

	bounds := geom.BoundsOfTriangles(cone)

	if bounds.Max[2] != 5 {
		t.Errorf("Cone height: max.Z = %v, want 5", bounds.Max[2])
	}
}

func TestCylinderTaperedRadius(t *testing.T) {
	base := geom.Vec3{0, 0, 0}
	triangles := Cylinder(base, 10, 2, 1, 8, 1)

	bounds := geom.BoundsOfTriangles(triangles)

	baseRadius := math.Sqrt(bounds.Min[0]*bounds.Min[0] + bounds.Min[1]*bounds.Min[1])
	topRadius := math.Sqrt(
		(bounds.Max[0]*bounds.Max[0] + bounds.Max[1]*bounds.Max[1]) / 2)

	if baseRadius < topRadius {
		t.Errorf("Tapered cylinder: base radius %v should be >= top radius %v",
			baseRadius, topRadius)
	}
}

func TestCylinderInvalidSides(t *testing.T) {
	triangles := Cylinder(geom.Vec3{0, 0, 0}, 1, 1, 1, 2, 1)

	expectedCount := 3 * 2
	if len(triangles) != expectedCount {
		t.Errorf("Cylinder with sides=2 should use minimum 3, got %d triangles",
			len(triangles))
	}
}

func TestCylinderZeroHeight(t *testing.T) {
	triangles := Cylinder(geom.Vec3{0, 0, 0}, 0, 1, 1, 8, 1)

	if len(triangles) != 0 {
		t.Errorf("Cylinder with zero height returned triangles")
	}
}

func TestCylinderZeroRadius(t *testing.T) {
	triangles := Cylinder(geom.Vec3{0, 0, 0}, 1, 0, 0, 8, 1)

	if len(triangles) != 0 {
		t.Errorf("Cylinder with zero radii returned triangles")
	}
}
