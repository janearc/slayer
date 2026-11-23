package mesh

import (
	"unsafe"

	"github.com/janearc/slayer/scene/storage"
)

// Storage counts the geometry retained so far, without copying faces or
// building a tree. The tree is retained only once a ray or a nearest
// point has grown it, so its nodes and bytes count then, and not before.
func (made *Mesh) Storage() storage.Geometry {
	made.treeMu.RLock()
	defer made.treeMu.RUnlock()
	return storage.Geometry{Vertices: len(made.points), Faces: len(made.faces),
		Normals: len(made.normals), Nodes: len(made.tree.nodes),
		Immutable: true,
		Bytes: uint64(unsafe.Sizeof(*made)) + capacityBytes(made.points) +
			capacityBytes(made.rest) + capacityBytes(made.faces) +
			capacityBytes(made.normals) + capacityBytes(made.tree.nodes) +
			capacityBytes(made.tree.order)}
}

// capacityBytes is the bytes a slice holds in all its capacity, used or
// not.
func capacityBytes[Held any](slice []Held) uint64 {
	var one Held
	return uint64(cap(slice)) * uint64(unsafe.Sizeof(one))
}
