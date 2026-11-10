//go:build darwin

package metal

/*
#include <stdlib.h>
#include "bridge.h"
*/
import "C"

import (
	"encoding/binary"
	"errors"
	"math"
	"runtime"
	"unsafe"
)

// ErrNoRayTracing means this device cannot build the native hierarchy.
var ErrNoRayTracing = errors.New("metal: native ray tracing is unavailable")

// Acceleration describes an acyclic hierarchy of nonindexed triangles.
// Vertices are borrowed and must stay unchanged until Accelerate returns.
// Each vertex begins with a little-endian float3 at the supplied byte stride.
// Apple's WWDC2023/10128 describes the nested instance representation.
type Acceleration struct {
	vertices  []byte
	stride    int
	nodes     []accelerationNode
	instances []Placement
}

type accelerationNode struct{ kind, first, count int }

// Placement is a child structure's transform and visibility mask.
type Placement struct {
	child  int
	matrix [12]float32
	mask   uint32
}

// NewAcceleration starts a plan with nonindexed, finite triangle positions.
func NewAcceleration(vertices []byte, stride int) (*Acceleration, error) {
	if stride < 12 || stride > math.MaxInt32 || stride%4 != 0 || len(vertices)%stride != 0 ||
		(len(vertices)/stride)%3 != 0 || len(vertices) > math.MaxInt32 {
		return nil, errors.New("metal: invalid triangle vertex layout")
	}
	for at := 0; at < len(vertices); at += stride {
		for axis := range 3 {
			word := binary.LittleEndian.Uint32(vertices[at+axis*4:])
			if !finiteFloat(math.Float32frombits(word)) {
				return nil, errors.New("metal: nonfinite triangle position")
			}
		}
	}
	return &Acceleration{vertices: vertices, stride: stride}, nil
}

// Triangles adds a checked triangle range and returns its node index.
func (plan *Acceleration) Triangles(first, count int) (int, error) {
	if plan == nil || plan.stride < 12 {
		return 0, errors.New("metal: no acceleration plan")
	}
	total := len(plan.vertices) / plan.stride / 3
	if first < 0 || count < 0 || first > total || count > total-first ||
		len(plan.nodes) == math.MaxInt32 {
		return 0, errors.New("metal: triangle range outside the vertices")
	}
	index := len(plan.nodes)
	plan.nodes = append(plan.nodes, accelerationNode{0, first, count})
	return index, nil
}

// Placed describes a child with an invertible column-major 4 by 3 matrix.
func Placed(child int, matrix [12]float32, mask uint32) (Placement, error) {
	if child < 0 || !validMatrix(matrix) {
		return Placement{}, errors.New("metal: invalid instance transform or child")
	}
	return Placement{child: child, matrix: matrix, mask: mask}, nil
}

// finiteFloat excludes values the native hierarchy cannot bound.
func finiteFloat(value float32) bool {
	return !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0)
}

// validMatrix checks finiteness and the determinant without a size threshold.
func validMatrix(matrix [12]float32) bool {
	var values [12]float64
	for i, value := range matrix {
		if !finiteFloat(value) {
			return false
		}
		values[i] = float64(value)
	}
	m := values
	determinant := m[0]*(m[4]*m[8]-m[7]*m[5]) -
		m[3]*(m[1]*m[8]-m[7]*m[2]) + m[6]*(m[1]*m[5]-m[4]*m[2])
	return determinant != 0
}

// Instances adds placements referring only to earlier nodes, ruling out cycles.
func (plan *Acceleration) Instances(places ...Placement) (int, error) {
	if plan == nil || plan.stride < 12 || len(plan.nodes) == math.MaxInt32 {
		return 0, errors.New("metal: no acceleration plan or too many nodes")
	}
	if !instancePlacementCount(len(plan.instances), len(places)) {
		return 0, errors.New("metal: too many instance placements")
	}
	for _, place := range places {
		if place.child < 0 || place.child >= len(plan.nodes) ||
			!validMatrix(place.matrix) {
			return 0, errors.New("metal: instance needs an earlier child and valid transform")
		}
	}
	index := len(plan.nodes)
	plan.nodes = append(plan.nodes, accelerationNode{1, len(plan.instances), len(places)})
	plan.instances = append(plan.instances, places...)
	return index, nil
}

// Accelerate builds and retains an instance root. Grid binds it after the
// origin, at buffer index len(inputs)+2. Close releases every structure.
// A second build is refused without changing the first; calls are serial.
func (kernel *Kernel) Accelerate(plan *Acceleration, root int) error {
	if kernel.ctx == nil {
		return errors.New("metal: the kernel is closed")
	}
	if plan == nil || root < 0 || root >= len(plan.nodes) ||
		plan.nodes[root].kind != 1 {
		return errors.New("metal: acceleration needs an instance root")
	}
	if !accelerationCounts(len(plan.nodes), len(plan.instances)) {
		return errors.New("metal: acceleration staging exceeds the host or signed native range")
	}
	nodes := make([]int32, 0, len(plan.nodes)*3)
	for _, node := range plan.nodes {
		if !accelerationRange(node.first, node.count) {
			return errors.New("metal: acceleration node exceeds the signed native range")
		}
		nodes = append(nodes, int32(node.kind), int32(node.first), int32(node.count))
	}
	matrices := make([]float32, len(plan.instances)*12)
	references := make([]uint32, len(plan.instances)*2)
	for index, place := range plan.instances {
		copy(matrices[index*12:], place.matrix[:])
		references[index*2], references[index*2+1] = uint32(place.child), place.mask
	}
	// The bridge copies positions into a Metal-owned buffer before use.
	// The pointer-free Go bytes remain pinned for this synchronous C call;
	// runtime.KeepAlive below also retains their owner through completion.
	vertices := unsafe.Pointer(unsafe.SliceData(plan.vertices))
	var why [1024]C.char
	var unavailable C.int
	failed := C.slayer_metal_accelerate(kernel.ctx, vertices,
		C.long(len(plan.vertices)), C.int(plan.stride),
		(*C.int32_t)(unsafe.Pointer(unsafe.SliceData(nodes))), C.int(len(plan.nodes)),
		(*C.float)(unsafe.Pointer(unsafe.SliceData(matrices))),
		(*C.uint32_t)(unsafe.Pointer(unsafe.SliceData(references))), C.int(root),
		&unavailable, &why[0], C.int(len(why)))
	runtime.KeepAlive(plan)
	runtime.KeepAlive(kernel)
	if unavailable != 0 {
		return ErrNoRayTracing
	}
	if failed != 0 {
		return errors.New("metal: " + C.GoString(&why[0]))
	}
	return nil
}

// accelerationCounts checks the C int counts as well as Go staging products.
func accelerationCounts(nodes, instances int) bool {
	return nodes >= 0 && nodes <= math.MaxInt32 && nodes <= math.MaxInt/3 &&
		instances >= 0 && instances <= math.MaxInt32 && instances <= math.MaxInt/12
}

// accelerationRange checks each word before narrowing a node to int32.
func accelerationRange(first, count int) bool {
	return first >= 0 && first <= math.MaxInt32 && count >= 0 && count <= math.MaxInt32
}

// instancePlacementCount checks the signed 32-bit native placement budget.
// Both arguments are slice lengths, so they are nonnegative.
func instancePlacementCount(held, added int) bool {
	return added <= math.MaxInt32-held
}
