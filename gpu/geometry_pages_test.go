//go:build draw && darwin

package gpu

import (
	"bytes"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// pageWorld includes a branched mesh, a smooth mesh, two shared group
// members, and repeated placements of both a mesh and the group.
func pageWorld(t *testing.T) *thing.World {
	t.Helper()
	triangle := func(x float64, smooth bool) *shape.Mesh {
		vertices := []maths.Vec{maths.V(x, 0, 0), maths.V(x+1, 0, 0), maths.V(x, 1, 0)}
		var normals []maths.Vec
		if smooth {
			normals = []maths.Vec{maths.ZAxis, maths.ZAxis, maths.ZAxis}
		}
		made, err := shape.NewMesh(vertices, [][3]int{{0, 1, 2}}, normals, smooth)
		if err != nil {
			t.Fatal(err)
		}
		return made
	}
	shared := triangle(0, false)
	var corners []maths.Vec
	var faces [][3]int
	for i := range 6 {
		x := float64(3 * i)
		index := len(corners)
		corners = append(corners, maths.V(x, 0, 0), maths.V(x+1, 0, 0), maths.V(x, 1, 0))
		faces = append(faces, [3]int{index, index + 1, index + 2})
	}
	branch, err := shape.NewMesh(corners, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: shared}, {Mesh: branch}})
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shared})
	world.Add(thing.Thing{Shape: group})
	world.Add(thing.Thing{Shape: triangle(40, true)})
	world.Add(thing.Thing{Shape: group, Motion: thing.Still(maths.Subspace{Origin: maths.V(30, 0, 0)})})
	world.Add(thing.Thing{Shape: shared, Motion: thing.Still(maths.Subspace{Origin: maths.V(50, 0, 0)})})
	return world
}

func pendingFor(t *testing.T, world *thing.World) *scene {
	t.Helper()
	made := newScene()
	for _, each := range world.Things {
		if err := made.place(each, 0); err != nil {
			t.Fatal(err)
		}
	}
	return made
}

func TestGeometryPagesMatchScenePoolAndNativeLayout(t *testing.T) {
	world := pageWorld(t)
	pending := pendingFor(t, world)
	if len(pending.pending) != 3 || len(pending.groups) != 1 || len(pending.standings) != 5 {
		t.Fatal("fixture lost its shared group or distinct meshes")
	}
	var budget int
	for _, m := range pending.pending {
		size := len(m.tree.triangles)*triangleStruct.size() + len(m.tree.nodes)*nodeStruct.size()
		for _, face := range m.tree.triangles {
			if face.smooth {
				size += bendStruct.size()
			}
		}
		budget = max(budget, size)
	}
	original, err := encode(world, maths.Subspace{}, nil, 0, 4, 3)
	if err != nil {
		t.Fatal(err)
	}
	plan, root, err := original.accelerationPlan()
	if err != nil {
		t.Fatal(err)
	}
	firsts := bytes.Clone(original.nativeFirsts())
	var triangles, bends, nodes []byte
	var pages, mesh, face, bend, node int
	err = visitGeometryPages(pending.pending, budget, func(p geometryPage) error {
		pages++
		if p.firstMesh != mesh || p.firstTriangle != face || p.firstBend != bend || p.firstNode != node {
			t.Fatalf("wrong offsets in page %+v", p)
		}
		if len(p.triangles)+len(p.bends)+len(p.nodes) > budget {
			t.Fatal("page over budget")
		}
		mesh += p.meshes
		face += len(p.triangles) / triangleStruct.size()
		bend += len(p.bends) / bendStruct.size()
		node += len(p.nodes) / nodeStruct.size()
		triangles = append(triangles, p.triangles...)
		bends = append(bends, p.bends...)
		nodes = append(nodes, p.nodes...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if pages < 2 || mesh != len(pending.pending) {
		t.Fatalf("pages=%d meshes=%d", pages, mesh)
	}
	if !bytes.Equal(triangles, original.triangles.bytes) || !bytes.Equal(bends, original.bends.bytes) || !bytes.Equal(nodes, original.nodes.bytes) {
		t.Fatal("joining pages changes the scene encoder's packed bytes")
	}
	// The real host-side native plan must still read the same global IDs.
	original.triangles.bytes, original.bends.bytes, original.nodes.bytes = triangles, bends, nodes
	joinedPlan, joinedRoot, err := original.accelerationPlan()
	if err != nil || joinedRoot != root || !plan.Layout().Same(joinedPlan.Layout()) || !bytes.Equal(firsts, original.nativeFirsts()) {
		t.Fatalf("page reassembly changed the native plan or IDs: %v", err)
	}
	// Anti-test: a real byte change cannot pass the packed-byte comparison.
	triangles[0] ^= 1
	if bytes.Equal(triangles, planTrianglesForTest(t, world)) {
		t.Fatal("comparison missed corrupted geometry")
	}
}

func planTrianglesForTest(t *testing.T, world *thing.World) []byte {
	t.Helper()
	encoded, err := encode(world, maths.Subspace{}, nil, 0, 4, 3)
	if err != nil {
		t.Fatal(err)
	}
	return encoded.triangles.bytes
}

func TestGeometryPagesRefuseBeforeEmitting(t *testing.T) {
	pending := pendingFor(t, pageWorld(t)).pending
	called := 0
	visit := func(geometryPage) error { called++; return nil }
	for _, budget := range []int{-1, 0, 1} {
		if err := visitGeometryPages(pending, budget, visit); err == nil || called != 0 {
			t.Fatalf("budget %d: err=%v emitted=%d", budget, err, called)
		}
	}
	bad := append([]pendingMesh(nil), pending...)
	bad[0].tree.triangles = append([]triangle(nil), bad[0].tree.triangles...)
	bad[0].tree.triangles[0].corners[0].X = math.NaN()
	if err := visitGeometryPages(bad, 1<<20, visit); err == nil || called != 0 {
		t.Fatalf("nonfinite mesh: err=%v emitted=%d", err, called)
	}
	if err := visitGeometryPages(pending, 1<<20, nil); err == nil || !strings.Contains(err.Error(), "visitor") {
		t.Fatalf("nil visitor: %v", err)
	}
	sentinel := errors.New("stop")
	err := visitGeometryPages(pending, 1<<20, func(geometryPage) error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Fatalf("visitor error: %v", err)
	}
	if err := visitGeometryPages(nil, 1, visit); err != nil || called != 0 {
		t.Fatalf("empty: %v, %d", err, called)
	}
}

func TestGeometryPageCountAndByteLimits(t *testing.T) {
	bytes, err := packedPageBytes(sizedPage{triangles: 2, bends: 1, nodes: 3})
	if err != nil || bytes != 2*triangleStruct.size()+bendStruct.size()+3*nodeStruct.size() {
		t.Fatalf("ordinary bytes %d: %v", bytes, err)
	}
	if _, err := packedPageBytes(sizedPage{triangles: math.MaxInt}); err == nil {
		t.Fatal("accepted overflowing packed size")
	}
	sum, err := addPageCounts(sizedPage{triangles: 1, bends: 2, nodes: 3}, sizedPage{triangles: 4, bends: 5, nodes: 6})
	if err != nil || sum.triangles != 5 || sum.bends != 7 || sum.nodes != 9 {
		t.Fatalf("ordinary counts %+v: %v", sum, err)
	}
	for _, pair := range [][2]sizedPage{
		{{triangles: math.MaxInt32}, {triangles: 1}},
		{{nodes: math.MaxInt32}, {nodes: 1}},
		{{bends: mostBends}, {bends: 1}},
	} {
		if _, err := addPageCounts(pair[0], pair[1]); err == nil {
			t.Fatalf("accepted overflow: %+v", pair)
		}
	}
	if _, err := addPageCounts(sizedPage{triangles: math.MaxInt32 - 1, bends: mostBends - 1, nodes: math.MaxInt32 - 1}, sizedPage{triangles: 1, bends: 1, nodes: 1}); err != nil {
		t.Fatalf("rejected valid boundary: %v", err)
	}
}

func TestGeometryPagesPreflightLaterFailureAndVisitorStop(t *testing.T) {
	pending := pendingFor(t, pageWorld(t)).pending
	last := len(pending) - 1
	bad := append([]pendingMesh(nil), pending...)
	bad[last].tree.triangles = append([]triangle(nil), bad[last].tree.triangles...)
	bad[last].tree.triangles[0].corners[0].X = math.NaN()
	calls := 0
	if err := visitGeometryPages(bad, 1<<20, func(geometryPage) error { calls++; return nil }); err == nil || calls != 0 {
		t.Fatalf("later invalid mesh: err=%v, emitted=%d", err, calls)
	}
	var largest int
	for _, m := range pending {
		size, err := pageMeshSize(m)
		if err != nil {
			t.Fatal(err)
		}
		largest = max(largest, size.bytes)
	}
	stop := errors.New("stop after first")
	err := visitGeometryPages(pending, largest, func(geometryPage) error { calls++; return stop })
	if !errors.Is(err, stop) || calls != 1 {
		t.Fatalf("did not stop after visitor error: %v, %d", err, calls)
	}
}

func TestGeometryPagePreflightChecksTotalIndices(t *testing.T) {
	for _, sizes := range [][]sizedPage{
		{{triangles: math.MaxInt32}, {triangles: 1}},
		{{nodes: math.MaxInt32}, {nodes: 1}},
		{{bends: mostBends}, {bends: 1}},
	} {
		if err := checkPageSizes(sizes, 1); err == nil {
			t.Fatalf("accepted overflow: %+v", sizes)
		}
	}
	if err := checkPageSizes([]sizedPage{{bytes: 2}}, 1); err == nil {
		t.Fatal("accepted oversized mesh")
	}
	if err := checkPageSizes([]sizedPage{{bytes: 1}, {bytes: 1}}, 1); err != nil {
		t.Fatalf("rejected separate full pages: %v", err)
	}
}
