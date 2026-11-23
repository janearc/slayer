package gpu

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

type pageSize struct{ triangles, bends, nodes int }

var triangleRows = planRows(triangleStruct, "corner_a", "corner_b", "corner_c")
var normalRows = planRows(bendStruct, "normal_a", "normal_b", "normal_c")
var nodeRows = planRows(nodeStruct, "low", "high", "info")

// geometryCapacity checks all packed-array sizes before reserving any of them.
func (encoded *scene) geometryCapacity(faces, bends, nodes int) error {
	for _, request := range []struct {
		buffer *packer
		count  int
		stride int
	}{
		{&encoded.triangles, faces, triangleStruct.size()},
		{&encoded.bends, bends, bendStruct.size()},
		{&encoded.nodes, nodes, nodeStruct.size()},
	} {
		if len(request.buffer.bytes) > math.MaxInt32 ||
			request.count > (math.MaxInt32-len(request.buffer.bytes))/request.stride {
			return fmt.Errorf("gpu: packed geometry exceeds the Metal input byte limit")
		}
	}
	return nil
}

// packPending packs the trees in line, one after another, into the
// scene's triangles, bends and boxes, each tree where it lies: a skin's
// faces and boxes are read from its mesh and no float64 copy of them is
// made. It notes where each tree's faces went, by its root, for the
// native plan. The checks all come before anything is reserved or
// written, as packWithin's do.
func (encoded *scene) packPending(waiting []pendingMesh) error {
	var total pageSize
	for _, each := range waiting {
		total.triangles += each.tree.faceCount()
		total.bends += each.tree.bendCount()
		total.nodes += each.tree.nodeCount()
	}
	if total.bends > mostBends {
		return unsupported{fmt.Sprintf("more than %d faces shaded "+
			"smooth", mostBends)}
	}
	for _, each := range waiting {
		if !each.tree.finite() {
			return unsupported{"a mesh face outside finite float32 " +
				"coordinates"}
		}
	}
	if err := encoded.geometryCapacity(total.triangles, total.bends,
		total.nodes); err != nil {
		return err
	}
	encoded.reserveGeometry(total.triangles, total.bends, total.nodes)
	encoded.runs = make(map[int]faceRun, len(waiting))
	var place pageSize
	for _, each := range waiting {
		encoded.packTree(each.tree, place)
		encoded.runs[place.nodes] = faceRun{first: place.triangles,
			count: each.tree.faceCount()}
		place.triangles += each.tree.faceCount()
		place.bends += each.tree.bendCount()
		place.nodes += each.tree.nodeCount()
	}
	encoded.counts.triangles = total.triangles
	encoded.counts.nodes = total.nodes
	return nil
}

// reserveGeometry allocates each final payload once instead of growing it by
// row. geometryCapacity has already checked the products and combined byte
// lengths.
func (encoded *scene) reserveGeometry(faces, bends, nodes int) {
	encoded.triangles.reserveWords(faces * triangleStruct.size())
	encoded.bends.reserveWords(bends * bendStruct.size())
	encoded.nodes.reserveWords(nodes * nodeStruct.size())
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
		encoded.packedBend(face.normals, face.unlifted)
	}
	encoded.packedCorners(face.corners, bend)
}

// packedBend packs a smooth face's three corner normals. the first
// normal's spare number says whether the kernel is to leave the point
// unlifted: nought, as it always was, lifts.
func (encoded *scene) packedBend(normals [3]maths.Vec, unlifted bool) {
	normalRows.write(&encoded.bends,
		vectorRow(normals[0], float64(flag(unlifted))),
		vectorRow(normals[1], 0), vectorRow(normals[2], 0))
}

// packedCorners packs a face's three corners with the place of its bend,
// or -1 for a face shaded flat, in the first corner's fourth number.
func (encoded *scene) packedCorners(corners [3]maths.Vec, bend int) {
	triangleRows.write(&encoded.triangles, vectorRow(corners[0], bendNumber(bend)),
		vectorRow(corners[1], 0), vectorRow(corners[2], 0))
}

// packedNode keeps named ABI resolution outside the per-node packing loop.
func (encoded *scene) packedNode(branch node) {
	encoded.packedBox(branch.bounds.low, branch.bounds.high, branch.first,
		branch.count)
}

// packedBox packs one box of a tree: its corners, and its first and
// count.
func (encoded *scene) packedBox(low, high maths.Vec, first, count int) {
	nodeRows.write(&encoded.nodes, vectorRow(low, 0), vectorRow(high, 0),
		[4]float64{float64(first), float64(count)})
}
