package manufacture

import (
	"testing"

	"github.com/janearc/slayer/geom"
)

func TestConiferDefault(t *testing.T) {
	params := DefaultConiferParams()
	conifer := Conifer(geom.Vec3{0, 0, 0}, params)

	if len(conifer) == 0 {
		t.Errorf("Conifer with default params returned no triangles")
	}
}

func TestConiferTriangleCount(t *testing.T) {
	params := ConiferParams{
		Height:      5,
		BaseRadius:  1,
		Depth:       2,
		TaperRate:   0.7,
		DepthScale:  0.7,
		BranchStart: 0.3,
		Sides:       4,
	}
	conifer := Conifer(geom.Vec3{0, 0, 0}, params)

	if len(conifer) < 4*2 {
		t.Errorf("Conifer depth 2 should have at least trunk + branches")
	}
}

func TestConiferBounds(t *testing.T) {
	params := DefaultConiferParams()
	params.Height = 10
	params.Depth = 1

	conifer := Conifer(geom.Vec3{0, 0, 0}, params)
	bounds := geom.BoundsOfTriangles(conifer)

	if bounds.Max[2] < params.Height*0.9 {
		t.Errorf("Conifer should reach height %v, got %v",
			params.Height, bounds.Max[2])
	}
}

func TestConiferDepthZero(t *testing.T) {
	params := ConiferParams{
		Height:     5,
		BaseRadius: 1,
		Depth:      0,
		Sides:      4,
	}
	conifer := Conifer(geom.Vec3{0, 0, 0}, params)

	if len(conifer) == 0 {
		t.Errorf("Conifer depth 0 should have at least trunk")
	}

	expectedTrunkTriangles := 4
	if len(conifer) != expectedTrunkTriangles {
		t.Errorf("Conifer depth 0 with sides 4 should have %d triangles, got %d",
			expectedTrunkTriangles, len(conifer))
	}
}

func TestConiferNegativeDepth(t *testing.T) {
	params := ConiferParams{Depth: -1}
	conifer := Conifer(geom.Vec3{0, 0, 0}, params)

	if len(conifer) != 0 {
		t.Errorf("Conifer with negative depth should return empty")
	}
}

func TestConiferPositionOffset(t *testing.T) {
	params := DefaultConiferParams()
	params.Depth = 1

	base := geom.Vec3{5, 10, 15}
	conifer := Conifer(base, params)
	bounds := geom.BoundsOfTriangles(conifer)

	if bounds.Min[0] < 3 || bounds.Min[1] < 8 || bounds.Min[2] < 14 {
		t.Errorf("Conifer at position %v should be within reasonable offset",
			base)
	}
}

func TestConiferTaperRate(t *testing.T) {
	params1 := DefaultConiferParams()
	params1.Depth = 1
	params1.TaperRate = 0.5

	params2 := DefaultConiferParams()
	params2.Depth = 1
	params2.TaperRate = 0.9

	conifer1 := Conifer(geom.Vec3{0, 0, 0}, params1)
	conifer2 := Conifer(geom.Vec3{0, 0, 0}, params2)

	bounds1 := geom.BoundsOfTriangles(conifer1)
	bounds2 := geom.BoundsOfTriangles(conifer2)

	size1 := bounds1.Size()
	size2 := bounds2.Size()

	if size1[0] >= size2[0] {
		t.Errorf("More aggressive taper should produce thinner tree")
	}
}

func BenchmarkConifer(b *testing.B) {
	params := DefaultConiferParams()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Conifer(geom.Vec3{0, 0, 0}, params)
	}
}
