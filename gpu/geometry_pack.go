package gpu

import (
	"fmt"
	"math"
)

var triangleRows = planRows(triangleStruct, "corner_a", "corner_b", "corner_c")
var normalRows = planRows(bendStruct, "normal_a", "normal_b", "normal_c")
var nodeRows = planRows(nodeStruct, "low", "high", "info")

// geometryCapacity checks all packed-array sizes before reserving any of them.
func (encoded *scene) geometryCapacity(grown tree, bends int) error {
	for _, request := range []struct {
		buffer *packer
		count  int
		stride int
	}{
		{&encoded.triangles, len(grown.triangles), triangleStruct.size()},
		{&encoded.bends, bends, bendStruct.size()},
		{&encoded.nodes, len(grown.nodes), nodeStruct.size()},
	} {
		if len(request.buffer.bytes) > math.MaxInt32 ||
			request.count > (math.MaxInt32-len(request.buffer.bytes))/request.stride {
			return fmt.Errorf("gpu: packed geometry exceeds the Metal input byte limit")
		}
	}
	return nil
}

// reserveGeometry allocates each final payload once instead of growing it by
// row. geometryCapacity has already checked the products and combined byte
// lengths.
func (encoded *scene) reserveGeometry(grown tree, bends int) {
	encoded.triangles.reserveWords(len(grown.triangles) * triangleStruct.size())
	encoded.bends.reserveWords(bends * bendStruct.size())
	encoded.nodes.reserveWords(len(grown.nodes) * nodeStruct.size())
}

// reserveWords preserves existing contents and reserves exact additional bytes.
func (buffer *packer) reserveWords(additional int) {
	wanted := len(buffer.bytes) + additional
	if cap(buffer.bytes) >= wanted {
		return
	}
	kept := make([]byte, len(buffer.bytes), wanted)
	copy(kept, buffer.bytes)
	buffer.bytes = kept
}

// packedFace retains the existing triangle and optional smooth-normal layout.
func (encoded *scene) packedFace(face triangle, bend int) {
	if face.smooth {
		normalRows.write(&encoded.bends, vectorRow(face.normals[0], 0),
			vectorRow(face.normals[1], 0), vectorRow(face.normals[2], 0))
	}
	triangleRows.write(&encoded.triangles, vectorRow(face.corners[0], float64(bend)),
		vectorRow(face.corners[1], 0), vectorRow(face.corners[2], 0))
}

// packedNode keeps named ABI resolution outside the per-node packing loop.
func (encoded *scene) packedNode(branch node) {
	nodeRows.write(&encoded.nodes, vectorRow(branch.bounds.low, 0),
		vectorRow(branch.bounds.high, 0), [4]float64{float64(branch.first), float64(branch.count)})
}
