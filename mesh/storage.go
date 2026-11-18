package mesh

import (
	"github.com/janearc/slayer/scene/storage"
	"unsafe"
)

// Storage counts retained geometry without copying faces or building a tree.
func (made *Mesh) Storage() storage.Geometry {
	return storage.Geometry{Vertices: len(made.points), Faces: len(made.faces),
		Normals: len(made.normals), Nodes: len(made.tree.nodes), Immutable: true,
		Bytes: uint64(unsafe.Sizeof(*made)) +
			uint64(cap(made.points))*uint64(unsafe.Sizeof(made.points[0])) +
			uint64(cap(made.faces))*uint64(unsafe.Sizeof(face{})) +
			uint64(cap(made.normals))*uint64(unsafe.Sizeof(made.points[0])) +
			uint64(cap(made.tree.nodes))*uint64(unsafe.Sizeof(node{})) +
			uint64(cap(made.tree.order))*uint64(unsafe.Sizeof(int(0)))}
}
