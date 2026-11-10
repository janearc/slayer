# Geometry preparation without keeping every intermediate

This code draft changes host preparation only. It has not been built,
tested or measured during the current code-only phase. It introduces
no MSL transport change, field ABI change or density threshold.

## Packing

Triangles, smooth normals, hierarchy nodes and instances use a row plan.
It resolves supplied field names against the shared ABI declaration once.
The hot loops then pass fixed arrays instead of allocating and validating
a named-field map for every record.

The declaration remains authoritative for field order and scalar type.
Missing, duplicate or unknown planned fields are code faults. An ABI
field reorder follows its names; an added field requires updating the
writer instead of quietly writing a zero.

Geometry checks run before output reservation: smooth-index capacity,
finite float32 positions/used normals, and Metal's per-input byte limit.
Packed triangle, normal and node buffers reserve their final capacities.
Instance and top-level node buffers reserve their final sizes too.

Native acceleration staging now also allocates its matrix and reference arrays
once at their final counts. Its synchronous bridge call passes the
pointer-free packed vertex bytes directly to Metal's copying buffer
constructor, avoiding a temporary C allocation of the whole vertex payload.

The plan remains alive through completion; the bridge retains Metal-owned
storage, not a Go pointer. The copying and cgo references are linked in
[streamed readback](streamed-readback.md).

The transform arithmetic and float32 conversion remain the original ones.
Packed-byte comparisons against independent ABI fixtures are required at
review; they have not run for these new writers.

Native staging also needs actual empty/populated hierarchy, failure cleanup
and stricter cgo pointer checks before this path is accepted.

## Ownership after preparation

Once serialized, the renderer releases its duplicate float64 triangle values.
Native planning uses node ranges and packed triangle positions. After native
identifiers and the acceleration plan have been built, the renderer also
releases construction-only nodes, groups, standings and lookup maps.

Software traversal needs only the packed buffers.

Source meshes and shared prototypes stay alive through the scene.
Textures needed for CPU finishing and boxed fields needed for baking
also stay alive. The change drops redundant encoder intermediates; it
does not discard geometry needed by a later stage.

Dropping references does not promise immediate collection or lower
process RSS. Go heap management, Metal allocations, staging geometry,
image development and retained frame resources still need a measured
preparation/memory study. No forced global garbage collection is added.

The [streamed readback draft](streamed-readback.md) separately bounds
the raw-sample array during tracing. Together these remove two sources
of duplicate data, but do not establish an eight-hour animation budget.
