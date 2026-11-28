# Model preparation and resource inventory

`catalogue resources [KIND...]` constructs each selected model at every
advertised detail, seed 1, and emits JSON lines without rendering. A failed
representation remains an error row; the command continues and eventually
returns failure. A successful row is not certification that the model is
optimized, physically accurate or supported by every native path.

The audit never substitutes a cheaper representation. Compare the same kind,
seed, detail, geometry, materials and animation when measuring improvements.

## Inspect an actual scene

to count the geometry in the first scene frame without drawing:

```sh
game build slayer
game run slayer -- assess sets/webb-at-l2.yaml --census out/webb-census.json
```

the census counts all placed geometry, including objects outside the
camera view. for more frame selections, the ordinary `game build` also
builds `scene-complexity`. see the
[scene reporter guide](../cmd/scene-complexity/README.md) for its job and
frame options.

For the set, scene and direct Go authoring surfaces, see the
[scene constructor reference](scene-constructor-reference.md).

## Owned storage and placements

`mesh.Mesh.Storage`, `shape.Mesh.Storage` and `shape.MeshGroup.Storage` count
their own structures and backing-array capacities without copying geometry.
`thing.Storage` counts each shared mesh/group once across the entire input.
Referenced faces count every placement; stored faces count each prototype.
The two counts must not be conflated, nor added as memory costs.

Owned geometry bytes exclude allocator overhead, construction forms,
materials, media, packed GPU inputs, native acceleration structures, scratch,
readback and frame buffers. Unknown shape implementations remain named in the
report. The allocation counter is cumulative Go allocation during construction
and preparation, not simultaneous memory usage or native allocation.

Geometry payload is therefore a lower-level accounting fact, not a safe job
reservation. Use measured process footprint and native allocation/scratch peaks
for admission, including the overlap of cold preparation and retained geometry.

## Prepare once

`made.Prepare()` creates the same local meshes that `made.Things` creates,
retains light descriptions, and releases the need to retain construction forms.
The returned preparation has `Things(at)`, `Lights(at)` and `Resources()`.
Repeated placements allocate new scene objects while sharing the original mesh
prototypes. Material sharing matches the original API. Nothing may mutate
shared geometry or materials during a frame.

Resources explicitly mark shared materials and the need to snapshot changing
frame state. Independent placement is not a claim that mutating the shared
materials while another frame is in flight is safe. Multi-frame pipelines must
separate immutable geometry from each frame's motion, material and light state.

Resources state whether local geometry is immutable and independently placeable.
They always state that native measurement is still needed. They do not silently
freeze time-varying materials, assert cache hits, or provide a made-up native
memory estimate. Scene animation still determines which placements, surfaces,
deformed vertices and topology change.

The caller must retain the preparation if reuse is intended. This API does not
install a process-global unbounded cache or select an eviction policy.

## Persistent caches and job probes

Slayer already keeps CPU face trees in bbolt; see [caches](caches.md).
Small meshes below the current disk threshold can still reuse immutable local
trees in a retained frame session. A model's geometry being cacheable does not
mean a particular render actually enabled that cache.

Persistent entries need content, recipe/seed/detail and format/algorithm keys
appropriate to their payload. Material/volume changes, lens/raster/quality,
device and build must also invalidate a workload profile where they affect it.
Do not cache opaque process handles as if they were portable geometry data.
Cache decoding and CPU reconstruction belong in cold/warm timing separately.

At job start, representative scene-boundary, active-motion and high-complexity
frames should measure construction, encoding, native build/update, tracing,
finish and simultaneous working set. A quiet hold is not a motion benchmark.
A small raster still constructs the full geometry; an extrapolated pixel cost
does not predict acceleration time or peak memory. Reuse sound profile records,
observe subsequent frames, and update a prediction when the scene exceeds it.

Admission must account for retained shared state plus incremental frame and
transient work. Several frames within one process can share geometry; separate
processes each own a copy. Low GPU utilization alone does not establish room
for another copy. Growing swap and live pressure must be considered before
adding work. Do not hide pressure by reducing model complexity.
