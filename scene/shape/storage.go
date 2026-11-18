package shape

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/storage"
	"unsafe"
)

// Storage counts a reference mesh's arrays. Its public fields can change,
// so identity alone cannot establish that its geometry is unchanged.
func (made *Mesh) Storage() storage.Geometry {
	return storage.Geometry{Vertices: len(made.Vertices), Faces: len(made.Faces),
		Normals: len(made.Normals), Nodes: len(made.nodes),
		Bytes: uint64(unsafe.Sizeof(*made)) +
			uint64(cap(made.Vertices)+cap(made.Normals))*uint64(unsafe.Sizeof(maths.Vec{})) +
			uint64(cap(made.Faces))*uint64(unsafe.Sizeof([3]int{})) +
			uint64(cap(made.tris))*uint64(unsafe.Sizeof(tri{})) +
			uint64(cap(made.nodes))*uint64(unsafe.Sizeof(node{}))}
}

// Storage counts only a group's own instance arrays and local hierarchy.
// Its member meshes must be counted separately, once per unique prototype.
func (group *MeshGroup) Storage() storage.Geometry {
	return storage.Geometry{Instances: len(group.members), Nodes: len(group.nodes),
		Immutable: true, Bytes: uint64(unsafe.Sizeof(*group)) +
			uint64(cap(group.members))*uint64(unsafe.Sizeof(MeshMember{})) +
			uint64(cap(group.boxes))*uint64(unsafe.Sizeof(maths.Box{})) +
			uint64(cap(group.nodes))*uint64(unsafe.Sizeof(groupNode{}))}
}
