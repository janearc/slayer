//go:build darwin

package metal

import (
	"encoding/binary"
	"math"
	"slices"
	"testing"
)

// A real native query, with a mask and finite segment carried by each ray.
const acceleratedProbe = `
#include <metal_stdlib>
#include <metal_raytracing>
using namespace metal;
using namespace metal::raytracing;
kernel void probe(device const float4 *rays [[buffer(0)]],
                  device float *out [[buffer(1)]],
                  constant uint2 &origin [[buffer(2)]],
                  acceleration_structure<instancing> scene [[buffer(3)]],
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

// triangleData is one triangle in z=0, in padded float4 vertex records.
func triangleData() []byte {
	var data []byte
	for _, number := range []float32{-1, -1, 0, 0, 1, -1, 0, 0, 0, 1, 0, 0} {
		data = binary.LittleEndian.AppendUint32(data, math.Float32bits(number))
	}
	return data
}

// translated is an affine matrix with no rotation or scaling.
func translated(x, y, z float32) [12]float32 {
	return [12]float32{1, 0, 0, 0, 1, 0, 0, 0, 1, x, y, z}
}

func TestNativeHierarchyKeepsTransformsMasksAndSegments(t *testing.T) {
	plan, err := NewAcceleration(triangleData(), 16)
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
	kernel, err := Open(acceleratedProbe, "probe")
	requireDevice(t, err)
	defer kernel.Close()
	if err := kernel.Accelerate(plan, root); err != nil {
		t.Fatal(err)
	}
	if kernel.GPUTime() <= 0 {
		t.Fatal("native construction did not report its GPU cost")
	}
	var rays []byte
	for _, probe := range [][3]float32{
		{3, 1, 6}, {3, 2, 6}, {-3, 2, 6}, {-3, 2, 4}, {0, 1, 6},
	} {
		for _, value := range []float32{probe[0], 0, 0, probe[1], 0, 0, -1, probe[2]} {
			rays = binary.LittleEndian.AppendUint32(rays, math.Float32bits(value))
		}
	}
	want := []float32{1, 0, 1, 0, 0}
	for range 2 {
		got, err := kernel.Grid(len(want), 1, 1, 1, [][]byte{rays})
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got, want) {
			t.Fatalf("native queries %v, want %v", got, want)
		}
	}
	if err := kernel.Accelerate(plan, root); err == nil {
		t.Fatal("a second build replaced the retained hierarchy")
	}
	got, err := kernel.Grid(len(want), 1, 1, 1, [][]byte{rays})
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("refused rebuild changed the hierarchy: %v, %v", got, err)
	}
	kernel.Close()
	if err := kernel.Accelerate(plan, root); err == nil {
		t.Fatal("a closed kernel accepted a hierarchy")
	}
}

func TestNativePlanRefusesInvalidGeometryAndReferences(t *testing.T) {
	for _, stride := range []int{-1, 0, 8, 13} {
		if _, err := NewAcceleration(triangleData(), stride); err == nil {
			t.Fatalf("accepted vertex stride %d", stride)
		}
	}
	plan, err := NewAcceleration(triangleData(), 16)
	requireDevice(t, err)
	for _, pair := range [][2]int{{-1, 1}, {0, -1}, {1, 1}, {0, 2}, {math.MaxInt, 1}} {
		if _, err := plan.Triangles(pair[0], pair[1]); err == nil {
			t.Fatalf("accepted triangle range %v", pair)
		}
	}
	missing, err := Placed(0, translated(0, 0, 0), 3)
	requireDevice(t, err)
	if _, err := plan.Instances(missing); err == nil {
		t.Fatal("accepted a missing or cyclic child")
	}
	if _, err := Placed(-1, translated(0, 0, 0), 3); err == nil {
		t.Fatal("accepted a negative child")
	}
	for _, matrix := range [][12]float32{{}, translated(float32(math.NaN()), 0, 0), translated(0, float32(math.Inf(1)), 0)} {
		if _, err := Placed(0, matrix, 3); err == nil {
			t.Fatal("accepted a singular or nonfinite matrix")
		}
	}
}

func TestEmptyNativeHierarchyDoesNotInventAnOccluder(t *testing.T) {
	plan, err := NewAcceleration(nil, 16)
	requireDevice(t, err)
	root, err := plan.Instances()
	requireDevice(t, err)
	kernel, err := Open(acceleratedProbe, "probe")
	requireDevice(t, err)
	defer kernel.Close()
	if err := kernel.Accelerate(plan, root); err != nil {
		t.Fatal(err)
	}
	var ray []byte
	for _, value := range []float32{0, 0, 0, 3, 0, 0, -1, 100} {
		ray = binary.LittleEndian.AppendUint32(ray, math.Float32bits(value))
	}
	got, err := kernel.Grid(1, 1, 1, 1, [][]byte{ray})
	if err != nil || len(got) != 1 || got[0] != 0 {
		t.Fatalf("empty hierarchy invented a hit: %v, %v", got, err)
	}
}

func TestNativeBuilderRefusesBadRootsBeforeChangingTheKernel(t *testing.T) {
	plan, err := NewAcceleration(triangleData(), 16)
	requireDevice(t, err)
	triangle, err := plan.Triangles(0, 1)
	requireDevice(t, err)
	kernel, err := Open(acceleratedProbe, "probe")
	requireDevice(t, err)
	defer kernel.Close()
	for _, root := range []int{-1, triangle, math.MaxInt} {
		if err := kernel.Accelerate(plan, root); err == nil {
			t.Fatalf("accepted invalid instance root %d", root)
		}
		if kernel.GPUTime() != 0 {
			t.Fatal("an invalid root reached the GPU")
		}
	}
	root, err := plan.Instances()
	requireDevice(t, err)
	if err := kernel.Accelerate(plan, root); err != nil {
		t.Fatalf("a refused root damaged the kernel: %v", err)
	}
}
