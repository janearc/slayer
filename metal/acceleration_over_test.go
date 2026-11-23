//go:build draw && darwin

package metal

import (
	"encoding/binary"
	"math"
	"slices"
	"testing"
)

// overProbe is acceleratedProbe with the triangles bound as an input
// beside the rays, as a grid stream holds them: the rays at 0, the
// triangles at 1, then the output, the origin and the hierarchy.
const overProbe = `
#include <metal_stdlib>
#include <metal_raytracing>
using namespace metal;
using namespace metal::raytracing;
kernel void probe(device const float4 *rays [[buffer(0)]],
                  device const float4 *triangles [[buffer(1)]],
                  device float *out [[buffer(2)]],
                  constant uint2 &origin [[buffer(3)]],
                  acceleration_structure<instancing> scene [[buffer(4)]],
                  uint2 at [[thread_position_in_grid]]) {
	ray r;
	r.origin = rays[2 * at.x].xyz;
	r.direction = rays[2 * at.x + 1].xyz;
	r.min_distance = 0;
	r.max_distance = rays[2 * at.x + 1].w;
	intersector<instancing, max_levels<3>> query;
	query.assume_geometry_type(geometry_type::triangle);
	query.accept_any_intersection(true);
	auto met = query.intersect(r, scene, uint(rays[2 * at.x].w));
	out[at.x] = met.type == intersection_type::triangle ? 1 : 0;
}
`

// the places of the probe's inputs in a stream
const (
	overRays      = 0
	overTriangles = 1
)

// overRaysFor is one ray down from x, at the mask, as long as reach.
func overRaysFor(probes ...[3]float32) []byte {
	var rays []byte
	for _, probe := range probes {
		for _, value := range []float32{probe[0], 0, 0, probe[1], 0, 0, -1,
			probe[2]} {
			rays = binary.LittleEndian.AppendUint32(rays,
				math.Float32bits(value))
		}
	}
	return rays
}

// overPlan is the one triangle placed twice under a root, as
// TestNativeHierarchyKeepsTransformsMasksAndSegments places it: at x of
// three with a mask of one, and at minus three with a mask of three.
func overPlan(t *testing.T, vertices []byte) (*Acceleration, int) {
	t.Helper()
	plan, err := NewAcceleration(vertices, 16)
	requireDevice(t, err)
	triangle, err := plan.Triangles(0, 1)
	requireDevice(t, err)
	leaf, err := Placed(triangle, translated(0, 0, -5), 3)
	requireDevice(t, err)
	group, err := plan.Instances(leaf)
	requireDevice(t, err)
	clear, err := Placed(group, translated(3, 0, 0), 1)
	requireDevice(t, err)
	opaque, err := Placed(group, translated(-3, 0, 0), 3)
	requireDevice(t, err)
	root, err := plan.Instances(clear, opaque)
	requireDevice(t, err)
	return plan, root
}

// overOpened is a kernel of the probe and a stream of its two inputs.
func overOpened(t *testing.T, rays []byte) (*Kernel, *GridStream) {
	t.Helper()
	kernel, err := Open(overProbe, "probe")
	requireDevice(t, err)
	t.Cleanup(kernel.Close)
	stream, err := kernel.OpenGrid(5, 1, 1, 1,
		[][]byte{rays, triangleData()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(stream.Close)
	return kernel, stream
}

// overProbes are the rays and what each should meet, as the held
// hierarchy test has them.
var overProbes = [][3]float32{
	{3, 1, 6}, {3, 2, 6}, {-3, 2, 6}, {-3, 2, 4}, {0, 1, 6},
}

var overWanted = []float32{1, 0, 1, 0, 0}

// TestABuildOverAnInputAnswersAsACopyDoes builds over the stream's
// triangles and draws: the queries are the ones a copied build answers.
func TestABuildOverAnInputAnswersAsACopyDoes(t *testing.T) {
	kernel, stream := overOpened(t, overRaysFor(overProbes...))
	plan, root := overPlan(t, triangleData())
	sent, err := kernel.AccelerateOver(plan, root, stream, overTriangles)
	if err != nil {
		t.Fatal(err)
	}
	if sent != 0 {
		t.Errorf("the input already held the vertices, yet %d bytes "+
			"were sent", sent)
	}
	got, err := stream.Draw()
	if err != nil || !slices.Equal(got, overWanted) {
		t.Fatalf("queries over the input %v (%v), want %v", got, err,
			overWanted)
	}
}

// TestTheHierarchyDoesNotReadTheInputAfterTheBuild replaces the input
// with zeros of another length after the build, which a hierarchy that
// kept a reference to the old buffer, or read the new one, would show.
func TestTheHierarchyDoesNotReadTheInputAfterTheBuild(t *testing.T) {
	kernel, stream := overOpened(t, overRaysFor(overProbes...))
	plan, root := overPlan(t, triangleData())
	if _, err := kernel.AccelerateOver(plan, root, stream,
		overTriangles); err != nil {
		t.Fatal(err)
	}
	if err := stream.Update(overTriangles, make([]byte, 64)); err != nil {
		t.Fatal(err)
	}
	got, err := stream.Draw()
	if err != nil || !slices.Equal(got, overWanted) {
		t.Fatalf("queries after the input changed %v (%v), want %v", got,
			err, overWanted)
	}
}

// movedTriangle is triangleData moved by x along the x axis.
func movedTriangle(x float32) []byte {
	var moved []byte
	for _, number := range []float32{-1 + x, -1, 0, 0, 1 + x, -1, 0, 0,
		x, 1, 0, 0} {
		moved = binary.LittleEndian.AppendUint32(moved,
			math.Float32bits(number))
	}
	return moved
}

// TestARebuildOverSendsTheMovedVerticesOnce moves the triangle and
// rebuilds: the input comes to hold the new bytes, counted once, and a
// rebuild of the same plan sends nothing.
func TestARebuildOverSendsTheMovedVerticesOnce(t *testing.T) {
	kernel, stream := overOpened(t, overRaysFor(overProbes...))
	plan, root := overPlan(t, triangleData())
	if _, err := kernel.AccelerateOver(plan, root, stream,
		overTriangles); err != nil {
		t.Fatal(err)
	}
	moved, movedRoot := overPlan(t, movedTriangle(0.25))
	sent, err := kernel.RebuildOver(moved, movedRoot, stream, overTriangles)
	if err != nil || sent != int64(len(movedTriangle(0.25))) {
		t.Fatalf("a moved rebuild sent %d bytes (%v), want %d", sent, err,
			len(movedTriangle(0.25)))
	}
	same, err := stream.Same(overTriangles, movedTriangle(0.25))
	if err != nil || !same {
		t.Fatalf("the input does not hold the moved vertices (%v)", err)
	}
	if sent, err = kernel.RebuildOver(moved, movedRoot, stream,
		overTriangles); err != nil || sent != 0 {
		t.Fatalf("an unmoved rebuild sent %d bytes (%v)", sent, err)
	}
	got, err := stream.Draw()
	if err != nil || !slices.Equal(got, overWanted) {
		t.Fatalf("queries after the rebuild %v (%v), want %v", got, err,
			overWanted)
	}
}

// TestABuildOverRefusesWhatItCannotRead is the near misses: an input
// that is not there, a stream another kernel opened, a closed stream and
// no stream. None changes the kernel, which still builds afterward.
func TestABuildOverRefusesWhatItCannotRead(t *testing.T) {
	kernel, stream := overOpened(t, overRaysFor(overProbes...))
	_, strange := overOpened(t, overRaysFor(overProbes...))
	opened, shut := overOpened(t, overRaysFor(overProbes...))
	shut.Close()
	plan, root := overPlan(t, triangleData())
	for name, attempt := range map[string]func() (int64, error){
		"an input below the first": func() (int64, error) {
			return kernel.AccelerateOver(plan, root, stream, -1)
		},
		"an input past the last": func() (int64, error) {
			return kernel.AccelerateOver(plan, root, stream, 2)
		},
		"a stream of another kernel": func() (int64, error) {
			return kernel.AccelerateOver(plan, root, strange, overTriangles)
		},
		"a closed stream": func() (int64, error) {
			return opened.AccelerateOver(plan, root, shut, overTriangles)
		},
		"no stream": func() (int64, error) {
			return kernel.AccelerateOver(plan, root, nil, overTriangles)
		},
	} {
		if _, err := attempt(); err == nil {
			t.Errorf("a build over %s was accepted", name)
		}
	}
	if _, err := kernel.AccelerateOver(plan, root, stream,
		overTriangles); err != nil {
		t.Fatalf("a refused build left the kernel unable to build: %v", err)
	}
}

// TestABuildOverAnEmptyPlanCopiesNothingAndBuilds is an empty plan: the
// input is empty, there is nothing to lend, and the hierarchy is built
// as an empty one is.
func TestABuildOverAnEmptyPlanCopiesNothingAndBuilds(t *testing.T) {
	kernel, stream := overOpened(t, overRaysFor(overProbes...))
	plan, err := NewAcceleration(nil, 16)
	requireDevice(t, err)
	root, err := plan.Instances()
	requireDevice(t, err)
	if _, err := kernel.AccelerateOver(plan, root, stream,
		overTriangles); err != nil {
		t.Fatal(err)
	}
	got, err := stream.Draw()
	if err != nil || !slices.Equal(got, make([]float32, len(overWanted))) {
		t.Fatalf("an empty hierarchy answered %v (%v)", got, err)
	}
}
