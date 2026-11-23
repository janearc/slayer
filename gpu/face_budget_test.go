//go:build draw

package gpu

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// TestABendsPlaceReadsBackFromItsBits says that every place up to the
// budget, past the old 2^24, survives the float64 the rows carry and the
// float32 the kernel reads, as bits that are neither subnormal nor
// negative, and that a flat face keeps -1.
func TestABendsPlaceReadsBackFromItsBits(t *testing.T) {
	for _, place := range []int{0, 1, 1 << 24, 1<<24 + 1, 1 << 30,
		mostBends - 1} {
		carried := float32(bendNumber(place))
		bits := math.Float32bits(carried)
		if bits < bendBias || carried < 0 || int(bits-bendBias) != place {
			t.Errorf("place %d carried as bits %#x", place, bits)
		}
	}
	if bendNumber(-1) != -1 {
		t.Error("a flat face does not keep -1")
	}
}

// TestTheBendBudgetStopsBeforeInfinity is the anti-test: one place past
// the budget would carry infinity's bits.
func TestTheBendBudgetStopsBeforeInfinity(t *testing.T) {
	if !math.IsInf(float64(math.Float32frombits(uint32(mostBends)+bendBias)),
		1) {
		t.Error("the place past the budget is not infinity's bits")
	}
}

func TestSmoothFaceBudgetRefusesBeforeAllocatingPackedBuffers(t *testing.T) {
	// Exercise the production packing path under a small explicit cap.
	// The full float-index capacity is checked independently above.
	const limit = 3
	faces := make([]triangle, limit+1)
	for i := range faces {
		faces[i].smooth = true
	}
	// Without the budget refusal this fails as a coordinate error, so it
	// also checks that the guard runs before either buffer is serialized.
	faces[0].corners[0].X = math.NaN()
	encoded := newScene()
	if err := encoded.treeWithin(tree{triangles: faces}, limit); err == nil ||
		!strings.Contains(err.Error(), "faces shaded smooth") {
		t.Fatalf("smooth-face budget did not precede serialization: %v", err)
	}
	if len(encoded.triangles.bytes) != 0 || len(encoded.bends.bytes) != 0 ||
		len(encoded.nodes.bytes) != 0 {
		t.Fatal("over-budget geometry allocated a partial output")
	}
}

func TestSmoothFaceBudgetAcceptsItsEndpointAndIgnoresFlatFaces(t *testing.T) {
	for _, limit := range []int{0, 1, 3} {
		faces := make([]triangle, limit+2)
		for i := range faces {
			faces[i].corners = [3]maths.Vec{{}, maths.XAxis, maths.YAxis}
			faces[i].smooth = i < limit
		}
		grown := grow(faces)
		encoded := newScene()
		if err := encoded.treeWithin(grown, limit); err != nil {
			t.Fatalf("exactly %d smooth faces plus flat faces refused: %v", limit, err)
		}
		if len(encoded.bends.bytes) != limit*bendStruct.size() ||
			len(encoded.triangles.bytes) != len(faces)*triangleStruct.size() {
			t.Fatal("accepted geometry lost its smooth or flat records")
		}
		unlimited := newScene()
		if err := unlimited.tree(grown); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(encoded.bends.bytes, unlimited.bends.bytes) ||
			!bytes.Equal(encoded.triangles.bytes, unlimited.triangles.bytes) ||
			!bytes.Equal(encoded.nodes.bytes, unlimited.nodes.bytes) {
			t.Fatal("explicit cap changed accepted geometry")
		}
	}
}

func TestPackingRefusesNonfiniteVerticesAndUsedNormals(t *testing.T) {
	for _, damage := range []func(*triangle){
		func(face *triangle) { face.corners[0].X = math.NaN() },
		func(face *triangle) { face.corners[1].Z = math.MaxFloat64 },
		func(face *triangle) { face.smooth = true; face.normals[2].Y = math.Inf(1) },
	} {
		face := triangle{corners: [3]maths.Vec{maths.Vec{}, maths.XAxis, maths.YAxis}}
		damage(&face)
		encoded := newScene()
		if err := encoded.tree(tree{triangles: []triangle{face}}); err == nil || !strings.Contains(err.Error(), "finite float32") {
			t.Fatalf("invalid mesh data reached serialization: %v", err)
		}
	}
}

func TestFlatFacesDoNotReadUnusedNormals(t *testing.T) {
	face := triangle{corners: [3]maths.Vec{maths.Vec{}, maths.XAxis, maths.YAxis}}
	plain := newScene()
	if err := plain.tree(tree{triangles: []triangle{face}}); err != nil {
		t.Fatal(err)
	}
	face.normals[1].Y = math.NaN()
	unused := newScene()
	if err := unused.tree(tree{triangles: []triangle{face}}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plain.triangles.bytes, unused.triangles.bytes) || len(unused.bends.bytes) != 0 {
		t.Fatal("unused flat normals changed packed geometry")
	}
}
