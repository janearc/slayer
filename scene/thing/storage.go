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
//
// OwnedGeometryBytes counts each mesh and group once, a sphere's value, and
// a height field's samples with the shading slopes it carries. A shape it
// does not know is named in Unknown and adds no bytes.
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

// The arrays a height field owns, counted one float64 to a sample.
const (
	// sampleArrays is defined: the one array of samples every height
	// field has.
	sampleArrays = 1
	// slopeArrays is defined: the two arrays of vertex slopes, along x and
	// along z, that a height field shaded by approximate normals carries
	// as well.
	slopeArrays = 2
)

// storageSample is one sample of a height field, which is a float64.
var storageSample float64

// storageSampleBytes is defined: the bytes of one sample, a float64.
const storageSampleBytes = unsafe.Sizeof(storageSample)

// ledger is an inventory being drawn up: the totals so far, the prototypes
// already counted, and the kinds of shape it cannot count.
type ledger struct {
	totals  GeometryStorage
	counted map[any]bool
	unknown map[string]bool
}

// Storage inventories shared geometry by identity. Repeated placements count
// their faces as references, while storage counts each prototype only once.
func Storage(things []Thing) GeometryStorage {
	books := &ledger{totals: GeometryStorage{Things: len(things)},
		counted: map[any]bool{}, unknown: map[string]bool{}}
	for _, part := range things {
		books.count(part.Shape)
	}
	for name := range books.unknown {
		books.totals.Unknown = append(books.totals.Unknown, name)
	}
	sort.Strings(books.totals.Unknown)
	return books.totals
}

// first says whether a prototype has not been counted yet, and marks it
// counted.
func (books *ledger) first(key any) bool {
	if books.counted[key] {
		return false
	}
	books.counted[key] = true
	return true
}

// add takes one prototype's own arrays into the totals, once. A group is
// counted as a group and anything else as a mesh.
func (books *ledger) add(key any, own storage.Geometry, group bool) {
	if !books.first(key) {
		return
	}
	books.totals.OwnedGeometryBytes += own.Bytes
	books.totals.Vertices += own.Vertices
	books.totals.HierarchyNodes += own.Nodes
	books.totals.StoredFaces += own.Faces
	books.totals.Instances += own.Instances
	if group {
		books.totals.UniqueGroups++
		return
	}
	books.totals.UniqueMeshes++
	if !own.Immutable {
		books.totals.MutableMeshes++
	}
}

// count takes one placed shape into the totals by what it is.
func (books *ledger) count(solid shape.Shape) {
	switch solid := solid.(type) {
	case *mesh.Mesh:
		books.countMesh(solid, solid.Storage())
	case *shape.Mesh:
		books.countMesh(solid, solid.Storage())
	case *shape.MeshGroup:
		books.countGroup(solid)
	case *shape.HeightField:
		if books.first(solid) {
			books.totals.OwnedGeometryBytes += heightFieldBytes(solid)
		}
	case shape.Sphere:
		// Analytic values live in the enclosing thing's interface.
		books.totals.OwnedGeometryBytes += uint64(unsafe.Sizeof(solid))
	case shape.Plane:
	default:
		books.unknown[fmt.Sprintf("%T", solid)] = true
	}
}

// countMesh takes a placed mesh into the totals: its faces as referenced at
// this placement, and its own arrays once.
func (books *ledger) countMesh(key any, own storage.Geometry) {
	books.totals.ReferencedFaces += own.Faces
	books.add(key, own, false)
}

// countGroup takes a mesh group and each distinct mesh it instances into
// the totals. Its faces count as referenced at every placement.
func (books *ledger) countGroup(group *shape.MeshGroup) {
	books.totals.ReferencedFaces += group.FaceCount()
	if books.counted[group] {
		return
	}
	books.add(group, group.Storage(), true)
	for index := range group.MemberCount() {
		switch skin := group.Member(index).Mesh.(type) {
		case *mesh.Mesh:
			books.add(skin, skin.Storage(), false)
		case *shape.Mesh:
			books.add(skin, skin.Storage(), false)
		}
	}
}

// heightFieldBytes is the bytes of the arrays a height field owns that
// Pack and its shading record show: its samples, and two more arrays of as
// many when it carries vertex slopes for shading.
func heightFieldBytes(field *shape.HeightField) uint64 {
	heights, _, _, _, _, _ := field.Pack()
	arrays := sampleArrays
	if field.ApproximateShadingNormalMode() != "" {
		arrays += slopeArrays
	}
	return uint64(len(heights)) * uint64(arrays) *
		uint64(storageSampleBytes)
}
