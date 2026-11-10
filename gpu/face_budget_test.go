package gpu

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestSmoothFaceLimitIsTheConsecutiveFloatIndexCapacity(t *testing.T) {
	// Find the first integer whose successor rounds back to itself.
	// The packed bend lane must name every earlier integer without gaps.
	firstGap := float32(1)
	for firstGap+1 != firstGap {
		firstGap *= 2
	}
	if mostBends != int(firstGap) {
		t.Fatalf("smooth-face budget %d, consecutive float indices end at %g", mostBends, firstGap)
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
