// Package storage describes geometry ownership without choosing a renderer
// or estimating the operating system's resident working set.
package storage

// Geometry counts one object's own arrays, excluding referenced prototypes.
// Bytes includes its struct and the capacity of its owned backing arrays;
// it excludes allocator overhead, materials, native resources and temporaries.
type Geometry struct {
	Vertices  int    `json:"vertices"`
	Faces     int    `json:"faces"`
	Normals   int    `json:"normals"`
	Nodes     int    `json:"hierarchy_nodes"`
	Instances int    `json:"instances"`
	Bytes     uint64 `json:"owned_geometry_bytes"`
	Immutable bool   `json:"immutable"`
}
