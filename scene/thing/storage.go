package thing

import (
	"fmt"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/storage"
	"sort"
	"unsafe"
)

// GeometryStorage is an inventory of the current geometry, not a peak-memory
// prediction. Native acceleration/scratch, packing, materials and media need
// their own measured accounting before a scheduler can reserve a working set.
type GeometryStorage struct {
	Things             int      `json:"things"`
	UniqueMeshes       int      `json:"unique_meshes"`
	MutableMeshes      int      `json:"mutable_meshes"`
	UniqueGroups       int      `json:"unique_groups"`
	Instances          int      `json:"local_instances"`
	StoredFaces        int      `json:"stored_faces"`
	ReferencedFaces    int      `json:"referenced_faces"`
	Vertices           int      `json:"vertices"`
	HierarchyNodes     int      `json:"hierarchy_nodes"`
	OwnedGeometryBytes uint64   `json:"owned_geometry_bytes"`
	Unknown            []string `json:"unaccounted_shape_types,omitempty"`
}

// Storage inventories shared geometry by identity. Repeated placements count
// their faces as references, while storage counts each prototype only once.
func Storage(things []Thing) GeometryStorage {
	one := GeometryStorage{Things: len(things)}
	seen := map[any]bool{}
	unknown := map[string]bool{}
	add := func(key any, own storage.Geometry, group bool) {
		if seen[key] {
			return
		}
		seen[key] = true
		one.OwnedGeometryBytes += own.Bytes
		one.Vertices += own.Vertices
		one.HierarchyNodes += own.Nodes
		one.StoredFaces += own.Faces
		one.Instances += own.Instances
		if group {
			one.UniqueGroups++
		} else {
			one.UniqueMeshes++
			if !own.Immutable {
				one.MutableMeshes++
			}
		}
	}
	for _, part := range things {
		switch solid := part.Shape.(type) {
		case *mesh.Mesh:
			own := solid.Storage()
			one.ReferencedFaces += own.Faces
			add(solid, own, false)
		case *shape.Mesh:
			own := solid.Storage()
			one.ReferencedFaces += own.Faces
			add(solid, own, false)
		case *shape.MeshGroup:
			one.ReferencedFaces += solid.FaceCount()
			if seen[solid] {
				continue
			}
			add(solid, solid.Storage(), true)
			for index := range solid.MemberCount() {
				member := solid.Member(index)
				switch skin := member.Mesh.(type) {
				case *mesh.Mesh:
					add(skin, skin.Storage(), false)
				case *shape.Mesh:
					add(skin, skin.Storage(), false)
				}
			}
		case shape.Sphere:
			// Analytic values live in the enclosing thing's interface.
			one.OwnedGeometryBytes += uint64(unsafe.Sizeof(solid))
		case shape.Plane:
		default:
			unknown[fmt.Sprintf("%T", solid)] = true
		}
	}
	for name := range unknown {
		one.Unknown = append(one.Unknown, name)
	}
	sort.Strings(one.Unknown)
	return one
}
